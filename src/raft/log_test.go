package raft

import "testing"

func TestLogString(t *testing.T) {
	log := Log{}
	log.makeEmptyLog()
	t.Logf("%+v", log)
}

func TestAEargs(t *testing.T) {
	entries := Log{}
	entries.makeEmptyLog()
	entries.Entries = append(entries.Entries, Entry{1, 1})
	args := AppendEntrisArgs{
		Entires: entries,
	}
	t.Logf("%+v", args)
}
