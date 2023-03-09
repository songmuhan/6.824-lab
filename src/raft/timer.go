package raft

import (
	"math/rand"
	"time"
)

//	"bytes"
//	"6.5840/labgob"

const (
	ElectionTimeMin      = 300
	ElectionTimeInterval = 400
	HeartBeatInterval    = 70
)

func (rf *Raft) SetElectionTimer() {
	Debug(dTimer, "S%d set election timer at T:%d", rf.me, rf.CurrentTerm)
	t := time.Now()
	ms := time.Duration(ElectionTimeMin+rand.Int63()%ElectionTimeInterval) * time.Millisecond
	rf.electionTime = t.Add(ms)
}

func (rf *Raft) tick() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	//	Debug(dTimer, "S%d tiker check at T:%d", rf.me, rf.currentTerm)
	if rf.state == leader {
		rf.SetElectionTimer()
		rf.SendAppendsL(true)
	}
	if time.Now().After(rf.electionTime) {
		rf.startElectionL()
	}
}

// ticker periodicly check heartbeat and election timeout
func (rf *Raft) ticker() {
	for rf.killed() == false {
		rf.tick()
		time.Sleep(time.Duration(HeartBeatInterval) * time.Millisecond)
	}
}
