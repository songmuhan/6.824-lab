package raft

import "fmt"

// AppendEntris RPC argument
type AppendEntrisArgs struct {
	Term     int // leader's term
	LeaderId int
}

type AppendEntrisReply struct {
	Term    int  // currentTerm, for leader to update itself
	Success bool //
}

func (a *AppendEntrisArgs) String() string {
	return fmt.Sprintf("T:%d Leader:S%d", a.Term, a.LeaderId)
}

func (a *AppendEntrisReply) String() string {
	return fmt.Sprintf("T:%d Success:%t", a.Term, a.Success)
}

func (rf *Raft) AppendEntries(args *AppendEntrisArgs, reply *AppendEntrisReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// do we need to consider this situation
	reply.Success = false
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		return
	}

	if args.Term > rf.currentTerm {
		rf.becomeFollowerL(args.Term)
	}
	rf.SetElectionTimer()
	reply.Term = rf.currentTerm

	Debug(dAppend, "S%d AE -> S%d, reply %v", rf.me, args.LeaderId, reply)

}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntrisArgs, reply *AppendEntrisReply) bool {
	Debug(dAppend, "S%d AE -> S%d, arg %v", rf.me, server, args)
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) SendAppendsL() {
	args := AppendEntrisArgs{
		Term:     rf.currentTerm,
		LeaderId: rf.me,
	}
	for peer := range rf.peers {
		if peer != rf.me {
			go func(peer int) {
				rf.sendAppend(peer, args)
			}(peer)
		}
	}
}

func (rf *Raft) sendAppend(peer int, args AppendEntrisArgs) {
	reply := AppendEntrisReply{}
	ok := rf.sendAppendEntries(peer, &args, &reply)
	if ok {
		rf.mu.Lock()
		Debug(dAppend, "S%d <- AE S%d, got reply", rf.me, peer)
		defer rf.mu.Unlock()
		if reply.Term > rf.currentTerm {
			rf.becomeFollowerL(reply.Term)
		}
	}

}
