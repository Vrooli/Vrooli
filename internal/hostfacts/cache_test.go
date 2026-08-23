package hostfacts

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestReaderWarmCacheAndBootInvalidation(t *testing.T) {
	path := t.TempDir() + "/facts.json"
	var calls int32
	now := time.Unix(10, 0)
	r := &Reader{Path: path, TTL: map[string]time.Duration{"platform": time.Hour}, BootID: func() string { return "boot-a" }, Now: func() time.Time { return now }, Probe: func(context.Context, string) (json.RawMessage, error) {
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
		t.Fatalf("warm cache calls=%d", calls)
	}
	r.BootID = func() string { return "boot-b" }
	if _, err := r.Read(context.Background(), "platform"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("boot stale calls=%d", calls)
	}
}

func TestReaderConcurrentRefreshesSingleFlight(t *testing.T) {
	var calls int32
	r := &Reader{Path: t.TempDir() + "/facts.json", TTL: map[string]time.Duration{"gpu": time.Hour}, Probe: func(context.Context, string) (json.RawMessage, error) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(20 * time.Millisecond)
		return json.RawMessage(`1`), nil
	}}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = r.Read(context.Background(), "gpu") }()
	}
	wg.Wait()
	if calls != 1 {
		t.Fatalf("concurrent calls=%d", calls)
	}
}

func TestReaderKeepsIndependentFactClassesInOneFile(t *testing.T) {
	path := t.TempDir() + "/facts.json"
	var calls int32
	r := &Reader{Path: path, TTL: map[string]time.Duration{"inventory": time.Hour, "gpu": time.Hour}, BootID: func() string { return "boot-a" }, Now: func() time.Time { return time.Unix(10, 0) }, Probe: func(_ context.Context, class string) (json.RawMessage, error) {
		atomic.AddInt32(&calls, 1)
		return json.RawMessage(`{"class":"` + class + `"}`), nil
	}}
	if _, err := r.Read(context.Background(), "inventory"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Read(context.Background(), "gpu"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Read(context.Background(), "inventory"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("class refresh calls=%d, want two cold probes and a warm inventory read", calls)
	}
}

func TestIndependentReadersShareOneRefreshThroughFileLock(t *testing.T) {
	path := t.TempDir() + "/facts.json"
	var calls int32
	probe := func(context.Context, string) (json.RawMessage, error) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(40 * time.Millisecond)
		return json.RawMessage(`{"ok":true}`), nil
	}
	newReader := func() *Reader {
		return &Reader{Path: path, TTL: map[string]time.Duration{"inventory": time.Hour}, BootID: func() string { return "boot-a" }, Probe: probe}
	}
	var wg sync.WaitGroup
	for _, reader := range []*Reader{newReader(), newReader()} {
		wg.Add(1)
		go func(r *Reader) { defer wg.Done(); _, _ = r.Read(context.Background(), "inventory") }(reader)
	}
	wg.Wait()
	if calls != 1 {
		t.Fatalf("cross-reader refresh calls=%d, want one", calls)
	}
}
