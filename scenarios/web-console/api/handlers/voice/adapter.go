package voice

import (
	"context"
	"fmt"
	"log"
	"time"

	"web-console/internal/capabilities"
)

// Audio size limits enforced at the transport layer. The constants are
// duplicated here (also in package main) intentionally: this package owns
// its own transport-level contract and shouldn't import main.
const (
	maxAudioSize                  = 10 << 20 // 10 MB
	maxSpeakerEnrollmentAudioSize = 20 << 20 // 20 MB
)

// SpeakerDecision is the transport-friendly view of the speaker-verification
// gate's outcome. Package main's evaluator returns a richer internal struct;
// the Deps shim converts it to this shape.
type SpeakerDecision struct {
	Enabled      bool
	Applied      bool
	Allowed      bool
	Matched      bool
	ProfileID    string
	Score        float64
	Threshold    float64
	Mode         string
	ErrorMessage string
}

// Backend is the seam the Adapter depends on. Methods speak in transport-
// neutral voice-package types; package main provides the shim that
// converts internal storage types and resource clients into these.
type Backend interface {
	// Capability and metrics
	WhisperAvailable(ctx context.Context) bool
	IncrSkipVerification()
	SpeakerCapability(ctx context.Context) (status string, label string)

	// Transcribe path
	EvaluateSpeaker(ctx context.Context, audio []byte) SpeakerDecision
	FormatSpeakerDecisionError(d SpeakerDecision) string
	Transcribe(ctx context.Context, audio []byte, language string) (string, error)
	IsWhisperHallucination(text string) bool

	// Stream config (validation + persistence happens inside Save*)
	GetStreamConfig() StreamConfig
	SaveStreamConfig(c StreamConfig) error

	// Wake word
	GetWakeWord() WakeWordConfig
	SetWakeWord(templateJSON string) (WakeWordConfig, error)
	ClearWakeWord() error

	// Speaker config (validation + persistence happens inside Save*)
	GetSpeakerConfig() SpeakerConfig
	SaveSpeakerConfig(c SpeakerConfig) error
	DefaultSpeakerThreshold() float64
	DefaultSpeakerProfileID() string

	// Speaker resource client
	SpeakerClientConfigured() bool
	SpeakerReady(ctx context.Context) bool
	ListSpeakerProfiles(ctx context.Context) ([]SpeakerProfile, int, error)
	SpeakerInfo(ctx context.Context) (SpeakerResourceInfo, bool)
	EnrollSpeaker(ctx context.Context, audio []byte, profileID, displayName, notes string) (SpeakerEnrollment, error)
	DeleteSpeakerBackend(ctx context.Context, profileID string) error
}

// Adapter is the production Service implementation. Constructed in
// api/main.go with a typed Deps and passed to Module.
type Adapter struct {
	Backend Backend
	Logger  *log.Logger
}

func (a *Adapter) logger() *log.Logger {
	if a.Logger != nil {
		return a.Logger
	}
	return log.Default()
}

// -----------------------------------------------------------------------------
// Transcribe
// -----------------------------------------------------------------------------

func (a *Adapter) Transcribe(ctx context.Context, in TranscribeInput) (string, error) {
	if !a.Backend.WhisperAvailable(ctx) {
		return "", fmt.Errorf("%w: whisper transcription is currently unavailable", ErrUnavailable)
	}
	if len(in.Audio) == 0 {
		return "", fmt.Errorf("%w: audio is required", ErrInvalidArgument)
	}
	if len(in.Audio) > maxAudioSize {
		return "", fmt.Errorf("%w: audio exceeds %d bytes", ErrInvalidArgument, maxAudioSize)
	}

	if in.SkipSpeakerVerification {
		a.Backend.IncrSkipVerification()
		a.logger().Printf("voice-http: speaker verification bypassed bytes=%d", len(in.Audio))
	} else {
		verifyCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		decision := a.Backend.EvaluateSpeaker(verifyCtx, in.Audio)
		cancel()
		if decision.Enabled {
			if decision.Applied {
				a.logger().Printf(
					"voice-http: speaker decision matched=%v allowed=%v score=%.3f threshold=%.3f profile=%s mode=%s",
					decision.Matched, decision.Allowed, decision.Score, decision.Threshold,
					decision.ProfileID, decision.Mode,
				)
			} else if decision.ErrorMessage != "" {
				a.logger().Printf("voice-http: %s", a.Backend.FormatSpeakerDecisionError(decision))
			}
			if !decision.Allowed {
				return "", nil
			}
		}
	}

	text, err := a.Backend.Transcribe(ctx, in.Audio, in.Language)
	if err != nil {
		a.logger().Printf("voice-http: whisper failed: %v", err)
		return "", fmt.Errorf("%w: whisper request failed", ErrInternal)
	}
	if a.Backend.IsWhisperHallucination(text) {
		a.logger().Printf("voice-http: filtered hallucination: %q", text)
		text = ""
	}
	return text, nil
}

// -----------------------------------------------------------------------------
// Stream config
// -----------------------------------------------------------------------------

func (a *Adapter) GetStreamConfig(_ context.Context) (StreamConfig, error) {
	return a.Backend.GetStreamConfig(), nil
}

func (a *Adapter) UpdateStreamConfig(_ context.Context, patch StreamConfigPatch) (StreamConfig, error) {
	current := a.Backend.GetStreamConfig()
	if patch.FlushIntervalMs != nil {
		current.FlushIntervalMs = *patch.FlushIntervalMs
	}
	if patch.MinDeltaBytes != nil {
		current.MinDeltaBytes = *patch.MinDeltaBytes
	}
	if patch.OverlapBytes != nil {
		current.OverlapBytes = *patch.OverlapBytes
	}
	if patch.PersistentMode != nil {
		current.PersistentMode = *patch.PersistentMode
	}
	if patch.WakeWordEnabled != nil {
		current.WakeWordEnabled = *patch.WakeWordEnabled
	}
	if patch.WakeWordThreshold != nil {
		current.WakeWordThreshold = *patch.WakeWordThreshold
	}
	if patch.SegmentSilenceMs != nil {
		current.SegmentSilenceMs = *patch.SegmentSilenceMs
	}
	if err := a.Backend.SaveStreamConfig(current); err != nil {
		return StreamConfig{}, err
	}
	a.logger().Printf("voice-config: updated: flush=%dms delta=%d overlap=%d",
		current.FlushIntervalMs, current.MinDeltaBytes, current.OverlapBytes)
	return current, nil
}

// -----------------------------------------------------------------------------
// Wake word
// -----------------------------------------------------------------------------

func (a *Adapter) GetWakeWordConfig(_ context.Context) (WakeWordConfig, error) {
	return a.Backend.GetWakeWord(), nil
}

func (a *Adapter) UpdateWakeWordTemplate(_ context.Context, templateJSON string) (WakeWordConfig, error) {
	return a.Backend.SetWakeWord(templateJSON)
}

func (a *Adapter) DeleteWakeWordTemplate(_ context.Context) (WakeWordConfig, error) {
	if err := a.Backend.ClearWakeWord(); err != nil {
		a.logger().Printf("wakeword: delete failed: %v", err)
	}
	a.logger().Printf("wakeword: template cleared")
	return WakeWordConfig{Configured: false}, nil
}

// -----------------------------------------------------------------------------
// Speaker config
// -----------------------------------------------------------------------------

func (a *Adapter) GetSpeakerConfig(_ context.Context) (SpeakerConfig, error) {
	return a.Backend.GetSpeakerConfig(), nil
}

func (a *Adapter) UpdateSpeakerConfig(_ context.Context, patch SpeakerConfigPatch) (SpeakerConfig, error) {
	current := a.Backend.GetSpeakerConfig()
	if patch.Enabled != nil {
		current.Enabled = *patch.Enabled
	}
	if patch.ProfileIDs != nil {
		current.ProfileIDs = append([]string(nil), (*patch.ProfileIDs)...)
	}
	if patch.Threshold != nil {
		current.Threshold = *patch.Threshold
	}
	if patch.Mode != nil {
		current.Mode = *patch.Mode
	}
	if patch.RejectBehavior != nil {
		current.RejectBehavior = *patch.RejectBehavior
	}
	if patch.FallbackWithoutVerification != nil {
		current.FallbackWithoutVerification = *patch.FallbackWithoutVerification
	}
	if patch.ExtractionEnabled != nil {
		current.ExtractionEnabled = *patch.ExtractionEnabled
	}
	if current.Mode == "" {
		current.Mode = "filter"
	}
	if current.RejectBehavior == "" {
		current.RejectBehavior = "drop"
	}
	if err := a.Backend.SaveSpeakerConfig(current); err != nil {
		return SpeakerConfig{}, err
	}
	return current, nil
}

func (a *Adapter) GetSpeakerStatus(ctx context.Context) (SpeakerStatus, error) {
	cfg := a.Backend.GetSpeakerConfig()
	out := SpeakerStatus{
		Config:            cfg,
		Capability:        string(capabilities.StatusUnknown),
		ProfileConfigured: len(cfg.ProfileIDs) > 0,
		CheckedAt:         time.Now().UTC().Format(time.RFC3339),
	}

	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	status, label := a.Backend.SpeakerCapability(probeCtx)
	out.Capability = status
	out.CapabilityLabel = label

	if !a.Backend.SpeakerClientConfigured() || out.Capability != string(capabilities.StatusAvailable) {
		return out, nil
	}

	if a.Backend.SpeakerReady(probeCtx) {
		out.ResourceReady = true
	}

	profiles, count, err := a.Backend.ListSpeakerProfiles(probeCtx)
	if err == nil {
		out.ProfileCount = count
		out.Profiles = profiles
		configured := make(map[string]struct{}, len(cfg.ProfileIDs))
		for _, id := range cfg.ProfileIDs {
			configured[id] = struct{}{}
		}
		for _, p := range profiles {
			if _, ok := configured[p.ID]; ok {
				out.ProfileExists = true
				break
			}
		}
	}

	if info, ok := a.Backend.SpeakerInfo(probeCtx); ok {
		out.Info = &info
	}
	return out, nil
}

func (a *Adapter) ListSpeakerProfiles(ctx context.Context) ([]SpeakerProfile, int, error) {
	if !a.Backend.SpeakerClientConfigured() {
		return nil, 0, fmt.Errorf("%w: speaker verification resource is not configured", ErrUnavailable)
	}
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	profiles, count, err := a.Backend.ListSpeakerProfiles(probeCtx)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: list speaker profiles: %s", ErrInternal, err.Error())
	}
	return profiles, count, nil
}

func (a *Adapter) EnrollSpeakerProfile(ctx context.Context, in EnrollInput) (SpeakerEnrollment, SpeakerConfig, error) {
	if !a.Backend.SpeakerClientConfigured() {
		return SpeakerEnrollment{}, SpeakerConfig{}, fmt.Errorf("%w: speaker verification resource is not configured", ErrUnavailable)
	}
	if len(in.Audio) > maxSpeakerEnrollmentAudioSize {
		return SpeakerEnrollment{}, SpeakerConfig{}, fmt.Errorf("%w: audio exceeds %d bytes", ErrInvalidArgument, maxSpeakerEnrollmentAudioSize)
	}

	profileID := in.ProfileID
	if profileID == "" {
		profileID = a.Backend.DefaultSpeakerProfileID()
	}
	displayName := in.DisplayName
	if displayName == "" {
		displayName = "My Voice"
	}
	addToActive := true
	if in.AddToActive != nil {
		addToActive = *in.AddToActive
	}
	enable := true
	if in.Enable != nil {
		enable = *in.Enable
	}

	enrollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	enrollment, err := a.Backend.EnrollSpeaker(enrollCtx, in.Audio, profileID, displayName, in.Notes)
	if err != nil {
		a.logger().Printf("speaker-verification-enroll: %v", err)
		return SpeakerEnrollment{}, SpeakerConfig{}, fmt.Errorf("%w: failed to enroll speaker profile", ErrInternal)
	}

	cfg := a.Backend.GetSpeakerConfig()
	if addToActive && !containsString(cfg.ProfileIDs, profileID) {
		cfg.ProfileIDs = append(cfg.ProfileIDs, profileID)
	}
	if enable {
		cfg.Enabled = true
		if cfg.Mode == "" {
			cfg.Mode = "filter"
		}
		if cfg.RejectBehavior == "" {
			cfg.RejectBehavior = "drop"
		}
		if cfg.Threshold == 0 {
			cfg.Threshold = a.Backend.DefaultSpeakerThreshold()
		}
	}
	if err := a.Backend.SaveSpeakerConfig(cfg); err != nil {
		return SpeakerEnrollment{}, SpeakerConfig{}, fmt.Errorf("%w: enrollment succeeded, but speaker verification config is invalid", ErrInternal)
	}

	return enrollment, cfg, nil
}

func (a *Adapter) ClearSpeakerProfileBinding(_ context.Context) (SpeakerConfig, error) {
	cfg := a.Backend.GetSpeakerConfig()
	cfg.Enabled = false
	cfg.ProfileIDs = nil
	if err := a.Backend.SaveSpeakerConfig(cfg); err != nil {
		return SpeakerConfig{}, err
	}
	return cfg, nil
}

func (a *Adapter) RemoveSpeakerProfile(_ context.Context, profileID string) (SpeakerConfig, error) {
	cfg := a.Backend.GetSpeakerConfig()
	cfg.ProfileIDs = removeString(cfg.ProfileIDs, profileID)
	if len(cfg.ProfileIDs) == 0 {
		cfg.Enabled = false
	}
	if err := a.Backend.SaveSpeakerConfig(cfg); err != nil {
		return SpeakerConfig{}, err
	}
	return cfg, nil
}

func (a *Adapter) DeleteSpeakerProfile(ctx context.Context, profileID string) (SpeakerConfig, error) {
	if !a.Backend.SpeakerClientConfigured() {
		return SpeakerConfig{}, fmt.Errorf("%w: speaker verification resource is not configured", ErrUnavailable)
	}
	delCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := a.Backend.DeleteSpeakerBackend(delCtx, profileID); err != nil {
		a.logger().Printf("speaker-verification-delete: %v", err)
		return SpeakerConfig{}, fmt.Errorf("%w: failed to delete speaker profile from resource", ErrInternal)
	}
	cfg := a.Backend.GetSpeakerConfig()
	cfg.ProfileIDs = removeString(cfg.ProfileIDs, profileID)
	if len(cfg.ProfileIDs) == 0 {
		cfg.Enabled = false
	}
	if err := a.Backend.SaveSpeakerConfig(cfg); err != nil {
		return SpeakerConfig{}, fmt.Errorf("%w: %s", ErrInternal, err.Error())
	}
	return cfg, nil
}

func containsString(s []string, target string) bool {
	for _, v := range s {
		if v == target {
			return true
		}
	}
	return false
}

func removeString(s []string, target string) []string {
	out := s[:0]
	for _, v := range s {
		if v != target {
			out = append(out, v)
		}
	}
	return out
}
