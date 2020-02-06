package raft

import (
	"net"
	"sync"
)

// NodeState represent different states of a raft node
type NodeState uint8

const (
	Follower NodeState = iota
	Candidate
	Leader
)

// leaderState is state that is used while we are a leader
type leaderState struct {
	replState map[string]*followerReplication
}

type followerReplication struct {
	nextIndex uint64
	matchIndex uint64
}

type Node struct {

	conf *Config

	// raft node state
	state NodeState

	// current term of a raft node
	currentTerm uint64

	// unique identifier of a raft node to distinguish it from peers
	nodeID int

	// list of other raft nodes
	peers []net.Addr

	// logs stored in this raft node
	logs LogStore

	// index of highest entry known to be committed
	commitIndex uint64

	// index of highest entry applied to state machine
	lastApplied uint64

	// states for a leader raft node
	leaderState leaderState

	// RPC chan comes from the transport layer
	rpcCh <-chan RPC

	// allows raft node to communicate to other peers
	trans Transport

	// channel dedicated to shutdown signal to raft node
	shutdownCh chan struct{}

	// lock dedicated to concurrent safe the data
	Lock sync.Mutex
}