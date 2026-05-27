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
