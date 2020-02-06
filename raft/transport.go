package raft

import "net"

type RPCResponse struct {
	Response interface{}
	Error error
}

type RPC struct {
	Command interface{}
	RespChan chan<- RPCResponse
}

func (r* RPC) Respond(resp interface{}, err error) {
	r.RespChan <- RPCResponse{resp, err}
}

// Transport provides an interface for network transports
// to allow Raft to communicate with other nodes
type Transport interface {
	// Consumer returns a channel that can be used to
	// consume and respond to RPC requests.
	Consumer() <-chan RPC

	// AppendEntries sends the appropriate RPC to the target node
	AppendEntries(target net.Addr, args *AppendEntriesRequest, resp *AppendEntriesResponse) error

	// RequestVote sends the appropriate RPC to the target node
	RequestVote(target net.Addr, args *RequestVoteRequest, resp *RequestVoteResponse) error
}
