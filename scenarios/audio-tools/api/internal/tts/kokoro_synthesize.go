package tts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"audio-tools/internal/httpc"
)

// seam: Synthesizer is the local-TTS engine seam (SEAMS.md row
// "tts.Synthesizer"). Production wires the kokoro HTTP synthesizer;
// tests wire fakes.
//
// Synthesizer is the testability seam for TTS synthesis.
type Synthesizer interface {
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
	Doer    httpc.Doer
}

func (k *KokoroSynthesizer) Synthesize(ctx context.Context, req SynthesizeRequest) (io.ReadCloser, string, error) {
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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, k.BaseURL+"/v1/audio/speech", strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := k.Doer.Do(httpReq)
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
		contentType = responseFormatContentType(req.ResponseFormat)
	}

	return resp.Body, contentType, nil
}

func responseFormatContentType(format string) string {
	switch format {
	case "wav":
		return "audio/wav"
	case "opus":
		return "audio/opus"
	case "flac":
		return "audio/flac"
	default:
		return "audio/mpeg"
	}
}
