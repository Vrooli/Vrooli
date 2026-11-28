package tailer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
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

type decodeFailureSource struct{ singleFileSource }

func (decodeFailureSource) DecodeLine(path string, line []byte) ([]Event, error) {
	if string(line) == "broken\n" {
		return nil, errors.New("temporary decode failure")
	}
	return testSource{}.DecodeLine(path, line)
}

type eofRecordTestSource struct{ ref FileRef }

func (s eofRecordTestSource) DiscoverFiles(context.Context) ([]FileRef, error) {
	return []FileRef{s.ref}, nil
}
func (eofRecordTestSource) DecodeLine(_ string, line []byte) ([]Event, error) {
	return []Event{{Role: "assistant", Text: string(line), Commit: true}}, nil
}
func (eofRecordTestSource) CaptureAgentInfo(string, string)            {}
func (eofRecordTestSource) CompleteEOFLine(_ string, line []byte) bool { return json.Valid(line) }

type documentTestSource struct{ ref FileRef }

func (s documentTestSource) DiscoverFiles(context.Context) ([]FileRef, error) {
	return []FileRef{s.ref}, nil
}
func (documentTestSource) DecodeLine(string, []byte) ([]Event, error) { return nil, nil }
func (documentTestSource) CaptureAgentInfo(string, string)            {}
func (documentTestSource) HandlesDocument(string) bool                { return true }
func (documentTestSource) DecodeDocument(_ string, document []byte) ([]Event, bool, error) {
	var value map[string]any
	if json.Unmarshal(document, &value) != nil {
		return nil, false, nil
	}
	return []Event{{Role: "assistant", Text: "complete document", Commit: true}}, true, nil
}

func TestEngineRetriesPartialWholeDocument(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "conversation.json")
	if err := os.WriteFile(path, []byte("{\"messages\":["), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &testCheckpointStore{cp: make(map[string]Checkpoint)}
	var got []string
	newEngine := func() *Engine {
		return New(Config{
			Name: "document-source", Source: documentTestSource{ref: FileRef{Path: path, SessionID: "session"}}, Checkpoints: store,
			Dispatch: func(event Event, _ string) bool { got = append(got, event.Text); return true },
		})
	}
	e := newEngine()
	e.Scan(ctx)
	e.Stop()
	if len(got) != 0 {
		t.Fatalf("partial document dispatched: %q", got)
	}
	if _, ok, _ := store.Get(ctx, "document-source", path); ok {
		t.Fatal("partial document advanced the checkpoint")
	}
	complete := []byte(`{"messages":[]}`)
	if err := os.WriteFile(path, complete, 0o600); err != nil {
		t.Fatal(err)
	}
	e = newEngine()
	e.Scan(ctx)
	e.Stop()
	if len(got) != 1 || got[0] != "complete document" {
		t.Fatalf("complete document events = %q", got)
	}
	if cp, ok, err := store.Get(ctx, "document-source", path); err != nil || !ok || cp.Cursor != strconv.Itoa(len(complete)) {
		t.Fatalf("document checkpoint = %+v, ok=%v, err=%v", cp, ok, err)
	}
}

func TestEngineCommitsCompleteEOFRecordAndRetriesPartialEOF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "message.json")
	partial := []byte(`{"info":`)
	if err := os.WriteFile(path, partial, 0o600); err != nil {
		t.Fatal(err)
	}
	store := &testCheckpointStore{cp: make(map[string]Checkpoint)}
	var got []string
	newEngine := func() *Engine {
		return New(Config{
			Name: "eof-source", Source: eofRecordTestSource{ref: FileRef{Path: path, SessionID: "session"}}, Checkpoints: store,
			Dispatch: func(event Event, _ string) bool { got = append(got, event.Text); return true },
		})
	}
	engine := newEngine()
	engine.Scan(context.Background())
	engine.Stop()
	if len(got) != 0 {
		t.Fatalf("partial EOF record dispatched: %q", got)
	}
	if _, ok, _ := store.Get(context.Background(), "eof-source", path); ok {
		t.Fatal("partial EOF record advanced the checkpoint")
	}
	if err := os.WriteFile(path, []byte(`{"info":{"role":"assistant"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	engine = newEngine()
	engine.Scan(context.Background())
	engine.Stop()
	if len(got) != 1 || !json.Valid([]byte(got[0])) {
		t.Fatalf("completed EOF record = %q", got)
	}
	checkpoint, ok, err := store.Get(context.Background(), "eof-source", path)
	if err != nil || !ok || checkpoint.Cursor != "29" {
		t.Fatalf("EOF checkpoint = %+v ok=%t err=%v", checkpoint, ok, err)
	}
}

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

func TestEngineStartScansBeforeFirstPollInterval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.jsonl")
	if err := os.WriteFile(path, []byte("already here\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	indexed := make(chan string, 1)
	e := New(Config{
		Name: "startup-scan-source", Source: singleFileSource{ref: FileRef{Path: path, SessionID: "session-1"}},
		PollInterval: time.Hour,
		Dispatch:     func(event Event, _ string) bool { indexed <- event.Text; return true },
	})
	e.Start()
	defer e.Stop()
	select {
	case got := <-indexed:
		if got != "already here" {
			t.Fatalf("initial event = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("initial scan waited for the recurring poll interval")
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

func TestEngineDoesNotCheckpointPastDecodeFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "updates.jsonl")
	if err := os.WriteFile(path, []byte("broken\nlater\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &testCheckpointStore{cp: make(map[string]Checkpoint)}
	var got []string
	newEngine := func() *Engine {
		return New(Config{
			Name: "decode-retry-source", Source: decodeFailureSource{singleFileSource{ref: FileRef{Path: path, SessionID: "session-1"}}}, Checkpoints: store,
			Dispatch: func(event Event, _ string) bool { got = append(got, event.Text); return true },
		})
	}
	engine := newEngine()
	engine.Scan(context.Background())
	engine.Stop()
	if len(got) != 0 {
		t.Fatalf("advanced past decode failure and dispatched later records: %v", got)
	}
	if _, ok, err := store.Get(context.Background(), "decode-retry-source", path); err != nil || ok {
		t.Fatalf("decode failure advanced checkpoint: ok=%t err=%v", ok, err)
	}

	// Repair the failed record and restart. Both it and the later
	// complete record must be projected; no later cursor may hide the repair.
	if err := os.WriteFile(path, []byte("repaired\nlater\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	engine = newEngine()
	engine.Scan(context.Background())
	engine.Stop()
	if len(got) != 2 || got[0] != "repaired" || got[1] != "later" {
		t.Fatalf("replayed records = %v", got)
	}
	if cp, ok, err := store.Get(context.Background(), "decode-retry-source", path); err != nil || !ok || cp.Cursor != "15" {
		t.Fatalf("checkpoint = %+v ok=%t err=%v", cp, ok, err)
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
