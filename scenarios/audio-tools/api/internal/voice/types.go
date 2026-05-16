package voice

import (
	"context"
	"errors"
)

// Sentinel errors used by service implementations and mapped by the transport
// layer (handlers/voice connect handler).
var (
	ErrInvalidArgument    = errors.New("invalid argument")
	ErrUnavailable        = errors.New("voice unavailable")
	ErrInternal           = errors.New("internal error")
	ErrNotFound           = errors.New("not found")
	ErrFailedPrecondition = errors.New("failed precondition")
)

// HandlerService is the domain contract consumed by the Connect handler.
// Methods speak in voice-package types only — no transport package types
// reach the domain.
//
// seam: HandlerService — Connect-RPC adapter -> voice domain.
type HandlerService interface {
	Transcribe(ctx context.Context, in TranscribeInput) (string, error)

	GetStreamConfig(ctx context.Context) (Config, error)
	UpdateStreamConfig(ctx context.Context, patch ConfigPatch) (Config, error)

	GetWakeWordConfig(ctx context.Context) (WakeWordConfig, error)
	UpdateWakeWordTemplate(ctx context.Context, templateJSON string) (WakeWordConfig, error)
	DeleteWakeWordTemplate(ctx context.Context) (WakeWordConfig, error)

	GetSpeakerConfig(ctx context.Context) (SpeakerConfig, error)
	UpdateSpeakerConfig(ctx context.Context, patch SpeakerConfigPatch) (SpeakerConfig, error)
	GetSpeakerStatus(ctx context.Context) (SpeakerStatus, error)
	ListSpeakerProfiles(ctx context.Context) ([]SpeakerProfile, int, error)
	EnrollSpeakerProfile(ctx context.Context, in EnrollInput) (SpeakerEnrollment, SpeakerConfig, error)
	ClearSpeakerProfileBinding(ctx context.Context) (SpeakerConfig, error)
	RemoveSpeakerProfile(ctx context.Context, profileID string) (SpeakerConfig, error)
	DeleteSpeakerProfile(ctx context.Context, profileID string) (SpeakerConfig, error)
}

// Backend is the storage/state seam the Adapter depends on. Production
// implementation is *Service in this package; tests substitute fakes.
//
// seam: Backend — Adapter -> service state, persistence, and resource clients.
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
	GetStreamConfig() Config
	SaveStreamConfig(c Config) error

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

// TranscribeInput is the Service-layer input for Transcribe.
type TranscribeInput struct {
	Audio                   []byte
	ContentType             string
	Language                string
	SkipSpeakerVerification bool
}

// StreamConfig is an alias for the canonical voice stream Config so older
// transport-layer code can keep using the legacy name during extraction prep.
type StreamConfig = Config

// StreamConfigPatch is an alias for ConfigPatch.
type StreamConfigPatch = ConfigPatch

// WakeWordConfig is the transport-visible wake-word state. The underlying
// template lives as *WakeWordTemplate in service state; TemplateJSON is the
// opaque payload the UI engine round-trips.
type WakeWordConfig struct {
	Configured   bool
	TemplateJSON string
}

// SpeakerStatus is the transport-visible speaker-verification status snapshot.
// Info is nil when the speaker resource is not configured.
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

// SpeakerEnrollment is the transport view of a successful enroll call. It is
// a field-for-field projection of SpeakerEnrollmentResponse — the JSON-tagged
// shape used on the resource wire.
type SpeakerEnrollment struct {
	ProfileID              string
	DisplayName            string
	EmbeddingDim           int
	SampleRate             int
	EnrollmentAudioSeconds float64
	ModelName              string
	CreatedAt              string
}

// EnrollInput carries the EnrollSpeakerProfile fields. AddToActive and Enable
// use pointer semantics because both default to true server-side when omitted;
// the Connect handler translates the has_* flags.
type EnrollInput struct {
	Audio       []byte
	ContentType string
	ProfileID   string
	DisplayName string
	Notes       string
	AddToActive *bool
	Enable      *bool
}
