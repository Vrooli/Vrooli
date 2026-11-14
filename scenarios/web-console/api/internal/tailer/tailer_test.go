package tailer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type testCheckpointStore struct {
	mu sync.Mutex
	cp map[string]Checkpoint
}

func (s *testCheckpointStore) Get(_ context.Context, source, key string) (Checkpoint, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp, ok := s.cp[source+"\x00"+key]
	return cp, ok, nil
}

func (s *testCheckpointStore) Save(_ context.Context, cp Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cp[cp.Source+"\x00"+cp.SourceKey] = cp
	return nil
}

type testSource struct{}

func (testSource) DiscoverFiles(_ context.Context) ([]FileRef, error) { return nil, nil }
func (testSource) DecodeLine(_ string, line []byte) ([]Event, error) {
	return []Event{{Role: "assistant", Text: string(line[:len(line)-1]), Commit: true}}, nil
}
func (testSource) CaptureAgentInfo(string, string) {}

func TestEngineTailsCompleteLinesAndPersistsByteCursor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "updates.jsonl")
	if err := os.WriteFile(path, []byte("one\ntwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	store := &testCheckpointStore{cp: make(map[string]Checkpoint)}
	var mu sync.Mutex
	var got []string
	e := New(Config{
		Name:        "test-source",
		Source:      testSource{},
		Checkpoints: store,
		Dispatch: func(event Event, sessionID string) bool {
			if sessionID != "session-1" {
				t.Errorf("session id = %q", sessionID)
			}
			mu.Lock()
			got = append(got, event.Text)
			mu.Unlock()
			return true
		},
	})

	// Scan is deliberately driven directly here; production callers can use
	// Start for the same engine after wiring a discovering source.
	e.source = singleFileSource{ref: FileRef{Path: path, SessionID: "session-1"}}
	e.Scan(context.Background())
	e.Stop()

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("events = %v", got)
	}
	cp, ok, err := store.Get(context.Background(), "test-source", path)
	if err != nil || !ok || cp.Cursor != "8" {
		t.Fatalf("checkpoint = %+v, ok=%v, err=%v", cp, ok, err)
	}
}

func TestEngineRetriesLineAfterDispatchFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "updates.jsonl")
	if err := os.WriteFile(path, []byte("retry\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &testCheckpointStore{cp: make(map[string]Checkpoint)}
	accepted := false
	var got []string
	e := New(Config{
		Name:         "retry-source",
		Source:       singleFileSource{ref: FileRef{Path: path, SessionID: "session-1"}},
		Checkpoints:  store,
		TailInterval: time.Millisecond,
		Dispatch: func(event Event, _ string) bool {
			if !accepted {
				return false
			}
			got = append(got, event.Text)
			return true
		},
	})
	e.Scan(context.Background())
	accepted = true
	e.Scan(context.Background())
	e.Stop()

	if len(got) != 1 || got[0] != "retry" {
		t.Fatalf("expected failed line to be retried, got %v", got)
	}
	if cp, ok, err := store.Get(context.Background(), "retry-source", path); err != nil || !ok || cp.Cursor != "6" {
		t.Fatalf("checkpoint = %+v, ok=%v, err=%v", cp, ok, err)
	}
}

type singleFileSource struct{ ref FileRef }

func (s singleFileSource) DiscoverFiles(_ context.Context) ([]FileRef, error) {
	return []FileRef{s.ref}, nil
}

func (singleFileSource) DecodeLine(_ string, line []byte) ([]Event, error) {
	return testSource{}.DecodeLine("", line)
}
func (singleFileSource) CaptureAgentInfo(string, string) {}

func TestEngineStopsIdempotently(t *testing.T) {
	e := New(Config{Source: singleFileSource{ref: FileRef{Path: t.TempDir(), SessionID: "s"}}, PollInterval: time.Millisecond})
	e.Stop()
	e.Stop()
}

type snapshotSource struct {
	singleFileSource
	files []FileRef
	err   error
}

func (s *snapshotSource) DiscoverFiles(context.Context) ([]FileRef, error) {
	return s.files, s.err
}

func TestAuthoritativeDiscoveryStopsReadersAndResumesFromCheckpoint(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "retained.jsonl")
	if err := os.WriteFile(path, []byte("before\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ref := FileRef{Path: path, SessionID: "pane"}
	source := &snapshotSource{files: []FileRef{ref}}
	store := &testCheckpointStore{cp: make(map[string]Checkpoint)}
	var mu sync.Mutex
	var got []string
	e := New(Config{Name: "snapshot", Source: source, Checkpoints: store,
		StopUndiscovered: true, TailInterval: time.Hour,
		Dispatch: func(event Event, _ string) bool {
			mu.Lock()
			defer mu.Unlock()
			got = append(got, event.Text)
			return true
		},
	})
	defer e.Stop()
	e.Scan(ctx)
	e.mu.Lock()
	original := e.watchers[path]
	e.mu.Unlock()
	if original == nil {
		t.Fatal("initial reader missing")
	}

	// An unavailable inventory must not be interpreted as an empty inventory.
	source.files = nil
	source.err = errors.New("inventory unavailable")
	e.Scan(ctx)
	select {
	case <-original.done:
		t.Fatal("discovery failure stopped a retained reader")
	default:
	}
	source.err = nil
	e.Scan(ctx)
	select {
	case <-original.done:
	default:
		t.Fatal("successful empty snapshot left the reader running")
	}
	if cp, ok, err := store.Get(ctx, "snapshot", path); err != nil || !ok || cp.Cursor != "7" {
		t.Fatalf("stopping reader lost checkpoint: %+v, %v, %v", cp, ok, err)
	}
	// Data appended while excluded stays in the file for an explicit restore.
	if err := os.WriteFile(path, []byte("before\nafter\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	source.files = []FileRef{ref}
	e.Scan(ctx)
	e.Scan(ctx)
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 || got[0] != "before" || got[1] != "after" {
		t.Fatalf("restored projection lost or duplicated content: %v", got)
	}
	if cp, ok, err := store.Get(ctx, "snapshot", path); err != nil || !ok || cp.Cursor != "13" {
		t.Fatalf("restore did not advance checkpoint: %+v, %v, %v", cp, ok, err)
	}
}

func TestDefaultDiscoveryRetainsUndiscoveredReaders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "retained.jsonl")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	source := &snapshotSource{files: []FileRef{{Path: path, SessionID: "pane"}}}
	e := New(Config{Source: source, TailInterval: time.Hour})
	defer e.Stop()
	e.Scan(context.Background())
	e.mu.Lock()
	original := e.watchers[path]
	e.mu.Unlock()
	if original == nil {
		t.Fatal("initial reader missing")
	}
	source.files = nil
	e.Scan(context.Background())
	select {
	case <-original.done:
		t.Fatal("partial discovery unexpectedly stopped a reader")
	default:
	}
}
