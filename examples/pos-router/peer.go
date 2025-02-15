package main

import(
//	"errors"
//	"fmt"
//	"crypto"
	"encoding/json"
)

/*
var (
	ErrEmptyPeerID = errors.New("Empty peer ID")
	ErrNoPublicKey = errors.New("public key is not embedded in peer ID")
)
*/

type NearbyPeer struct {
	Address		string  //ip:port/hash
	PublicKey	[]byte
	Sign		[]byte
	Ts		int64
	index		int
}

type RemotePeer struct {
	PublicKey	[]byte
	Sign		[]byte
	Ts		int64
	index		int
}

type PeerInfo struct {
	//NP	[]*NearbyPeer
	NP	NPeerHeap
	//RP	[]*RemotePeer
	RP	RPeerHeap
	ID	string
}

func NewNearbyPeer(Address string, PublicKey []byte, sign []byte, ts int64) (*NearbyPeer, error) {
	return &NearbyPeer{
		Address:	Address,
		PublicKey:	PublicKey,
		Sign:		sign,
		Ts:		ts,
		index:		-1,
	}, nil
}

func NewRemotePeer(PublicKey []byte, Sign []byte, ts int64) (*RemotePeer, error) {
	return &RemotePeer{
		PublicKey:	PublicKey,
		Sign:		Sign,
		Ts:		ts,
		index:		-1,
	}, nil
}

func NewPeerInfo(ID string, NP NPeerHeap, RP RPeerHeap) (*PeerInfo, error) {
	return &PeerInfo{
		ID: ID,
		NP: NP,
		RP: RP,
	}, nil
}

func(peer PeerInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"NP":	peer.NP,
		"RP":	peer.RP,
		"ID":	peer.ID})
}

func(peer PeerInfo) UnmarshalJSON(b []byte) error{
	temp := struct {
		ID	string
		//NP	[]*NearbyPeer
		NP	NPeerHeap
		//RP	[]*RemotePeer
		RP	RPeerHeap
	}{}

	err := json.Unmarshal(b, &temp)
	if err != nil {
		return err
	}

	if len(temp.ID) > 0 {
		peer.ID = temp.ID
	}
	peer.NP	= temp.NP
	peer.RP = temp.RP
	return nil
}

