package opencode

import (
	"context"
	"fmt"
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

	sessions, err := c.ListSessions(ctx)
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
