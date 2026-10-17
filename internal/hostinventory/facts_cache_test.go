package hostinventory

import (
	"context"
	"encoding/json"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestHostFactsReaderWarmCacheAndBootInvalidation(t *testing.T) {
	var calls int32
	now := time.Unix(10, 0)
	r := &hostFactsReader{Path: t.TempDir() + "/facts.json", TTL: map[string]time.Duration{"platform": time.Hour}, BootID: func() string { return "boot-a" }, Now: func() time.Time { return now }, Probe: func(context.Context, string) (json.RawMessage, error) {
		atomic.AddInt32(&calls, 1)
		return json.RawMessage(`{"ok":true}`), nil
	}}
	if _, err := r.Read(context.Background(), "platform"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Read(context.Background(), "platform"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("warm cache calls = %d", calls)
	}
	r.BootID = func() string { return "boot-b" }
	if _, err := r.Read(context.Background(), "platform"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("boot invalidation calls = %d", calls)
	}
}

// TestHostFactsReaderSurvivesOrphanedLockfile is the regression for the
// 2026-08-27 incident: a process died between creating hostfacts.json.lock
// and removing it, and the O_EXCL protocol then made every cache miss on the
// host spin its full retry budget. A pre-existing lock file must not delay or
// block a reader.
func TestHostFactsReaderSurvivesOrphanedLockfile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/facts.json"
	if err := os.WriteFile(path+".lock", nil, 0o600); err != nil {
		t.Fatal(err)
	}
	var calls int32
	r := &hostFactsReader{Path: path, TTL: map[string]time.Duration{"platform": time.Hour}, BootID: func() string { return "boot-a" }, Probe: func(context.Context, string) (json.RawMessage, error) {
		atomic.AddInt32(&calls, 1)
		return json.RawMessage(`{"ok":true}`), nil
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	started := time.Now()
	if _, err := r.Read(ctx, "platform"); err != nil {
		t.Fatalf("Read with orphaned lockfile: %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("Read spun %s on an orphaned lockfile", elapsed)
	}
	if calls != 1 {
		t.Fatalf("probe calls = %d", calls)
	}
}

// TestHostFactsReaderCrossProcessLockWaitsAndReusesRefresh proves the lock
// seam: a reader that waits for the lock re-checks the cache before probing,
// so one refresh serves every waiter.
func TestHostFactsReaderCrossProcessLockWaitsAndReusesRefresh(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/facts.json"
	var calls int32
	locked := make(chan struct{})
	proceed := make(chan struct{})
	writer := &hostFactsReader{Path: path, TTL: map[string]time.Duration{"platform": time.Hour}, BootID: func() string { return "boot-a" }, Probe: func(context.Context, string) (json.RawMessage, error) {
		atomic.AddInt32(&calls, 1)
		close(locked)
		<-proceed
		return json.RawMessage(`{"ok":true}`), nil
	}}
	done := make(chan error, 1)
	go func() {
		_, err := writer.Read(context.Background(), "platform")
		done <- err
	}()
	<-locked
	close(proceed)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	// A second reader over the same file sees the stored value without probing.
	reader := &hostFactsReader{Path: path, TTL: map[string]time.Duration{"platform": time.Hour}, BootID: func() string { return "boot-a" }, Probe: func(context.Context, string) (json.RawMessage, error) {
		atomic.AddInt32(&calls, 1)
		return json.RawMessage(`{"ok":false}`), nil
	}}
	value, err := reader.Read(context.Background(), "platform")
	if err != nil {
		t.Fatal(err)
	}
	if string(value) != `{"ok":true}` {
		t.Fatalf("second reader value = %s, want the first refresh", value)
	}
	if calls != 1 {
		t.Fatalf("probe calls = %d, want one shared refresh", calls)
	}
}
