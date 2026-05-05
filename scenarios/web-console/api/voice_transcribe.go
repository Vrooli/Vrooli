package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const maxAudioSize = 10 << 20 // 10 MB

// resolveWhisperURL returns the Whisper ASR endpoint URL from WHISPER_URL env
// var with a sensible default for cross-platform portability.

func resolveWhisperURL() string {
	base := "http://localhost:8090"
	if v := os.Getenv("WHISPER_URL"); v != "" {
		base = v
	}
	return base + "/asr?output=json"
}

func (s *Server) handleVoiceTranscribe(w http.ResponseWriter, r *http.Request) {
	reqStart := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if !s.capabilities.IsAvailable(ctx, "whisper-stt") {
		writeCatalogError(w, "voice_unavailable", "Voice transcription is currently unavailable")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAudioSize)
	if err := r.ParseMultipartForm(maxAudioSize); err != nil {
		writeCatalogError(w, "invalid_body", "Failed to parse multipart form: "+err.Error())
		return
	}

	// Empty language = Whisper auto-detects.
	language := r.URL.Query().Get("language")

	// Speaker-verification bypass — strictly the literal "true". Any other
	// value (including "1", "yes", "TRUE", trailing whitespace, or omitted)
	// keeps the verification gate active. Explicit, typo-safe.
	skipSpeakerVerification := r.URL.Query().Get("skip_speaker_verification") == "true"

	file, _, err := r.FormFile("audio_file")
	if err != nil {
		writeCatalogError(w, "invalid_body", "Missing audio_file field")
		return
	}
	defer file.Close()

	// Buffer the uploaded audio so we can transcode before forwarding.
	raw, err := io.ReadAll(file)
	if err != nil {
		log.Printf("voice transcribe: read audio: %v", err)
		writeCatalogError(w, "voice_transcribe_failed", "Failed to read audio data")
		return
	}
	log.Printf("voice-http: received %d bytes, parse took %dms", len(raw), time.Since(reqStart).Milliseconds())

	if skipSpeakerVerification {
		// User-initiated retry overriding a prior false rejection. Skip the
		// verification gate entirely and proceed straight to Whisper. The
		// metric lets operators see bypass usage without touching logs.
		if s.metrics != nil {
			s.metrics.VoiceSkipVerificationTotal.Add(1)
		}
		log.Printf("voice-http: speaker verification bypassed bytes=%d", len(raw))
	} else {
		verifyCtx, verifyCancel := context.WithTimeout(ctx, 10*time.Second)
		decision := s.evaluateSpeakerVerification(verifyCtx, raw)
		verifyCancel()
		if decision.Enabled {
			if decision.Applied {
				log.Printf(
					"voice-http: speaker decision matched=%v allowed=%v score=%.3f threshold=%.3f profile=%s mode=%s",
					decision.Matched,
					decision.Allowed,
					decision.Score,
					decision.Threshold,
					decision.ProfileID,
					decision.Mode,
				)
			} else if decision.ErrorMessage != "" {
				log.Printf("voice-http: %s", formatSpeakerDecisionError(decision))
			}
			if !decision.Allowed {
				writeJSON(w, http.StatusOK, map[string]string{"text": ""})
				return
			}
		}
	}

	whisperStart := time.Now()
	text, err := s.transcribeBytes(ctx, raw, language, true, "")
	if err != nil {
		log.Printf("voice-http: whisper failed after %dms: %v", time.Since(whisperStart).Milliseconds(), err)
		writeCatalogError(w, "voice_transcribe_failed", "Whisper request failed")
		return
	}

	if isWhisperHallucination(text) {
		log.Printf("voice-http: filtered hallucination: %q", text)
		text = ""
	}
	log.Printf("voice-http: transcribed %d bytes -> %d chars in %dms (total %dms)",
		len(raw), len(text), time.Since(whisperStart).Milliseconds(), time.Since(reqStart).Milliseconds())
	writeJSON(w, http.StatusOK, map[string]string{"text": text})
}
