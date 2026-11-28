package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"audio-tools/internal/audio"
	"audio-tools/internal/audioformat"
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

// SpeechSynthesizer proxies the shared OpenAI-compatible local speech
// contract. Kokoro-FastAPI, the sherpa-onnx Kokoro adapter, and the isolated
// Kale adapter implement this endpoint, so backend choice stays in composition
// rather than the TTS service or cache path.
type SpeechSynthesizer struct {
	BaseURL        string
	Model          string
	SpeedMode      string
	Doer           httpc.Doer
	Transcode      func(context.Context, []byte, string, string) ([]byte, error)
	SpeedTransform func(context.Context, []byte, float64) ([]byte, error)
}

type preparedSpeechRequest struct {
	model      string
	wireFormat string
	speed      float64
	applyTempo bool
	body       []byte
}

// KokoroSynthesizer is retained as a source-compatible name for existing
// callers; new wiring should use SpeechSynthesizer.
type KokoroSynthesizer = SpeechSynthesizer

func (k *SpeechSynthesizer) Synthesize(ctx context.Context, req SynthesizeRequest) (io.ReadCloser, string, error) {
	prepared, err := k.prepareRequest(req)
	if err != nil {
		return nil, "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, k.BaseURL+"/v1/audio/speech", bytes.NewReader(prepared.body))
	if err != nil {
		return nil, "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := k.Doer.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, "", err
		}
		return nil, "", fmt.Errorf("%w: request %s: %v", ErrUnavailable, prepared.model, err)
	}
	if err := speechStatusError(prepared.model, resp); err != nil {
		return nil, "", err
	}
	return k.readAndConvertResponse(ctx, resp, prepared, req.ResponseFormat)
}

func (k *SpeechSynthesizer) prepareRequest(req SynthesizeRequest) (preparedSpeechRequest, error) {
	model := k.model()
	wireFormat := req.ResponseFormat
	speedMode := k.speedMode()
	if speedMode != SpeedModeNative && speedMode != SpeedModePostprocess {
		return preparedSpeechRequest{}, fmt.Errorf("unsupported TTS speed mode %q", speedMode)
	}
	usesLocalTempo := speedMode == SpeedModePostprocess
	if usesLocalTempo {
		wireFormat = "wav"
	}
	speed := req.Speed
	if speed <= 0 {
		speed = 1
	}
	body := speechRequestBody{Input: req.Input, Voice: req.Voice, ResponseFormat: wireFormat, Model: model}
	if !usesLocalTempo && req.Speed > 0 {
		body.Speed = &speed
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return preparedSpeechRequest{}, err
	}
	return preparedSpeechRequest{model: model, wireFormat: wireFormat, speed: speed, applyTempo: usesLocalTempo && speed != 1, body: encoded}, nil
}

type speechRequestBody struct {
	Input          string   `json:"input"`
	Voice          string   `json:"voice"`
	ResponseFormat string   `json:"response_format"`
	Speed          *float64 `json:"speed,omitempty"`
	Model          string   `json:"model"`
}

func speechStatusError(model string, resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	resp.Body.Close()
	if isTransientSpeechStatus(resp.StatusCode) {
		return fmt.Errorf("%w: %s returned status %d: %s", ErrUnavailable, model, resp.StatusCode, string(errBody))
	}
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity {
		return fmt.Errorf("%w: speech backend %s rejected the request (status %d): %s", ErrInvalidArgument, model, resp.StatusCode, string(errBody))
	}
	return fmt.Errorf("speech backend %s returned status %d: %s", model, resp.StatusCode, string(errBody))
}

func isTransientSpeechStatus(status int) bool {
	switch status {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout, http.StatusTooManyRequests:
		return true
	default:
		return false
	}
}

func (k *SpeechSynthesizer) readAndConvertResponse(ctx context.Context, resp *http.Response, prepared preparedSpeechRequest, requestedFormat string) (io.ReadCloser, string, error) {
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = responseFormatContentType(prepared.wireFormat)
	}
	if prepared.wireFormat == requestedFormat && !prepared.applyTempo {
		// Ownership of the streaming response body transfers to the caller. The
		// TTS service consumes and closes this reader after synthesis.
		return resp.Body, contentType, nil
	}
	defer resp.Body.Close()
	data, err := readBoundedResponse(resp.Body, maxSynthesizeAudioBytes)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, "", err
		}
		return nil, "", fmt.Errorf("%w: read %s response: %v", ErrUnavailable, prepared.model, err)
	}
	return k.convertSpeechResponse(ctx, data, contentType, prepared, requestedFormat)
}

func (k *SpeechSynthesizer) convertSpeechResponse(ctx context.Context, data []byte, contentType string, prepared preparedSpeechRequest, requestedFormat string) (io.ReadCloser, string, error) {
	if prepared.applyTempo {
		transform := k.SpeedTransform
		if transform == nil {
			transform = audio.ChangeSpeed
		}
		var err error
		data, err = transform(ctx, data, prepared.speed)
		if err != nil {
			return nil, "", fmt.Errorf("change %s speech speed: %w", prepared.model, err)
		}
		contentType = "audio/wav"
	}
	if prepared.wireFormat == requestedFormat {
		return io.NopCloser(bytes.NewReader(data)), contentType, nil
	}
	transcode := k.Transcode
	if transcode == nil {
		transcode = func(ctx context.Context, input []byte, inFormat, outFormat string) ([]byte, error) {
			return audio.TranscodeWithInput(ctx, input, inFormat, outFormat, 0, 0, 0)
		}
	}
	converted, err := transcode(ctx, data, "wav", requestedFormat)
	if err != nil {
		return nil, "", fmt.Errorf("convert %s audio to %s: %w", prepared.model, requestedFormat, err)
	}
	return io.NopCloser(bytes.NewReader(converted)), responseFormatContentType(requestedFormat), nil
}

func (k *SpeechSynthesizer) model() string {
	if k != nil && strings.TrimSpace(k.Model) != "" {
		return strings.TrimSpace(k.Model)
	}
	return "kokoro"
}

func (k *SpeechSynthesizer) speedMode() string {
	if k != nil && strings.TrimSpace(k.SpeedMode) != "" {
		return strings.TrimSpace(k.SpeedMode)
	}
	return SpeedModeNative
}

// responseFormatContentType is the fallback content type when the kokoro
// resource omits its own Content-Type header. It delegates to the
// audioformat substrate so the mapping is defined in exactly one place.
func responseFormatContentType(format string) string {
	if f, ok := audioformat.OutputFormatFromString(format); ok {
		return f.ContentType()
	}
	return audioformat.OutputMP3.ContentType()
}
