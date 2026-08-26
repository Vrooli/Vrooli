package tailer

import (
	"context"
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
		Dispatch: func(event Event, sessionID string) {
			if sessionID != "session-1" {
				t.Errorf("session id = %q", sessionID)
			}
			mu.Lock()
			got = append(got, event.Text)
			mu.Unlock()
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
