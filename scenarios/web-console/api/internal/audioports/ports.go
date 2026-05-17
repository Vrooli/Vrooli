// Package audioports defines the web-console-owned capability ports for
// audio behaviour the future scenarios/audio-tools will own end-to-end.
//
// The ports are intentionally narrow — they expose only the operations
// web-console actually uses against the audio domain today. The local
// implementation (in package main wiring) is backed by internal/tts and
// internal/voice; the future audio-tools implementation will be backed by an
// HTTP/Connect/WebSocket client. Conversation, terminal, and hook code talk
// to these ports, never to the underlying internal services directly, so the
// adoption swap is a single wiring change.
//
// seam: SpeechToText, TextToSpeech, SpeechTextProcessor — web-console
// orchestration -> audio capability provider.
package audioports

import "context"

// SpeechToText converts raw audio bytes into text. Mirrors the Connect-RPC
// VoiceService.Transcribe surface but exposes only what web-console
// orchestration needs.
type SpeechToText interface {
	Transcribe(ctx context.Context, audio []byte, opts STTOptions) (STTResult, error)
}

// STTOptions controls a single transcription. Optional fields are zero-valued
// to mean "use provider default".
type STTOptions struct {
	Language                string
	SkipSpeakerVerification bool
	InitialPrompt           string
}

// STTResult is the deduplicated, hallucination-filtered transcription.
type STTResult struct {
	Text string
}

// TextToSpeech renders text to audio bytes and lists available voices.
// Conversation orchestration uses Synthesize / cache lookup, never the
// concrete Kokoro client.
type TextToSpeech interface {
	Synthesize(ctx context.Context, req TTSRequest) (TTSResult, error)
	ListVoices(ctx context.Context) ([]Voice, error)
	GetCached(ctx context.Context, key CacheLookup) (TTSResult, bool)
}

// TTSRequest carries the synthesize parameters; EventID/Version, when set,
// trigger cache-on-write inside the provider.
type TTSRequest struct {
	Input          string
	Voice          string
	ResponseFormat string
	Speed          float64
	EventID        string
	Version        string
}

// TTSResult is what Synthesize / GetCached return.
type TTSResult struct {
	Audio       []byte
	ContentType string
}

// CacheLookup is the GetCached request shape.
type CacheLookup struct {
	EventID string
	Voice   string
	Speed   float64
	Version string
}

// Voice is the listable voice catalogue entry.
type Voice struct {
	ID   string
	Name string
}

// SpeechTextProcessor runs the markdown-to-speech text pipeline (normalize +
// paragraph split + optional summarization). Pure-function ports today
// (normalize/split) plus a context-bearing summarize that talks to a
// remote model.
type SpeechTextProcessor interface {
	NormalizeForSpeech(text string) string
	SplitIntoParagraphs(text string) []string
}

// PassthroughSpeechTextProcessor is a no-op SpeechTextProcessor for tests
// and the in-memory ConversationStore default. Production wiring replaces
// it with RemoteSpeechTextProcessor backed by audio-tools.
type PassthroughSpeechTextProcessor struct{}

func (PassthroughSpeechTextProcessor) NormalizeForSpeech(text string) string { return text }
func (PassthroughSpeechTextProcessor) SplitIntoParagraphs(text string) []string {
	if text == "" {
		return nil
	}
	return []string{text}
}

// -----------------------------------------------------------------------------
// Admin ports (added Phase B of "UI↔own-API only" rollout).
//
// These mirror audio-tools' STTAdminService surface but expose typed
// domain structs from contracts.go so handler/test code never imports an
// audio-tools proto package. Remote* implementations live in
// remote_*_admin.go.
// -----------------------------------------------------------------------------

// FieldMask is the path-list update mask shared by every Update* admin
// method. Empty mask is invalid (handlers reject InvalidArgument).
type FieldMask struct {
	Paths []string
}

// StreamConfigAdmin owns the StreamConfig CRUD surface.
type StreamConfigAdmin interface {
	GetStreamConfig(ctx context.Context) (StreamConfig, error)
	UpdateStreamConfig(ctx context.Context, mask FieldMask, cfg StreamConfig) (StreamConfig, error)
}

// WakeWordAdmin owns the wake-word template surface.
type WakeWordAdmin interface {
	GetWakeWordConfig(ctx context.Context) (WakeWordConfig, error)
	UpdateWakeWordTemplate(ctx context.Context, t WakeWordTemplate) (WakeWordConfig, error)
	DeleteWakeWordTemplate(ctx context.Context) (WakeWordConfig, error)
}

// SpeakerAdmin owns speaker-verification config + profile lifecycle.
type SpeakerAdmin interface {
	GetSpeakerConfig(ctx context.Context) (SpeakerConfig, error)
	UpdateSpeakerConfig(ctx context.Context, mask FieldMask, cfg SpeakerConfig) (SpeakerConfig, error)
	GetSpeakerStatus(ctx context.Context) (SpeakerStatus, error)
	ListSpeakerProfiles(ctx context.Context) ([]SpeakerProfile, error)
	EnrollSpeakerProfile(ctx context.Context, in EnrollSpeakerInput) (SpeakerEnrollResult, error)
	ClearSpeakerProfileBinding(ctx context.Context) (SpeakerConfig, error)
	UnbindSpeakerProfile(ctx context.Context, profileID string) (SpeakerConfig, error)
	DeleteSpeakerProfile(ctx context.Context, profileID string) (SpeakerConfig, error)
}

// EnrollSpeakerInput is the EnrollSpeakerProfile request shape.
// AddToActive / Enable are tri-state (nil = unset).
type EnrollSpeakerInput struct {
	Audio       []byte
	Format      AudioFormat
	ProfileID   string
	DisplayName string
	Notes       string
	AddToActive *bool
	Enable      *bool
}

// TTSConfigAdmin owns the audio-tools TTS Config surface.
type TTSConfigAdmin interface {
	GetTTSConfig(ctx context.Context) (TTSConfig, error)
	UpdateTTSConfig(ctx context.Context, mask FieldMask, cfg TTSConfig) (TTSConfig, error)
}

// SummarizeConfigAdmin owns the SummarizeConfig surface (separate from the
// per-call Summarizer above).
type SummarizeConfigAdmin interface {
	GetSummarizeConfig(ctx context.Context) (SummarizeConfig, error)
	UpdateSummarizeConfig(ctx context.Context, mask FieldMask, cfg SummarizeConfig) (SummarizeConfig, error)
}

// PlaybackEventRecorder forwards UI-emitted TTS playback events
// (start/stop/error) to audio-tools' RecordPlaybackEvent RPC.
type PlaybackEventRecorder interface {
	RecordPlaybackEvent(ctx context.Context, ev PlaybackEvent) error
}

// PlaybackEvent mirrors audio-tools' PlaybackEvent message.
type PlaybackEvent struct {
	Source    string
	Stage     string
	Backend   string
	SessionID string
	Message   string
	EventID   string
}
