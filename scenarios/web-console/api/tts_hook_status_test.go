package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gorilla/mux"
)

// minimalHookServer wires only the state + handlers the hook-status tests
// exercise. Avoiding the full NewServer keeps the test hermetic against
// audio-tools connectivity and SQLite migrations.
func minimalHookServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "tts-hook-config.json")
	router := mux.NewRouter()
	srv := &Server{
		router:          router,
		lastTTSBySource: map[string]conversationAppendSnapshot{},
		lastTTSAckBySrc: map[string]ttsAckSnapshot{},
		ttsHookConfigState: hookConfigState{
			cfg:  DefaultTTSHookConfig(),
			path: cfgPath,
		},
	}
	// Override hookAuthToken so getClaudeHookStatus' file-read path is exercised
	// without poking real Claude settings.
	srv.hookAuthToken = "test-token"
	srv.registerTTSHookRoutes()
	return srv
}

func TestTTSHookStatus_GET_ReturnsDefaults(t *testing.T) {
	srv := minimalHookServer(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tts-hook/status", nil)
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp ttsHookStatusDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Config.Backend != "auto" {
		t.Errorf("expected default backend=auto, got %q", resp.Config.Backend)
	}
	if resp.Config.AutoEnabled {
		t.Errorf("expected default autoEnabled=false")
	}
	if !resp.Config.StartMuted {
		t.Errorf("expected default startMuted=true")
	}
}

func TestTTSHookStatus_PUTConfig_PersistsToDisk(t *testing.T) {
	srv := minimalHookServer(t)
	body, _ := json.Marshal(TTSHookConfigPatch{
		AutoEnabled: boolPtr(true),
		Backend:     stringPtr("kokoro"),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tts-hook/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var updated TTSHookConfig
	if err := json.Unmarshal(rr.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !updated.AutoEnabled || updated.Backend != "kokoro" {
		t.Errorf("patch not applied: %+v", updated)
	}
	// File on disk should reflect the patch — proves the persist seam.
	data, err := os.ReadFile(srv.ttsHookConfigState.path)
	if err != nil {
		t.Fatalf("read persisted: %v", err)
	}
	var onDisk TTSHookConfig
	if err := json.Unmarshal(data, &onDisk); err != nil {
		t.Fatalf("decode on-disk: %v", err)
	}
	if !onDisk.AutoEnabled || onDisk.Backend != "kokoro" {
		t.Errorf("disk state stale: %+v", onDisk)
	}
}

func TestTTSHookStatus_PUTConfig_RejectsBadBackend(t *testing.T) {
	srv := minimalHookServer(t)
	body, _ := json.Marshal(TTSHookConfigPatch{Backend: stringPtr("totallyfake")})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tts-hook/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	// Bad backend value silently drops; default remains.
	cfg := srv.getTTSHookConfig()
	if cfg.Backend != "auto" {
		t.Errorf("expected backend to stay auto, got %q", cfg.Backend)
	}
}

func TestTTSHookStatus_POSTAck_RecordsState(t *testing.T) {
	srv := minimalHookServer(t)
	body, _ := json.Marshal(ttsClientAckDTO{
		EventID:   "evt-1",
		Source:    "claude_hook",
		SessionID: "sess-1",
		Stage:     "playback_succeeded",
		Backend:   "kokoro",
		Message:   "ok",
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tts-hook/ack", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	ack, _ := srv.getLastTTSAckBySource("claude_hook")
	if ack == nil || ack.EventID != "evt-1" || ack.Stage != "playback_succeeded" {
		t.Errorf("ack not recorded: %+v", ack)
	}
}

func TestTTSHookStatus_POSTPlayback_RecordsState(t *testing.T) {
	srv := minimalHookServer(t)
	body, _ := json.Marshal(ttsPlaybackEventDTO{
		Source:  "settings_test",
		Stage:   "success",
		Backend: "browser",
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tts-hook/playback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	ev, _ := srv.getLastTTSPlaybackEvent()
	if ev == nil || ev.Stage != "success" || ev.Backend != "browser" {
		t.Errorf("playback not recorded: %+v", ev)
	}
}

func TestTTSHookStatus_RoutingState_FlowsToResponse(t *testing.T) {
	srv := minimalHookServer(t)
	srv.recordLastTTSRouting(ConversationAppendResult{
		Appended:  true,
		Code:      "tts_appended",
		Reason:    "fanout routed",
		Source:    "claude_hook",
		SessionID: "sess-7",
		EventID:   "evt-99",
		Sequence:  3,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tts-hook/status", nil)
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp ttsHookStatusDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.LastHookRouting == nil || resp.LastHookRouting.EventID != "evt-99" {
		t.Errorf("hook routing not surfaced: %+v", resp.LastHookRouting)
	}
}

func TestTTSHookStatus_ConcurrentACKs_DontDeadlock(t *testing.T) {
	srv := minimalHookServer(t)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body, _ := json.Marshal(ttsClientAckDTO{
				EventID: "evt-x",
				Source:  "claude_hook",
				Stage:   "ok",
			})
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tts-hook/ack", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			srv.router.ServeHTTP(rr, req)
		}(i)
	}
	wg.Wait()
}

func boolPtr(b bool) *bool       { return &b }
func stringPtr(s string) *string { return &s }
