// Package voice is the HTTP-handler home for the voice domain. It exposes
// the generated Connect-RPC VoiceService (proto schema:
// packages/proto/schemas/web-console/v1/voice).
package voice

import (
	"context"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	voiceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/voice/voice_v1connect"

	"web-console/internal/module"
)

// Service is the seam the Connect handler depends on. The concrete
// implementation lives in package main and wraps voice config storage, the
// Whisper transcribe pipeline, the wake-word template store, and the
// speaker-verification config/profile surface.
type Service interface {
	Transcribe(ctx context.Context, in TranscribeInput) (string, error)

	GetStreamConfig(ctx context.Context) (StreamConfig, error)
	UpdateStreamConfig(ctx context.Context, patch StreamConfigPatch) (StreamConfig, error)

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

// TranscribeInput is the Service-layer input for Transcribe.
type TranscribeInput struct {
	Audio                   []byte
	ContentType             string
	Language                string
	SkipSpeakerVerification bool
}

// StreamConfig mirrors the legacy VoiceStreamConfig JSON shape.
type StreamConfig struct {
	FlushIntervalMs   int
	MinDeltaBytes     int
	OverlapBytes      int
	PersistentMode    bool
	WakeWordEnabled   bool
	WakeWordThreshold float64
	SegmentSilenceMs  int
}

// StreamConfigPatch carries optional updates. Pointer fields preserve the
// "not provided vs zero" distinction.
type StreamConfigPatch struct {
	FlushIntervalMs   *int
	MinDeltaBytes     *int
	OverlapBytes      *int
	PersistentMode    *bool
	WakeWordEnabled   *bool
	WakeWordThreshold *float64
	SegmentSilenceMs  *int
}

// WakeWordConfig is the response shape for wake-word endpoints. TemplateJSON
// is the opaque WakeWordTemplate JSON owned by the UI engine.
type WakeWordConfig struct {
	Configured   bool
	TemplateJSON string
}

// SpeakerConfig mirrors the legacy SpeakerVerificationConfig.
type SpeakerConfig struct {
	Enabled                     bool
	ProfileIDs                  []string
	Threshold                   float64
	Mode                        string
	RejectBehavior              string
	FallbackWithoutVerification bool
	ExtractionEnabled           bool
}

// SpeakerConfigPatch is the optional-update variant for SpeakerConfig.
type SpeakerConfigPatch struct {
	Enabled                     *bool
	ProfileIDs                  *[]string
	Threshold                   *float64
	Mode                        *string
	RejectBehavior              *string
	FallbackWithoutVerification *bool
	ExtractionEnabled           *bool
}

// SpeakerProfile mirrors SpeakerVerificationProfile.
type SpeakerProfile struct {
	ID                     string
	DisplayName            string
	CreatedAt              string
	UpdatedAt              string
	ModelName              string
	EmbeddingDim           int
	SampleRate             int
	EnrollmentAudioSeconds float64
	Notes                  string
}

// SpeakerResourceInfo mirrors SpeakerVerificationResourceInfo.
type SpeakerResourceInfo struct {
	Backend      string
	Model        string
	Device       string
	SampleRate   int
	Version      string
	EmbeddingDim int
}

// SpeakerStatus mirrors SpeakerVerificationStatusResponse. Info pointer
// encodes "not present" without exposing protobuf optionals.
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

// SpeakerEnrollment mirrors SpeakerVerificationEnrollmentResponse.
type SpeakerEnrollment struct {
	ProfileID              string
	DisplayName            string
	EmbeddingDim           int
	SampleRate             int
	EnrollmentAudioSeconds float64
	ModelName              string
	CreatedAt              string
}

// EnrollInput carries the EnrollSpeakerProfile fields. AddToActive and
// Enable use pointer semantics because they both default to true server-side
// when omitted; the Connect handler translates the has_* flags.
type EnrollInput struct {
	Audio       []byte
	ContentType string
	ProfileID   string
	DisplayName string
	Notes       string
	AddToActive *bool
	Enable      *bool
}

// Module wires the voice domain into the API server.
func Module(svc Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := voiceconnect.NewVoiceServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "voice",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
