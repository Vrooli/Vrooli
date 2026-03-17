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

func TestDefaultVoiceStreamConfig(t *testing.T) {
	cfg := DefaultVoiceStreamConfig()
	if cfg.FlushIntervalMs != 500 {
		t.Errorf("FlushIntervalMs = %d, want 500", cfg.FlushIntervalMs)
	}
	if cfg.MinDeltaBytes != 4096 {
		t.Errorf("MinDeltaBytes = %d, want 4096", cfg.MinDeltaBytes)
	}
	if cfg.OverlapBytes != 2048 {
		t.Errorf("OverlapBytes = %d, want 2048", cfg.OverlapBytes)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("defaults should be valid, got: %v", err)
	}
}

func TestVoiceStreamConfig_Validate(t *testing.T) {
	valid := DefaultVoiceStreamConfig()

	tests := []struct {
		name    string
		modify  func(*VoiceStreamConfig)
		wantErr string
	}{
		// FlushIntervalMs bounds
		{"flush_at_min", func(c *VoiceStreamConfig) { c.FlushIntervalMs = 100 }, ""},
		{"flush_at_max", func(c *VoiceStreamConfig) { c.FlushIntervalMs = 5000 }, ""},
		{"flush_below_min", func(c *VoiceStreamConfig) { c.FlushIntervalMs = 99 }, "flushIntervalMs"},
		{"flush_above_max", func(c *VoiceStreamConfig) { c.FlushIntervalMs = 5001 }, "flushIntervalMs"},

		// MinDeltaBytes bounds
		{"delta_at_min", func(c *VoiceStreamConfig) { c.MinDeltaBytes = 512 }, ""},
		{"delta_at_max", func(c *VoiceStreamConfig) { c.MinDeltaBytes = 32768 }, ""},
		{"delta_below_min", func(c *VoiceStreamConfig) { c.MinDeltaBytes = 511 }, "minDeltaBytes"},
		{"delta_above_max", func(c *VoiceStreamConfig) { c.MinDeltaBytes = 32769 }, "minDeltaBytes"},

		// OverlapBytes bounds
		{"overlap_at_min", func(c *VoiceStreamConfig) { c.OverlapBytes = 0 }, ""},
		{"overlap_at_max", func(c *VoiceStreamConfig) { c.OverlapBytes = 16384 }, ""},
		{"overlap_below_min", func(c *VoiceStreamConfig) { c.OverlapBytes = -1 }, "overlapBytes"},
		{"overlap_above_max", func(c *VoiceStreamConfig) { c.OverlapBytes = 16385 }, "overlapBytes"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := valid // copy
			tc.modify(&cfg)
			err := cfg.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Errorf("expected valid, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error %q should contain %q", err.Error(), tc.wantErr)
				}
			}
		})
	}
}

func TestVoiceStreamConfig_PatchApply(t *testing.T) {
	base := DefaultVoiceStreamConfig()

	t.Run("partial_update", func(t *testing.T) {
		flush := 200
		patch := VoiceStreamConfigPatch{FlushIntervalMs: &flush}
		result := patch.Apply(base)
		if result.FlushIntervalMs != 200 {
			t.Errorf("FlushIntervalMs = %d, want 200", result.FlushIntervalMs)
		}
		// Other fields unchanged
		if result.MinDeltaBytes != base.MinDeltaBytes {
			t.Errorf("MinDeltaBytes changed: %d != %d", result.MinDeltaBytes, base.MinDeltaBytes)
		}
		if result.OverlapBytes != base.OverlapBytes {
			t.Errorf("OverlapBytes changed: %d != %d", result.OverlapBytes, base.OverlapBytes)
		}
	})

	t.Run("full_update", func(t *testing.T) {
		flush := 300
		delta := 8192
		overlap := 4096
		patch := VoiceStreamConfigPatch{
			FlushIntervalMs: &flush,
			MinDeltaBytes:   &delta,
			OverlapBytes:    &overlap,
		}
		result := patch.Apply(base)
		if result.FlushIntervalMs != 300 || result.MinDeltaBytes != 8192 ||
			result.OverlapBytes != 4096 {
			t.Errorf("unexpected result: %+v", result)
		}
	})

	t.Run("empty_patch", func(t *testing.T) {
		patch := VoiceStreamConfigPatch{}
		result := patch.Apply(base)
		if result != base {
			t.Errorf("empty patch should not change config: %+v != %+v", result, base)
		}
	})
}

func TestVoiceConfig_FileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "voice-config.json")

	cfg := VoiceStreamConfig{
		FlushIntervalMs: 200,
		MinDeltaBytes:   8192,
		OverlapBytes:    1024,
	}

	if err := saveVoiceConfig(path, cfg); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := loadVoiceConfig(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded != cfg {
		t.Errorf("round-trip mismatch: got %+v, want %+v", loaded, cfg)
	}
}

func TestVoiceConfig_LoadMissing(t *testing.T) {
	t.Run("missing_file", func(t *testing.T) {
		dir := t.TempDir()
		cfg, err := loadVoiceConfig(filepath.Join(dir, "nonexistent.json"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg != DefaultVoiceStreamConfig() {
			t.Errorf("expected defaults, got %+v", cfg)
		}
	})

	t.Run("missing_directory", func(t *testing.T) {
		cfg, err := loadVoiceConfig("/tmp/nonexistent-dir-12345/voice-config.json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg != DefaultVoiceStreamConfig() {
			t.Errorf("expected defaults, got %+v", cfg)
		}
	})
}

func TestVoiceConfig_SaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "voice-config.json")

	cfg := DefaultVoiceStreamConfig()
	if err := saveVoiceConfig(path, cfg); err != nil {
		t.Fatalf("save to nested path failed: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

// voiceConfigTestServer creates a minimal Server with voice config for handler tests.
func voiceConfigTestServer(t *testing.T) *Server {
	t.Helper()
	srv := serverWithCapability(true)
	srv.voiceConfigPath = filepath.Join(t.TempDir(), "voice-config.json")
	return srv
}

func TestHandleGetVoiceConfig(t *testing.T) {
	srv := voiceConfigTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/voice/config", nil)
	rec := httptest.NewRecorder()
	srv.handleGetVoiceConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var cfg VoiceStreamConfig
	if err := json.NewDecoder(rec.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg != DefaultVoiceStreamConfig() {
		t.Errorf("expected defaults, got %+v", cfg)
	}
}

func TestHandleUpdateVoiceConfig_Valid(t *testing.T) {
	srv := voiceConfigTestServer(t)

	body := `{"flushIntervalMs": 200, "overlapBytes": 0}`
	req := httptest.NewRequest("PUT", "/api/v1/voice/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleUpdateVoiceConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var cfg VoiceStreamConfig
	if err := json.NewDecoder(rec.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.FlushIntervalMs != 200 {
		t.Errorf("FlushIntervalMs = %d, want 200", cfg.FlushIntervalMs)
	}
	if cfg.OverlapBytes != 0 {
		t.Errorf("OverlapBytes = %d, want 0", cfg.OverlapBytes)
	}
	// Unchanged fields should retain defaults
	if cfg.MinDeltaBytes != 4096 {
		t.Errorf("MinDeltaBytes = %d, want 4096 (unchanged)", cfg.MinDeltaBytes)
	}
	// Verify in-memory config was updated
	mem := srv.getVoiceConfig()
	if mem.FlushIntervalMs != 200 || mem.OverlapBytes != 0 {
		t.Errorf("in-memory config not updated: %+v", mem)
	}

	// Verify file was persisted
	loaded, err := loadVoiceConfig(srv.voiceConfigPath)
	if err != nil {
		t.Fatalf("load persisted: %v", err)
	}
	if loaded != cfg {
		t.Errorf("persisted config mismatch: got %+v, want %+v", loaded, cfg)
	}
}

func TestHandleUpdateVoiceConfig_OutOfRange(t *testing.T) {
	srv := voiceConfigTestServer(t)

	body := `{"flushIntervalMs": 50}`
	req := httptest.NewRequest("PUT", "/api/v1/voice/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleUpdateVoiceConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body: %s", rec.Code, rec.Body.String())
	}

	// Config should be unchanged
	mem := srv.getVoiceConfig()
	if mem != DefaultVoiceStreamConfig() {
		t.Errorf("config should not have changed: %+v", mem)
	}
}

func TestHandleUpdateVoiceConfig_InvalidJSON(t *testing.T) {
	srv := voiceConfigTestServer(t)

	req := httptest.NewRequest("PUT", "/api/v1/voice/config", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleUpdateVoiceConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleGetVoiceConfig_AfterPut(t *testing.T) {
	srv := voiceConfigTestServer(t)

	// PUT to change config
	putBody := `{"minDeltaBytes": 8192}`
	putReq := httptest.NewRequest("PUT", "/api/v1/voice/config", strings.NewReader(putBody))
	putReq.Header.Set("Content-Type", "application/json")
	putRec := httptest.NewRecorder()
	srv.handleUpdateVoiceConfig(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200", putRec.Code)
	}

	// GET should reflect the update
	getReq := httptest.NewRequest("GET", "/api/v1/voice/config", nil)
	getRec := httptest.NewRecorder()
	srv.handleGetVoiceConfig(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", getRec.Code)
	}

	var cfg VoiceStreamConfig
	if err := json.NewDecoder(getRec.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.MinDeltaBytes != 8192 {
		t.Errorf("MinDeltaBytes = %d, want 8192", cfg.MinDeltaBytes)
	}
}
