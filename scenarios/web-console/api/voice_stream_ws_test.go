package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// setupVoiceWSServer creates a test environment for voice streaming WebSocket tests.
// It returns the test HTTP server (for WS dialing) and the Server instance.
// The whisperHandler receives transcription requests and must respond with
// JSON {"text": "..."}.
func setupVoiceWSServer(t *testing.T, whisperHandler http.Handler) (*httptest.Server, *Server) {
	t.Helper()

	whisper := httptest.NewServer(whisperHandler)
	t.Cleanup(whisper.Close)

	origURL := whisperURL
	whisperURL = whisper.URL + "/asr?output=json"
	t.Cleanup(func() { whisperURL = origURL })

	// Bypass audio transcoding in tests so ffmpeg availability doesn't matter.
	origTranscode := transcodeAudio
	transcodeAudio = func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil }
	t.Cleanup(func() { transcodeAudio = origTranscode })

	srv := serverWithCapability(true)
	srv.router = mux.NewRouter()
	srv.router.HandleFunc("/api/v1/voice/stream", srv.handleVoiceStreamWS).Methods("GET")

	ts := httptest.NewServer(srv.router)
	t.Cleanup(ts.Close)
	return ts, srv
}

// echoWhisperHandler returns a Whisper mock that responds with the given text.
func echoWhisperHandler(text string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Consume the body so the pipe writer doesn't block
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": text})
	})
}

// countingWhisperHandler returns a Whisper mock that responds with
// "prefix1", "prefix2", etc. Each call produces unique text so that
// deduplicateOverlap doesn't suppress subsequent partial messages.
func countingWhisperHandler(prefix string) (*atomic.Int64, http.Handler) {
	var counter atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		n := counter.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": fmt.Sprintf("%s%d", prefix, n)})
	})
	return &counter, handler
}

// trackingWhisperHandler records the size and URL of each received audio payload.
// Each call returns "response1", "response2", etc. to avoid deduplication
// suppressing identical partial messages.
type trackingWhisperHandler struct {
	mu       sync.Mutex
	sizes    []int
	urls     []string
	counter  int
	response string
}

func (h *trackingWhisperHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("audio_file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	data, _ := io.ReadAll(file)
	file.Close()

	h.mu.Lock()
	h.sizes = append(h.sizes, len(data))
	h.urls = append(h.urls, r.URL.String())
	h.counter++
	text := fmt.Sprintf("%s%d", h.response, h.counter)
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"text": text})
}

func (h *trackingWhisperHandler) getSizes() []int {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]int, len(h.sizes))
	copy(out, h.sizes)
	return out
}

func (h *trackingWhisperHandler) getURLs() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, len(h.urls))
	copy(out, h.urls)
	return out
}

func TestVoiceStreamWS_RejectWhenUnavailable(t *testing.T) {
	srv := serverWithCapability(false)
	srv.router = mux.NewRouter()
	srv.setupRoutes()

	req := httptest.NewRequest("GET", "/api/v1/voice/stream", nil)
	rec := httptest.NewRecorder()
	srv.handleVoiceStreamWS(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for unavailable whisper, got %d", rec.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != "voice_unavailable" {
		t.Errorf("code = %q, want %q", resp.Code, "voice_unavailable")
	}
}

func TestVoiceStreamWS_BasicTranscription(t *testing.T) {
	ts, _ := setupVoiceWSServer(t, echoWhisperHandler("hello world"))

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send some audio data
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, 1024)); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	// Signal done
	if err := conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone}); err != nil {
		t.Fatalf("write done: %v", err)
	}

	// Read messages until we get a final
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read message: %v", err)
		}
		if msg.Type == VoiceMsgFinal {
			if msg.Text != "hello world" {
				t.Errorf("final text = %q, want %q", msg.Text, "hello world")
			}
			return
		}
		if msg.Type == VoiceMsgError {
			t.Fatalf("unexpected error: %s", msg.Text)
		}
		// Partial — continue reading
	}
}

func TestVoiceStreamWS_EmptyBuffer(t *testing.T) {
	ts, _ := setupVoiceWSServer(t, echoWhisperHandler(""))

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send done immediately with no audio
	if err := conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone}); err != nil {
		t.Fatalf("write done: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var msg VoiceStreamMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read message: %v", err)
	}
	if msg.Type != VoiceMsgFinal {
		t.Errorf("expected final, got type=%s", msg.Type)
	}
	if msg.Text != "" {
		t.Errorf("expected empty text for empty buffer, got %q", msg.Text)
	}
}

func TestVoiceStreamWS_PartialTranscripts(t *testing.T) {
	ts, _ := setupVoiceWSServer(t, echoWhisperHandler("partial text"))

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send enough audio data to exceed MinDeltaBytes and trigger a partial flush
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, 4096+1024)); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	// Wait for the flush interval to fire (1s + processing time)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var gotPartial bool
	for i := 0; i < 10; i++ {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			// Timeout or close — break
			break
		}
		if msg.Type == VoiceMsgPartial {
			gotPartial = true
			if msg.Text != "partial text" {
				t.Errorf("partial text = %q, want %q", msg.Text, "partial text")
			}
			break
		}
	}

	if !gotPartial {
		t.Error("expected at least one partial transcript")
	}

	// Cleanup: send done
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
}

func TestVoiceStreamWS_DeltaPartials(t *testing.T) {
	// Disable overlap and skip-final so this test isolates delta-only behaviour.
	tracker := &trackingWhisperHandler{response: "delta"}
	ts, srv := setupVoiceWSServer(t, tracker)

	cfg := srv.getVoiceConfig()
	cfg.OverlapBytes = 0
	srv.setVoiceConfig(cfg)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	batchSize := 4096 + 1024 // each batch exceeds MinDeltaBytes

	// Send 3 batches with pauses to allow ticks between them.
	for batch := 0; batch < 3; batch++ {
		if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, batchSize)); err != nil {
			t.Fatalf("write binary batch %d: %v", batch, err)
		}
		// Wait for the tick to fire and process
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			var msg VoiceStreamMessage
			if readErr := conn.ReadJSON(&msg); readErr != nil {
				t.Fatalf("read partial batch %d: %v", batch, readErr)
			}
			if msg.Type == VoiceMsgPartial {
				break
			}
		}
	}

	// Send done and wait for final
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		if msg.Type == VoiceMsgFinal || msg.Type == VoiceMsgError {
			break
		}
	}

	sizes := tracker.getSizes()
	// Expect at least 3 partial calls + 1 final call
	if len(sizes) < 4 {
		t.Fatalf("expected at least 4 Whisper calls (3 partial + 1 final), got %d", len(sizes))
	}

	// Each partial call should receive approximately batchSize (delta), not the full accumulated buffer
	for i := 0; i < 3; i++ {
		if sizes[i] > batchSize+1024 {
			t.Errorf("partial call %d sent %d bytes, want ~%d (delta only)", i, sizes[i], batchSize)
		}
	}

	// Final call should use the full buffer
	totalBytes := batchSize * 3
	finalSize := sizes[len(sizes)-1]
	if finalSize < totalBytes {
		t.Errorf("final call sent %d bytes, want >= %d (full buffer)", finalSize, totalBytes)
	}
}

func TestVoiceStreamWS_SkipsUnchangedTicks(t *testing.T) {
	var callCount atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "test"})
	})

	ts, _ := setupVoiceWSServer(t, handler)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send enough data to exceed MinDeltaBytes
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, 4096+1024)); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	// Wait for the first partial tick to fire
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatal("timed out waiting for partial")
		}
		if msg.Type == VoiceMsgPartial {
			break
		}
	}

	countAfterFirst := callCount.Load()

	// Wait another 3 flush intervals without sending new data
	time.Sleep(1500 * time.Millisecond)

	countAfterWait := callCount.Load()
	if countAfterWait > countAfterFirst {
		t.Errorf("Whisper called %d more time(s) with no new data", countAfterWait-countAfterFirst)
	}

	// Cleanup
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
}

func TestVoiceStreamWS_DefaultValues(t *testing.T) {
	defaults := DefaultVoiceStreamConfig()
	if defaults.FlushIntervalMs != 500 {
		t.Errorf("FlushIntervalMs = %d, want 500", defaults.FlushIntervalMs)
	}
	if defaults.MinDeltaBytes != 4096 {
		t.Errorf("MinDeltaBytes = %d, want 4096", defaults.MinDeltaBytes)
	}
	if defaults.OverlapBytes != 2048 {
		t.Errorf("OverlapBytes = %d, want 2048", defaults.OverlapBytes)
	}
}

func TestVoiceStreamWS_FlushInterval(t *testing.T) {
	tests := []struct {
		name       string
		intervalMs int
	}{
		{"100ms", 100},
		{"200ms", 200},
		{"1s", 1000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tracker := &trackingWhisperHandler{response: "flush-test"}
			ts, srv := setupVoiceWSServer(t, tracker)

			cfg := srv.getVoiceConfig()
			cfg.FlushIntervalMs = tc.intervalMs
			srv.setVoiceConfig(cfg)

			dialer := websocket.Dialer{}
			conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
			if err != nil {
				t.Fatalf("ws dial: %v", err)
			}
			defer conn.Close()

			// Send enough data to trigger a partial
			if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, 4096+1024)); err != nil {
				t.Fatalf("write: %v", err)
			}

			// Wait for partial within a generous deadline
			_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			start := time.Now()
			interval := time.Duration(tc.intervalMs) * time.Millisecond
			for {
				var msg VoiceStreamMessage
				if err := conn.ReadJSON(&msg); err != nil {
					t.Fatalf("read: %v", err)
				}
				if msg.Type == VoiceMsgPartial {
					elapsed := time.Since(start)
					// Partial should arrive within ~2x the interval (interval + processing)
					maxExpected := interval*2 + 500*time.Millisecond
					if elapsed > maxExpected {
						t.Errorf("partial arrived after %v, want within %v", elapsed, maxExpected)
					}
					break
				}
			}

			_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
		})
	}
}

func TestVoiceStreamWS_TranscodeSeamCalled(t *testing.T) {
	var transcodeCalls atomic.Int64
	origTranscode := transcodeAudio
	t.Cleanup(func() { transcodeAudio = origTranscode })

	ts, _ := setupVoiceWSServer(t, echoWhisperHandler("transcoded"))

	// Final transcription always runs (full retranscribe with transcode=true).

	// Override the passthrough installed by setupVoiceWSServer with a tracker.
	transcodeAudio = func(_ context.Context, audio []byte) ([]byte, error) {
		transcodeCalls.Add(1)
		return audio, nil
	}

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send enough data to trigger a partial (>= MinDeltaBytes)
	audioData := make([]byte, 4096+1024)
	if err := conn.WriteMessage(websocket.BinaryMessage, audioData); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	// Wait for partial tick
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgPartial {
			break
		}
	}

	// After partial, transcode should not have been called
	if calls := transcodeCalls.Load(); calls != 0 {
		t.Errorf("transcodeAudio called %d time(s) during partial, want 0", calls)
	}

	// Signal done
	if err := conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone}); err != nil {
		t.Fatalf("write done: %v", err)
	}

	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgFinal {
			break
		}
	}

	// After final, transcode should have been called exactly once
	if calls := transcodeCalls.Load(); calls != 1 {
		t.Errorf("transcodeAudio called %d time(s) for final, want 1", calls)
	}
}

func TestVoiceStreamWS_LanguageAutoDetect(t *testing.T) {
	var receivedURL string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "auto"})
	})

	ts, _ := setupVoiceWSServer(t, handler)

	// Connect without ?language= param (auto-detect mode)
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, 512)); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone}); err != nil {
		t.Fatalf("write done: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgFinal || msg.Type == VoiceMsgError {
			break
		}
	}

	if strings.Contains(receivedURL, "language=") {
		t.Errorf("Whisper URL %q should NOT contain language= for auto-detect", receivedURL)
	}
}

func TestVoiceStreamWS_LanguagePassthrough(t *testing.T) {
	var receivedURL string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "bonjour"})
	})

	ts, _ := setupVoiceWSServer(t, handler)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream?language=fr"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, 512)); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone}); err != nil {
		t.Fatalf("write done: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgFinal || msg.Type == VoiceMsgError {
			break
		}
	}

	if !strings.Contains(receivedURL, "language=fr") {
		t.Errorf("Whisper URL %q should contain language=fr", receivedURL)
	}
}

func TestLastNWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		n        int
		expected string
	}{
		{"empty string", "", 3, ""},
		{"fewer words than n", "hello world", 5, "hello world"},
		{"exactly n words", "one two three", 3, "one two three"},
		{"more than n words", "alpha beta gamma delta epsilon", 3, "gamma delta epsilon"},
		{"single word", "hello", 1, "hello"},
		{"n is zero", "hello world", 0, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := lastNWords(tc.input, tc.n)
			if got != tc.expected {
				t.Errorf("lastNWords(%q, %d) = %q, want %q", tc.input, tc.n, got, tc.expected)
			}
		})
	}
}

func TestVoiceStreamWS_PartialSkipsTranscode(t *testing.T) {
	var transcodeCalls atomic.Int64
	origTranscode := transcodeAudio
	t.Cleanup(func() { transcodeAudio = origTranscode })

	ts, _ := setupVoiceWSServer(t, echoWhisperHandler("partial-no-transcode"))

	// Final transcription always runs with transcode=true.

	// Override the passthrough with a call counter.
	transcodeAudio = func(_ context.Context, audio []byte) ([]byte, error) {
		transcodeCalls.Add(1)
		return audio, nil
	}

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send enough audio to exceed MinDeltaBytes
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, 4096+1024)); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	// Wait for partial
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgPartial {
			break
		}
	}

	// Transcode should NOT have been called for partial
	if calls := transcodeCalls.Load(); calls != 0 {
		t.Errorf("transcodeAudio called %d time(s) during partial, want 0", calls)
	}

	// Send done and wait for final
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgFinal {
			break
		}
	}

	// Transcode SHOULD have been called for final
	if calls := transcodeCalls.Load(); calls != 1 {
		t.Errorf("transcodeAudio called %d time(s) for final, want 1", calls)
	}
}

func TestVoiceStreamWS_InitialPromptPassthrough(t *testing.T) {
	tracker := &trackingWhisperHandler{response: "word1 word2 word3"}
	ts, _ := setupVoiceWSServer(t, tracker)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	batchSize := 4096 + 1024

	// Send first batch and wait for partial
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, batchSize)); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read first partial: %v", err)
		}
		if msg.Type == VoiceMsgPartial {
			break
		}
	}

	// Send second batch and wait for partial
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, batchSize)); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read second partial: %v", err)
		}
		if msg.Type == VoiceMsgPartial {
			break
		}
	}

	// Cleanup
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		if msg.Type == VoiceMsgFinal || msg.Type == VoiceMsgError {
			break
		}
	}

	urls := tracker.getURLs()
	if len(urls) < 2 {
		t.Fatalf("expected at least 2 Whisper calls, got %d", len(urls))
	}

	// First partial should NOT have initial_prompt
	if strings.Contains(urls[0], "initial_prompt") {
		t.Errorf("first partial URL %q should not contain initial_prompt", urls[0])
	}

	// Second partial SHOULD have initial_prompt with words from the mock response
	if !strings.Contains(urls[1], "initial_prompt=") {
		t.Errorf("second partial URL %q should contain initial_prompt", urls[1])
	}
}

func TestVoiceStreamWS_MinDeltaSize(t *testing.T) {
	// The first tick bypasses the MinDeltaBytes gate (eager first partial).
	// This test verifies that SUBSEQUENT ticks still enforce the gate.
	callCount, handler := countingWhisperHandler("test")

	ts, _ := setupVoiceWSServer(t, handler)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send enough data to trigger the eager first partial
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, 4096+1024)); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	// Wait for the first (eager) partial
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read first partial: %v", err)
		}
		if msg.Type == VoiceMsgPartial {
			break
		}
	}

	countAfterFirst := callCount.Load()

	// Now send a tiny drip (< MinDeltaBytes) — should NOT trigger another partial
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, 1024)); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	// Wait for a tick interval + margin
	time.Sleep(750 * time.Millisecond)

	if calls := callCount.Load(); calls > countAfterFirst {
		t.Errorf("Whisper called %d extra time(s) with delta < MinDeltaBytes after first partial",
			calls-countAfterFirst)
	}

	// Send enough to exceed MinDeltaBytes for the next partial
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, 4096)); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgPartial {
			break
		}
	}

	if calls := callCount.Load(); calls <= countAfterFirst {
		t.Error("Whisper was never called after exceeding MinDeltaBytes")
	}

	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
}

// --- Phase 3: Audio Overlap Tests ---

func TestVoiceStreamWS_OverlapIncluded(t *testing.T) {
	tracker := &trackingWhisperHandler{response: "overlap"}
	ts, srv := setupVoiceWSServer(t, tracker)

	cfg := srv.getVoiceConfig()
	cfg.OverlapBytes = 2048
	srv.setVoiceConfig(cfg)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	batch1Size := 4096 + 1024
	batch2Size := 4096 + 1024

	// Send batch1 and wait for partial
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, batch1Size)); err != nil {
		t.Fatalf("write batch1: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read partial 1: %v", err)
		}
		if msg.Type == VoiceMsgPartial {
			break
		}
	}

	// Send batch2 and wait for partial
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, batch2Size)); err != nil {
		t.Fatalf("write batch2: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read partial 2: %v", err)
		}
		if msg.Type == VoiceMsgPartial {
			break
		}
	}

	// Send done and wait for final
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		if msg.Type == VoiceMsgFinal || msg.Type == VoiceMsgError {
			break
		}
	}

	sizes := tracker.getSizes()
	if len(sizes) < 2 {
		t.Fatalf("expected at least 2 partial Whisper calls, got %d", len(sizes))
	}

	// Partial 1: no prior data to overlap from, so size ≈ batch1Size
	if sizes[0] < batch1Size-512 || sizes[0] > batch1Size+512 {
		t.Errorf("partial 1 size = %d, want ~%d (no overlap for first delta)", sizes[0], batch1Size)
	}

	// Partial 2: should include 2048 bytes of overlap + batch2Size
	expectedMin := 2048 + batch2Size - 512
	expectedMax := 2048 + batch2Size + 512
	if sizes[1] < expectedMin || sizes[1] > expectedMax {
		t.Errorf("partial 2 size = %d, want ~%d (overlap + delta)", sizes[1], 2048+batch2Size)
	}
}

func TestVoiceStreamWS_OverlapClampedToStart(t *testing.T) {
	tracker := &trackingWhisperHandler{response: "clamped"}
	ts, srv := setupVoiceWSServer(t, tracker)

	cfg := srv.getVoiceConfig()
	cfg.OverlapBytes = 1 << 20 // huge overlap — should clamp to buffer start
	srv.setVoiceConfig(cfg)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	batchSize := 4096 + 1024

	// Send one batch and wait for partial
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, batchSize)); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read partial: %v", err)
		}
		if msg.Type == VoiceMsgPartial {
			break
		}
	}

	sizes := tracker.getSizes()
	if len(sizes) < 1 {
		t.Fatal("expected at least 1 Whisper call")
	}
	// First delta has no prior data, so no overlap — size ≈ batchSize
	if sizes[0] < batchSize-512 || sizes[0] > batchSize+512 {
		t.Errorf("partial size = %d, want ~%d (clamped to start)", sizes[0], batchSize)
	}

	// Cleanup
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
}

func TestVoiceStreamWS_OverlapZeroDisabled(t *testing.T) {
	tracker := &trackingWhisperHandler{response: "no-overlap"}
	ts, srv := setupVoiceWSServer(t, tracker)

	cfg := srv.getVoiceConfig()
	cfg.OverlapBytes = 0
	srv.setVoiceConfig(cfg)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	batchSize := 4096 + 1024

	// Send 2 batches, waiting for each partial
	for batch := 0; batch < 2; batch++ {
		if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, batchSize)); err != nil {
			t.Fatalf("write batch %d: %v", batch, err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			var msg VoiceStreamMessage
			if err := conn.ReadJSON(&msg); err != nil {
				t.Fatalf("read partial %d: %v", batch, err)
			}
			if msg.Type == VoiceMsgPartial {
				break
			}
		}
	}

	sizes := tracker.getSizes()
	if len(sizes) < 2 {
		t.Fatalf("expected at least 2 partial calls, got %d", len(sizes))
	}

	// With overlap=0, each partial should be approximately batchSize
	for i := 0; i < 2; i++ {
		if sizes[i] < batchSize-512 || sizes[i] > batchSize+512 {
			t.Errorf("partial %d size = %d, want ~%d (no overlap)", i, sizes[i], batchSize)
		}
	}

	// Cleanup
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
}

// --- Phase 4: Final Transcription Tests ---

func TestVoiceStreamWS_AlwaysFullRetranscribe(t *testing.T) {
	tracker := &trackingWhisperHandler{response: "final-result"}
	ts, _ := setupVoiceWSServer(t, tracker)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send data < MinDeltaBytes and done immediately. No partials fire.
	// The full retranscribe should always run.
	dataSize := 4096 / 2
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, dataSize)); err != nil {
		t.Fatalf("write: %v", err)
	}

	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read final: %v", err)
		}
		if msg.Type == VoiceMsgFinal {
			break
		}
	}

	// Expect exactly 1 call: the full-buffer retranscribe (no partials, no tail)
	sizes := tracker.getSizes()
	if len(sizes) != 1 {
		t.Errorf("Whisper call count = %d, want 1 (full retranscribe only)", len(sizes))
	}
	if len(sizes) >= 1 && sizes[0] != dataSize {
		t.Errorf("final payload size = %d, want %d", sizes[0], dataSize)
	}
}

func TestVoiceStreamWS_FinalOverridesPartials(t *testing.T) {
	// Partials return different text than the full retranscribe.
	// Verifies the final message uses the retranscribe result, not accumulated partials.
	var callCount atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		n := callCount.Add(1)
		// First call (partial) returns "partial text"
		// Second call (full retranscribe) returns "correct final"
		text := "partial text"
		if n > 1 {
			text = "correct final"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": text})
	})

	ts, srv := setupVoiceWSServer(t, handler)

	cfg := srv.getVoiceConfig()
	cfg.FlushIntervalMs = 100
	srv.setVoiceConfig(cfg)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send enough data for one partial
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, 4096+1024)); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Wait for partial
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read partial: %v", err)
		}
		if msg.Type == VoiceMsgPartial {
			if !strings.Contains(msg.Text, "partial text") {
				t.Errorf("partial text = %q, want it to contain 'partial text'", msg.Text)
			}
			break
		}
	}

	// Send done, read final
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var finalText string
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read final: %v", err)
		}
		if msg.Type == VoiceMsgFinal {
			finalText = msg.Text
			break
		}
	}

	// Final should be from retranscribe, not from accumulated partials
	if finalText != "correct final" {
		t.Errorf("final text = %q, want %q (from full retranscribe, not partials)", finalText, "correct final")
	}
}

// --- Phase 3: Eager First Partial Tests ---

func TestVoiceStreamWS_EagerFirstPartial(t *testing.T) {
	var callCount atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "eager"})
	})

	ts, _ := setupVoiceWSServer(t, handler)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send audio SMALLER than MinDeltaBytes — should still trigger on first tick
	smallSize := 4096 / 4
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, smallSize)); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Wait for the first flush interval + margin
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatal("timed out waiting for eager first partial")
		}
		if msg.Type == VoiceMsgPartial {
			if msg.Text != "eager" {
				t.Errorf("partial text = %q, want %q", msg.Text, "eager")
			}
			break
		}
	}

	if calls := callCount.Load(); calls != 1 {
		t.Errorf("Whisper call count = %d, want 1 (eager first tick)", calls)
	}

	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
}

func TestVoiceStreamWS_EagerFirstPartialEmptyBuffer(t *testing.T) {
	var callCount atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "should-not-happen"})
	})

	ts, _ := setupVoiceWSServer(t, handler)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send zero bytes — currentLen == 0 blocks even on first tick
	time.Sleep(750 * time.Millisecond)

	if calls := callCount.Load(); calls != 0 {
		t.Errorf("Whisper called %d time(s) with empty buffer, want 0", calls)
	}

	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
}

func TestVoiceStreamWS_SubsequentTicksStillGated(t *testing.T) {
	var callCount atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "gated"})
	})

	ts, _ := setupVoiceWSServer(t, handler)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	smallSize := 4096 / 4

	// Send small chunk — triggers eager first partial
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, smallSize)); err != nil {
		t.Fatalf("write: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatal("timed out waiting for eager first partial")
		}
		if msg.Type == VoiceMsgPartial {
			break
		}
	}

	countAfterEager := callCount.Load()

	// Send another small chunk (< MinDeltaBytes) — should NOT trigger
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, smallSize)); err != nil {
		t.Fatalf("write: %v", err)
	}

	time.Sleep(750 * time.Millisecond)

	if calls := callCount.Load(); calls > countAfterEager {
		t.Errorf("Whisper called %d extra time(s) after eager partial with sub-MinDeltaBytes data",
			calls-countAfterEager)
	}

	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
}

// (Tail transcription and coverage-skip tests removed — the pipeline now
// always uses full retranscription for the final result.)

func TestVoiceStreamWS_FullFinalWhenNoPartials(t *testing.T) {
	tracker := &trackingWhisperHandler{response: "full-final"}
	ts, srv := setupVoiceWSServer(t, tracker)
	cfg := srv.getVoiceConfig()
	cfg.FlushIntervalMs = 5000
	srv.setVoiceConfig(cfg)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send small data, then done immediately. Force a long flush interval so
	// the test deterministically exercises the no-partials path even under
	// heavy suite load.
	dataSize := 1024
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, dataSize)); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgFinal {
			if !strings.Contains(msg.Text, "full-final") {
				t.Errorf("final text = %q, want it to contain %q", msg.Text, "full-final")
			}
			break
		}
	}

	// No partial tick should have fired, so only the final full retranscribe
	// should reach Whisper.
	sizes := tracker.getSizes()
	if len(sizes) != 1 {
		t.Errorf("Whisper call count = %d, want 1 (final retranscribe only)", len(sizes))
	}
}

// --- findWebMInitEnd tests ---

func TestFindWebMInitEnd_ValidWebM(t *testing.T) {
	// Simulate a WebM buffer: some header bytes followed by Cluster element ID.
	header := []byte{0x1A, 0x45, 0xDF, 0xA3, 0x00, 0x00, 0x00, 0x10} // EBML header (fake)
	clusterID := []byte{0x1F, 0x43, 0xB6, 0x75}
	audioData := make([]byte, 100)
	buf := append(append(header, clusterID...), audioData...)

	got := findWebMInitEnd(buf)
	want := len(header)
	if got != want {
		t.Errorf("findWebMInitEnd = %d, want %d", got, want)
	}
}

func TestFindWebMInitEnd_NoCluster(t *testing.T) {
	buf := []byte{0x1A, 0x45, 0xDF, 0xA3, 0x00, 0x00, 0x00, 0x10}
	got := findWebMInitEnd(buf)
	if got != 0 {
		t.Errorf("findWebMInitEnd = %d, want 0 (no cluster found)", got)
	}
}

// --- deduplicateOverlap tests ---

func TestDeduplicateOverlap(t *testing.T) {
	tests := []struct {
		name        string
		accumulated string
		newText     string
		want        string
	}{
		{
			name:        "no overlap",
			accumulated: "hello",
			newText:     "world",
			want:        "hello world",
		},
		{
			name:        "single word overlap",
			accumulated: "the quick brown",
			newText:     "brown fox",
			want:        "the quick brown fox",
		},
		{
			name:        "multi word overlap",
			accumulated: "alpha beta gamma delta",
			newText:     "gamma delta epsilon",
			want:        "alpha beta gamma delta epsilon",
		},
		{
			name:        "full overlap — newText adds nothing",
			accumulated: "hello world",
			newText:     "hello world",
			want:        "hello world",
		},
		{
			name:        "empty accumulated",
			accumulated: "",
			newText:     "hello world",
			want:        "hello world",
		},
		{
			name:        "empty newText",
			accumulated: "hello world",
			newText:     "",
			want:        "hello world",
		},
		{
			name:        "both empty",
			accumulated: "",
			newText:     "",
			want:        "",
		},
		{
			name:        "case insensitive merge",
			accumulated: "Hello",
			newText:     "hello world",
			want:        "Hello world",
		},
		{
			name:        "punctuation overlap",
			accumulated: "hello world,",
			newText:     "world foo",
			want:        "hello world, foo",
		},
		{
			name:        "case and punctuation combined",
			accumulated: "The Quick Brown,",
			newText:     "brown fox",
			want:        "The Quick Brown, fox",
		},
		{
			name:        "newText is suffix subset",
			accumulated: "one two three four",
			newText:     "three four",
			want:        "one two three four",
		},
		{
			name:        "long overlap at boundary",
			accumulated: "a b c d e f",
			newText:     "d e f g h",
			want:        "a b c d e f g h",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := deduplicateOverlap(tc.accumulated, tc.newText)
			if got != tc.want {
				t.Errorf("deduplicateOverlap(%q, %q) = %q, want %q",
					tc.accumulated, tc.newText, got, tc.want)
			}
		})
	}
}

func TestVoiceStreamWS_DeduplicationIntegration(t *testing.T) {
	// Mock Whisper returns overlapping text for partials, and correct text for final.
	// Verifies that partial messages show deduplicated text during recording,
	// and the final message comes from the full retranscribe (not partial assembly).
	var callCount atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		idx := int(callCount.Add(1)) - 1
		// Partials: overlapping text that exercises dedup
		// Final retranscribe: authoritative result
		responses := []string{"the quick brown", "brown fox jumps", "the quick brown fox jumps"}
		text := responses[min(idx, len(responses)-1)]
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": text})
	})

	ts, srv := setupVoiceWSServer(t, handler)

	cfg := srv.getVoiceConfig()
	cfg.FlushIntervalMs = 100
	srv.setVoiceConfig(cfg)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send two batches to trigger two partials.
	var lastPartialText string
	for batch := 0; batch < 2; batch++ {
		if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, 4096+512)); err != nil {
			t.Fatalf("write batch %d: %v", batch, err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			var msg VoiceStreamMessage
			if readErr := conn.ReadJSON(&msg); readErr != nil {
				t.Fatalf("read partial batch %d: %v", batch, readErr)
			}
			if msg.Type == VoiceMsgPartial {
				lastPartialText = msg.Text
				break
			}
		}
	}

	// Partial messages should have deduplicated "brown" (appears once)
	if count := strings.Count(lastPartialText, "brown"); count != 1 {
		t.Errorf("expected 'brown' once in partial, got %d times in %q", count, lastPartialText)
	}

	// Signal done and collect the final transcript.
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var finalText string
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read final: %v", err)
		}
		if msg.Type == VoiceMsgFinal {
			finalText = msg.Text
			break
		}
		if msg.Type == VoiceMsgError {
			t.Fatalf("unexpected error: %s", msg.Text)
		}
	}

	// Final should come from full retranscribe
	if finalText != "the quick brown fox jumps" {
		t.Errorf("final text = %q, want %q (from full retranscribe)", finalText, "the quick brown fox jumps")
	}
}

func TestFindWebMInitEnd_EmptyBuffer(t *testing.T) {
	got := findWebMInitEnd(nil)
	if got != 0 {
		t.Errorf("findWebMInitEnd(nil) = %d, want 0", got)
	}
}

func TestFindWebMInitEnd_ClusterAtStart(t *testing.T) {
	// Cluster ID right at offset 0 — init segment is empty.
	buf := []byte{0x1F, 0x43, 0xB6, 0x75, 0x00, 0x00}
	got := findWebMInitEnd(buf)
	if got != 0 {
		t.Errorf("findWebMInitEnd = %d, want 0", got)
	}
}

// TestVoiceStreamWS_PartialCancelledOnDone verifies that when the client sends
// "done", any in-flight partial transcription is cancelled so the final
// retranscribe isn't blocked by a slow partial.
func TestVoiceStreamWS_PartialCancelledOnDone(t *testing.T) {
	var callCount atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		if n == 1 {
			// First call (partial) — block until cancelled or 10s.
			select {
			case <-r.Context().Done():
				// Cancelled — return error so transcribeBytes sees a failure.
				http.Error(w, "cancelled", http.StatusServiceUnavailable)
				return
			case <-time.After(10 * time.Second):
				// Should not reach here.
			}
		}
		// Subsequent calls (final retranscribe) — respond immediately.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "final result"})
	})

	ts, srv := setupVoiceWSServer(t, handler)
	cfg := srv.getVoiceConfig()
	cfg.FlushIntervalMs = 100 // fast tick to ensure partial fires
	srv.setVoiceConfig(cfg)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send enough data to trigger a partial (> MinDeltaBytes).
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, 8192)); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Wait for the partial to start (mock will block on it).
	time.Sleep(300 * time.Millisecond)

	// Signal done — should cancel in-flight partial.
	start := time.Now()
	if err := conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone}); err != nil {
		t.Fatalf("write done: %v", err)
	}

	// Final should arrive quickly (not blocked by the 10s partial).
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgFinal {
			elapsed := time.Since(start)
			if elapsed > 3*time.Second {
				t.Errorf("final took %v, want < 3s (partial should have been cancelled)", elapsed)
			}
			if msg.Text != "final result" {
				t.Errorf("final text = %q, want %q", msg.Text, "final result")
			}
			return
		}
		// Skip any partial messages that arrived before cancellation.
	}
}

func TestTruncateForLog_Runes(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"ascii short", "hello", 10, "hello"},
		{"ascii exact", "hello", 5, "hello"},
		{"ascii over", "hello world", 5, "hello…"},
		{"multibyte under", "café", 10, "café"},
		{"multibyte truncate", "世界你好", 2, "世界…"},
		{"empty", "", 10, ""},
		{"zero max", "hello", 0, "…"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateForLog(tc.input, tc.maxLen)
			if got != tc.want {
				t.Errorf("truncateForLog(%q, %d) = %q, want %q", tc.input, tc.maxLen, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Whisper hallucination filter
// ---------------------------------------------------------------------------

func TestIsWhisperHallucination(t *testing.T) {
	positives := []string{
		"Thank you", "thank you", "THANK YOU", "Thank you.",
		"Thanks", "thanks.", "Thanks for watching.",
		"Thank you for watching", "Subscribe", "Bye.", "goodbye",
		"You", "The end.", "so", "...", "",
		// Exclamation and mixed punctuation variants
		"Thanks for watching!", "Thank you!", "Thank you for watching!",
		"Thank you very much", "Thank you very much.", "Thank you very much!",
		"Goodbye!", "Bye!",
		"Please subscribe", "Please subscribe.",
	}
	for _, s := range positives {
		if !isWhisperHallucination(s) {
			t.Errorf("expected hallucination for %q", s)
		}
	}

	negatives := []string{
		"Hello world", "Thank you for the great presentation",
		"Please subscribe to the newsletter and leave a comment",
		"I said goodbye to them", "The meeting starts at 3pm",
	}
	for _, s := range negatives {
		if isWhisperHallucination(s) {
			t.Errorf("unexpected hallucination for %q", s)
		}
	}
}

// ---------------------------------------------------------------------------
// VAD gating: partials suppressed during silence
// ---------------------------------------------------------------------------

func TestVoiceStreamWS_VadGatingSuppressesPartialsOnSilence(t *testing.T) {
	counter, handler := countingWhisperHandler("word")
	ts, _ := setupVoiceWSServer(t, handler)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Opt into VAD gating by signaling speech-start then speech-end.
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgVadSpeechStart})
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgVadSpeechEnd})

	// Send audio while VAD says "silent" — should NOT produce partials.
	_ = conn.WriteMessage(websocket.BinaryMessage, make([]byte, 8192))

	// Wait long enough for 2 ticker ticks (default FlushIntervalMs = 500).
	time.Sleep(1200 * time.Millisecond)

	// Whisper should not have been called during silence.
	if n := counter.Load(); n != 0 {
		t.Errorf("Whisper called %d time(s) during VAD silence, want 0", n)
	}

	// Now signal speech-start — partials should resume.
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgVadSpeechStart})
	_ = conn.WriteMessage(websocket.BinaryMessage, make([]byte, 8192))

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var gotPartial bool
	for i := 0; i < 10; i++ {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		if msg.Type == VoiceMsgPartial {
			gotPartial = true
			break
		}
	}
	if !gotPartial {
		t.Error("expected partial after vad-speech-start, got none")
	}

	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
}

// ---------------------------------------------------------------------------
// Hallucination filter: "Thank you" from silence is suppressed
// ---------------------------------------------------------------------------

func TestVoiceStreamWS_HallucinationFilteredFromFinal(t *testing.T) {
	// Whisper returns "Thank you" (common hallucination on silence).
	ts, _ := setupVoiceWSServer(t, echoWhisperHandler("Thank you"))

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	_ = conn.WriteMessage(websocket.BinaryMessage, make([]byte, 1024))
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgFinal {
			if msg.Text != "" {
				t.Errorf("final text = %q, want empty (hallucination filtered)", msg.Text)
			}
			return
		}
		if msg.Type == VoiceMsgError {
			t.Fatalf("unexpected error: %s", msg.Text)
		}
	}
}

// ---------------------------------------------------------------------------
// VAD speech-start trims leading silence from segment audio
// ---------------------------------------------------------------------------

func TestVoiceStreamWS_SpeechStartTrimsSilenceWithLookback(t *testing.T) {
	// Track the size of audio Whisper receives for segment-final transcription.
	tracker := &trackingWhisperHandler{response: "hello world"}
	ts, _ := setupVoiceWSServer(t, tracker)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// lookbackBytes = audioBitrateBps/8 * vadLookbackMs/1000 = 48000/8 * 600/1000 = 3600
	expectedLookback := audioBitrateBps / 8 * vadLookbackMs / 1000

	// 1. Send 8KB of "silence" audio (before speech).
	silenceAudio := make([]byte, 8192)
	_ = conn.WriteMessage(websocket.BinaryMessage, silenceAudio)

	// 2. Signal speech start — this should advance segmentStartOffset
	//    but keep a lookback margin to preserve speech onset.
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgVadSpeechStart})

	// 3. Send 4KB of "speech" audio.
	speechAudio := make([]byte, 4096)
	for i := range speechAudio {
		speechAudio[i] = byte(i % 256)
	}
	_ = conn.WriteMessage(websocket.BinaryMessage, speechAudio)

	// 4. Signal speech end and segment boundary.
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgVadSpeechEnd})
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgSegmentBoundary})

	// Wait for segment-final processing.
	time.Sleep(800 * time.Millisecond)

	// 5. Close the session.
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})

	// Read messages until final.
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		if msg.Type == VoiceMsgFinal {
			break
		}
	}

	// Verify: segment-final should include speech audio (4096) + lookback (3600)
	// = ~7696 bytes, NOT the full 12288 (all silence + speech). The lookback
	// ensures the beginning of speech isn't clipped while still trimming most
	// of the leading silence.
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	if len(tracker.sizes) == 0 {
		t.Fatal("expected at least one Whisper call")
	}
	segmentSize := tracker.sizes[0]
	expectedMax := 4096 + expectedLookback + 500 // small margin for framing
	expectedMin := 4096                          // at minimum, all speech audio
	if segmentSize < expectedMin {
		t.Errorf("segment-final too small: %d bytes < %d (speech audio lost)", segmentSize, expectedMin)
	}
	if segmentSize > expectedMax {
		t.Errorf("segment-final too large: %d bytes > %d (silence not trimmed)", segmentSize, expectedMax)
	}
	if segmentSize >= 8192+4096 {
		t.Errorf("segment-final received full buffer (%d bytes) — lookback trim is not working", segmentSize)
	}
}

func TestVoiceStreamWS_InterWordSilenceBouncesPreserveAudio(t *testing.T) {
	// Simulates counting "1 2 3 4 5" where each word has a brief silence gap.
	// Multiple vad-speech-start signals should NOT discard prior speech audio
	// within the same segment.
	tracker := &trackingWhisperHandler{response: "one two three four five"}
	ts, _ := setupVoiceWSServer(t, tracker)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	chunk := make([]byte, 2048)
	for i := range chunk {
		chunk[i] = byte(i % 256)
	}

	// Simulate 5 words with silence bounces between them:
	// speech → silence → speech → silence → ... → segment-boundary
	for word := 0; word < 5; word++ {
		_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgVadSpeechStart})
		_ = conn.WriteMessage(websocket.BinaryMessage, chunk) // "word" audio
		_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgVadSpeechEnd})
		_ = conn.WriteMessage(websocket.BinaryMessage, chunk[:512]) // brief silence
	}

	// Trigger segment-final
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgSegmentBoundary})
	time.Sleep(800 * time.Millisecond)

	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		if msg.Type == VoiceMsgFinal {
			break
		}
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	if len(tracker.sizes) == 0 {
		t.Fatal("expected at least one Whisper call")
	}

	// Total audio sent: 5 words × 2048 bytes + 5 silences × 512 bytes = 12800 bytes.
	// The first speech-start trims leading silence (none here, since we start with speech).
	// All subsequent speech-starts should NOT discard prior audio.
	// The segment-final should contain ALL the audio (≥ 12800 bytes).
	segmentSize := tracker.sizes[0]
	totalAudio := 5*2048 + 5*512 // 12800
	if segmentSize < totalAudio {
		t.Errorf("segment-final only has %d bytes, expected ≥ %d — inter-word speech-start signals are discarding audio", segmentSize, totalAudio)
	}
}

func TestVoiceStreamWS_StopMidSpeechInPersistentMode(t *testing.T) {
	// Simulates persistent mode where:
	// 1. First segment is delivered via segment-boundary
	// 2. User speaks more, then presses stop (done) without waiting for silence
	// The "done" handler should transcribe only the un-segmented tail audio
	// and return it as the final message.
	callIdx := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		callIdx++
		var text string
		switch callIdx {
		case 1:
			text = "segment one" // segment-final #0
		default:
			text = "tail speech" // final retranscription of un-segmented tail
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": text})
	})
	ts, _ := setupVoiceWSServer(t, handler)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	chunk := make([]byte, 4096)
	for i := range chunk {
		chunk[i] = byte(i % 256)
	}

	// Phase 1: First segment — speech, silence, segment-boundary.
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgVadSpeechStart})
	_ = conn.WriteMessage(websocket.BinaryMessage, chunk)
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgVadSpeechEnd})
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgSegmentBoundary})
	time.Sleep(500 * time.Millisecond)

	// Phase 2: More speech, then immediate stop (no silence/segment-boundary).
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgVadSpeechStart})
	_ = conn.WriteMessage(websocket.BinaryMessage, chunk)
	// User presses stop immediately — no vad-speech-end, no segment-boundary.
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})

	// Collect all server messages.
	var segmentFinals []string
	var finalText string
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		if msg.Type == VoiceMsgSegmentFinal {
			segmentFinals = append(segmentFinals, msg.Text)
		}
		if msg.Type == VoiceMsgFinal {
			finalText = msg.Text
			break
		}
	}

	// Verify: segment-final delivered the first segment.
	if len(segmentFinals) == 0 {
		t.Error("expected at least one segment-final message")
	}

	// Verify: final message contains the tail speech (not empty, not the full recording).
	if finalText == "" {
		t.Error("final message is empty — un-segmented tail speech was lost")
	}
	if finalText == "segment one" {
		t.Error("final message duplicates segment content instead of tail")
	}
}

// --- Speaker Verification WebSocket Tests ---

// fakeSpeakerVerificationServer creates a test HTTP server that simulates the
// speaker-verification resource. The matched parameter controls whether
// verify requests return a match or mismatch.
func fakeSpeakerVerificationServer(t *testing.T, matched bool, score float64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/verify" {
			// Consume multipart body
			_ = r.ParseMultipartForm(10 << 20)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(SpeakerVerificationResult{
				ProfileID: "default",
				Matched:   matched,
				Score:     score,
				Threshold: 0.85,
			})
			return
		}
		http.NotFound(w, r)
	}))
}

// setupVoiceWSServerWithSpeaker creates a test environment with speaker
// verification enabled. Returns test server, Server instance, and the
// fake speaker verification server.
func setupVoiceWSServerWithSpeaker(
	t *testing.T,
	whisperHandler http.Handler,
	matched bool,
	score float64,
	mode string,
) (*httptest.Server, *Server) {
	t.Helper()

	ts, srv := setupVoiceWSServer(t, whisperHandler)

	svSrv := fakeSpeakerVerificationServer(t, matched, score)
	t.Cleanup(svSrv.Close)

	srv.speakerVerification = &SpeakerVerificationResourceClient{
		BaseURL: svSrv.URL,
		Client:  svSrv.Client(),
	}
	srv.speakerVerificationConfig = SpeakerVerificationConfig{
		Enabled:                     true,
		ProfileID:                   "default",
		Threshold:                   0.85,
		Mode:                        mode,
		RejectBehavior:              "drop",
		FallbackWithoutVerification: false,
	}

	return ts, srv
}

func TestVoiceStreamWS_SpeakerVerification_SendsStatusOnConnect(t *testing.T) {
	ts, _ := setupVoiceWSServerWithSpeaker(t, echoWhisperHandler("hello"), true, 0.95, "filter")

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// First message should be speaker-status
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var msg VoiceStreamMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg.Type != VoiceMsgSpeakerStatus {
		t.Fatalf("expected speaker-status, got %s", msg.Type)
	}
	if !msg.Enabled {
		t.Error("expected enabled=true")
	}
	if !msg.ProfileConfigured {
		t.Error("expected profileConfigured=true")
	}
	if msg.ProfileID != "default" {
		t.Errorf("profileId = %q, want %q", msg.ProfileID, "default")
	}

	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
}

func TestVoiceStreamWS_SpeakerVerification_AcceptedFinal(t *testing.T) {
	ts, _ := setupVoiceWSServerWithSpeaker(t, echoWhisperHandler("accepted text"), true, 0.95, "filter")

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Read speaker-status
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var statusMsg VoiceStreamMessage
	_ = conn.ReadJSON(&statusMsg)

	// Send audio + done
	_ = conn.WriteMessage(websocket.BinaryMessage, make([]byte, 1024))
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})

	// Read until final — should contain transcription text (verification passed)
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgFinal {
			if msg.Text != "accepted text" {
				t.Errorf("final text = %q, want %q", msg.Text, "accepted text")
			}
			return
		}
		if msg.Type == VoiceMsgError {
			t.Fatalf("unexpected error: %s", msg.Text)
		}
	}
}

func TestVoiceStreamWS_SpeakerVerification_RejectedFinal(t *testing.T) {
	ts, _ := setupVoiceWSServerWithSpeaker(t, echoWhisperHandler("should not appear"), false, 0.3, "filter")

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Read speaker-status
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var statusMsg VoiceStreamMessage
	_ = conn.ReadJSON(&statusMsg)

	// Send audio + done
	_ = conn.WriteMessage(websocket.BinaryMessage, make([]byte, 1024))
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})

	// Read until final — should be empty (verification rejected)
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgFinal {
			if msg.Text != "" {
				t.Errorf("expected empty final for rejected speaker, got %q", msg.Text)
			}
			return
		}
		if msg.Type == VoiceMsgError {
			t.Fatalf("unexpected error: %s", msg.Text)
		}
	}
}

func TestVoiceStreamWS_SpeakerVerification_AdvisoryModeAllowsThrough(t *testing.T) {
	ts, _ := setupVoiceWSServerWithSpeaker(t, echoWhisperHandler("advisory text"), false, 0.3, "advisory")

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Read speaker-status
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var statusMsg VoiceStreamMessage
	_ = conn.ReadJSON(&statusMsg)

	// Send audio + done
	_ = conn.WriteMessage(websocket.BinaryMessage, make([]byte, 1024))
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})

	// Advisory mode: mismatch should still allow text through
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgFinal {
			if msg.Text != "advisory text" {
				t.Errorf("final text = %q, want %q", msg.Text, "advisory text")
			}
			return
		}
		if msg.Type == VoiceMsgError {
			t.Fatalf("unexpected error: %s", msg.Text)
		}
	}
}

func TestVoiceStreamWS_SpeakerVerification_SegmentRejected(t *testing.T) {
	ts, _ := setupVoiceWSServerWithSpeaker(t, echoWhisperHandler("segment text"), false, 0.3, "filter")

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Read speaker-status
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var statusMsg VoiceStreamMessage
	_ = conn.ReadJSON(&statusMsg)

	// Send enough audio to exceed minSpeakerVerifyBytes, trigger segment boundary
	_ = conn.WriteMessage(websocket.BinaryMessage, make([]byte, 16384))
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgSegmentBoundary})

	// Wait for segment-rejected message
	var gotSegmentRejected bool
	deadline := time.Now().Add(10 * time.Second)
	_ = conn.SetReadDeadline(deadline)
	for time.Now().Before(deadline) {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		if msg.Type == VoiceMsgSegmentRejected {
			gotSegmentRejected = true
			if msg.SegmentIndex != 0 {
				t.Errorf("segmentIndex = %d, want 0", msg.SegmentIndex)
			}
			break
		}
		if msg.Type == VoiceMsgSegmentFinal {
			t.Fatal("unexpected segment-final for rejected segment")
		}
	}

	if !gotSegmentRejected {
		t.Error("expected segment-rejected message for non-matching speaker")
	}

	// Cleanup
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
}

func TestVoiceStreamWS_SpeakerVerification_SegmentAccepted(t *testing.T) {
	ts, _ := setupVoiceWSServerWithSpeaker(t, echoWhisperHandler("accepted segment"), true, 0.95, "filter")

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Read speaker-status
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var statusMsg VoiceStreamMessage
	_ = conn.ReadJSON(&statusMsg)

	// Send enough audio to exceed minSpeakerVerifyBytes, trigger segment boundary
	_ = conn.WriteMessage(websocket.BinaryMessage, make([]byte, 16384))
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgSegmentBoundary})

	// Should get segment-accepted + segment-final
	var gotAccepted, gotSegmentFinal bool
	deadline := time.Now().Add(10 * time.Second)
	_ = conn.SetReadDeadline(deadline)
	for time.Now().Before(deadline) {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		if msg.Type == VoiceMsgSegmentAccepted {
			gotAccepted = true
			if msg.SegmentIndex != 0 {
				t.Errorf("accepted segmentIndex = %d, want 0", msg.SegmentIndex)
			}
		}
		if msg.Type == VoiceMsgSegmentFinal {
			gotSegmentFinal = true
			if msg.Text != "accepted segment" {
				t.Errorf("segment-final text = %q, want %q", msg.Text, "accepted segment")
			}
		}
		if gotAccepted && gotSegmentFinal {
			break
		}
		if msg.Type == VoiceMsgSegmentRejected {
			t.Fatal("unexpected segment-rejected for matching speaker")
		}
	}

	if !gotAccepted {
		t.Error("expected segment-accepted message for matching speaker")
	}
	if !gotSegmentFinal {
		t.Error("expected segment-final after accepted segment")
	}

	// Cleanup
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
}

func TestVoiceStreamWS_SpeakerVerification_DisabledNoStatusMessage(t *testing.T) {
	ts, _ := setupVoiceWSServer(t, echoWhisperHandler("no verification"))

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send audio + done
	_ = conn.WriteMessage(websocket.BinaryMessage, make([]byte, 1024))
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})

	// Should get partial/final but no speaker-status
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgSpeakerStatus {
			t.Fatal("unexpected speaker-status when verification is disabled")
		}
		if msg.Type == VoiceMsgFinal {
			if msg.Text != "no verification" {
				t.Errorf("final text = %q, want %q", msg.Text, "no verification")
			}
			return
		}
	}
}

func TestVoiceStreamWS_SpeakerVerification_FallbackPolicy(t *testing.T) {
	// Create a speaker verification server that always errors
	svSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(svSrv.Close)

	ts, srv := setupVoiceWSServer(t, echoWhisperHandler("fallback text"))

	srv.speakerVerification = &SpeakerVerificationResourceClient{
		BaseURL: svSrv.URL,
		Client:  svSrv.Client(),
	}
	// Fallback=false: should reject when verification errors
	srv.speakerVerificationConfig = SpeakerVerificationConfig{
		Enabled:                     true,
		ProfileID:                   "default",
		Threshold:                   0.85,
		Mode:                        "filter",
		RejectBehavior:              "drop",
		FallbackWithoutVerification: false,
	}

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	// Read speaker-status
	var statusMsg VoiceStreamMessage
	_ = conn.ReadJSON(&statusMsg)

	_ = conn.WriteMessage(websocket.BinaryMessage, make([]byte, 1024))
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})

	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgFinal {
			if msg.Text != "" {
				t.Errorf("expected empty final when fallback=false and resource errors, got %q", msg.Text)
			}
			return
		}
		if msg.Type == VoiceMsgError {
			t.Fatalf("unexpected error: %s", msg.Text)
		}
	}
}

func TestVoiceStreamWS_SpeakerVerification_FallbackAllowed(t *testing.T) {
	// Create a speaker verification server that always errors
	svSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(svSrv.Close)

	ts, srv := setupVoiceWSServer(t, echoWhisperHandler("fallback allowed"))

	srv.speakerVerification = &SpeakerVerificationResourceClient{
		BaseURL: svSrv.URL,
		Client:  svSrv.Client(),
	}
	// Fallback=true: should allow through when verification errors
	srv.speakerVerificationConfig = SpeakerVerificationConfig{
		Enabled:                     true,
		ProfileID:                   "default",
		Threshold:                   0.85,
		Mode:                        "filter",
		RejectBehavior:              "drop",
		FallbackWithoutVerification: true,
	}

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	// Read speaker-status
	var statusMsg VoiceStreamMessage
	_ = conn.ReadJSON(&statusMsg)

	_ = conn.WriteMessage(websocket.BinaryMessage, make([]byte, 1024))
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})

	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgFinal {
			if msg.Text != "fallback allowed" {
				t.Errorf("final text = %q, want %q", msg.Text, "fallback allowed")
			}
			return
		}
		if msg.Type == VoiceMsgError {
			t.Fatalf("unexpected error: %s", msg.Text)
		}
	}
}

func TestVoiceStreamWS_LegitTextNotFiltered(t *testing.T) {
	ts, _ := setupVoiceWSServer(t, echoWhisperHandler("Thank you for the great presentation"))

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	_ = conn.WriteMessage(websocket.BinaryMessage, make([]byte, 1024))
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read: %v", err)
		}
		if msg.Type == VoiceMsgFinal {
			if msg.Text != "Thank you for the great presentation" {
				t.Errorf("final text = %q, want %q", msg.Text, "Thank you for the great presentation")
			}
			return
		}
		if msg.Type == VoiceMsgError {
			t.Fatalf("unexpected error: %s", msg.Text)
		}
	}
}
