// Package tts is the HTTP-handler home for the tts domain. It exposes
// the generated Connect-RPC TTSService (proto schema:
// packages/proto/schemas/web-console/v1/tts).
package tts

import (
	"context"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/tts/tts_v1connect"

	"web-console/internal/module"
)

// Service is the seam the Connect handler depends on. The concrete
// implementation lives in package main and wraps the existing TTS config,
// summarization config, synthesizer, cache, voice lister, and runtime
// status accessors.
type Service interface {
	GetConfig(ctx context.Context) (Config, error)
	UpdateConfig(ctx context.Context, patch ConfigPatch) (Config, error)

	GetStatus(ctx context.Context) (Status, error)
	RecordPlaybackEvent(ctx context.Context, event PlaybackEvent) error

	GetSummarizeConfig(ctx context.Context) (SummarizeConfig, error)
	UpdateSummarizeConfig(ctx context.Context, patch SummarizeConfigPatch) (SummarizeConfig, error)

	Synthesize(ctx context.Context, in SynthesizeInput) (SynthesizeResult, error)
	GetCache(ctx context.Context, in CacheLookup) (SynthesizeResult, error)
	ListVoices(ctx context.Context) ([]Voice, error)
}

// Config mirrors the legacy TTSConfig JSON shape.
type Config struct {
	AutoEnabled bool
	Backend     string
	KokoroVoice string
	KokoroSpeed float64
}

// ConfigPatch carries optional updates. Pointer fields preserve the legacy
// "not provided vs zero" distinction; the Connect handler builds this from
// the proto request's has_* flags.
type ConfigPatch struct {
	AutoEnabled *bool
	Backend     *string
	KokoroVoice *string
	KokoroSpeed *float64
}

// SummarizeConfig mirrors the legacy TTSSummarizeConfig.
type SummarizeConfig struct {
	Enabled        bool
	CharThreshold  int
	Level          string
	Model          string
	TimeoutSeconds int
}

// SummarizeConfigPatch is the optional-update variant for SummarizeConfig.
type SummarizeConfigPatch struct {
	Enabled        *bool
	CharThreshold  *int
	Level          *string
	Model          *string
	TimeoutSeconds *int
}

// AppendResult mirrors the routing-result snapshot embedded in Status.
type AppendResult struct {
	Appended  bool
	Code      string
	Reason    string
	Source    string
	SessionID string
	EventID   string
	Sequence  int64
	Duplicate bool
}

// ClientAck mirrors TTSClientAck.
type ClientAck struct {
	EventID   string
	Source    string
	SessionID string
	Stage     string
	Backend   string
	Message   string
}

// PlaybackEvent is the body of RecordPlaybackEvent.
type PlaybackEvent struct {
	Source    string
	Stage     string
	Backend   string
	SessionID string
	Message   string
}

// Status is the full /tts/status snapshot. Pointer fields encode
// "not-set-yet" without exposing protobuf optionals to the service layer.
type Status struct {
	Config           Config
	HookRegistered   bool
	HookCode         string
	HookReason       string
	HookSettingsPath string

	LastRouting         *AppendResult
	LastRoutingAt       string
	LastHookRouting     *AppendResult
	LastHookRoutingAt   string
	LastTailerRouting   *AppendResult
	LastTailerRoutingAt string

	LastAck         *ClientAck
	LastAckAt       string
	LastHookAck     *ClientAck
	LastHookAckAt   string
	LastTailerAck   *ClientAck
	LastTailerAckAt string

	LastPlaybackEvent *PlaybackEvent
	LastPlaybackAt    string

	KokoroCapability      string
	KokoroCapabilityLabel string
}

// SynthesizeInput carries the on-the-wire synthesize parameters plus the
// optional event_id/version pair that triggers cache-on-write.
type SynthesizeInput struct {
	Input          string
	Voice          string
	ResponseFormat string
	Speed          float64
	EventID        string
	Version        string
}

// SynthesizeResult is what Synthesize and GetCache return.
type SynthesizeResult struct {
	Audio       []byte
	ContentType string
}

// CacheLookup is the GetCache request shape.
type CacheLookup struct {
	EventID string
	Voice   string
	Speed   float64
	Version string
}

// Voice mirrors TTSVoice.
type Voice struct {
	ID   string
	Name string
}

// Module wires the tts domain into the API server.
func Module(svc Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := ttsconnect.NewTTSServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "tts",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
