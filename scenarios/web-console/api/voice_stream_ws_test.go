package main

import (
	"context"
	"encoding/json"
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

// trackingWhisperHandler records the size and URL of each received audio payload.
type trackingWhisperHandler struct {
	mu       sync.Mutex
	sizes    []int
	urls     []string
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
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"text": h.response})
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

	// Send enough audio data to exceed minDeltaSize and trigger a partial flush
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, minDeltaSize+1024)); err != nil {
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
	origOverlap := overlapBytes
	overlapBytes = 0
	t.Cleanup(func() { overlapBytes = origOverlap })

	origThreshold := skipFinalCoverageThreshold
	skipFinalCoverageThreshold = 2.0 // never skip — exceeds maximum possible coverage
	t.Cleanup(func() { skipFinalCoverageThreshold = origThreshold })

	tracker := &trackingWhisperHandler{response: "delta"}
	ts, _ := setupVoiceWSServer(t, tracker)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	batchSize := minDeltaSize + 1024 // each batch exceeds minDeltaSize

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

	// Send enough data to exceed minDeltaSize
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, minDeltaSize+1024)); err != nil {
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

func TestVoiceStreamFlushInterval(t *testing.T) {
	if voiceStreamFlushInterval != 500*time.Millisecond {
		t.Errorf("voiceStreamFlushInterval = %v, want 500ms", voiceStreamFlushInterval)
	}
}

func TestVoiceStreamFlushInterval_Overridable(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
	}{
		{"100ms", 100 * time.Millisecond},
		{"200ms", 200 * time.Millisecond},
		{"1s", 1 * time.Second},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			orig := voiceStreamFlushInterval
			voiceStreamFlushInterval = tc.interval
			t.Cleanup(func() { voiceStreamFlushInterval = orig })

			tracker := &trackingWhisperHandler{response: "flush-test"}
			ts, _ := setupVoiceWSServer(t, tracker)

			dialer := websocket.Dialer{}
			conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
			if err != nil {
				t.Fatalf("ws dial: %v", err)
			}
			defer conn.Close()

			// Send enough data to trigger a partial
			if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, minDeltaSize+1024)); err != nil {
				t.Fatalf("write: %v", err)
			}

			// Wait for partial within a generous deadline
			_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			start := time.Now()
			for {
				var msg VoiceStreamMessage
				if err := conn.ReadJSON(&msg); err != nil {
					t.Fatalf("read: %v", err)
				}
				if msg.Type == VoiceMsgPartial {
					elapsed := time.Since(start)
					// Partial should arrive within ~2x the interval (interval + processing)
					maxExpected := tc.interval*2 + 500*time.Millisecond
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
	// Ensure final transcription always runs so transcode is invoked.
	origThreshold := skipFinalCoverageThreshold
	skipFinalCoverageThreshold = 2.0
	t.Cleanup(func() { skipFinalCoverageThreshold = origThreshold })

	var transcodeCalls atomic.Int64
	origTranscode := transcodeAudio
	t.Cleanup(func() { transcodeAudio = origTranscode })

	ts, _ := setupVoiceWSServer(t, echoWhisperHandler("transcoded"))

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

	// Send enough data to trigger a partial (>= minDeltaSize)
	audioData := make([]byte, minDeltaSize+1024)
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
	// Ensure final transcription always runs so we can verify transcode is called for it.
	origThreshold := skipFinalCoverageThreshold
	skipFinalCoverageThreshold = 2.0
	t.Cleanup(func() { skipFinalCoverageThreshold = origThreshold })

	var transcodeCalls atomic.Int64
	origTranscode := transcodeAudio
	t.Cleanup(func() { transcodeAudio = origTranscode })

	ts, _ := setupVoiceWSServer(t, echoWhisperHandler("partial-no-transcode"))

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

	// Send enough audio to exceed minDeltaSize
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, minDeltaSize+1024)); err != nil {
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

	batchSize := minDeltaSize + 1024

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
	// The first tick bypasses the minDeltaSize gate (eager first partial).
	// This test verifies that SUBSEQUENT ticks still enforce the gate.
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

	// Send enough data to trigger the eager first partial
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, minDeltaSize+1024)); err != nil {
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

	// Now send a tiny drip (< minDeltaSize) — should NOT trigger another partial
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, 1024)); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	// Wait for a tick interval + margin
	time.Sleep(750 * time.Millisecond)

	if calls := callCount.Load(); calls > countAfterFirst {
		t.Errorf("Whisper called %d extra time(s) with delta < minDeltaSize after first partial",
			calls-countAfterFirst)
	}

	// Send enough to exceed minDeltaSize for the next partial
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, minDeltaSize)); err != nil {
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
		t.Error("Whisper was never called after exceeding minDeltaSize")
	}

	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
}

// --- Phase 3: Audio Overlap Tests ---

func TestVoiceStreamWS_OverlapIncluded(t *testing.T) {
	origOverlap := overlapBytes
	overlapBytes = 2048
	t.Cleanup(func() { overlapBytes = origOverlap })

	// Disable skip-final so we get a clean final call.
	origThreshold := skipFinalCoverageThreshold
	skipFinalCoverageThreshold = 2.0
	t.Cleanup(func() { skipFinalCoverageThreshold = origThreshold })

	tracker := &trackingWhisperHandler{response: "overlap"}
	ts, _ := setupVoiceWSServer(t, tracker)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	batch1Size := minDeltaSize + 1024
	batch2Size := minDeltaSize + 1024

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
	origOverlap := overlapBytes
	overlapBytes = 1 << 20 // huge overlap — should clamp to buffer start
	t.Cleanup(func() { overlapBytes = origOverlap })

	tracker := &trackingWhisperHandler{response: "clamped"}
	ts, _ := setupVoiceWSServer(t, tracker)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	batchSize := minDeltaSize + 1024

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
	origOverlap := overlapBytes
	overlapBytes = 0
	t.Cleanup(func() { overlapBytes = origOverlap })

	tracker := &trackingWhisperHandler{response: "no-overlap"}
	ts, _ := setupVoiceWSServer(t, tracker)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	batchSize := minDeltaSize + 1024

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

// --- Phase 4: Skip Final Transcription Tests ---

func TestVoiceStreamWS_SkipFinalWhenCoverageHigh(t *testing.T) {
	origThreshold := skipFinalCoverageThreshold
	skipFinalCoverageThreshold = 0.0 // always skip
	t.Cleanup(func() { skipFinalCoverageThreshold = origThreshold })

	// Disable overlap to simplify size tracking.
	origOverlap := overlapBytes
	overlapBytes = 0
	t.Cleanup(func() { overlapBytes = origOverlap })

	tracker := &trackingWhisperHandler{response: "partial-text"}
	ts, _ := setupVoiceWSServer(t, tracker)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	batchSize := minDeltaSize + 1024

	// Send 3 batches, waiting for each partial
	for batch := 0; batch < 3; batch++ {
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

	// Send done and read final
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

	// Final text should be the accumulated partials
	if !strings.Contains(finalText, "partial-text") {
		t.Errorf("final text = %q, want it to contain %q", finalText, "partial-text")
	}

	// Should be exactly 3 calls (partials only, no extra final call)
	sizes := tracker.getSizes()
	if len(sizes) != 3 {
		t.Errorf("Whisper call count = %d, want 3 (no extra final call)", len(sizes))
	}
}

func TestVoiceStreamWS_FullFinalWhenCoverageLow(t *testing.T) {
	origThreshold := skipFinalCoverageThreshold
	skipFinalCoverageThreshold = 2.0 // never skip — exceeds maximum possible coverage
	t.Cleanup(func() { skipFinalCoverageThreshold = origThreshold })

	tracker := &trackingWhisperHandler{response: "final-result"}
	ts, _ := setupVoiceWSServer(t, tracker)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send data < minDeltaSize and done immediately. The done message
	// arrives before the first tick fires, so no partials trigger.
	// However, the pipeline tail transcribes the un-processed audio.
	dataSize := minDeltaSize / 2
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

	// Expect: 1 tail call + 1 final full-buffer call = 2
	// (threshold is 2.0, so coverage check always fails → full re-transcription runs)
	sizes := tracker.getSizes()
	if len(sizes) != 2 {
		t.Errorf("Whisper call count = %d, want 2 (tail + final)", len(sizes))
	}
	// Final call should use the full buffer
	if len(sizes) >= 2 && sizes[1] != dataSize {
		t.Errorf("final payload size = %d, want %d", sizes[1], dataSize)
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

	// Send audio SMALLER than minDeltaSize — should still trigger on first tick
	smallSize := minDeltaSize / 4
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

	smallSize := minDeltaSize / 4

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

	// Send another small chunk (< minDeltaSize) — should NOT trigger
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, smallSize)); err != nil {
		t.Fatalf("write: %v", err)
	}

	time.Sleep(750 * time.Millisecond)

	if calls := callCount.Load(); calls > countAfterEager {
		t.Errorf("Whisper called %d extra time(s) after eager partial with sub-minDeltaSize data",
			calls-countAfterEager)
	}

	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})
}

// --- Phase 5: Pipeline Tail Transcription Tests ---

func TestVoiceStreamWS_PipelineTailTranscription(t *testing.T) {
	// Disable overlap to simplify tracking.
	origOverlap := overlapBytes
	overlapBytes = 0
	t.Cleanup(func() { overlapBytes = origOverlap })

	// Force full re-transcription so we can verify the tail is included.
	origThreshold := skipFinalCoverageThreshold
	skipFinalCoverageThreshold = 2.0
	t.Cleanup(func() { skipFinalCoverageThreshold = origThreshold })

	// Handler returns different text per call to distinguish partial vs tail vs final.
	var callCount atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		n := callCount.Add(1)
		text := "partial"
		if n > 1 {
			text = "tail"
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

	// Send enough for one partial
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, minDeltaSize+1024)); err != nil {
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

	// Send more audio (creates un-transcribed tail) then done immediately
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, minDeltaSize+1024)); err != nil {
		t.Fatalf("write tail audio: %v", err)
	}
	_ = conn.WriteJSON(VoiceStreamMessage{Type: VoiceMsgDone})

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg VoiceStreamMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read final: %v", err)
		}
		if msg.Type == VoiceMsgFinal {
			// The tail should have been transcribed (call count includes tail call)
			calls := callCount.Load()
			// Expect: 1 partial + 1 tail + 1 final = 3
			if calls < 3 {
				t.Errorf("Whisper call count = %d, want >= 3 (partial + tail + final)", calls)
			}
			break
		}
		if msg.Type == VoiceMsgError {
			t.Fatalf("unexpected error: %s", msg.Text)
		}
	}
}

func TestVoiceStreamWS_PipelineTailBoostsCoverage(t *testing.T) {
	// Disable overlap to simplify tracking.
	origOverlap := overlapBytes
	overlapBytes = 0
	t.Cleanup(func() { overlapBytes = origOverlap })

	// Set threshold high enough that partials alone won't reach it,
	// but partials + tail will.
	origThreshold := skipFinalCoverageThreshold
	skipFinalCoverageThreshold = 0.90
	t.Cleanup(func() { skipFinalCoverageThreshold = origThreshold })

	var callCount atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "word"})
	})

	ts, _ := setupVoiceWSServer(t, handler)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	batchSize := minDeltaSize + 1024

	// Send 3 batches and wait for each partial (covers ~75% of total with 4 batches)
	for batch := 0; batch < 3; batch++ {
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

	// Send 4th batch as tail (not waited for partial tick) then done
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, batchSize)); err != nil {
		t.Fatalf("write tail: %v", err)
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
		if msg.Type == VoiceMsgError {
			t.Fatalf("unexpected error: %s", msg.Text)
		}
	}

	// With 3 partials + 1 tail covering all 4 batches, coverage = 100% > 90%.
	// So the full re-transcription should be skipped.
	// Expect: 3 partials + 1 tail = 4 calls (no final full-buffer call)
	calls := callCount.Load()
	if calls > 4 {
		t.Errorf("Whisper call count = %d, want <= 4 (3 partials + 1 tail, no final)", calls)
	}
}

func TestVoiceStreamWS_PipelineTailEmpty(t *testing.T) {
	// Disable overlap to simplify.
	origOverlap := overlapBytes
	overlapBytes = 0
	t.Cleanup(func() { overlapBytes = origOverlap })

	// Force full final so we can count calls precisely.
	origThreshold := skipFinalCoverageThreshold
	skipFinalCoverageThreshold = 2.0
	t.Cleanup(func() { skipFinalCoverageThreshold = origThreshold })

	var callCount atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "result"})
	})

	ts, _ := setupVoiceWSServer(t, handler)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send one batch and wait for the partial tick to consume all audio
	if err := conn.WriteMessage(websocket.BinaryMessage, make([]byte, minDeltaSize+1024)); err != nil {
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

	countAfterPartial := callCount.Load()

	// Send done immediately — no new audio since last partial
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

	// Expect: 1 partial + 0 tail + 1 final = 2 (no tail call since tailLen == 0)
	finalCalls := callCount.Load()
	expectedCalls := countAfterPartial + 1 // only the final full-buffer call
	if finalCalls != expectedCalls {
		t.Errorf("Whisper call count = %d, want %d (no tail call when tail is empty)",
			finalCalls, expectedCalls)
	}
}

func TestVoiceStreamWS_FullFinalWhenNoPartials(t *testing.T) {
	tracker := &trackingWhisperHandler{response: "full-final"}
	ts, _ := setupVoiceWSServer(t, tracker)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/voice/stream"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send small data, then done immediately (before any tick fires).
	// Pipeline tail transcribes the un-processed audio, but with
	// default threshold the accumulated partial coverage (from tail
	// alone) reaches 100% and the full re-transcription is skipped.
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
			if msg.Text != "full-final" {
				t.Errorf("final text = %q, want %q", msg.Text, "full-final")
			}
			break
		}
	}

	// Tail pipeline transcribes the audio and coverage = 100% >= threshold,
	// so the full re-transcription is skipped. Expect: 1 call (tail only).
	sizes := tracker.getSizes()
	if len(sizes) != 1 {
		t.Errorf("Whisper call count = %d, want 1 (tail only, final skipped)", len(sizes))
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
