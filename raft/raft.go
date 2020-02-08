package raft

import (
	"go.uber.org/zap"
	"sync"
)

// NodeState represent different states of a raft node
type NodeState uint8

const (
	Follower NodeState = iota
	Candidate
	Leader
)

const (
	NULL int64 = -1
)

// leaderState is state that is used while we are a leader
type leaderState struct {
	replState map[string]*followerReplication
}

type followerReplication struct {
	nextIndex  uint64
	matchIndex uint64
}

type Node struct {
	conf *Config

	// raft node state
	state NodeState

	// current term of a raft node
	currentTerm uint64

	// unique identifier of a raft node to distinguish it from peers
	nodeID uint64

	// unique identifier of a raft node for which this node voted
	votedFor int64

	// list of other raft nodes
	peers []Peer

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

	// lock to protect shared access to this node's state
	lock sync.Mutex

	// logger to produce logs
	logger *zap.Logger

	// use to manage go routines
	routinesBucket sync.WaitGroup
}

func NewNode(nodeID uint64, peers []Peer, logs LogStore, trans Transport,
	shutdownCh chan struct{}) (*Node, error) {
	conf := DefaultConfig()

	logger, _ := zap.NewDevelopment()

	r := &Node{
		conf:           conf,
		state:          Follower,
		currentTerm:    0,
		nodeID:         nodeID,
		votedFor:       NULL,
		peers:          peers,
		logs:           logs,
		commitIndex:    0,
		lastApplied:    0,
		trans:          trans,
		rpcCh:          trans.Consumer(),
		shutdownCh:     shutdownCh,
		lock:           sync.Mutex{},
		logger:         logger,
		routinesBucket: sync.WaitGroup{},
	}

	r.goFunc(r.run)
	return r, nil
}

// gracefully starts goroutines in sync with main go routine
func (r *Node) goFunc(f func()) {
	r.routinesBucket.Add(1)
	go func() {
		defer r.routinesBucket.Done()
		f()
	}()
}

// run is a long goroutine which kick starts a raft node
// based on its current state
func (r *Node) run() {
	r.logger.Info("Starting raft node", zap.Uint64("nodeID", r.nodeID))

	for {
		switch r.state {
		case Candidate:
			r.goFunc(r.runCandidate)
		}
	}
}

// requests votes from other peers
func (r *Node) askVoteFromPeer(peer Peer, voteCh chan<- *RequestVoteResp) {
	r.goFunc(func() {
		args := &RequestVoteArgs{
			Term:         r.currentTerm,
			CandidateID:  r.nodeID,
			LastLogIndex: r.logs.LastIndex(),
			LastLogTerm:  r.logs.LastTerm(),
		}

		resp := new(RequestVoteResp)

		err := r.trans.RequestVote(peer.Addr, args, resp)
		if err != nil {
			r.logger.Error("vote request denied",
				zap.Uint64("candidateID", r.nodeID),
				zap.Uint64("peerID", peer.PeerID),
				zap.Reflect("addr", peer.Addr))
		}
		voteCh <- resp
	})
}

// runCandidate is a long goroutine which governs the operation of
// raft node in Candidate state
func (r *Node) runCandidate() {

	// use to collect vote from other peers and its own
	voteCh := make(chan *RequestVoteResp, len(r.peers))
	// increase the current term
	r.currentTerm += 1

	// vote for itself
	r.votedFor = int64(r.nodeID)

	// peers represents list of peers
	for _, peer := range r.peers {
		if peer.PeerID == r.nodeID {
			continue
		}
		r.askVoteFromPeer(peer, voteCh)
	}

	votesGranted := 0

	// Votes needed win an election
	votesNeeded := (len(r.peers) / 2) + 1
	for r.state == Candidate {

		select {
		case vote := <-voteCh:

			if !vote.VoteGranted {
				votesGranted++
			}

			if votesGranted >= votesNeeded {
				// Election won
				// TODO
			}

		}
	}
}
