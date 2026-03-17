package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

	req := httptest.NewRequest("GET", "/api/v1/tts/config", nil)
	rec := httptest.NewRecorder()
	srv.handleGetTTSConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var cfg TTSConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !cfg.AutoEnabled {
		t.Error("expected autoEnabled=true")
	}
}

func TestHandleUpdateTTSConfig(t *testing.T) {
	dir := t.TempDir()
	srv := newFakeTestServer()
	srv.ttsConfig = DefaultTTSConfig()
	srv.ttsConfigPath = filepath.Join(dir, "tts-config.json")

	body := strings.NewReader(`{"autoEnabled": true}`)
	req := httptest.NewRequest("PUT", "/api/v1/tts/config", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleUpdateTTSConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var cfg TTSConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !cfg.AutoEnabled {
		t.Error("expected autoEnabled=true after patch")
	}

	// Verify persisted to disk
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
