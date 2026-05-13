package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	voicev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/voice"
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

func TestDefaultVoiceStreamConfig_PersistentModeDefaults(t *testing.T) {
	cfg := DefaultVoiceStreamConfig()
	if cfg.PersistentMode {
		t.Error("PersistentMode should default to false")
	}
	if cfg.WakeWordEnabled {
		t.Error("WakeWordEnabled should default to false")
	}
	if cfg.WakeWordThreshold != 0.65 {
		t.Errorf("WakeWordThreshold = %f, want 0.65", cfg.WakeWordThreshold)
	}
	if cfg.SegmentSilenceMs != 1500 {
		t.Errorf("SegmentSilenceMs = %d, want 1500", cfg.SegmentSilenceMs)
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

		// SegmentSilenceMs bounds
		{"segment_at_min", func(c *VoiceStreamConfig) { c.SegmentSilenceMs = 800 }, ""},
		{"segment_at_max", func(c *VoiceStreamConfig) { c.SegmentSilenceMs = 3000 }, ""},
		{"segment_below_min", func(c *VoiceStreamConfig) { c.SegmentSilenceMs = 799 }, "segmentSilenceMs"},
		{"segment_above_max", func(c *VoiceStreamConfig) { c.SegmentSilenceMs = 3001 }, "segmentSilenceMs"},
		{"segment_zero_allowed", func(c *VoiceStreamConfig) { c.SegmentSilenceMs = 0 }, ""},

		// WakeWordThreshold bounds
		{"wakeword_threshold_at_min", func(c *VoiceStreamConfig) { c.WakeWordThreshold = 0.1 }, ""},
		{"wakeword_threshold_at_max", func(c *VoiceStreamConfig) { c.WakeWordThreshold = 0.95 }, ""},
		{"wakeword_threshold_below_min", func(c *VoiceStreamConfig) { c.WakeWordThreshold = 0.05 }, "wakeWordThreshold"},
		{"wakeword_threshold_above_max", func(c *VoiceStreamConfig) { c.WakeWordThreshold = 0.96 }, "wakeWordThreshold"},
		{"wakeword_threshold_zero_allowed", func(c *VoiceStreamConfig) { c.WakeWordThreshold = 0 }, ""},
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

	t.Run("persistent_mode_patch", func(t *testing.T) {
		boolVal := true
		wwEnabled := true
		wwThreshold := 0.8
		segMs := 2000
		patch := VoiceStreamConfigPatch{
			PersistentMode:    &boolVal,
			WakeWordEnabled:   &wwEnabled,
			WakeWordThreshold: &wwThreshold,
			SegmentSilenceMs:  &segMs,
		}
		result := patch.Apply(base)
		if !result.PersistentMode {
			t.Error("PersistentMode should be true")
		}
		if !result.WakeWordEnabled {
			t.Error("WakeWordEnabled should be true")
		}
		if result.WakeWordThreshold != 0.8 {
			t.Errorf("WakeWordThreshold = %f, want 0.8", result.WakeWordThreshold)
		}
		if result.SegmentSilenceMs != 2000 {
			t.Errorf("SegmentSilenceMs = %d, want 2000", result.SegmentSilenceMs)
		}
		// Original fields unchanged
		if result.FlushIntervalMs != base.FlushIntervalMs {
			t.Errorf("FlushIntervalMs changed unexpectedly")
		}
	})
}

func TestVoiceConfig_FileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "voice-config.json")

	cfg := VoiceStreamConfig{
		FlushIntervalMs:   200,
		MinDeltaBytes:     8192,
		OverlapBytes:      1024,
		PersistentMode:    true,
		WakeWordEnabled:   true,
		WakeWordThreshold: 0.75,
		SegmentSilenceMs:  2000,
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
	cfg, err := callGetVoiceStreamConfig(t, srv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	def := DefaultVoiceStreamConfig()
	if int(cfg.GetFlushIntervalMs()) != def.FlushIntervalMs ||
		int(cfg.GetMinDeltaBytes()) != def.MinDeltaBytes ||
		int(cfg.GetOverlapBytes()) != def.OverlapBytes {
		t.Errorf("expected defaults, got %+v", cfg)
	}
}

func TestHandleUpdateVoiceConfig_Valid(t *testing.T) {
	srv := voiceConfigTestServer(t)

	cfg, err := callUpdateVoiceStreamConfig(t, srv, &voicev1.UpdateStreamConfigRequest{
		FlushIntervalMs:    200,
		HasFlushIntervalMs: true,
		OverlapBytes:       0,
		HasOverlapBytes:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GetFlushIntervalMs() != 200 {
		t.Errorf("FlushIntervalMs = %d, want 200", cfg.GetFlushIntervalMs())
	}
	if cfg.GetOverlapBytes() != 0 {
		t.Errorf("OverlapBytes = %d, want 0", cfg.GetOverlapBytes())
	}
	if cfg.GetMinDeltaBytes() != 4096 {
		t.Errorf("MinDeltaBytes = %d, want 4096 (unchanged)", cfg.GetMinDeltaBytes())
	}
	mem := srv.getVoiceConfig()
	if mem.FlushIntervalMs != 200 || mem.OverlapBytes != 0 {
		t.Errorf("in-memory config not updated: %+v", mem)
	}
	loaded, err := loadVoiceConfig(srv.voiceConfigPath)
	if err != nil {
		t.Fatalf("load persisted: %v", err)
	}
	if loaded.FlushIntervalMs != 200 || loaded.OverlapBytes != 0 {
		t.Errorf("persisted config mismatch: got %+v", loaded)
	}
}

func TestHandleUpdateVoiceConfig_OutOfRange(t *testing.T) {
	srv := voiceConfigTestServer(t)

	_, err := callUpdateVoiceStreamConfig(t, srv, &voicev1.UpdateStreamConfigRequest{
		FlushIntervalMs:    50,
		HasFlushIntervalMs: true,
	})
	if connectCode(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v (err=%v)", connectCode(err), err)
	}
	mem := srv.getVoiceConfig()
	if mem != DefaultVoiceStreamConfig() {
		t.Errorf("config should not have changed: %+v", mem)
	}
}

func TestHandleUpdateVoiceConfig_PersistentMode(t *testing.T) {
	srv := voiceConfigTestServer(t)

	cfg, err := callUpdateVoiceStreamConfig(t, srv, &voicev1.UpdateStreamConfigRequest{
		PersistentMode:       true,
		HasPersistentMode:    true,
		WakeWordEnabled:      true,
		HasWakeWordEnabled:   true,
		WakeWordThreshold:    0.7,
		HasWakeWordThreshold: true,
		SegmentSilenceMs:     2000,
		HasSegmentSilenceMs:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.GetPersistentMode() {
		t.Error("PersistentMode should be true")
	}
	if !cfg.GetWakeWordEnabled() {
		t.Error("WakeWordEnabled should be true")
	}
	if cfg.GetWakeWordThreshold() != 0.7 {
		t.Errorf("WakeWordThreshold = %f, want 0.7", cfg.GetWakeWordThreshold())
	}
	if cfg.GetSegmentSilenceMs() != 2000 {
		t.Errorf("SegmentSilenceMs = %d, want 2000", cfg.GetSegmentSilenceMs())
	}
}

func TestHandleUpdateVoiceConfig_SegmentSilenceOutOfRange(t *testing.T) {
	srv := voiceConfigTestServer(t)
	_, err := callUpdateVoiceStreamConfig(t, srv, &voicev1.UpdateStreamConfigRequest{
		SegmentSilenceMs:    500,
		HasSegmentSilenceMs: true,
	})
	if connectCode(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", connectCode(err))
	}
}

func TestVoiceStreamMessage_SegmentFinalJSON(t *testing.T) {
	msg := VoiceStreamMessage{
		Type:         VoiceMsgSegmentFinal,
		Text:         "hello world",
		SegmentIndex: 2,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var decoded VoiceStreamMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if decoded.Type != "segment-final" {
		t.Errorf("Type = %q, want 'segment-final'", decoded.Type)
	}
	if decoded.SegmentIndex != 2 {
		t.Errorf("SegmentIndex = %d, want 2", decoded.SegmentIndex)
	}
}

func TestVoiceStreamMessage_SegmentIndexOmitEmpty(t *testing.T) {
	msg := VoiceStreamMessage{Type: VoiceMsgPartial, Text: "test"}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	// SegmentIndex should be omitted when 0
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if _, exists := raw["segmentIndex"]; exists {
		t.Error("segmentIndex should be omitted when 0")
	}
}

func TestHandleGetVoiceConfig_AfterPut(t *testing.T) {
	srv := voiceConfigTestServer(t)

	if _, err := callUpdateVoiceStreamConfig(t, srv, &voicev1.UpdateStreamConfigRequest{
		MinDeltaBytes:    8192,
		HasMinDeltaBytes: true,
	}); err != nil {
		t.Fatalf("PUT: %v", err)
	}
	cfg, err := callGetVoiceStreamConfig(t, srv)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	if cfg.GetMinDeltaBytes() != 8192 {
		t.Errorf("MinDeltaBytes = %d, want 8192", cfg.GetMinDeltaBytes())
	}
}
