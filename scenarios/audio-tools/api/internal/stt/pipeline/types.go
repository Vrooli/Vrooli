package pipeline

import (
	"errors"
)

// Sentinel errors used by the pipeline and mapped by the STT Connect
// handler.
var (
	ErrInvalidArgument    = errors.New("invalid argument")
	ErrUnavailable        = errors.New("voice unavailable")
	ErrInternal           = errors.New("internal error")
	ErrNotFound           = errors.New("not found")
	ErrFailedPrecondition = errors.New("failed precondition")

	// ErrSTTBackendUnavailable marks a transcribe failure caused by the speech
	// backend (whisper/kyutai-stt) being down or unreachable — the honest typed
	// replacement for the raw `dial tcp …: connection refused` transport string
	// (plan L2). The Connect handler maps a transient/starting case to
	// CodeUnavailable and an operator-action case to CodeFailedPrecondition via
	// errors.As(*STTBackendError). Mirrors the ErrFfmpegExec sentinel precedent —
	// classification is errors.Is/typed, never a string compare.
	ErrSTTBackendUnavailable = errors.New("speech backend unavailable")
)

// WakeWordConfig is the transport-visible wake-word state. The
// underlying template lives as *WakeWordTemplate in service state;
// TemplateJSON is the opaque payload the UI engine round-trips.
type WakeWordConfig struct {
	Configured   bool
	TemplateJSON string
}

// SpeakerStatus is the transport-visible speaker-verification status
// snapshot. Info is nil when the speaker resource is not configured.
type SpeakerStatus struct {
	Config            SpeakerConfig
	Capability        string
	CapabilityLabel   string
	ResourceReady     bool
	ProfileConfigured bool
	ProfileExists     bool
	ProfileCount      int
	Profiles          []SpeakerProfile
	Info              *SpeakerResourceInfo
	CheckedAt         string
}
