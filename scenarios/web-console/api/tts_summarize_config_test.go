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

func TestTTSSummarizeConfig_LoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tts-summarize-config.json")

	cfg := TTSSummarizeConfig{
		Enabled:        true,
		CharThreshold:  300,
		Level:          "heavy",
		Model:          "test-model",
		TimeoutSeconds: 10,
	}

	if err := saveTTSSummarizeConfig(path, cfg); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := loadTTSSummarizeConfig(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.Enabled != cfg.Enabled {
		t.Errorf("enabled: got %v, want %v", loaded.Enabled, cfg.Enabled)
	}
	if loaded.CharThreshold != cfg.CharThreshold {
		t.Errorf("charThreshold: got %d, want %d", loaded.CharThreshold, cfg.CharThreshold)
	}
	if loaded.Level != cfg.Level {
		t.Errorf("level: got %q, want %q", loaded.Level, cfg.Level)
	}
	if loaded.Model != cfg.Model {
		t.Errorf("model: got %q, want %q", loaded.Model, cfg.Model)
	}
	if loaded.TimeoutSeconds != cfg.TimeoutSeconds {
		t.Errorf("timeoutSeconds: got %d, want %d", loaded.TimeoutSeconds, cfg.TimeoutSeconds)
	}
}

func TestTTSSummarizeConfig_DefaultsWhenMissing(t *testing.T) {
	cfg, err := loadTTSSummarizeConfig("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg.Enabled {
		t.Error("expected disabled by default")
	}
	if cfg.CharThreshold != 500 {
		t.Errorf("expected default charThreshold 500, got %d", cfg.CharThreshold)
	}
	if cfg.Level != "moderate" {
		t.Errorf("expected default level moderate, got %q", cfg.Level)
	}
}

func TestTTSSummarizeConfig_PatchSemantics(t *testing.T) {
	base := DefaultTTSSummarizeConfig()

	enabled := true
	patch := TTSSummarizeConfigPatch{Enabled: &enabled}
	result := patch.Apply(base)

	if !result.Enabled {
		t.Error("expected enabled after patch")
	}
	if result.CharThreshold != base.CharThreshold {
		t.Error("charThreshold should be preserved when not patched")
	}
	if result.Level != base.Level {
		t.Error("level should be preserved when not patched")
	}
}

func TestTTSSummarizeConfig_LoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(path, []byte("{invalid json"), 0o644)

	cfg, err := loadTTSSummarizeConfig(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	// Should return defaults on parse error
	if cfg.Level != "moderate" {
		t.Errorf("expected default level on error, got %q", cfg.Level)
	}
}

func TestTTSSummarizeConfig_LoadFillsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// Write minimal config with zero values
	data, _ := json.Marshal(map[string]any{"enabled": true})
	_ = os.WriteFile(path, data, 0o644)

	cfg, err := loadTTSSummarizeConfig(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.CharThreshold != 500 {
		t.Errorf("expected default charThreshold, got %d", cfg.CharThreshold)
	}
	if cfg.Level != "moderate" {
		t.Errorf("expected default level, got %q", cfg.Level)
	}
	if cfg.TimeoutSeconds != 30 {
		t.Errorf("expected default timeoutSeconds, got %d", cfg.TimeoutSeconds)
	}
}

func TestHandleGetTTSSummarizeConfig(t *testing.T) {
	srv := newFakeTestServer()
	srv.ttsSummarizeConfig = DefaultTTSSummarizeConfig()

	req := httptest.NewRequest("GET", "/api/v1/tts/summarize/config", nil)
	rec := httptest.NewRecorder()
	srv.handleGetTTSSummarizeConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var cfg TTSSummarizeConfig
	if err := json.NewDecoder(rec.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.Level != "moderate" {
		t.Errorf("expected default level moderate, got %q", cfg.Level)
	}
}

func TestHandleUpdateTTSSummarizeConfig(t *testing.T) {
	dir := t.TempDir()
	srv := newFakeTestServer()
	srv.ttsSummarizeConfig = DefaultTTSSummarizeConfig()
	srv.ttsSummarizePath = filepath.Join(dir, "config.json")

	body := strings.NewReader(`{"enabled":true,"charThreshold":200,"level":"heavy"}`)
	req := httptest.NewRequest("PUT", "/api/v1/tts/summarize/config", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleUpdateTTSSummarizeConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var cfg TTSSummarizeConfig
	if err := json.NewDecoder(rec.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !cfg.Enabled {
		t.Error("expected enabled")
	}
	if cfg.CharThreshold != 200 {
		t.Errorf("expected charThreshold 200, got %d", cfg.CharThreshold)
	}
	if cfg.Level != "heavy" {
		t.Errorf("expected level heavy, got %q", cfg.Level)
	}
}

func TestHandleUpdateTTSSummarizeConfig_InvalidLevel(t *testing.T) {
	srv := newFakeTestServer()
	srv.ttsSummarizeConfig = DefaultTTSSummarizeConfig()

	body := strings.NewReader(`{"level":"extreme"}`)
	req := httptest.NewRequest("PUT", "/api/v1/tts/summarize/config", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleUpdateTTSSummarizeConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
