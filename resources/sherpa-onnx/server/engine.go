package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

const streamingSampleRate = 16_000

// TTS is the small resource-local seam around sherpa-onnx's offline TTS C API.
// The HTTP layer does not know whether the platform implementation is cgo or
// a future native binding; cgo remains confined to this resource binary.
type TTS interface {
	Synthesize(context.Context, string, int, float32) (Audio, error)
	Close()
}

// StreamingSTT is the resource-local seam for sherpa-onnx online recognition.
// It deliberately exposes PCM/event data rather than C handles so the HTTP
// and WebSocket layers remain usable in the fail-closed non-cgo build.
type StreamingSTT interface {
	NewStream() (STTStream, error)
	Close()
}

type STTStream interface {
	AcceptPCM([]byte) ([]STTEvent, error)
	Finish() ([]STTEvent, error)
	Close()
}

type STTEvent struct {
	Text        string
	Final       bool
	StartSample int64
	EndSample   int64
}

// SpeakerRuntime is optional so TTS/STT can remain available when the larger
// speaker model bundle is not installed. The native implementation owns the
// profile store and all speaker model handles behind this resource boundary.
type SpeakerRuntime interface {
	http.Handler
	Close()
}

type Audio struct {
	Samples    []float32
	SampleRate int
}

type UnsupportedEngine struct{}

func (UnsupportedEngine) Synthesize(context.Context, string, int, float32) (Audio, error) {
	return Audio{}, fmt.Errorf("sherpa-onnx native TTS is unavailable in this build")
}

func (UnsupportedEngine) Close() {}

type UnsupportedStreaming struct{}

func (UnsupportedStreaming) NewStream() (STTStream, error) {
	return nil, fmt.Errorf("sherpa-onnx native streaming STT is unavailable in this build")
}

func (UnsupportedStreaming) Close() {}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
