package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/tts"
)

func TestTTSSummarizeConfig_LoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tts-summarize-config.json")

	cfg := TTSSummarizeConfig{
		Enabled:        true,
		CharThreshold:  300,
		Level:          "heavy",
		Model:          "test-model",
		TimeoutSeconds: 45,
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

func TestTTSSummarizeConfig_ClampsUndersizedTimeout(t *testing.T) {
	// REGRESSION: a stale config with timeoutSeconds=5 (from before we
	// switched to reasoning models that emit 300+ <think> tokens) caused
	// every real-sized summary request to time out before Ollama returned.
	// The loader now clamps anything below the minimum up to the default.
	dir := t.TempDir()
	path := filepath.Join(dir, "tts-summarize-config.json")
	if err := os.WriteFile(path, []byte(`{"enabled":true,"charThreshold":500,"level":"moderate","model":"qwen3:1.7b","timeoutSeconds":5}`), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	cfg, err := loadTTSSummarizeConfig(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.TimeoutSeconds < minSummarizeTimeoutSeconds {
		t.Errorf("timeoutSeconds=%d, want clamped to >= %d", cfg.TimeoutSeconds, minSummarizeTimeoutSeconds)
	}
}

func TestTTSSummarizeConfig_DefaultsWhenMissing(t *testing.T) {
	cfg, err := loadTTSSummarizeConfig("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if !cfg.Enabled {
		t.Error("expected enabled by default (summarization is the happy path)")
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
	if cfg.TimeoutSeconds != 120 {
		t.Errorf("expected default timeoutSeconds 120, got %d", cfg.TimeoutSeconds)
	}
}

func TestHandleGetTTSSummarizeConfig(t *testing.T) {
	srv := newFakeTestServer()
	srv.ttsSummarizeConfig = DefaultTTSSummarizeConfig()

	cfg, err := callGetTTSSummarizeConfig(t, srv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GetLevel() != "moderate" {
		t.Errorf("expected default level moderate, got %q", cfg.GetLevel())
	}
}

func TestHandleUpdateTTSSummarizeConfig(t *testing.T) {
	dir := t.TempDir()
	srv := newFakeTestServer()
	srv.ttsSummarizeConfig = DefaultTTSSummarizeConfig()
	srv.ttsSummarizePath = filepath.Join(dir, "config.json")

	cfg, err := callUpdateTTSSummarizeConfig(t, srv, &ttsv1.UpdateSummarizeConfigRequest{
		Enabled: true, HasEnabled: true,
		CharThreshold: 200, HasCharThreshold: true,
		Level: "heavy", HasLevel: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.GetEnabled() {
		t.Error("expected enabled")
	}
	if cfg.GetCharThreshold() != 200 {
		t.Errorf("expected charThreshold 200, got %d", cfg.GetCharThreshold())
	}
	if cfg.GetLevel() != "heavy" {
		t.Errorf("expected level heavy, got %q", cfg.GetLevel())
	}
}

func TestHandleUpdateTTSSummarizeConfig_TimeoutAcceptsLargeValues(t *testing.T) {
	dir := t.TempDir()
	srv := newFakeTestServer()
	srv.ttsSummarizeConfig = DefaultTTSSummarizeConfig()
	srv.ttsSummarizePath = filepath.Join(dir, "config.json")

	cfg, err := callUpdateTTSSummarizeConfig(t, srv, &ttsv1.UpdateSummarizeConfigRequest{
		TimeoutSeconds: 120, HasTimeoutSeconds: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GetTimeoutSeconds() != 120 {
		t.Errorf("expected timeoutSeconds 120, got %d", cfg.GetTimeoutSeconds())
	}
}

func TestHandleUpdateTTSSummarizeConfig_TimeoutRejectsOver300(t *testing.T) {
	srv := newFakeTestServer()
	srv.ttsSummarizeConfig = DefaultTTSSummarizeConfig()

	_, err := callUpdateTTSSummarizeConfig(t, srv, &ttsv1.UpdateSummarizeConfigRequest{
		TimeoutSeconds: 301, HasTimeoutSeconds: true,
	})
	if connectCode(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v (err=%v)", connectCode(err), err)
	}
}

func TestDefaultTTSSummarizeConfig_TimeoutSufficientForColdStart(t *testing.T) {
	cfg := DefaultTTSSummarizeConfig()
	if cfg.TimeoutSeconds < 60 {
		t.Errorf("default timeout %ds is too short for Ollama cold model loads; need >= 60s",
			cfg.TimeoutSeconds)
	}
}

func TestHandleUpdateTTSSummarizeConfig_InvalidLevel(t *testing.T) {
	srv := newFakeTestServer()
	srv.ttsSummarizeConfig = DefaultTTSSummarizeConfig()

	_, err := callUpdateTTSSummarizeConfig(t, srv, &ttsv1.UpdateSummarizeConfigRequest{
		Level: "extreme", HasLevel: true,
	})
	if connectCode(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v (err=%v)", connectCode(err), err)
	}
}
