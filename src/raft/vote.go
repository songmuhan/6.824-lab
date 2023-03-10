package raft

import "fmt"

//	"bytes"

//	"6.5840/labgob"

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int // candidate's term
	CandidateId  int // candidate requesting vote
	LastLogIndex int // index of candidate's last log entry
	LastLogTerm  int // term of candidate's last log entry
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int  // currentTerm, for candidate to update itself
	VoteGranted bool // true means candidate received vote
}

func (r *RequestVoteArgs) String() string {
	return fmt.Sprintf("{T:%d Candidate:S%d LastLog[%d,%d]}", r.Term, r.CandidateId, r.LastLogIndex, r.LastLogTerm)
}

func (r *RequestVoteReply) String() string {
	return fmt.Sprintf("{T:%d Vote:%t}", r.Term, r.VoteGranted)
}

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
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	Debug(dVote, "S%d RV -> S%d, arg %+v", rf.me, server, args)
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	reply.VoteGranted = false

	if rf.CurrentTerm < args.Term {
		rf.becomeFollowerL(args.Term)
		//rf.SetElectionTimer()
	}
	if args.Term < rf.CurrentTerm {
		reply.Term = rf.CurrentTerm
		return
	} else if rf.CurrentTerm == args.Term {
		uptodate := (args.LastLogTerm > rf.Log.lastTerm()) ||
			((args.LastLogTerm == rf.Log.lastTerm()) && (args.LastLogIndex >= rf.Log.lastIndex()))
		if (rf.VoteFor == -1 || rf.VoteFor == args.CandidateId) && uptodate {
			rf.VoteFor = args.CandidateId
			rf.persist()
			reply.VoteGranted = true
			rf.SetElectionTimer()
		}
	}
	reply.Term = rf.CurrentTerm
	Debug(dVote, "S%d RV -> S%d, reply %+v", rf.me, args.CandidateId, reply)

}
func (rf *Raft) SendRequestVotesL() {
	args := RequestVoteArgs{
		Term:         rf.CurrentTerm,
		CandidateId:  rf.me,
		LastLogIndex: rf.Log.lastIndex(),
		LastLogTerm:  rf.Log.lastTerm(),
	}
	voteCounter := 1

	for peer := range rf.peers {
		if peer != rf.me {
			go func(peer int) {
				rf.SendRequestVote(peer, args, &voteCounter)
			}(peer)
		}
	}
}

func (rf *Raft) SendRequestVote(peer int, args RequestVoteArgs, voteCounter *int) {
	reply := RequestVoteReply{}
	ok := rf.sendRequestVote(peer, &args, &reply)
	if ok {
		rf.mu.Lock()
		defer rf.mu.Unlock()
		Debug(dVote, "S%d <- RV S%d, got reply", rf.me, peer)
		if reply.Term > rf.CurrentTerm {
			rf.becomeFollowerL(reply.Term)
			return
		}
		// check whether we are still in the same term and this peer is still candidate
		sameTermAndState := (args.Term == rf.CurrentTerm && rf.state == candidate)
		if reply.VoteGranted && sameTermAndState {
			*voteCounter++
			if *voteCounter > len(rf.peers)/2 {
				rf.becomeLeaderL()
			}
		}
	}

}

func (rf *Raft) startElectionL() {
	rf.state = candidate
	rf.VoteFor = rf.me
	rf.CurrentTerm++
	rf.persist()
	rf.SetElectionTimer()
	Debug(dVote, "S%d start new election at T:%d", rf.me, rf.CurrentTerm)
	rf.SendRequestVotesL()
}
