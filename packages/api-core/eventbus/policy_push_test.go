package eventbus

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPolicyPushAtomicallyAppliesCompleteSnapshot(t *testing.T) {
	snapshotSent := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/policies/subscribe" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		fmt.Fprint(w, "event: snapshot\ndata: {\"version\":\"policy-v4\",\"receipt_capture_policies\":[{\"policy_id\":\"p\",\"enabled\":true,\"selector\":{\"target_scenario\":\"b\",\"operation\":\"POST /x\",\"protocol\":\"connect\",\"event_type\":\"vrooli.events.receipt.v1\"},\"response_projection_paths\":[\"id\"],\"version\":\"policy-v4\"}]}\n\n")
		flusher.Flush()
		close(snapshotSent)
		<-r.Context().Done()
	}))
	defer server.Close()

	cache := NewCache()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); _ = Client{BaseURL: server.URL}.consumePolicyStream(ctx, cache) }()
	<-snapshotSent
	deadline := time.Now().Add(time.Second)
	for {
		if version, _, ok := cache.Health(time.Now()); ok && version == "policy-v4" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("snapshot was not applied")
		}
		time.Sleep(time.Millisecond)
	}
	if projected, _, ok := cache.ProjectReceipt("a", "b", "POST /x", map[string]any{"id": "x"}); !ok || projected["id"] != "x" {
		t.Fatalf("pushed receipt projection was not applied: %#v, %t", projected, ok)
	}
	cancel()
	<-done
}
