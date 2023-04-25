package raft

import (
	"fmt"
)

type Entry struct {
	Command interface{}
	Term    int
}

type Log struct {
	Entries []Entry
}

func (l Log) String() string {
	str := ""
	start := 0
	if len(l.Entries) > 5 {
		start = len(l.Entries) - 5
		str = fmt.Sprintf("{%d: ...", len(l.Entries))
	}
	for _, entry := range l.Entries[start:] {
		cmd := fmt.Sprintf("%+v", entry.Command)
		if len(cmd) > 4 {
			cmd = cmd[:4]
		}
		str += fmt.Sprintf("[%4s %d]", cmd, entry.Term)
	}
	str += "}"
	return str
}

func (l *Log) makeEmptyLog() {
	l.Entries = append(l.Entries, Entry{Term: -1})
}

func (l *Log) lastIndex() int {
	return len(l.Entries) - 1
}

func (l *Log) entry(index int) Entry {
	return l.Entries[index]
}

func (l *Log) lastTerm() int {
	return l.Entries[len(l.Entries)-1].Term
}

func (l *Log) term(index int) int {
	if index < len(l.Entries) {
		return l.Entries[index].Term
	} else {
		// this should not happen in any circmustance
		str := fmt.Sprintf("log out of index, this should never happen")
		Panic(str)
	}
	return -1
}

func (l *Log) len() int {
	return len(l.Entries)
}

func (l *Log) append(entry Entry) {
	l.Entries = append(l.Entries, entry)
}

func (l *Log) getFirstIndexofTerm(term int) int {
	for i, entry := range l.Entries {
		if entry.Term == term {
			return i
		}
	}
	return -1
}

func (l *Log) getLastIndexofTerm(term int) int {
	first := l.getFirstIndexofTerm(term)
	if first != -1 {
		for i, entry := range l.Entries[first:] {
			if entry.Term != term {
				// if i != len(l.Entries[first:])-1 {
				// 	return i - 1
				// }
				// return 1
				return i - 1
			}
		}
	}
	return -1
}
