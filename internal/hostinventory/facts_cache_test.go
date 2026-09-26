package hostinventory

import (
	"context"
	"encoding/json"
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
