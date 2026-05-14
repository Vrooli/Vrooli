package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/tts"

	"web-console/internal/capabilities"
)

func TestLoadTTSConfig_MissingFile(t *testing.T) {
	cfg, err := loadTTSConfig("/nonexistent/path/tts-config.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := DefaultTTSConfig()
	if cfg != want {
		t.Errorf("expected defaults %+v, got %+v", want, cfg)
	}
}

func TestTTSConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tts-config.json")

	cfg := TTSConfig{AutoEnabled: true, Backend: "kokoro", KokoroVoice: "af_heart", KokoroSpeed: 1.0}
	if err := saveTTSConfig(path, cfg); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := loadTTSConfig(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded != cfg {
		t.Errorf("expected %+v, got %+v", cfg, loaded)
	}
}

func TestTTSConfigPatch_PreservesUnsetFields(t *testing.T) {
	base := TTSConfig{AutoEnabled: true, Backend: "kokoro", KokoroVoice: "af_heart", KokoroSpeed: 1.0}
	// Empty patch — no fields set
	patch := TTSConfigPatch{}
	result := patch.Apply(base)
	if result.AutoEnabled != true {
		t.Error("expected AutoEnabled to remain true with empty patch")
	}
	if result.Backend != "kokoro" {
		t.Error("expected Backend to remain kokoro with empty patch")
	}
	if result.KokoroVoice != "af_heart" {
		t.Error("expected KokoroVoice to remain af_heart with empty patch")
	}
	if result.KokoroSpeed != 1.0 {
		t.Error("expected KokoroSpeed to remain 1.0 with empty patch")
	}
}

func TestTTSConfigPatch_AppliesField(t *testing.T) {
	base := TTSConfig{AutoEnabled: false, Backend: "browser", KokoroVoice: "af_heart", KokoroSpeed: 1.0}
	v := true
	backend := "kokoro"
	voice := "bf_emma"
	speed := 1.5
	patch := TTSConfigPatch{AutoEnabled: &v, Backend: &backend, KokoroVoice: &voice, KokoroSpeed: &speed}
	result := patch.Apply(base)
	if result.AutoEnabled != true {
		t.Error("expected AutoEnabled to be set to true")
	}
	if result.Backend != "kokoro" {
		t.Errorf("expected Backend kokoro, got %s", result.Backend)
	}
	if result.KokoroVoice != "bf_emma" {
		t.Errorf("expected KokoroVoice bf_emma, got %s", result.KokoroVoice)
	}
	if result.KokoroSpeed != 1.5 {
		t.Errorf("expected KokoroSpeed 1.5, got %f", result.KokoroSpeed)
	}
}

func TestHandleGetTTSConfig(t *testing.T) {
	srv := newFakeTestServer()
	srv.ttsConfig = TTSConfig{AutoEnabled: true, Backend: "kokoro", KokoroVoice: "af_heart", KokoroSpeed: 1.0}
	srv.ttsConfigPath = filepath.Join(t.TempDir(), "tts-config.json")

	cfg, err := callGetTTSConfig(t, srv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.GetAutoEnabled() {
		t.Error("expected autoEnabled=true")
	}
}

func TestHandleUpdateTTSConfig(t *testing.T) {
	dir := t.TempDir()
	srv := newFakeTestServer()
	srv.ttsConfig = DefaultTTSConfig()
	srv.ttsConfigPath = filepath.Join(dir, "tts-config.json")

	cfg, err := callUpdateTTSConfig(t, srv, &ttsv1.UpdateConfigRequest{
		AutoEnabled: true, HasAutoEnabled: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.GetAutoEnabled() {
		t.Error("expected autoEnabled=true after patch")
	}

	data, err := os.ReadFile(srv.ttsConfigPath)
	if err != nil {
		t.Fatalf("config file not written: %v", err)
	}
	var diskCfg TTSConfig
	if err := json.Unmarshal(data, &diskCfg); err != nil {
		t.Fatalf("invalid disk JSON: %v", err)
	}
	if !diskCfg.AutoEnabled {
		t.Error("expected disk config autoEnabled=true")
	}
}

func TestGetClaudeHookStatus_Registered(t *testing.T) {
	t.Setenv("API_PORT", "17086")
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(settingsPath, []byte(`{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "_id": "web-console-tts",
            "type": "http",
            "url": "http://localhost:17086/api/v1/hooks/stop",
            "headers": {
              "X-Hook-Token": "secret-token"
            },
            "timeout": 30
          }
        ]
      }
    ]
  }
}`), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	t.Setenv("CLAUDE_PROJECT_SETTINGS", settingsPath)
	srv := newFakeTestServer()
	srv.hookAuthToken = "secret-token"

	registered, code, reason, gotPath := srv.getClaudeHookStatus()
	if !registered {
		t.Fatalf("expected hook to be registered, got code=%s reason=%s", code, reason)
	}
	if code != "hook_registered" {
		t.Fatalf("expected hook_registered, got %s", code)
	}
	if gotPath != settingsPath {
		t.Fatalf("expected settings path %s, got %s", settingsPath, gotPath)
	}
}

func TestGetClaudeHookStatus_CommandHookRegistered(t *testing.T) {
	t.Setenv("API_PORT", "17086")
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(settingsPath, []byte(`{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "_id": "web-console-tts",
            "type": "command",
            "command": "bash /tmp/claude-stop-hook.sh --url http://localhost:17086/api/v1/hooks/stop --token secret-token",
            "timeout": 30
          }
        ]
      }
    ]
  }
}`), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	t.Setenv("CLAUDE_PROJECT_SETTINGS", settingsPath)
	srv := newFakeTestServer()
	srv.hookAuthToken = "secret-token"

	registered, code, reason, gotPath := srv.getClaudeHookStatus()
	if !registered {
		t.Fatalf("expected hook to be registered, got code=%s reason=%s", code, reason)
	}
	if code != "hook_registered" {
		t.Fatalf("expected hook_registered, got %s", code)
	}
	if gotPath != settingsPath {
		t.Fatalf("expected settings path %s, got %s", settingsPath, gotPath)
	}
}

func TestGetClaudeHookStatus_StaleToken(t *testing.T) {
	t.Setenv("API_PORT", "17086")
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(settingsPath, []byte(`{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "_id": "web-console-tts",
            "type": "http",
            "url": "http://localhost:17086/api/v1/hooks/stop",
            "headers": {
              "X-Hook-Token": "stale-token"
            },
            "timeout": 30
          }
        ]
      }
    ]
  }
}`), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	t.Setenv("CLAUDE_PROJECT_SETTINGS", settingsPath)
	srv := newFakeTestServer()
	srv.hookAuthToken = "fresh-token"

	registered, code, _, _ := srv.getClaudeHookStatus()
	if registered {
		t.Fatal("expected stale hook to be reported as not registered")
	}
	if code != "hook_stale" {
		t.Fatalf("expected hook_stale, got %s", code)
	}
}

func TestHandleGetTTSStatus_SeparatesHookAndTailerRoutingAndAck(t *testing.T) {
	srv := newFakeTestServer()
	srv.hookAuthToken = "secret-token"
	srv.ttsConfig = TTSConfig{AutoEnabled: true, Backend: "auto", KokoroVoice: "af_heart", KokoroSpeed: 1.0}
	srv.capabilities = capabilities.NewRegistry(capabilities.Known, map[string]capabilities.Checker{}, 0)
	srv.recordLastTTSRouting(ConversationAppendResult{
		Appended: true,
		Code:     "conversation_event_appended",
		Reason:   "hook appended",
		Source:   "claude_hook",
	})
	time.Sleep(10 * time.Millisecond)
	srv.recordLastTTSRouting(ConversationAppendResult{
		Appended: false,
		Code:     "conversation_target_missing",
		Reason:   "tailer skipped",
		Source:   "codex_tailer",
	})
	srv.recordTTSAck(TTSClientAck{
		EventID:   "evt-claude",
		Source:    "claude_hook",
		SessionID: "terminal-1",
		Stage:     "playback_succeeded",
		Backend:   "browser",
	})

	status, err := callGetTTSStatus(t, srv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.GetLastHookRouting() == nil || status.GetLastHookRouting().GetSource() != "claude_hook" {
		t.Fatalf("expected last hook routing, got %+v", status.GetLastHookRouting())
	}
	if status.GetLastTailerRouting() == nil || status.GetLastTailerRouting().GetSource() != "codex_tailer" {
		t.Fatalf("expected last tailer routing, got %+v", status.GetLastTailerRouting())
	}
	if status.GetLastHookAck() == nil || status.GetLastHookAck().GetSource() != "claude_hook" {
		t.Fatalf("expected last hook ack, got %+v", status.GetLastHookAck())
	}
}
