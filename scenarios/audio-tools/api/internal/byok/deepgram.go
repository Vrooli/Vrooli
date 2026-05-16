package byok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"audio-tools/internal/ai/sttchain"
)

// DeepgramSTT calls Deepgram's /v1/listen endpoint.
type DeepgramSTT struct {
	Endpoint   string
	HTTPClient *http.Client
}

func NewDeepgramSTT() *DeepgramSTT {
	return &DeepgramSTT{
		Endpoint:   "https://api.deepgram.com/v1/listen",
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (a *DeepgramSTT) ID() string    { return "deepgram" }
func (a *DeepgramSTT) Model() string { return "nova-2" }

func (a *DeepgramSTT) IsAvailable(ctx context.Context, key string) bool { return key != "" }

// StreamingCapability — Deepgram natively supports streaming via WSS.
// The streaming TranscribeStreaming implementation lands in Phase E.
// This file declares the capability surface so the interface satisfies
// the new sttchain.BYOKAdapter contract.
func (a *DeepgramSTT) StreamingCapability() bool { return false }

func (a *DeepgramSTT) TranscribeStreaming(_ context.Context, _ string, _ sttchain.StreamStart, _ <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
	return nil, nil
}

func (a *DeepgramSTT) Transcribe(ctx context.Context, key string, req sttchain.Request) (*sttchain.Result, error) {
	if key == "" {
		return nil, fmt.Errorf("deepgram: missing API key")
	}
	u, err := url.Parse(a.Endpoint)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("model", "nova-2")
	q.Set("smart_format", "true")
	if req.Language != "" {
		q.Set("language", req.Language)
	}
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(req.Audio))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Token "+key)
	httpReq.Header.Set("Content-Type", contentTypeFor(req.Format))

	start := time.Now()
	resp, err := a.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("deepgram: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("deepgram: HTTP %d: %s", resp.StatusCode, truncate(string(raw), 256))
	}

	var out struct {
		Results struct {
			Channels []struct {
				Alternatives []struct {
					Transcript string  `json:"transcript"`
					Confidence float64 `json:"confidence"`
				} `json:"alternatives"`
			} `json:"channels"`
		} `json:"results"`
		Metadata struct {
			Duration float64 `json:"duration"`
		} `json:"metadata"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("deepgram: decode response: %w", err)
	}
	text := ""
	if len(out.Results.Channels) > 0 && len(out.Results.Channels[0].Alternatives) > 0 {
		text = out.Results.Channels[0].Alternatives[0].Transcript
	}
	return &sttchain.Result{
		Text:             text,
		DetectedLanguage: req.Language,
		DurationSeconds:  out.Metadata.Duration,
		ModelID:          "nova-2",
		Latency:          time.Since(start),
	}, nil
}

func contentTypeFor(format string) string {
	switch format {
	case "wav":
		return "audio/wav"
	case "mp3":
		return "audio/mpeg"
	case "flac":
		return "audio/flac"
	case "ogg":
		return "audio/ogg"
	case "webm":
		return "audio/webm"
	default:
		return "application/octet-stream"
	}
}
