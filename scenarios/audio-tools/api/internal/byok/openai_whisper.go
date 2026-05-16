package byok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"audio-tools/internal/ai/sttchain"
)

// OpenAIWhisperSTT calls OpenAI's audio/transcriptions endpoint.
// Model: whisper-1. Audio bytes are uploaded as multipart form-data.
type OpenAIWhisperSTT struct {
	Endpoint   string
	HTTPClient *http.Client
}

func NewOpenAIWhisperSTT() *OpenAIWhisperSTT {
	return &OpenAIWhisperSTT{
		Endpoint:   "https://api.openai.com/v1/audio/transcriptions",
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (a *OpenAIWhisperSTT) ID() string    { return "openai-whisper" }
func (a *OpenAIWhisperSTT) Model() string { return "whisper-1" }

func (a *OpenAIWhisperSTT) IsAvailable(ctx context.Context, key string) bool {
	return key != ""
}

func (a *OpenAIWhisperSTT) Transcribe(ctx context.Context, key string, req sttchain.Request) (*sttchain.Result, error) {
	if key == "" {
		return nil, fmt.Errorf("openai-whisper: missing API key")
	}
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)

	fw, err := mw.CreateFormFile("file", "audio."+req.Format)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(fw, bytes.NewReader(req.Audio)); err != nil {
		return nil, err
	}
	_ = mw.WriteField("model", "whisper-1")
	if req.Language != "" {
		_ = mw.WriteField("language", req.Language)
	}
	if req.InitialPrompt != "" {
		_ = mw.WriteField("prompt", req.InitialPrompt)
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.Endpoint, body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+key)
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())

	start := time.Now()
	resp, err := a.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai-whisper: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai-whisper: HTTP %d: %s", resp.StatusCode, truncate(string(raw), 256))
	}

	var out struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("openai-whisper: decode response: %w", err)
	}
	return &sttchain.Result{
		Text:             out.Text,
		DetectedLanguage: req.Language,
		ModelID:          "whisper-1",
		Latency:          time.Since(start),
	}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
