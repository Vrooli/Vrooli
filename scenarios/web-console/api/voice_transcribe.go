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

// whisperURL is the Whisper ASR endpoint. Initialized from WHISPER_URL env var
// with a sensible default for cross-platform portability.
var whisperURL = resolveWhisperURL()

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

	whisperStart := time.Now()
	text, err := transcribeBytes(ctx, raw, language, true, "")
	if err != nil {
		log.Printf("voice-http: whisper failed after %dms: %v", time.Since(whisperStart).Milliseconds(), err)
		writeCatalogError(w, "voice_transcribe_failed", "Whisper request failed")
		return
	}

	log.Printf("voice-http: transcribed %d bytes -> %d chars in %dms (total %dms)",
		len(raw), len(text), time.Since(whisperStart).Milliseconds(), time.Since(reqStart).Milliseconds())
	writeJSON(w, http.StatusOK, map[string]string{"text": text})
}
