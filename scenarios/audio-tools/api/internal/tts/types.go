package tts

import (
	"context"
	"errors"
)

// Sentinel errors used by service implementations and mapped by the transport
// layer.
var (
	ErrNotFound           = errors.New("not found")
	ErrInvalidArgument    = errors.New("invalid argument")
	ErrFailedPrecondition = errors.New("failed precondition")
	ErrUnavailable        = errors.New("tts unavailable")
	ErrInternal           = errors.New("internal error")
)

// seam: HandlerService is the TTS application-layer seam (SEAMS.md row
// "tts.HandlerService"). Production wires the concrete tts.Service;
// tests wire fakes to drive the Connect handler.
//
// HandlerService is the domain contract consumed by the Connect handler.
type HandlerService interface {
	GetConfig(ctx context.Context) (Config, error)
	UpdateConfig(ctx context.Context, patch ConfigPatch) (Config, error)

	GetStatus(ctx context.Context) (Status, error)
	RecordPlaybackEvent(ctx context.Context, event PlaybackEvent) error

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
