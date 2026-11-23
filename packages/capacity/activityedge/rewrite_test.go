package activityedge

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRewriteNormalizesRequestsBeforeForwardingAndRejectsOnError(t *testing.T) {
	var calls atomic.Int32
	var forwarded atomic.Value
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		forwarded.Store(string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	h := New("rewrite-fixture", "127.0.0.1:0", strings.TrimPrefix(upstream.URL, "http://"), func(*http.Request) bool { return false })
	h.Rewrite = func(r *http.Request) (Completion, error) {
		if r.URL.Path == "/refuse" {
			return nil, errors.New("refused by owner")
		}
		r.Body = io.NopCloser(strings.NewReader("normalized"))
		r.ContentLength = int64(len("normalized"))
		return nil, nil
	}
	server, _, err := h.serverWithTrackerMode("127.0.0.1:0", upstream.URL, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	edge := httptest.NewServer(server.Handler)
	defer edge.Close()

	resp, err := http.Post(edge.URL+"/api/chat", "application/json", strings.NewReader("original"))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || forwarded.Load() != "normalized" {
		t.Fatalf("status=%s forwarded=%v, want the rewritten body", resp.Status, forwarded.Load())
	}

	resp, err = http.Post(edge.URL+"/refuse", "application/json", strings.NewReader("x"))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "refused by owner") {
		t.Fatalf("refusal status=%s body=%s", resp.Status, body)
	}
	if calls.Load() != 1 {
		t.Fatalf("upstream calls = %d, want only the accepted request", calls.Load())
	}
}

func TestRewriteCompletionReachesTheUpstreamDirectlyUntilDrained(t *testing.T) {
	var followUps atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/follow-up" {
			followUps.Add(1)
		}
		_, _ = io.WriteString(w, "ok")
	}))
	defer upstream.Close()

	h := New("rewrite-fixture", "127.0.0.1:0", strings.TrimPrefix(upstream.URL, "http://"), MatchPaths([]string{http.MethodPost}, "/work"))
	h.Exec = func(context.Context, string, ...string) ([]byte, error) { return nil, nil }
	results := make(chan error, 2)
	h.Rewrite = func(*http.Request) (Completion, error) {
		return func(u Upstream) {
			resp, err := u.Client.Post(u.BaseURL+"/follow-up", "application/json", strings.NewReader("{}"))
			if err == nil {
				_ = resp.Body.Close()
			}
			results <- err
		}, nil
	}
	server, _, err := h.serverWithTrackerMode("127.0.0.1:0", upstream.URL, time.Millisecond, false)
	if err != nil {
		t.Fatal(err)
	}
	edge := httptest.NewServer(server.Handler)
	defer edge.Close()

	resp, err := http.Post(edge.URL+"/work", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if err := <-results; err != nil || followUps.Load() != 1 {
		t.Fatalf("completion error = %v, follow-ups = %d; want one direct upstream request", err, followUps.Load())
	}

	if err := h.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	resp, err = http.Get(edge.URL + "/not-work")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if err := <-results; !errors.Is(err, errEdgeDrained) || followUps.Load() != 1 {
		t.Fatalf("completion while drained error = %v, follow-ups = %d; want errEdgeDrained and no request", err, followUps.Load())
	}
}
