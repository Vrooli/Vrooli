package activityedge

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/modelretention"
)

// The edge is the only place that sees every request to a model store, so it
// owns real last-use and the request half of the eviction fence.
func TestModelStoreEdgeRecordsRequestsAndHoldsTheEvictionFence(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	started := make(chan string, 4)
	release := make(chan struct{})
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		started <- string(body)
		<-release
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	h, err := ForResource("ollama")
	if err != nil {
		t.Fatal(err)
	}
	h.Exec = func(context.Context, string, ...string) ([]byte, error) { return nil, nil }
	h.startModelGate()
	server, _, err := h.serverWithTrackerMode("127.0.0.1:0", upstream.URL, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	edge := httptest.NewServer(server.Handler)
	defer edge.Close()
	paths, err := modelretention.StorePaths("ollama")
	if err != nil {
		t.Fatal(err)
	}
	fence := modelretention.Fence{Dir: paths.FenceDir}

	done := make(chan int, 1)
	const body = `{"model":"llama3.2:1b","prompt":"hello"}`
	go func() {
		resp, err := http.Post(edge.URL+"/api/generate", "application/json", strings.NewReader(body))
		if err != nil {
			done <- 0
			return
		}
		_ = resp.Body.Close()
		done <- resp.StatusCode
	}()
	select {
	case got := <-started:
		if got != body {
			t.Fatalf("upstream body = %q, want the unchanged request %q", got, body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("request did not reach the upstream")
	}
	if _, err := fence.AcquireEviction("llama3.2:1b"); !errors.Is(err, modelretention.ErrModelInUse) {
		t.Fatalf("eviction during an in-flight request: got %v, want ErrModelInUse", err)
	}
	close(release)
	if status := <-done; status != http.StatusOK {
		t.Fatalf("in-flight request status = %d", status)
	}

	evict, err := fence.AcquireEviction("llama3.2:1b")
	if err != nil {
		t.Fatalf("eviction after the request ended: %v", err)
	}
	resp, err := http.Post(edge.URL+"/api/chat", "application/json", strings.NewReader(`{"model":"llama3.2:1b","messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable || calls.Load() != 1 {
		t.Fatalf("request during eviction: status %d upstream calls %d, want 503 and no forward", resp.StatusCode, calls.Load())
	}
	evict()

	// An unload names a model but is not use.
	unload, err := http.Post(edge.URL+"/api/generate", "application/json", strings.NewReader(`{"model":"qwen3:4b","keep_alive":0}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = unload.Body.Close()
	<-started

	h.stopModelGate()
	doc, err := modelretention.ReadUsage(paths.UsageFile)
	if err != nil {
		t.Fatal(err)
	}
	if doc.TrackingSince.IsZero() {
		t.Fatal("usage file must record when tracking began")
	}
	if entry := doc.Models["llama3.2:1b"]; entry.Requests != 1 || entry.LastRequestAt.IsZero() {
		t.Fatalf("llama3.2:1b usage = %+v, want one served request (the fenced one never entered)", entry)
	}
	if _, ok := doc.Models["qwen3:4b"]; ok {
		t.Fatal("an unload must not count as use")
	}
}

func TestModelStoreEdgeAttributesUseToTheConnectionCallerAndNotDeletes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	h, err := ForResource("ollama")
	if err != nil {
		t.Fatal(err)
	}
	h.Exec = func(context.Context, string, ...string) ([]byte, error) { return nil, nil }
	var identified atomic.Int32
	h.IdentifyCaller = func(local, remote net.Addr) string {
		identified.Add(1)
		return "scenario:demo-start-api/demo-api"
	}
	h.startModelGate()
	server, _, err := h.serverWithTrackerMode("127.0.0.1:0", upstream.URL, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	edge := httptest.NewUnstartedServer(server.Handler)
	edge.Config.ConnContext = server.ConnContext
	edge.Start()
	defer edge.Close()

	client := edge.Client() // one keep-alive connection
	for _, req := range []struct{ method, path, body string }{
		{http.MethodPost, "/v1/chat/completions", `{"model":"qwen3.5:9b","messages":[{"role":"user","content":"x"}]}`},
		{http.MethodPost, "/api/chat", `{"model":"qwen3.5:9b","messages":[{"role":"user","content":"y"}]}`},
		{http.MethodDelete, "/api/delete", `{"name":"qwen3:4b"}`},
	} {
		r, err := http.NewRequest(req.method, edge.URL+req.path, strings.NewReader(req.body))
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
	paths, err := modelretention.StorePaths("ollama")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	var entry modelretention.UsageEntry
	for time.Now().Before(deadline) {
		if err := h.modelGateFor().recorder.Flush(context.Background()); err != nil {
			t.Fatal(err)
		}
		doc, err := modelretention.ReadUsage(paths.UsageFile)
		if err != nil {
			t.Fatal(err)
		}
		entry = doc.Models["qwen3.5:9b"]
		if entry.Callers["scenario:demo-start-api/demo-api"].Requests == 2 {
			if _, deleted := doc.Models["qwen3:4b"]; deleted {
				t.Fatal("a delete must not count as use")
			}
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	h.stopModelGate()
	if entry.Requests != 2 || entry.Callers["scenario:demo-start-api/demo-api"].Requests != 2 {
		t.Fatalf("qwen3.5:9b usage = %+v, want 2 requests attributed to the demo caller", entry)
	}
	if identified.Load() != 1 {
		t.Fatalf("caller identified %d times, want once for the connection", identified.Load())
	}
}
