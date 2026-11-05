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

func TestHostFactsFutureFetchIsNeverCacheFresh(t *testing.T) {
	for _, mode := range []string{"initial", "after-lock"} {
		t.Run(mode, func(t *testing.T) {
			now := time.Unix(100, 0)
			calls := 0
			r := &hostFactsReader{Path: t.TempDir() + "/facts.json", TTL: map[string]time.Duration{"gpu": time.Minute}, BootID: func() string { return "boot" }, Now: func() time.Time { return now }, Probe: func(context.Context, string) (json.RawMessage, error) {
				calls++
				return json.RawMessage(`{"current":true}`), nil
			}}
			future := hostFactsEntry{Schema: hostFactsSchemaVersion, BootID: "boot", FetchedAt: now.Add(time.Hour), Class: "gpu", Value: json.RawMessage(`{"future":true}`)}
			if mode == "initial" {
				if err := r.store("gpu", future); err != nil {
					t.Fatal(err)
				}
			} else {
				r.AcquireLock = func(context.Context, string) (func(), error) {
					if err := r.store("gpu", future); err != nil {
						t.Fatal(err)
					}
					return func() {}, nil
				}
			}
			got, err := r.Read(context.Background(), "gpu")
			if err != nil {
				t.Fatal(err)
			}
			if calls != 1 || string(got) != `{"current":true}` {
				t.Fatalf("future cache accepted mode=%s calls=%d value=%s", mode, calls, got)
			}
		})
	}
}

func TestHostFactsReaderRechecksClockAfterLockWait(t *testing.T) {
	now := time.Unix(100, 0)
	calls := 0
	r := &hostFactsReader{Path: t.TempDir() + "/facts.json", TTL: map[string]time.Duration{"gpu": time.Minute}, BootID: func() string { return "boot" }, Now: func() time.Time { return now }, Probe: func(context.Context, string) (json.RawMessage, error) {
		calls++
		return json.RawMessage(`{"extra_probe":true}`), nil
	}}
	r.AcquireLock = func(context.Context, string) (func(), error) {
		now = now.Add(10 * time.Second)
		if err := r.store("gpu", hostFactsEntry{Schema: hostFactsSchemaVersion, BootID: "boot", FetchedAt: now, Class: "gpu", Value: json.RawMessage(`{"refreshed_while_waiting":true}`)}); err != nil {
			t.Fatal(err)
		}
		return func() {}, nil
	}
	value, err := r.Read(context.Background(), "gpu")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 || string(value) != `{"refreshed_while_waiting":true}` {
		t.Fatalf("real waiter refresh was mistaken for future cache: probes=%d value=%s", calls, value)
	}
}

func TestHostFactsReaderRechecksClockAfterLocalMutexWait(t *testing.T) {
	for _, mode := range []string{"expired", "refreshed"} {
		t.Run(mode, func(t *testing.T) {
			var seconds atomic.Int64
			seconds.Store(100)
			reachedBeforeMutex := make(chan struct{})
			probes, locks := 0, 0
			r := &hostFactsReader{Path: t.TempDir() + "/facts.json", TTL: map[string]time.Duration{"gpu": time.Minute}, Now: func() time.Time { return time.Unix(seconds.Load(), 0) }, BootID: func() string { close(reachedBeforeMutex); return "boot" }, Probe: func(context.Context, string) (json.RawMessage, error) {
				probes++
				return json.RawMessage(`{"current":true}`), nil
			}, AcquireLock: func(context.Context, string) (func(), error) { locks++; return func() {}, nil }}
			entry := hostFactsEntry{Schema: hostFactsSchemaVersion, BootID: "boot", FetchedAt: time.Unix(100, 0), Class: "gpu", Value: json.RawMessage(`{"expired_after_wait":true}`)}
			if err := r.store("gpu", entry); err != nil {
				t.Fatal(err)
			}
			r.mu.Lock()
			type result struct {
				value json.RawMessage
				err   error
			}
			done := make(chan result, 1)
			go func() { value, err := r.Read(context.Background(), "gpu"); done <- result{value, err} }()
			// The reader has captured its pre-wait instant and cannot acquire the mutex.
			<-reachedBeforeMutex
			seconds.Store(220)
			if mode == "refreshed" {
				entry.FetchedAt = time.Unix(220, 0)
				entry.Value = json.RawMessage(`{"refreshed_while_waiting":true}`)
				if err := r.store("gpu", entry); err != nil {
					r.mu.Unlock()
					<-done
					t.Fatal(err)
				}
			}
			r.mu.Unlock()
			got := <-done
			if got.err != nil {
				t.Fatal(got.err)
			}
			if mode == "expired" {
				if probes != 1 || locks != 1 || string(got.value) != `{"current":true}` {
					t.Fatalf("expired local-wait data reused: probes=%d locks=%d value=%s", probes, locks, got.value)
				}
			} else if probes != 0 || locks != 0 || string(got.value) != `{"refreshed_while_waiting":true}` {
				t.Fatalf("fresh local-wait refresh not reused: probes=%d locks=%d value=%s", probes, locks, got.value)
			}
		})
	}
}
