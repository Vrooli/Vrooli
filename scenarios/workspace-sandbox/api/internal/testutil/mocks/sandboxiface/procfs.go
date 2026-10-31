package sandboxiface

import (
	"fmt"
	"sync"
	"time"

	"workspace-sandbox/internal/sandbox"
)

// FakeProcFS is a synthetic /proc fixture for daemon-reaper tests.
// Tests populate Entries with the per-PID fakeProcEntry data they
// want the reaper to see; FakeProcFS returns those values without
// touching the real /proc.
type FakeProcFS struct {
	mu sync.Mutex

	// Entries maps pid (string) -> entry data.
	Entries map[string]FakeProcEntry

	// ListErr / OpenErr override the default success path.
	ListErr error
	OpenErr error
}

// FakeProcEntry is the data backing one /proc/<pid> directory in the
// fake. Cmdline holds the NUL-separated argv buffer; StartTime is the
// process start time the reaper compares against the grace period.
type FakeProcEntry struct {
	Cmdline   []byte
	StartTime time.Time
}

// NewFakeProcFS returns a FakeProcFS with the given entries.
func NewFakeProcFS(entries map[string]FakeProcEntry) *FakeProcFS {
	if entries == nil {
		entries = make(map[string]FakeProcEntry)
	}
	return &FakeProcFS{Entries: entries}
}

// List returns the PIDs in the fixture (sorted by map iteration; tests
// shouldn't depend on order).
func (f *FakeProcFS) List() ([]string, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	pids := make([]string, 0, len(f.Entries))
	for k := range f.Entries {
		pids = append(pids, k)
	}
	return pids, nil
}

// Open returns the entry for the given PID.
func (f *FakeProcFS) Open(pid string) (sandbox.ProcEntry, error) {
	if f.OpenErr != nil {
		return nil, f.OpenErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.Entries[pid]
	if !ok {
		return nil, fmt.Errorf("pid %s not in fixture", pid)
	}
	return &fakeProcEntryView{data: e}, nil
}

// fakeProcEntryView is the per-PID view returned by Open. It reads
// from a snapshotted FakeProcEntry so concurrent updates to
// FakeProcFS.Entries don't race.
type fakeProcEntryView struct {
	data FakeProcEntry
}

func (e *fakeProcEntryView) Cmdline() []byte      { return e.data.Cmdline }
func (e *fakeProcEntryView) StartTime() time.Time { return e.data.StartTime }

var (
	_ sandbox.ProcFS    = (*FakeProcFS)(nil)
	_ sandbox.ProcEntry = (*fakeProcEntryView)(nil)
)
