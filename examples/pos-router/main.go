package main

import (
	"bufio"
	"flag"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/cipher/aead"
	"github.com/perlin-network/noise/handshake/ecdh"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/payload"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"
	"github.com/perlin-network/noise/signature/eddsa"
	"github.com/pkg/errors"
	"os"
	"strconv"
	"strings"
	"fmt"
	"time"
)

/** DEFINE MESSAGES **/
var (
	opcodeChat noise.Opcode
	_          noise.Message = (*chatMessage)(nil)
)

type chatMessage struct {
	text	string
}


func (chatMessage) Read(reader payload.Reader) (noise.Message, error) {
	text, err := reader.ReadString()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read chat msg")
	}

	return chatMessage{text: text}, nil
}

func (m chatMessage) Write() []byte {
	return payload.NewWriter(nil).WriteString(m.text).Bytes()
}

func genPeersData(node *noise.Node) ([]byte, error){
	/************ add peers info **************/
	//var npList []*NearbyPeer
	//var rpList []*RemotePeer
	var npList NPeerHeap
	var rpList RPeerHeap

	npeers := skademlia.FindNode(node, protocol.NodeID(node).(skademlia.ID), skademlia.BucketSize(), 1)
	//log.Info().Msgf("Bootstrapped with peers: %+v", npeers)
	rpeers := skademlia.FindNode(node, protocol.NodeID(node).(skademlia.ID), skademlia.BucketSize(), 8)
	//log.Info().Msgf("Bootstrapped with peers: %+v", rpeers)

	for k, v := range npeers {
		log.Info().Msgf("**** id:%v --> address:%s, publicKey:%x", k , v.Address(), v.PublicKey()[:])
		scheme := eddsa.New()
		ts := time.Now().Unix()
		now := fmt.Sprintf("%d:%x", ts, v.PublicKey()[:])
		log.Info().Msgf("**** time now and publicKey:%s", now)

		sign, err := scheme.Sign(node.Keys.PrivateKey(), []byte(now))
		if err != nil {
			panic(err)
		}

		log.Info().Msgf("**** sign:%x", string(sign[:]))
		np, _ := NewNearbyPeer(v.Address(), v.PublicKey(), sign, ts)
		//npList = append(npList, np)
		npList.Push(np)
	}

	for k, v := range rpeers {
		log.Info().Msgf("#### id:%v --> address:%s, publicKey:%x", k , v.Address(), v.PublicKey()[:])
		scheme := eddsa.New()
		ts := time.Now().Unix()
		now := fmt.Sprintf("%d:%x", ts, v.PublicKey()[:]) //FIXME: only ts
		log.Info().Msgf("#### time now and publicKey:%s", now)

		sign, err := scheme.Sign(node.Keys.PrivateKey(), []byte(now))
		if err != nil {
			panic(err)
		}

		log.Info().Msgf("#### sign:%x", string(sign[:]))
		rp, _ := NewRemotePeer(v.PublicKey(), sign, ts)
		//rpList = append(rpList, rp)
		rpList.Push(rp)
	}

	pid := fmt.Sprintf("%x", node.Keys.PublicKey()[:])
	pinfo, _ := NewPeerInfo(pid, npList, rpList)
	jsonPeerInfo, err := pinfo.MarshalJSON()
	if err != nil {
		panic(err)
	}
	//log.Info().Msgf("json peer info:%s", jsonPeerInfo)
	/************ add peers info end **************/
	return jsonPeerInfo, nil
}

/** ENTRY POINT **/
func setup(node *noise.Node) {
	opcodeChat = noise.RegisterMessage(noise.NextAvailableOpcode(), (*chatMessage)(nil))

	node.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
		peer.OnConnError(func(node *noise.Node, peer *noise.Peer, err error) error {
			log.Info().Msgf("Got an error: %v", err)

			return nil
		})

		peer.OnDisconnect(func(node *noise.Node, peer *noise.Peer) error {
			log.Info().Msgf("Peer %v has disconnected.", peer.RemoteIP().String()+":"+strconv.Itoa(int(peer.RemotePort())))

			return nil
		})

		log.Info().Msgf("** [localAddress:%s | remoteAddress:%s] **", peer.LocalAddress(), peer.RemoteAddress())

		go func() {
			for {
				msg := <-peer.Receive(opcodeChat)
				log.Info().Msgf("[%s]: %s", protocol.PeerID(peer), msg.(chatMessage).text)
				//TODO:
				//npList and rpList receive, verify peer data by publicKey, and then update the npList and rpList info
			}
		}()

		return nil
	})
}

func main() {
	hostFlag := flag.String("h", "127.0.0.1", "host to listen for peers on")
	portFlag := flag.Uint("p", 3000, "port to listen for peers on")
	target := flag.String("d", "", "target peer to dial")
	flag.Parse()

	params := noise.DefaultParams()
	//params.NAT = nat.NewPMP()
	params.Keys = skademlia.RandomKeys()
	params.Host = *hostFlag
	params.Port = uint16(*portFlag)

	node, err := noise.NewNode(params)
	if err != nil {
		panic(err)
	}
	defer node.Kill()

	p := protocol.New()
	p.Register(ecdh.New())
	p.Register(aead.New())
	p.Register(skademlia.New())
	p.Enforce(node)

	setup(node)
	go node.Listen()

	log.Info().Msgf("Listening for peers on port %d.", node.ExternalPort())

	/*** add timer to boradcast node access ***/
	report_ticker := time.NewTicker(time.Second * time.Duration(10))
	go func() {
		for {
			select {
			case <-report_ticker.C:
				data, err := genPeersData(node)
				if err != nil {
					panic(err)
				}
				log.Info().Msgf("@@ Broadcast this message:[%s]", data)
			}
		}
	}()
	/************ end timer **********/

	//var peers []skademlia.ID
	if *target != "" {
		peer, err := node.Dial(*target)
		if err != nil {
			panic(err)
		}
		skademlia.WaitUntilAuthenticated(peer)

		//peers = skademlia.FindNode(node, protocol.NodeID(node).(skademlia.ID), skademlia.BucketSize(), 8)
		//log.Info().Msgf("Bootstrapped with peers: %+v", peers)
	}
	reader := bufio.NewReader(os.Stdin)

	for {
		txt, err := reader.ReadString('\n')

		if err != nil {
			panic(err)
		}

		skademlia.BroadcastAsync(node, chatMessage{text: strings.TrimSpace(txt)})
	}
}
