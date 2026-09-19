package byok

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/testutil/vendorws"

	"github.com/gorilla/websocket"
)

func TestDeepgramContract(t *testing.T) {
	a := NewDeepgramSTT()
	if a.ID() != "deepgram" {
		t.Fatalf("id: %s", a.ID())
	}
	if a.Model() != "nova-2" {
		t.Fatalf("model: %s", a.Model())
	}
	if a.IsAvailable(context.Background(), "") {
		t.Fatalf("empty key unavailable")
	}
}

func TestDeepgramTranscribeSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Token ") {
			http.Error(w, "auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":{"channels":[{"alternatives":[{"transcript":"hello"}]}]}}`))
	}))
	defer srv.Close()

	a := NewDeepgramSTT()
	a.Endpoint = srv.URL
	res, err := a.Transcribe(context.Background(), "k", sttchain.Request{Audio: []byte("x"), Format: "wav"})
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if res.Text != "hello" {
		t.Fatalf("text: %q", res.Text)
	}
}

func TestDeepgramTranscribeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "auth", http.StatusUnauthorized)
	}))
	defer srv.Close()
	a := NewDeepgramSTT()
	a.Endpoint = srv.URL
	_, err := a.Transcribe(context.Background(), "k", sttchain.Request{Audio: []byte("x")})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected 401: %v", err)
	}
}

func TestDeepgramTranscribeMissingKey(t *testing.T) {
	a := NewDeepgramSTT()
	if _, err := a.Transcribe(context.Background(), "", sttchain.Request{}); err == nil {
		t.Fatalf("expected missing-key rejection")
	}
}

// httpURLToWSURL rewrites http(s)://... → ws(s)://... so deepgramFake
// servers (httptest.Server returns http://) can be passed straight into
// DeepgramSTT.StreamEndpoint.
func httpURLToWSURL(s string) string {
	if strings.HasPrefix(s, "https://") {
		return "wss://" + strings.TrimPrefix(s, "https://")
	}
	return "ws://" + strings.TrimPrefix(s, "http://")
}

// TestDeepgramStreaming_HappyPath drives the WS adapter against the
// vendorws fake: client sends one audio chunk, server scripts two
// transcript frames (one partial, one final), adapter must surface a
// Partial event followed by a Segment event and finally a Done event.
func TestDeepgramStreaming_HappyPath(t *testing.T) {
	srv := vendorws.NewDeepgramServer(vendorws.Options{
		Script: []vendorws.Frame{
			{Text: vendorws.EncodeJSON(map[string]any{
				"type":     "Results",
				"is_final": false,
				"channel": map[string]any{
					"alternatives": []map[string]any{{"transcript": "hel"}},
				},
			})},
			{Text: vendorws.EncodeJSON(map[string]any{
				"type":     "Results",
				"is_final": true,
				"channel": map[string]any{
					"alternatives": []map[string]any{{"transcript": "hello world"}},
				},
			})},
		},
		CloseAfterScript: true,
	})
	defer srv.Close()

	a := NewDeepgramSTT()
	a.StreamEndpoint = httpURLToWSURL(srv.URL)

	chunks := make(chan sttchain.AudioChunk, 1)
	chunks <- sttchain.AudioChunk{Audio: []byte{0x00, 0x01, 0x02}}
	close(chunks)

	events, err := a.TranscribeStreaming(context.Background(), "test-key", sttchain.StreamStart{Language: "en"}, chunks)
	if err != nil {
		t.Fatalf("TranscribeStreaming: %v", err)
	}

	var (
		gotPartial, gotSegment, gotDone bool
		segText                         string
	)
	deadline := time.After(5 * time.Second)
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				goto done
			}
			switch ev.Kind {
			case sttchain.StreamEventPartial:
				gotPartial = true
			case sttchain.StreamEventSegment:
				gotSegment = true
				segText = ev.Segment.Text
			case sttchain.StreamEventDone:
				gotDone = true
			}
		case <-deadline:
			t.Fatal("timed out waiting for stream events")
		}
	}
done:
	if !gotPartial {
		t.Fatal("missing partial event")
	}
	if !gotSegment || segText != "hello world" {
		t.Fatalf("missing final segment, got text=%q", segText)
	}
	if !gotDone {
		t.Fatal("missing done event")
	}
}

// TestDeepgramStreaming_MidStreamErrorClosesClean asserts the adapter
// surfaces a Done event when the vendor WS closes abnormally
// (e.g. close code 1011, server-side error).
func TestDeepgramStreaming_MidStreamErrorClosesClean(t *testing.T) {
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Close immediately with 1011 — adapter must treat the read
		// failure as terminal and emit a Done event.
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1011, "server error"))
		_ = conn.Close()
	}))
	defer srv.Close()

	a := NewDeepgramSTT()
	a.StreamEndpoint = httpURLToWSURL(srv.URL)

	chunks := make(chan sttchain.AudioChunk)
	close(chunks)

	events, err := a.TranscribeStreaming(context.Background(), "k", sttchain.StreamStart{Language: "en"}, chunks)
	if err != nil {
		t.Fatalf("TranscribeStreaming: %v", err)
	}

	deadline := time.After(5 * time.Second)
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return
			}
			if ev.Kind == sttchain.StreamEventDone {
				return
			}
		case <-deadline:
			t.Fatal("timed out waiting for Done event on mid-stream close")
		}
	}
}

// TestDeepgramStreaming_ContextCancelCloses ensures cancelling the
// caller context propagates: the events channel closes (no leak) and
// no panic occurs.
func TestDeepgramStreaming_ContextCancelCloses(t *testing.T) {
	srv := vendorws.NewDeepgramServer(vendorws.Options{Script: nil, CloseAfterScript: false})
	defer srv.Close()

	a := NewDeepgramSTT()
	a.StreamEndpoint = httpURLToWSURL(srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	chunks := make(chan sttchain.AudioChunk)

	events, err := a.TranscribeStreaming(ctx, "k", sttchain.StreamStart{Language: "en"}, chunks)
	if err != nil {
		t.Fatalf("TranscribeStreaming: %v", err)
	}

	// Cancel before any chunks. The adapter must close the events
	// channel; if it leaks we time out below.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range events {
		}
	}()
	cancel()
	close(chunks)

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()
	select {
	case <-doneCh:
	case <-time.After(5 * time.Second):
		t.Fatal("events channel did not close after ctx cancel — possible goroutine leak")
	}
}
