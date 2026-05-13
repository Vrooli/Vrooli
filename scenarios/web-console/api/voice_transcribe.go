package main

import (
	"os"
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

// The /voice/transcribe HTTP handler has moved to the Connect VoiceService
// (see voice_adapter.go).
