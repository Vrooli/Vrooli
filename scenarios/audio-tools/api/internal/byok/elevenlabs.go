package byok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/httpc"

	"github.com/vrooli/api-core/schedule"
)

// ElevenLabsTTS calls ElevenLabs' text-to-speech endpoint.
type ElevenLabsTTS struct {
	BaseURL string
	Doer    httpc.Doer
	Clock   schedule.Clock
}

func NewElevenLabsTTS() *ElevenLabsTTS {
	return &ElevenLabsTTS{
		BaseURL: "https://api.elevenlabs.io",
		Doer:    httpc.DefaultDoer(),
		Clock:   schedule.System(),
	}
}

func (a *ElevenLabsTTS) ID() string    { return "elevenlabs" }
func (a *ElevenLabsTTS) Model() string { return "eleven_multilingual_v2" }

func (a *ElevenLabsTTS) IsAvailable(ctx context.Context, key string) bool { return key != "" }

// StreamingCapability — ElevenLabs supports streaming WS but it is
// declared out of scope for this plan; return false so the chain
// negotiates a streaming-capable peer first.
func (a *ElevenLabsTTS) StreamingCapability() bool { return false }

func (a *ElevenLabsTTS) SynthesizeStreaming(_ context.Context, _ string, _ ttschain.Request) (<-chan ttschain.AudioFrame, error) {
	return nil, nil
}

// canonicalToElevenVoiceID returns the ElevenLabs voice_id for a canonical
// voice. Public ElevenLabs IDs are stable; users override via voice_overrides.
func canonicalToElevenVoiceID(canonical string, overrides map[string]string) string {
	if v, ok := overrides["byok:elevenlabs"]; ok && v != "" {
		return v
	}
	switch canonical {
	case "voice.feminine.warm":
		return "EXAVITQu4vr4xnSDxMaL" // "Bella" public ID
	case "voice.feminine.neutral":
		return "21m00Tcm4TlvDq8ikWAM" // "Rachel"
	case "voice.masculine.warm":
		return "ErXwobaYiN019PkySvjV" // "Antoni"
	case "voice.masculine.neutral":
		return "29vD33N1CtxCmqQRPOHJ" // "Drew"
	case "voice.neutral.default":
		return "21m00Tcm4TlvDq8ikWAM" // "Rachel"
	}
	return "21m00Tcm4TlvDq8ikWAM"
}

func (a *ElevenLabsTTS) Synthesize(ctx context.Context, key string, req ttschain.Request) (*ttschain.Result, error) {
	if key == "" {
		return nil, fmt.Errorf("elevenlabs: missing API key")
	}
	voiceID := canonicalToElevenVoiceID(req.Voice, req.VoiceOverrides)
	payload, _ := json.Marshal(map[string]any{
		"text":     req.Text,
		"model_id": "eleven_multilingual_v2",
	})
	endpoint := fmt.Sprintf("%s/v1/text-to-speech/%s", a.BaseURL, voiceID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("xi-api-key", key)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "audio/mpeg")

	audio, latency, err := DoAudioRequest(ctx, a.Doer, a.Clock, "elevenlabs", httpReq)
	if err != nil {
		return nil, err
	}
	return &ttschain.Result{
		Audio:       audio,
		ContentType: "audio/mpeg",
		ModelID:     "eleven_multilingual_v2",
		VoiceUsed:   voiceID,
		Latency:     latency,
	}, nil
}
