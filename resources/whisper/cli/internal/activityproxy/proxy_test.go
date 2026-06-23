package activityproxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestIsTranscription(t *testing.T) {
	cases := []struct {
		method, path string
		want         bool
	}{
		{http.MethodPost, "/asr", true},
		{http.MethodPost, "/asr?task=transcribe", true},
		{http.MethodPost, "/detect-language", true},
		{http.MethodGet, "/asr", false}, // GET is not work
		{http.MethodGet, "/", false},    // health/docs
		{http.MethodPost, "/", false},
	}
	for _, c := range cases {
		r, _ := http.NewRequest(c.method, "http://x"+c.path, nil)
		if got := isTranscription(r); got != c.want {
			t.Errorf("isTranscription(%s %s) = %v, want %v", c.method, c.path, got, c.want)
		}
	}
}

func TestProxyForwardsTransparentlyAndBrackets(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "yes")
		_, _ = io.WriteString(w, "ok:"+r.URL.Path)
	}))
	defer upstream.Close()

	var mu sync.Mutex
	var states []string
	h := &Handlers{
		Stdout: io.Discard, Stderr: io.Discard,
		GetEnv:   func(string) string { return "" },
		Debounce: 30 * time.Millisecond,
		Exec: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if len(args) >= 2 && args[1] == "list" {
				return []byte(`{"claims":[{"claim_id":"clm-w","owner_id":"whisper"}]}`), nil
			}
			if len(args) >= 2 && args[1] == "activity" {
				mu.Lock()
				for i, a := range args {
					if a == "--state" && i+1 < len(args) {
						states = append(states, args[i+1])
					}
				}
				mu.Unlock()
			}
			return []byte(`{}`), nil
		},
	}
	srv, err := h.server("127.0.0.1:0", strings.TrimPrefix(upstream.URL, "http://"), 30*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	front := httptest.NewServer(srv.Handler)
	defer front.Close()

	// GET / is transparent and does NOT bracket.
	resp, err := http.Get(front.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "ok:/" || resp.Header.Get("X-Upstream") != "yes" {
		t.Fatalf("GET / not transparent: body=%q header=%q", body, resp.Header.Get("X-Upstream"))
	}
	if got := snapshot(&mu, &states); len(got) != 0 {
		t.Fatalf("GET / must not report activity; got %v", got)
	}

	// POST /asr is transparent AND brackets: active now, idle after debounce.
	resp, err = http.Post(front.URL+"/asr", "audio/wav", strings.NewReader("audio"))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "ok:/asr" {
		t.Fatalf("POST /asr not forwarded: body=%q", body)
	}

	if !waitForState(&mu, &states, "active", time.Second) {
		t.Fatalf("active never reported; states=%v", snapshot(&mu, &states))
	}
	if !waitForState(&mu, &states, "idle", time.Second) {
		t.Fatalf("idle never reported after debounce; states=%v", snapshot(&mu, &states))
	}
}

// Fail-open: when every capacity call errors, transcription still forwards.
func TestProxyFailOpenOnCapacityError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "transcribed")
	}))
	defer upstream.Close()

	h := &Handlers{
		Stdout: io.Discard, Stderr: io.Discard,
		GetEnv:   func(string) string { return "" },
		Debounce: 10 * time.Millisecond,
		Exec: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, context.DeadlineExceeded // broker down / vrooli missing
		},
	}
	srv, _ := h.server("127.0.0.1:0", strings.TrimPrefix(upstream.URL, "http://"), 10*time.Millisecond)
	front := httptest.NewServer(srv.Handler)
	defer front.Close()

	resp, err := http.Post(front.URL+"/asr", "audio/wav", strings.NewReader("audio"))
	if err != nil {
		t.Fatalf("request must succeed even when capacity is down: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "transcribed" {
		t.Fatalf("body=%q, want transcribed (fail-open)", body)
	}
}

func snapshot(mu *sync.Mutex, states *[]string) []string {
	mu.Lock()
	defer mu.Unlock()
	return append([]string(nil), (*states)...)
}

func waitForState(mu *sync.Mutex, states *[]string, want string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, s := range snapshot(mu, states) {
			if s == want {
				return true
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}
