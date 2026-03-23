package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

// handleTTSSynthesize proxies synthesis to Kokoro and streams audio back.
// POST /api/v1/tts/synthesize
func (s *Server) handleTTSSynthesize(w http.ResponseWriter, r *http.Request) {
	if s.ttsSynthesizer == nil {
		writeCatalogError(w, "not_configured", "TTS synthesis is not configured")
		return
	}
	if !s.capabilities.IsAvailable(r.Context(), "kokoro-tts") {
		writeCatalogError(w, "tts_unavailable", "Kokoro TTS is not available")
		return
	}

	var req SynthesizeRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	// Validate
	req.Input = strings.TrimSpace(req.Input)
	if req.Input == "" {
		writeCatalogError(w, "tts_input_required", "input is required")
		return
	}
	if len(req.Input) > maxSynthesizeInputLength {
		log.Printf("tts-synthesize: input too long (%d chars, limit %d)", len(req.Input), maxSynthesizeInputLength)
		writeCatalogError(w, "tts_input_too_long", "input exceeds maximum length of 5000 characters")
		return
	}

	// Apply defaults
	if req.Voice == "" {
		cfg := s.getTTSConfig()
		req.Voice = cfg.KokoroVoice
		if req.Voice == "" {
			req.Voice = "af_heart"
		}
	}
	if req.ResponseFormat == "" {
		req.ResponseFormat = "mp3"
	}
	if _, ok := formatContentTypes[req.ResponseFormat]; !ok {
		writeCatalogError(w, "tts_invalid_format", "unsupported response_format; use mp3, wav, opus, or flac")
		return
	}
	const maxTTSSpeed = 4.0
	if req.Speed <= 0 {
		req.Speed = 1.0
	} else if req.Speed > maxTTSSpeed {
		req.Speed = maxTTSSpeed
	}

	audioBody, contentType, err := s.ttsSynthesizer.Synthesize(r.Context(), req)
	if err != nil {
		log.Printf("tts-synthesize: synthesis failed: %v", err)
		writeCatalogError(w, "tts_synthesis_failed", "synthesis failed")
		return
	}
	defer audioBody.Close()

	// If an eventId is provided, read the full response so we can both
	// stream it to the client and opportunistically cache it.
	eventID := r.URL.Query().Get("eventId")
	version := r.URL.Query().Get("version")
	if eventID != "" && s.ttsCache != nil {
		if version == "" {
			version = "active"
		}
		data, readErr := io.ReadAll(audioBody)
		if readErr != nil {
			log.Printf("tts-synthesize: read for cache: %v", readErr)
			writeCatalogError(w, "tts_synthesis_failed", "synthesis failed")
			return
		}
		if len(data) > 0 {
			s.ttsCache.Put(TTSCacheKey{
				EventID: eventID,
				Voice:   req.Voice,
				Speed:   req.Speed,
				Version: version,
			}, data, contentType)
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		if _, writeErr := w.Write(data); writeErr != nil {
			log.Printf("tts-synthesize: write cached response: %v", writeErr)
		}
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, audioBody); err != nil {
		log.Printf("tts-synthesize: streaming audio to client: %v", err)
	}
}
