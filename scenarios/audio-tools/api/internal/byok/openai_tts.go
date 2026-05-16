package byok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"audio-tools/internal/ai/ttschain"
)

// OpenAITTS calls OpenAI's audio/speech endpoint. Model: tts-1.
type OpenAITTS struct {
	Endpoint   string
	HTTPClient *http.Client
}

func NewOpenAITTS() *OpenAITTS {
	return &OpenAITTS{
		Endpoint:   "https://api.openai.com/v1/audio/speech",
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (a *OpenAITTS) ID() string    { return "openai-tts" }
func (a *OpenAITTS) Model() string { return "tts-1" }

func (a *OpenAITTS) IsAvailable(ctx context.Context, key string) bool { return key != "" }

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

	start := time.Now()
	resp, err := a.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai-tts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai-tts: HTTP %d: %s", resp.StatusCode, truncate(string(raw), 256))
	}
	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &ttschain.Result{
		Audio:       audio,
		ContentType: ttsContentType(format),
		ModelID:     "tts-1",
		VoiceUsed:   voice,
		Latency:     time.Since(start),
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
