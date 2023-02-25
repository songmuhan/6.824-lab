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
	Entries      Log //
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
		entry := " Entries:" + a.Entries.String() + " }"
		return str + entry
	*/
	return fmt.Sprintf("{T:%d Leader:S%d,PrevLog Index:%d Term:%d LeaderCommit:%d, Entries:%+v }",
		a.Term, a.LeaderId, a.PrevLogIndex, a.PrevLogTerm, a.LeaderCommit, a.Entries)
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
	if args.Term > rf.currentTerm {
		rf.becomeFollowerL(args.Term)
	}
	if args.Term == rf.currentTerm {
		rf.SetElectionTimer()
		if rf.log.len() <= args.PrevLogIndex || rf.log.entry(args.PrevLogIndex).Term != args.PrevLogTerm {
			Debug(dInfo, "S%d not match PrevLogIndex&Term", rf.me)
			reply.Term = rf.currentTerm
			return
		} else {
			Debug(dError, "S%d match PreLog{Idx:%d,T:%d}", rf.me, args.PrevLogIndex, args.PrevLogTerm)
			reply.Term = rf.currentTerm
			reply.Success = true
			logInsertIndex := args.PrevLogIndex + 1
			newEntriesIndex := 0

			for {
				if logInsertIndex >= rf.log.len() || newEntriesIndex >= args.Entries.len() {
					break
				}
				if rf.log.term(logInsertIndex) != args.Entries.term(newEntriesIndex) {
					break
				}
				logInsertIndex++
				newEntriesIndex++
			}

			if newEntriesIndex < args.Entries.len() {
				rf.log.Entries = append(rf.log.Entries[:logInsertIndex], args.Entries.Entries[newEntriesIndex:]...)
			}
			if args.LeaderCommit > rf.commitIndex {
				rf.commitIndex = min(args.LeaderCommit, rf.log.lastIndex())
				Debug(dCommit, "S%d commit %d", rf.me, rf.commitIndex)
				rf.cond.Signal()
			}
		}
	}

	Debug(dAppend, "S%d AE -> S%d, reply %+v", rf.me, args.LeaderId, reply)
	Debug(dInfo, "S%d after  AE log: %+v", rf.me, rf.log)
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntrisArgs, reply *AppendEntrisReply) bool {
	Debug(dAppend, "S%d AE -> S%d, arg %+v", rf.me, server, args)
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) SendAppendsL(heartbeat bool) {
	for i := 0; i < len(rf.peers); i++ {
		if i != rf.me {
			if rf.log.lastIndex() > rf.nextIndex[i] || heartbeat {
				go func(peer int) {
					rf.sendAppend(peer, heartbeat)
				}(i)
			}
		}
	}
}
func (rf *Raft) makeAppendEntriesArg(peer int, heartbeat bool) AppendEntrisArgs {
	rf.mu.Lock()
	entries := Log{}
	lastLogIndex := rf.nextIndex[peer] - 1
	if lastLogIndex < len(rf.log.Entries)-1 {
		// do we need deep copy here ??
		entries.Entries = make([]Entry, len(rf.log.Entries[lastLogIndex+1:]))
		copy(entries.Entries, rf.log.Entries[lastLogIndex+1:])
	}
	args := AppendEntrisArgs{
		Term:         rf.currentTerm,
		LeaderId:     rf.me,
		PrevLogIndex: lastLogIndex,
		PrevLogTerm:  rf.log.term(lastLogIndex),
		Entries:      entries,
		LeaderCommit: rf.commitIndex,
	}
	rf.mu.Unlock()
	return args
}

func (rf *Raft) sendAppend(peer int, heartbeat bool) {
	// construct the args of AppendEntries
	args := rf.makeAppendEntriesArg(peer, heartbeat)
	lastLogIndex := args.PrevLogIndex
	reply := AppendEntrisReply{}
	rf.mu.Lock()
	sameTermAndState := args.Term == rf.currentTerm && rf.state == leader
	rf.mu.Unlock()
	if sameTermAndState {
		ok := rf.sendAppendEntries(peer, &args, &reply)
		if ok {
			rf.mu.Lock()
			defer rf.mu.Unlock()
			Debug(dAppend, "S%d <- AE S%d,got reply %+v", rf.me, peer, reply)
			if reply.Term > rf.currentTerm {
				rf.becomeFollowerL(reply.Term)
				return
			} else if args.Term == rf.currentTerm {
				if reply.Success {
					nextIndex := lastLogIndex + args.Entries.len() + 1
					matchIndex := lastLogIndex + args.Entries.len()
					if nextIndex > rf.nextIndex[peer] {
						rf.nextIndex[peer] = nextIndex
					}
					if matchIndex > rf.matchIndex[peer] {
						rf.matchIndex[peer] = matchIndex
					}
					for index := args.LeaderCommit + 1; index <= rf.log.lastIndex(); index++ {
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
								//				Debug(dLeader, "S%d: leader matchIndex -> %+v", rf.me, rf.matchIndex)
								Debug(dLeader, "S%d: leader commitIndex %d -> %d", rf.me, args.LeaderCommit, index)
								rf.commitIndex = index
								rf.cond.Signal()
							}
						}
					}
				} else {
					if rf.nextIndex[peer] > 1 {
						rf.nextIndex[peer] -= 1
						Debug(dAppend, "S%d decrease S%d nextIndex to %d", rf.me, peer, rf.nextIndex[peer])
					}
				}

			}
		}
	}
}
