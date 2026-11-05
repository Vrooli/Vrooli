package opencode

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestHTTPClient_ListSessionsAndMessages(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/session", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `[{"id":"ses_a","directory":"/work","title":"t","time":{"created":1,"updated":2}}]`)
	})
	mux.HandleFunc("/session/ses_a/message", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `[{"info":{"id":"m1","role":"user","time":{"created":10}},"parts":[{"type":"text","text":"hi"}]}]`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewHTTPClient(srv.URL)
	ctx := context.Background()

	sessions, err := c.ListSessions(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].ID != "ses_a" || sessions[0].Directory != "/work" {
		t.Fatalf("unexpected sessions: %+v", sessions)
	}

	msgs, err := c.SessionMessages(ctx, "ses_a")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Info.Role != "user" || msgs[0].Parts[0].Text != "hi" {
		t.Fatalf("unexpected messages: %+v", msgs)
	}
}

func TestHTTPClient_ListSessionsScopesByDirectory(t *testing.T) {
	var mu sync.Mutex
	gotDirectory := ""
	mux := http.NewServeMux()
	mux.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotDirectory = r.URL.Query().Get("directory")
		mu.Unlock()
		fmt.Fprint(w, `[]`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewHTTPClient(srv.URL)
	if _, err := c.ListSessions(context.Background(), "/work dir"); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	directory := gotDirectory
	mu.Unlock()
	if directory != "/work dir" {
		t.Fatalf("directory query = %q, want %q", directory, "/work dir")
	}

	mu.Lock()
	gotDirectory = "unset"
	mu.Unlock()
	if _, err := c.ListSessions(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	directory = gotDirectory
	mu.Unlock()
	if directory != "" {
		t.Fatalf("empty directory must omit the query, got %q", directory)
	}
}

func TestHTTPClient_StatusAndRevert(t *testing.T) {
	var gotBody string
	mux := http.NewServeMux()
	mux.HandleFunc("/session/status", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `{"ses_a":{"type":"idle"}}`) })
	mux.HandleFunc("/session/ses_a/revert", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		fmt.Fprint(w, `true`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := NewHTTPClient(srv.URL)
	status, err := c.SessionStatus(context.Background())
	if err != nil || status["ses_a"].Type != "idle" {
		t.Fatalf("status = %+v, err = %v", status, err)
	}
	if err := c.RevertMessage(context.Background(), "ses_a", "msg_1", "part_1"); err != nil {
		t.Fatal(err)
	}
	if gotBody != `{"messageID":"msg_1","partID":"part_1"}` {
		t.Fatalf("revert body = %s", gotBody)
	}
}

func TestHTTPClient_EventsUnwrapsGlobalEnvelope(t *testing.T) {
	var mu sync.Mutex
	gotPath := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath = r.URL.Path
		mu.Unlock()
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("no flusher")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"directory\":\"/work\",\"project\":\"p\",\"payload\":{\"type\":\"session.created\",\"properties\":{\"sessionID\":\"ses_x\"}}}\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var mu2 sync.Mutex
	var got Event
	done := make(chan struct{})
	go func() {
		_ = c.Events(ctx, func(e Event) {
			mu2.Lock()
			got = e
			mu2.Unlock()
			cancel()
		})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Events did not return")
	}

	mu.Lock()
	path := gotPath
	mu.Unlock()
	if path != "/global/event" {
		t.Fatalf("stream path = %q, want /global/event", path)
	}
	mu2.Lock()
	defer mu2.Unlock()
	if got.Type != "session.created" || got.SessionID() != "ses_x" || got.Payload != nil {
		t.Fatalf("global event not unwrapped: %+v", got)
	}
}

func TestHTTPClient_EventsFallsBackToScopedStream(t *testing.T) {
	var mu sync.Mutex
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		if r.URL.Path == "/global/event" {
			http.NotFound(w, r)
			return
		}
		flusher, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"server.connected\",\"properties\":{}}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() {
		_ = c.Events(ctx, func(Event) { cancel() })
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Events did not return")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(paths) < 2 || paths[0] != "/global/event" || paths[1] != "/event" {
		t.Fatalf("expected global then scoped fallback, got %v", paths)
	}
}

func TestHTTPClient_EventsDecodesSSEFrames(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("no flusher")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"server.connected\",\"properties\":{}}\n\n")
		fmt.Fprint(w, "data: {\"type\":\"message.updated\",\"properties\":{\"sessionID\":\"ses_x\"}}\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var mu sync.Mutex
	var got []Event
	done := make(chan struct{})
	go func() {
		_ = c.Events(ctx, func(e Event) {
			mu.Lock()
			got = append(got, e)
			mu.Unlock()
			if e.Type == "message.updated" {
				cancel()
			}
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Events did not return")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(got) < 2 {
		t.Fatalf("expected at least 2 events, got %d: %+v", len(got), got)
	}
	if got[0].Type != "server.connected" {
		t.Fatalf("first event = %q", got[0].Type)
	}
	if got[1].SessionID() != "ses_x" {
		t.Fatalf("second event session = %q", got[1].SessionID())
	}
}
