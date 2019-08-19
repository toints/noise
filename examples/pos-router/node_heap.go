package main

import (
	"container/heap"
//	"fmt"
)

type NPeerHeap []*NearbyPeer
type RPeerHeap []*RemotePeer

func (nh NPeerHeap) Len() int {
	return len(nh)
}

func (nh NPeerHeap) Less(i, j int) bool {
	return nh[i].Ts < nh[j].Ts
}

func (nh NPeerHeap) Swap(i, j int) {
	nh[i], nh[j] = nh[j], nh[i]
	nh[i].index = i
	nh[j].index = j
}

func (nh *NPeerHeap) Push(x interface{}) {
	n := len(*nh)
	item := x.(*NearbyPeer)
	item.index = n
	*nh = append(*nh, item)
}

func (nh *NPeerHeap) Pop() interface{} {
	old := *nh
	n := len(old)
	item := old[n-1]
	item.index = -1 //for safety
	*nh = old[0:n-1]
	return item
}

func (nh *NPeerHeap) update(item *NearbyPeer, addr string, pkey []byte, sign []byte, ts int64) {
	item.Address = addr
	item.PublicKey = pkey
	item.Sign = sign
	item.Ts = ts
	heap.Fix(nh, item.index)
}

/*************** Remote Peer List ********************/
func (rh RPeerHeap) Len() int {
	return len(rh)
}

func (rh RPeerHeap) Less(i, j int) bool {
	return rh[i].Ts < rh[j].Ts
}

func (rh RPeerHeap) Swap(i, j int) {
	rh[i], rh[j] = rh[j], rh[i]
	rh[i].index = i
	rh[j].index = j
}

func (rh *RPeerHeap) Push(x interface{}) {
	n := len(*rh)
	item := x.(*RemotePeer)
	item.index = n
	*rh = append(*rh, item)
}

func (rh *RPeerHeap) Pop() interface{} {
	old := *rh
	n := len(old)
	item := old[n-1]
	item.index = -1 //for safety
	*rh = old[0:n-1]
	return item
}

func (rh *RPeerHeap) update(item *RemotePeer, pkey []byte, sign []byte, ts int64) {
	item.PublicKey = pkey
	item.Sign = sign
	item.Ts = ts
	heap.Fix(rh, item.index)
}
