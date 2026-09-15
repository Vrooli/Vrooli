package sandboxiface

import (
	"errors"
	"testing"
	"time"
)

func TestFakeProcFS_DefaultEmpty(t *testing.T) {
	fs := NewFakeProcFS(nil)
	pids, err := fs.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(pids) != 0 {
		t.Errorf("len = %d, want 0", len(pids))
	}
}

func TestFakeProcFS_RoundTrip(t *testing.T) {
	now := time.Now()
	fs := NewFakeProcFS(map[string]FakeProcEntry{
		"42": {Cmdline: []byte("fuse-overlayfs"), StartTime: now},
	})

	pids, err := fs.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(pids) != 1 || pids[0] != "42" {
		t.Errorf("List = %v", pids)
	}

	entry, err := fs.Open("42")
	if err != nil {
		t.Fatal(err)
	}
	if string(entry.Cmdline()) != "fuse-overlayfs" {
		t.Errorf("Cmdline = %q", entry.Cmdline())
	}
	if !entry.StartTime().Equal(now) {
		t.Errorf("StartTime mismatch")
	}

	if _, err := fs.Open("missing"); err == nil {
		t.Error("Open(missing) should error")
	}
}

func TestFakeProcFS_ListErr(t *testing.T) {
	fs := NewFakeProcFS(map[string]FakeProcEntry{"1": {}})
	fs.ListErr = errors.New("io")
	if _, err := fs.List(); err == nil {
		t.Error("ListErr should surface")
	}
}
