package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// TTSSynthesizer is the testability seam for TTS synthesis.
// DOC: docs/internal/SEAMS.md#tts-synthesizer-seam
type TTSSynthesizer interface {
	Synthesize(ctx context.Context, req SynthesizeRequest) (io.ReadCloser, string, error)
}

// SynthesizeRequest holds parameters for a TTS synthesis call.
type SynthesizeRequest struct {
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format"`
	Speed          float64 `json:"speed,omitempty"`
}

// KokoroSynthesizer proxies synthesis requests to a Kokoro-FastAPI instance.
type KokoroSynthesizer struct {
	BaseURL string
	Client  *http.Client
}

func (k *KokoroSynthesizer) Synthesize(ctx context.Context, req SynthesizeRequest) (io.ReadCloser, string, error) {
	// Build the Kokoro-FastAPI request body (OpenAI-compatible)
	body := struct {
		Input          string  `json:"input"`
		Voice          string  `json:"voice"`
		ResponseFormat string  `json:"response_format"`
		Speed          float64 `json:"speed,omitempty"`
		Model          string  `json:"model"`
	}{
		Input:          req.Input,
		Voice:          req.Voice,
		ResponseFormat: req.ResponseFormat,
		Speed:          req.Speed,
		Model:          "kokoro",
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", k.BaseURL+"/v1/audio/speech", strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := k.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, "", err
	}

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		return nil, "", fmt.Errorf("kokoro returned status %d: %s", resp.StatusCode, string(errBody))
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		// Default based on format
		switch req.ResponseFormat {
		case "wav":
			contentType = "audio/wav"
		case "opus":
			contentType = "audio/opus"
		case "flac":
			contentType = "audio/flac"
		default:
			contentType = "audio/mpeg"
		}
	}

	return resp.Body, contentType, nil
}

// Content-type mapping for response format
var formatContentTypes = map[string]string{
	"mp3":  "audio/mpeg",
	"wav":  "audio/wav",
	"opus": "audio/opus",
	"flac": "audio/flac",
}

const maxSynthesizeInputLength = 5000

// HTTP handler for /api/v1/tts/synthesize moved to handlers/tts. The
// validation, default-application, and cache-on-write logic now live in
// tts_adapter.go's Synthesize.
