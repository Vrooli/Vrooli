// DOC: docs/internal/SEAMS.md#voice-config-seam
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// VoiceStreamConfig holds tunable parameters for the voice streaming pipeline.
// Each field controls a tradeoff in the partial-transcription loop. Values are
// validated on update to stay within safe operating bounds.
//
// Config is read once per WebSocket session (snapshot pattern) so mid-session
// changes take effect on the next recording, not the one in progress.
type VoiceStreamConfig struct {
	// FlushIntervalMs controls how often accumulated audio is sent to Whisper
	// for partial transcription. Lower values produce faster partials but
	// increase Whisper load. At 48 kbps, 500ms ≈ 3 KB of new audio per tick.
	// Range: 100–5000ms.
	FlushIntervalMs int `json:"flushIntervalMs"`

	// MinDeltaBytes is the minimum audio delta (in bytes) before a partial
	// transcription is attempted. Below this threshold the tick is skipped
	// (except the first tick, which always fires for perceived latency).
	// At 48 kbps (~6 KB/s), 4096 bytes ≈ 0.67s of audio.
	// Range: 512–32768.
	MinDeltaBytes int `json:"minDeltaBytes"`

	// OverlapBytes is the trailing audio overlap prepended to each delta
	// chunk for Whisper context continuity. Combined with initial_prompt
	// text context, this avoids cutting mid-word at chunk boundaries.
	// Higher values improve word continuity but increase redundant audio.
	// At 48 kbps (~6 KB/s), 2048 bytes ≈ 0.33s.
	// Range: 0–16384.
	OverlapBytes int `json:"overlapBytes"`

	// PersistentMode enables always-on listening where the mic stays active
	// until explicitly toggled off. VAD silence triggers segment boundaries
	// instead of recording stop.
	PersistentMode bool `json:"persistentMode"`

	// CommandPrefix is the keyword prefix that identifies a voice command
	// (e.g., "hey do new tab"). Case-insensitive.
	CommandPrefix string `json:"commandPrefix"`

	// SegmentSilenceMs is the silence duration (ms) that triggers a segment
	// boundary in persistent mode. Must be less than the VAD auto-stop
	// silence timeout. Range: 800–3000.
	SegmentSilenceMs int `json:"segmentSilenceMs"`
}

// DefaultVoiceStreamConfig returns production defaults matching the original
// hardcoded values. These are known to work well on a local Whisper instance.
func DefaultVoiceStreamConfig() VoiceStreamConfig {
	return VoiceStreamConfig{
		FlushIntervalMs:  500,
		MinDeltaBytes:    4096,
		OverlapBytes:     2048,
		PersistentMode:   false,
		CommandPrefix:    "hey do",
		SegmentSilenceMs: 1500,
	}
}

// Validate checks that all fields are within safe operating bounds. Returns a
// descriptive error for the first field that fails validation, or nil if valid.
func (c VoiceStreamConfig) Validate() error {
	if c.FlushIntervalMs < 100 || c.FlushIntervalMs > 5000 {
		return fmt.Errorf("flushIntervalMs must be between 100 and 5000, got %d", c.FlushIntervalMs)
	}
	if c.MinDeltaBytes < 512 || c.MinDeltaBytes > 32768 {
		return fmt.Errorf("minDeltaBytes must be between 512 and 32768, got %d", c.MinDeltaBytes)
	}
	if c.OverlapBytes < 0 || c.OverlapBytes > 16384 {
		return fmt.Errorf("overlapBytes must be between 0 and 16384, got %d", c.OverlapBytes)
	}
	if c.SegmentSilenceMs != 0 && (c.SegmentSilenceMs < 800 || c.SegmentSilenceMs > 3000) {
		return fmt.Errorf("segmentSilenceMs must be between 800 and 3000, got %d", c.SegmentSilenceMs)
	}
	if c.CommandPrefix != "" && len(c.CommandPrefix) > 50 {
		return fmt.Errorf("commandPrefix must be at most 50 characters, got %d", len(c.CommandPrefix))
	}
	return nil
}

// VoiceStreamConfigPatch is the partial update type for voice config. Pointer
// fields allow distinguishing "not provided" from "set to zero value".
type VoiceStreamConfigPatch struct {
	FlushIntervalMs  *int    `json:"flushIntervalMs,omitempty"`
	MinDeltaBytes    *int    `json:"minDeltaBytes,omitempty"`
	OverlapBytes     *int    `json:"overlapBytes,omitempty"`
	PersistentMode   *bool   `json:"persistentMode,omitempty"`
	CommandPrefix    *string `json:"commandPrefix,omitempty"`
	SegmentSilenceMs *int    `json:"segmentSilenceMs,omitempty"`
}

// Apply merges non-nil patch fields into base, returning the updated config.
func (p VoiceStreamConfigPatch) Apply(base VoiceStreamConfig) VoiceStreamConfig {
	if p.FlushIntervalMs != nil {
		base.FlushIntervalMs = *p.FlushIntervalMs
	}
	if p.MinDeltaBytes != nil {
		base.MinDeltaBytes = *p.MinDeltaBytes
	}
	if p.OverlapBytes != nil {
		base.OverlapBytes = *p.OverlapBytes
	}
	if p.PersistentMode != nil {
		base.PersistentMode = *p.PersistentMode
	}
	if p.CommandPrefix != nil {
		base.CommandPrefix = *p.CommandPrefix
	}
	if p.SegmentSilenceMs != nil {
		base.SegmentSilenceMs = *p.SegmentSilenceMs
	}
	return base
}

// loadVoiceConfig reads a VoiceStreamConfig from the given JSON file path.
// If the file or directory does not exist, returns defaults without error.
func loadVoiceConfig(path string) (VoiceStreamConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultVoiceStreamConfig(), nil
		}
		return DefaultVoiceStreamConfig(), fmt.Errorf("read voice config: %w", err)
	}
	var cfg VoiceStreamConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultVoiceStreamConfig(), fmt.Errorf("parse voice config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return DefaultVoiceStreamConfig(), fmt.Errorf("voice config validation: %w", err)
	}
	return cfg, nil
}

// saveVoiceConfig writes a VoiceStreamConfig to the given JSON file path.
// The parent directory is created if it doesn't exist. Writes atomically
// via a temporary file to prevent corruption from concurrent writes or crashes.
func saveVoiceConfig(path string, cfg VoiceStreamConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal voice config: %w", err)
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) // clean up on rename failure
		return fmt.Errorf("rename config file: %w", err)
	}
	return nil
}

// voiceConfigMu and voiceConfig are embedded in the Server struct (see main.go).
// These accessor methods provide thread-safe read/write access.

func (s *Server) getVoiceConfig() VoiceStreamConfig {
	s.voiceConfigMu.RLock()
	defer s.voiceConfigMu.RUnlock()
	return s.voiceConfig
}

func (s *Server) setVoiceConfig(cfg VoiceStreamConfig) {
	s.voiceConfigMu.Lock()
	defer s.voiceConfigMu.Unlock()
	s.voiceConfig = cfg
}

// handleGetVoiceConfig returns the current voice streaming configuration.
// GET /api/v1/voice/config
func (s *Server) handleGetVoiceConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.getVoiceConfig())
}

// handleUpdateVoiceConfig applies a partial update to voice streaming config,
// validates the result, persists to disk, and returns the updated config.
// PUT /api/v1/voice/config
func (s *Server) handleUpdateVoiceConfig(w http.ResponseWriter, r *http.Request) {
	var patch VoiceStreamConfigPatch
	if !decodeJSON(w, r, &patch) {
		return
	}
	current := s.getVoiceConfig()
	updated := patch.Apply(current)
	if err := updated.Validate(); err != nil {
		writeCatalogError(w, "invalid_body", err.Error())
		return
	}
	s.setVoiceConfig(updated)
	if err := saveVoiceConfig(s.voiceConfigPath, updated); err != nil {
		log.Printf("voice-config: persist failed (in-memory updated): %v", err)
	}
	log.Printf("voice-config: updated: flush=%dms delta=%d overlap=%d",
		updated.FlushIntervalMs, updated.MinDeltaBytes, updated.OverlapBytes)
	writeJSON(w, http.StatusOK, updated)
}
