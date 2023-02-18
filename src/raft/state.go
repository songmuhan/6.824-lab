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
	defer Debug(dState, "S%d become Follower at old T:%d, new T:%d", rf.me, rf.currentTerm, term)
	rf.state = follower
	rf.currentTerm = term
	rf.voteFor = -1
	rf.SetElectionTimer()
}

func (rf *Raft) becomeLeaderL() {
	Debug(dState, "S%d become leader at T:%d", rf.me, rf.currentTerm)
	if rf.state == leader {
		msg := fmt.Sprintf("S%d is leader, want to become leader again", rf.me)
		panic(msg)
	}
	rf.state = leader
	rf.SetElectionTimer()
	rf.SendAppendsL() // issue heartbeat
}
