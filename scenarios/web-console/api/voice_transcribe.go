package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"
)

const maxAudioSize = 10 << 20 // 10 MB

var whisperURL = "http://localhost:8090/asr?output=json"

func (s *Server) handleVoiceTranscribe(w http.ResponseWriter, r *http.Request) {
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

	text, err := transcribeBytes(ctx, raw, language, true, "")
	if err != nil {
		log.Printf("voice transcribe: %v", err)
		writeCatalogError(w, "voice_transcribe_failed", "Whisper request failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"text": text})
}
