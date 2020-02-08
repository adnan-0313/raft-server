package raft

type RequestVoteArgs struct {
	// Candidate's Term
	Term uint64

	// Candidate's ID
	CandidateID uint64

	// Index of candidate's last log entry
	LastLogIndex uint64

	// Term of candidate's last log entry
	LastLogTerm uint64
}

type RequestVoteResp struct {
	// Current term for candidate to update itself
	Term uint64

	// Vote received status
	VoteGranted bool
}

type AppendEntriesArgs struct {
	// TODO
}

type AppendEntriesResp struct {
	// TODO
}
