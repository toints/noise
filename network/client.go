package network

import (
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/network/rpc"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"github.com/pkg/errors"
	"sync"
	"sync/atomic"
)

// PeerClient represents a single incoming peers client.
type PeerClient struct {
	Network *Network

	ID      *peer.ID
	Address string

	Requests     *sync.Map
	RequestNonce uint64

	stream StreamState
}

type StreamState struct {
	sync.Mutex
	buffer   []byte
	buffered chan struct{}
	closed   bool
}

// createPeerClient creates a stub peer client.
func createPeerClient(network *Network, address string) *PeerClient {
	client := &PeerClient{
		Network:      network,
		Address:      address,
		Requests:     new(sync.Map),
		RequestNonce: 0,
		stream: StreamState{
			buffer:   make([]byte, 0),
			buffered: make(chan struct{}),
		},
	}

	client.Network.Plugins.Each(func(plugin PluginInterface) {
		plugin.PeerConnect(client)
	})

	return client
}

// Close stops all sessions/streams and cleans up the nodes
// routing table. Errors if session fails to close.
func (c *PeerClient) Close() error {
	c.stream.Lock()
	c.stream.closed = true
	c.stream.Unlock()

	// Handle 'on peer disconnect' callback for plugins.
	c.Network.Plugins.Each(func(plugin PluginInterface) {
		plugin.PeerDisconnect(c)
	})

	// Remove entries from node's network.
	if c.ID != nil {
		c.Network.Peers.Delete(c.ID.Address)
		c.Network.Connections.Delete(c.ID.Address)
	}

	return nil
}

// Write asynchronously emit a message to a given peer.
func (c *PeerClient) Tell(message proto.Message) error {
	signed, err := c.Network.PrepareMessage(message)
	if err != nil {
		return errors.Wrap(err, "failed to sign message")
	}

	err = c.Network.Write(c.Address, signed)
	if err != nil {
		return errors.Wrapf(err, "failed to send message to %s", c.Address)
	}

	return nil
}

// Request requests for a response for a request sent to a given peer.
func (c *PeerClient) Request(req *rpc.Request) (proto.Message, error) {
	signed, err := c.Network.PrepareMessage(req.Message)
	if err != nil {
		return nil, err
	}

	signed.Nonce = atomic.AddUint64(&c.RequestNonce, 1)

	err = c.Network.Write(c.Address, signed)
	if err != nil {
		return nil, err
	}

	// Start tracking the request.
	channel := make(chan proto.Message, 1)
	c.Requests.Store(signed.Nonce, channel)

	// Stop tracking the request.
	defer close(channel)
	defer c.Requests.Delete(signed.Nonce)

	select {
	case res := <-channel:
		return res, nil
	case <-time.After(req.Timeout):
	}

	return nil, errors.New("request timed out")
}

// Reply is equivalent to Write() with an appended nonce to signal a reply.
func (c *PeerClient) Reply(nonce uint64, message proto.Message) error {
	signed, err := c.Network.PrepareMessage(message)
	if err != nil {
		return err
	}

	// Set the nonce.
	signed.Nonce = nonce

	err = c.Network.Write(c.Address, signed)
	if err != nil {
		return err
	}

	return nil
}

func (c *PeerClient) handleStreamPacket(pkt []byte) {
	c.stream.Lock()
	empty := len(c.stream.buffer) == 0
	c.stream.buffer = append(c.stream.buffer, pkt...)
	c.stream.Unlock()

	if empty {
		select {
		case c.stream.buffered <- struct{}{}:
		default:
		}
	}
}

// Read implement net.Conn by reading packets of bytes over a stream.
func (c *PeerClient) Read(out []byte) (int, error) {
	for {
		c.stream.Lock()
		closed := c.stream.closed
		n := copy(out, c.stream.buffer)
		c.stream.buffer = c.stream.buffer[n:]
		c.stream.Unlock()

		if closed {
			return n, errors.New("closed")
		}

		if n == 0 {
			select {
			case <-c.stream.buffered:
			case <-time.After(1 * time.Second):
			}
		} else {
			return n, nil
		}
	}
}

// Write implements net.Conn and sends packets of bytes over a stream.
func (c *PeerClient) Write(data []byte) (int, error) {
	err := c.Tell(&protobuf.StreamPacket{Data: data})
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

type NoiseAddr struct {
	Address string
}

func (a *NoiseAddr) Network() string {
	return "noise"
}

func (a *NoiseAddr) String() string {
	return a.Address
}

// LocalAddr implements net.Conn.
func (c *PeerClient) LocalAddr() net.Addr {
	return &NoiseAddr{Address: "[local]"}
}

// RemoteAddr implements net.Conn.
func (c *PeerClient) RemoteAddr() net.Addr {
	return &NoiseAddr{Address: c.Address}
}

// SetDeadline implements net.Conn.
func (c *PeerClient) SetDeadline(t time.Time) error {
	// TODO
	return nil
}

// SetReadDeadline implements net.Conn.
func (c *PeerClient) SetReadDeadline(t time.Time) error {
	// TODO
	return nil
}

// SetWriteDeadline implements net.Conn.
func (c *PeerClient) SetWriteDeadline(t time.Time) error {
	// TODO
	return nil
}
