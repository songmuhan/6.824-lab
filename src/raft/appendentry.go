package raft

import (
	"fmt"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// AppendEntris RPC argument
type AppendEntrisArgs struct {
	Term         int // leader's term
	LeaderId     int
	PrevLogIndex int // index of log entry immediately preceding new ones
	PrevLogTerm  int // term of PrevLogIndex entry
	Entires      Log //
	LeaderCommit int //leader's commitIndex
}

type AppendEntrisReply struct {
	Term    int  // currentTerm, for leader to update itself
	Success bool // true, if follower contained entry matching PrevLogIndex and PreLogTerm
}

func (a AppendEntrisArgs) String() string {
	/*
		str := fmt.Sprintf("{T:%d Leader:S%d,PrevLog Index:%d Term:%d LeaderCommit:%d",
			a.Term, a.LeaderId, a.PrevLogIndex, a.PrevLogTerm, a.LeaderCommit)
		entry := " Entries:" + a.Entires.String() + " }"
		return str + entry
	*/
	return fmt.Sprintf("{T:%d Leader:S%d,PrevLog Index:%d Term:%d LeaderCommit:%d, Entries:%+v }",
		a.Term, a.LeaderId, a.PrevLogIndex, a.PrevLogTerm, a.LeaderCommit, a.Entires)
}

func (a AppendEntrisReply) String() string {
	str := fmt.Sprintf("{T:%d", a.Term)
	if a.Success {
		str += " Success}"
	} else {
		str += " Fail}"
	}
	return str
}

func (rf *Raft) findConflict(args *AppendEntrisArgs) (int, int, bool) {
	logInsertIndex := args.PrevLogIndex + 1
	newEntriesIndex := 0
	for logInsertIndex < rf.log.len() && newEntriesIndex < args.Entires.len() {
		if rf.log.term(logInsertIndex) != args.Entires.term(newEntriesIndex) {
			break
		}
		logInsertIndex++
		newEntriesIndex++
	}
	if logInsertIndex == args.PrevLogIndex+1 && args.Entires.len() == 0 {
		return 0, 0, false
	} else {
		Debug(dError, "S%d log find conflict!", rf.me)
		return logInsertIndex, newEntriesIndex, true
	}
}

func (rf *Raft) AppendEntries(args *AppendEntrisArgs, reply *AppendEntrisReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	Debug(dInfo, "S%d before AE log: %+v", rf.me, rf.log)
	// do we need to consider this situation
	reply.Success = false
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		return
	}
	if rf.log.len() <= args.PrevLogIndex {
		Debug(dInfo, "S%d not match PrevLogIndex&Term", rf.me)
		reply.Term = rf.currentTerm
		return
	}
	if rf.log.entry(args.PrevLogIndex).Term != args.PrevLogTerm {
		Debug(dInfo, "S%d not match PrevLogIndex&Term", rf.me)
		reply.Term = rf.currentTerm
		rf.log.Entries = rf.log.Entries[:args.PrevLogIndex]
		return
	}
	if args.Term > rf.currentTerm {
		rf.becomeFollowerL(args.Term)
	}
	rf.SetElectionTimer()
	/*
		logInsertIndex, newEntriesIndex, needAppend := rf.findConflict(args)
		if needAppend {
			Debug(dInfo, "S%d logInsertIndex:%d newEntriesIndex:%d", rf.me, logInsertIndex, newEntriesIndex)
			if newEntriesIndex < args.Entires.len() {
				rf.log.Entries = append(rf.log.Entries[:logInsertIndex], args.Entires.Entries[newEntriesIndex:]...)
				Debug(dInfo, "S%d log after append:%+v", rf.me, rf.log)
			}
		}
	*/
	/*
		if args.Entires.len() > 0 {
			existingEntries := rf.log.Entries[args.PrevLogIndex+1:]
			var i int
			for i = 0; i < min(len(existingEntries), args.Entires.len()); i++ {
				if existingEntries[i].Term != args.Entires.term(i) {
					Debug(dInfo, "S%d ,Discard conflicts: %v", rf.me, rf.log.Entries[args.PrevLogIndex+1+i:])
					rf.log.Entries = rf.log.Entries[:args.PrevLogIndex+1+i]
					break
				}
			}
			if i < args.Entires.len() {
				// Append any new entries not already in the log
				rf.log.Entries = append(rf.log.Entries, args.Entires.Entries[i:]...)
			}
		}
	*/
	if args.Entires.len() > 0 {
		rf.log.Entries = append(rf.log.Entries[:args.PrevLogIndex+1], args.Entires.Entries...)
	}
	if args.LeaderCommit > rf.commitIndex {
		tmp := min(args.LeaderCommit, rf.log.lastIndex())
		if rf.log.term(tmp) == rf.currentTerm {
			Debug(dAppend, "S%d follower commitIndex:%d -> %d", rf.me, rf.commitIndex, tmp)
			rf.commitIndex = tmp
			rf.cond.Signal()
		}
	}

	reply.Term = rf.currentTerm
	reply.Success = true
	Debug(dAppend, "S%d AE -> S%d, reply %+v", rf.me, args.LeaderId, reply)
	Debug(dInfo, "S%d after  AE log:%+v", rf.me, rf.log)
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntrisArgs, reply *AppendEntrisReply) bool {
	Debug(dAppend, "S%d AE -> S%d, arg %+v", rf.me, server, args)
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) SendAppendsL(heartbeat bool) {
	//rf.mu.Lock()
	for i := 0; i < len(rf.peers); i++ {
		if i != rf.me {
			go func(peer int) {
				rf.sendAppend(peer, heartbeat)
			}(i)
		}
	}
	//rf.mu.Unlock()
}

func (rf *Raft) sendAppend(peer int, heartbeat bool) {
	// construct the args of AppendEntries

	rf.mu.Lock()
	entries := Log{}
	//	Debug(dLeader, "S%d log ->[ %+v ]", rf.me, rf.log)
	lastLogIndex := rf.nextIndex[peer] - 1
	commitIndexbeforeRPC := rf.commitIndex
	//Debug(dInfo, "S%d: lastlogIndex:%d log: %+v", rf.me, lastLogIndex, rf.log)
	if (!heartbeat) && lastLogIndex < len(rf.log.Entries)-1 {
		// do we need deep copy here ??
		entries.Entries = make([]Entry, len(rf.log.Entries[lastLogIndex+1:]))
		copy(entries.Entries, rf.log.Entries[lastLogIndex+1:])
	}
	//Debug(dInfo, "S%d: lastlogIndex:%d entries: %+v", rf.me, lastLogIndex, entries)

	args := AppendEntrisArgs{
		Term:         rf.currentTerm,
		LeaderId:     rf.me,
		PrevLogIndex: lastLogIndex,
		PrevLogTerm:  rf.log.term(lastLogIndex),
		Entires:      entries,
		LeaderCommit: rf.commitIndex,
	}
	//	Debug(dError, "entries: %+v", entries)
	//	Debug(dAppend, "args should be : %+v", args)
	rf.mu.Unlock()
	reply := AppendEntrisReply{}
	savedTerm := rf.currentTerm
	if savedTerm == rf.currentTerm {
		ok := rf.sendAppendEntries(peer, &args, &reply)
		if ok {
			rf.mu.Lock()
			Debug(dAppend, "S%d <- AE S%d,got reply %+v", rf.me, peer, reply)
			defer rf.mu.Unlock()
			if reply.Term > rf.currentTerm {
				rf.becomeFollowerL(reply.Term)
				return
			}

			if reply.Success {
				nextIndex := lastLogIndex + entries.len() + 1
				matchIndex := lastLogIndex + entries.len()
				if nextIndex > rf.nextIndex[peer] {
					//		Debug(dLeader, "S%d: nextIndex S%d [%d -> %d]", rf.me, peer, rf.nextIndex[peer], nextIndex)
					rf.nextIndex[peer] = nextIndex
				}
				if matchIndex > rf.matchIndex[peer] {
					//		Debug(dLeader, "S%d: matchIndex S%d [%d -> %d]", rf.me, peer, rf.matchIndex[peer], matchIndex)
					rf.matchIndex[peer] = matchIndex
				}
				for index := commitIndexbeforeRPC + 1; index <= rf.log.lastIndex(); index++ {
					if rf.log.term(index) == rf.currentTerm {
						counter := 1 // leader has a count
						for peer := range rf.peers {
							if peer != rf.me {
								if rf.matchIndex[peer] >= index {
									counter++
								}
							}

						}
						if counter > len(rf.peers)/2 {
							Debug(dLeader, "S%d: leader matchIndex -> %+v", rf.me, rf.matchIndex)
							Debug(dLeader, "S%d: leader commitIndex %d -> %d", rf.me, commitIndexbeforeRPC, index)
							rf.commitIndex = index
							rf.cond.Signal()
						}
					}
					// todo
					//Debug(dInfo, "S%d: next CommitIndex %d", rf.me, index)
				}

				// todo : do we need to preserve the previous

			} else {
				if rf.nextIndex[peer] > 1 {
					rf.nextIndex[peer] -= 1
					Debug(dAppend, "S%d decrease S%d nextIndex to %d", rf.me, peer, rf.nextIndex[peer])
				}
			}

		}
	}
}
