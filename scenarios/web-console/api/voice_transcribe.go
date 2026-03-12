package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"time"
)

const maxAudioSize = 10 << 20 // 10 MB

var whisperURL = "http://localhost:8090/asr?output=json&language=en"

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

	file, header, err := r.FormFile("audio_file")
	if err != nil {
		writeCatalogError(w, "invalid_body", "Missing audio_file field")
		return
	}
	defer file.Close()

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		part, err := writer.CreateFormFile("audio_file", header.Filename)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			pw.CloseWithError(err)
			return
		}
		writer.Close()
	}()

	req, err := http.NewRequestWithContext(ctx, "POST", whisperURL, pr)
	if err != nil {
		log.Printf("voice transcribe: create request: %v", err)
		writeCatalogError(w, "voice_transcribe_failed", "Failed to create transcription request")
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("voice transcribe: proxy request: %v", err)
		writeCatalogError(w, "voice_transcribe_failed", "Whisper request failed")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("voice transcribe: whisper returned %d", resp.StatusCode)
		writeCatalogError(w, "voice_transcribe_failed", "Whisper returned an error")
		return
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("voice transcribe: decode response: %v", err)
		writeCatalogError(w, "voice_transcribe_failed", "Failed to decode Whisper response")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"text": result.Text})
}
