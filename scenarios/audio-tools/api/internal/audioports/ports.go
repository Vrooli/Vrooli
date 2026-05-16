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
