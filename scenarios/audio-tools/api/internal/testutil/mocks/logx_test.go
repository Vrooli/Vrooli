package mocks_test

import (
	"testing"

	"audio-tools/internal/logx"
	"audio-tools/internal/testutil/mocks"
)

func TestFakeLogger_SatisfiesLogger(t *testing.T) {
	var _ logx.Logger = mocks.NewFakeLogger()
}

func TestFakeLogger_RecordsFormattedEntries(t *testing.T) {
	l := mocks.NewFakeLogger()
	l.Printf("hello %s %d", "world", 7)
	l.Printf("plain")
	got := l.Entries()
	if len(got) != 2 || got[0] != "hello world 7" || got[1] != "plain" {
		t.Errorf("Entries() = %v", got)
	}
}

func TestFakeLogger_Reset(t *testing.T) {
	l := mocks.NewFakeLogger()
	l.Printf("x")
	l.Reset()
	if got := l.Entries(); len(got) != 0 {
		t.Errorf("after Reset, Entries() = %v, want empty", got)
	}
}
