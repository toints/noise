package main

import (
	"fmt"
)

func main(){
	peerInfo := new(PeerInfo)
	peerInfo.ID = "0x123456"
	np := new(NearbyPeer)
	address := "/127.0.0.1:3000/adfdgbg"
	pubkey := "FSDBGFDKGGJKGDNFDFN"
	sign := "0xabcdefghijklmn"

	npinfo, err := np.NewNearbyPeer(address, pubkey, sign)
	if err != nil {
		fmt.Println(err)
	}
	rp := &RemotePeer{
		PublicKey: "XXXXXXXXXXXXXX",
		Sign:"0x000000000000000000",
	}

	peerInfo.NP = append(peerInfo.NP, npinfo)
	peerInfo.RP = append(peerInfo.RP, rp)

	jsonData, err := peerInfo.MarshalJSON()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("jsondata:\n", string(jsonData))
}
