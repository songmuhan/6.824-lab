package raft

import "fmt"

type State int

const (
	leader State = iota
	follower
	candidate
)

// must be called with rf.mu.lock()
func (rf *Raft) becomeFollowerL(term int) {
	defer Debug(dState, "S%d become Follower at old T:%d, new T:%d", rf.me, rf.CurrentTerm, term)
	rf.state = follower
	rf.CurrentTerm = term
	rf.VoteFor = -1
	// rf.SetElectionTimer()
	rf.persist()
}

func (rf *Raft) becomeLeaderL() {
	Debug(dState, "S%d become leader at T:%d", rf.me, rf.CurrentTerm)
	if rf.state == leader {
		msg := fmt.Sprintf("S%d is leader, want to become leader again", rf.me)
		panic(msg)
	}
	rf.state = leader
	//	rf.SetElectionTimer()

	// initialize nextIndex and matchIndex
	for peer := range rf.peers {
		if peer != rf.me {
			// initialized to leader last log index + 1
			rf.nextIndex[peer] = rf.Log.lastIndex() + 1
			// initialized to 0, increase monotonically
			rf.matchIndex[peer] = 0
		}
	}
	rf.SendAppendsL(false) // issue heartbeat
}
