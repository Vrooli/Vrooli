package byok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/clock"
	"audio-tools/internal/httpc"
)

// OpenAITTS calls OpenAI's audio/speech endpoint. Model: tts-1.
type OpenAITTS struct {
	Endpoint string
	Doer     httpc.Doer
	Clock    clock.Clock
}

func NewOpenAITTS() *OpenAITTS {
	return &OpenAITTS{
		Endpoint: "https://api.openai.com/v1/audio/speech",
		Doer:     httpc.DefaultDoer(),
		Clock:    clock.System{},
	}
}

func (a *OpenAITTS) ID() string    { return "openai-tts" }
func (a *OpenAITTS) Model() string { return "tts-1" }

func (a *OpenAITTS) IsAvailable(ctx context.Context, key string) bool { return key != "" }

// StreamingCapability — OpenAI TTS HTTP endpoint is unary; streaming
// goes through the Realtime API (out of scope here).
func (a *OpenAITTS) StreamingCapability() bool { return false }

func (a *OpenAITTS) SynthesizeStreaming(_ context.Context, _ string, _ ttschain.Request) (<-chan ttschain.AudioFrame, error) {
	return nil, nil
}

// canonicalToOpenAIVoice maps canonical voice IDs to OpenAI voice names.
// Overrides keyed "byok:openai-tts" win.
func canonicalToOpenAIVoice(canonical string, overrides map[string]string) string {
	if v, ok := overrides["byok:openai-tts"]; ok && v != "" {
		return v
	}
	switch canonical {
	case "voice.feminine.warm":
		return "shimmer"
	case "voice.feminine.neutral":
		return "nova"
	case "voice.masculine.warm":
		return "onyx"
	case "voice.masculine.neutral":
		return "echo"
	case "voice.neutral.default":
		return "alloy"
	}
	return "alloy"
}

func (a *OpenAITTS) Synthesize(ctx context.Context, key string, req ttschain.Request) (*ttschain.Result, error) {
	if key == "" {
		return nil, fmt.Errorf("openai-tts: missing API key")
	}
	voice := canonicalToOpenAIVoice(req.Voice, req.VoiceOverrides)
	format := req.ResponseFormat
	if format == "" {
		format = "mp3"
	}
	payload, _ := json.Marshal(map[string]any{
		"model":           "tts-1",
		"voice":           voice,
		"input":           req.Text,
		"response_format": format,
		"speed":           clampSpeed(req.Speed),
	})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+key)
	httpReq.Header.Set("Content-Type", "application/json")

	audio, latency, err := DoAudioRequest(ctx, a.Doer, a.Clock, "openai-tts", httpReq)
	if err != nil {
		return nil, err
	}
	return &ttschain.Result{
		Audio:       audio,
		ContentType: ttsContentType(format),
		ModelID:     "tts-1",
		VoiceUsed:   voice,
		Latency:     latency,
	}, nil
}

func clampSpeed(s float64) float64 {
	if s <= 0 {
		return 1.0
	}
	if s < 0.25 {
		return 0.25
	}
	if s > 4.0 {
		return 4.0
	}
	return s
}

func ttsContentType(format string) string {
	switch format {
	case "mp3":
		return "audio/mpeg"
	case "wav":
		return "audio/wav"
	case "opus":
		return "audio/opus"
	case "flac":
		return "audio/flac"
	case "aac":
		return "audio/aac"
	}
	return "application/octet-stream"
}
