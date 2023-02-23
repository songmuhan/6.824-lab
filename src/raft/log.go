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
	for _, entry := range l.Entries {
		str += fmt.Sprintf("[%+v %d]", entry.Command, entry.Term)
	}
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
	return l.Entries[index].Term
}

func (l *Log) len() int {
	return len(l.Entries)
}

func (l *Log) append(entry Entry) {
	l.Entries = append(l.Entries, entry)
}
