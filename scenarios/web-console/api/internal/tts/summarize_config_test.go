package tts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSummarizeConfig_LoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tts-summarize-config.json")

	cfg := SummarizeConfig{
		Enabled:        true,
		CharThreshold:  300,
		Level:          "heavy",
		Model:          "test-model",
		TimeoutSeconds: 45,
	}

	if err := SaveSummarizeConfig(path, cfg); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadSummarizeConfig(path)
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

func TestSummarizeConfig_ClampsUndersizedTimeout(t *testing.T) {
	// REGRESSION: a stale config with timeoutSeconds=5 caused every real-sized
	// summary request to time out before Ollama returned. The loader now
	// clamps anything below the minimum up to the default.
	dir := t.TempDir()
	path := filepath.Join(dir, "tts-summarize-config.json")
	if err := os.WriteFile(path, []byte(`{"enabled":true,"charThreshold":500,"level":"moderate","model":"qwen3:1.7b","timeoutSeconds":5}`), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	cfg, err := LoadSummarizeConfig(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.TimeoutSeconds < MinSummarizeTimeoutSeconds {
		t.Errorf("timeoutSeconds=%d, want clamped to >= %d", cfg.TimeoutSeconds, MinSummarizeTimeoutSeconds)
	}
}

func TestSummarizeConfig_DefaultsWhenMissing(t *testing.T) {
	cfg, err := LoadSummarizeConfig("/nonexistent/path/config.json")
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

func TestSummarizeConfig_PatchSemantics(t *testing.T) {
	base := DefaultSummarizeConfig()

	enabled := true
	patch := SummarizeConfigPatch{Enabled: &enabled}
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

func TestSummarizeConfig_LoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(path, []byte("{invalid json"), 0o644)

	cfg, err := LoadSummarizeConfig(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if cfg.Level != "moderate" {
		t.Errorf("expected default level on error, got %q", cfg.Level)
	}
}

func TestSummarizeConfig_LoadFillsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data, _ := json.Marshal(map[string]any{"enabled": true})
	_ = os.WriteFile(path, data, 0o644)

	cfg, err := LoadSummarizeConfig(path)
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

func TestDefaultSummarizeConfig_TimeoutSufficientForColdStart(t *testing.T) {
	cfg := DefaultSummarizeConfig()
	if cfg.TimeoutSeconds < 60 {
		t.Errorf("default timeout %ds is too short for Ollama cold model loads; need >= 60s",
			cfg.TimeoutSeconds)
	}
}
