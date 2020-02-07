package raft

import "net"

// Peer stores meta data of a peer
type Peer struct {
	Addr net.Addr
	PeerID uint64
}