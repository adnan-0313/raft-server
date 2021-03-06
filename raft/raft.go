package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, Term, isleader)
//   start agreement on a new log entry
// rf.GetState() (Term, isLeader)
//   ask a Raft for its current Term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"course/labrpc"
)

const (
	Follower = iota
	Candidate
	Leader

	ElectionTimeout = 300
	HearbeatTimeout = 100
)

// import "bytes"
// import "../labgob"

//
// as each Raft peer becomes aware that successive log Entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

// Log represents log entry recieved by followers.
type Log struct {
	Term    int
	Command string
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	// Persistent State
	currenTerm      int
	votedFor        int
	logs            []Log
	state           int
	electionTimeout *time.Timer

	// Volatile State
	commitIndex int
	lastApplied int

	// Leader's State
	nextIndex  []int
	matchIndex []int

	elecTimeoutSignal    chan struct{}
	voteRPCSignal        chan struct{}
	appendEntryRPCSignal chan struct{}
}

func randomElectionTimeout(fixedElectionTimeout int) time.Duration {
	timeout := fixedElectionTimeout + ((fixedElectionTimeout / 2) + rand.Int())
	return time.Duration(timeout)
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	var term int
	var isleader bool
	// Your code here (2A).
	term = rf.currenTerm
	isleader = rf.state == Leader
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateID  int
	LastLogIndex int
	LastLogTerm  int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

func (rf *Raft) isVoteReqValid(args *RequestVoteArgs) bool {
	lastLogTerm := rf.getLastLogTerm()
	lastLogIndex := rf.getLastLogIndex()

	// Check if candidates log is at least up to date with reciever's log
	if rf.votedFor != -1 && rf.votedFor != args.CandidateID {
		return false
	}else if lastLogTerm > args.LastLogTerm {
		return false
	}else if lastLogTerm == args.LastLogTerm && lastLogIndex > args.LastLogIndex {
		return false
	}else {
		return true
	}

}

func send(ch chan struct{}) {
	// If channel not empty consumed it
	select {
	case <-ch:
	default:
	}

	ch <- struct{}{}
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).

	reply.Term = rf.currenTerm

	//A higher Term spotted, revertback to follower
	if rf.currenTerm < args.Term {
		rf.beFollower(args.Term)
		send(rf.voteRPCSignal)
	}

	if rf.currenTerm > args.Term {
		reply.VoteGranted = false
	} else if ok := rf.isVoteReqValid(args); ok {
		rf.votedFor = args.CandidateID
		reply.VoteGranted = true
		//fmt.Printf("Server %d: Grants vote to: %d\n", rf.me, args.CandidateID)
		// Update it's own Term and state to match the candidate's
		rf.beFollower(args.Term)
	}

	// Reset the election timer
	send(rf.voteRPCSignal)
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// Term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

//
// the tester calls Kill() when a Raft instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead args.CandidateID(without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
//

// AppendEntriesArgs represent arguments for AppendEntriesRPC
type AppendEndtriesArgs struct {
	Term              int
	LeaderID          int
	PrevLogIndex      int
	PrevLogTerm       int
	Entries           []Log
	LeaderCommitIndex int
}

// AppendEntriesReply represent reply for AppendEntriesRPC
type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (rf *Raft) getPrevLogIndex(i int) int {
	return rf.nextIndex[i] - 1
}

func (rf *Raft) getPrevLogTerm(i int) int {
	index := rf.getPrevLogIndex(i)
	if index < 0 {
		return -1
	}
	return rf.logs[index].Term
}

// AppendEntriesRPC will send Entries to get appended in log
func (rf *Raft) AppendEntries(args *AppendEndtriesArgs, reply *AppendEntriesReply) {
	reply.Term = rf.currenTerm

	if rf.currenTerm < args.Term {
		// Discovered greater Term, revert back to follower
		reply.Success = false
		rf.beFollower(args.Term)
		send(rf.appendEntryRPCSignal)
		return
	}

	prevLogTerm := -1
	if args.PrevLogIndex >= 0 {
		prevLogTerm = rf.logs[args.PrevLogIndex].Term
	}

	if (rf.currenTerm > args.Term /*Got stale append entry*/) ||
		(prevLogTerm != args.PrevLogTerm /*Not uptodate with leader log*/) {
		reply.Success = false
	}

	send(rf.appendEntryRPCSignal)
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEndtriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) startAppendingEntries() {
	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}

		go func(server int) {
			if rf.state != Leader {
				// This block will make sure that as
				// soon as state changed from leader to follower
				// appendEntries should stop
				return
			}

			args := &AppendEndtriesArgs{
				Term:              rf.currenTerm,
				LeaderID:          rf.me,
				PrevLogIndex:      rf.getPrevLogIndex(server),
				PrevLogTerm:       rf.getPrevLogTerm(server),
				Entries:           []Log{},
				LeaderCommitIndex: rf.commitIndex,
			}

			reply := &AppendEntriesReply{}

			if ok := rf.sendAppendEntries(server, args, reply); ok {

				if rf.currenTerm < reply.Term {
					// Discovered higher Term, revert back to follower
					rf.beFollower(reply.Term)
					send(rf.appendEntryRPCSignal)
					return
				}
			}
		}(i)
	}
}

func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) beCandidate() {
	rf.state = Candidate
	rf.currenTerm += 1
	rf.votedFor = rf.me
}

func (rf *Raft) beFollower(newTerm int) {
	rf.state = Follower
	rf.votedFor = -1
	rf.currenTerm = newTerm
}

func (rf *Raft) beLeader() {
	if rf.state != Candidate {
		return
	}

	rf.state = Leader
	//Initialise leader states
	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))

	for i := 0; i < len(rf.peers); i++ {
		rf.nextIndex[i] = rf.getLastLogIndex() + 1
	}
}

func (rf *Raft) resetElectionTimer() {
	rf.electionTimeout.Reset(1 * time.Nanosecond)
	if !rf.electionTimeout.Stop() {
		<-rf.electionTimeout.C
		rf.electionTimeout.Reset(randomElectionTimeout(ElectionTimeout))
	}
}

func (rf *Raft) getLastLogTerm() int {
	index := rf.getLastLogIndex()
	if index < 0 {
		return -1
	}
	return rf.logs[index].Term
}

func (rf *Raft) getLastLogIndex() int {
	return len(rf.logs) - 1
}

func (rf *Raft) startElection() {
	args := RequestVoteArgs{
		Term:         rf.currenTerm,
		CandidateID:  rf.me,
		LastLogIndex: rf.getLastLogIndex(),
		LastLogTerm:  rf.getLastLogTerm(),
	}

	var voteCount int32 = 1

	for i := 0; i < len(rf.peers); i++ {

		if i == rf.me {
			continue
		}

		go func(server int) {

			reply := &RequestVoteReply{}

			if ok := rf.sendRequestVote(server, &args, reply); ok {

				if rf.currenTerm < reply.Term {
					// Discover gretear Term, revrt back to follower
					//Revert back to Follower
					rf.beFollower(reply.Term)
					send(rf.voteRPCSignal)
					return
				}

				//fmt.Printf("Server %d: Recieved vote of: %d, is %v\n", rf.me, server, reply.VoteGranted)

				if rf.state != Candidate {
					return
				}

				if reply.VoteGranted {
					atomic.AddInt32(&voteCount, 1)
				}

				if atomic.LoadInt32(&voteCount) > int32(len(rf.peers)/2) {
					// Become Leader
					rf.beLeader()
					send(rf.voteRPCSignal)
				}
			}
		}(i)
	}
}

func (rf *Raft) bootServer() {
	for {
		elec := time.Duration(rand.Int31n(100) + 300) * time.Millisecond
		switch rf.state {
		case Follower, Candidate:
			select {
			case <-rf.appendEntryRPCSignal:
			case <-rf.voteRPCSignal:
			case <-time.After(elec):
				// Reset Timer
				//rf.resetElectionTimer()

				// Become Candidate
				rf.beCandidate()
				// Start Election
				go rf.startElection()
				//fmt.Printf("Server %d: Starting election ..\n", rf.me)
			}
		case Leader:
			//fmt.Printf("Server %d: is Leader\n", rf.me)
			rf.startAppendingEntries()
			time.Sleep(HearbeatTimeout * time.Millisecond)
		}
	}
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.currenTerm = 0
	rf.votedFor = -1
	rf.logs = make([]Log, 0)

	rf.commitIndex = 0
	rf.lastApplied = 0

	rf.state = Follower
	rf.electionTimeout = time.NewTimer(randomElectionTimeout(ElectionTimeout))

	rf.elecTimeoutSignal = make(chan struct{})
	rf.appendEntryRPCSignal = make(chan struct{})
	rf.voteRPCSignal = make(chan struct{})

	go rf.bootServer()
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	return rf
}
