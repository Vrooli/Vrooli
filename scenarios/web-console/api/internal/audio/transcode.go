// Package audio provides audio-format transcoding for the web-console
// voice/transcription path. The Transcode function pipes audio through
// ffmpeg to produce 16 kHz mono WAV — the format the transcription
// providers expect. When ffmpeg is unavailable, the original payload is
// returned unchanged so the caller can fall back gracefully.
package audio

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"sync"
)

var (
	ffmpegOnce      sync.Once
	ffmpegAvailable bool
)

// hasFfmpeg lazily checks whether ffmpeg is available on $PATH. The result
// is cached for the process lifetime.
func hasFfmpeg() bool {
	ffmpegOnce.Do(func() {
		_, err := exec.LookPath("ffmpeg")
		ffmpegAvailable = err == nil
		if !ffmpegAvailable {
			log.Printf("voice: ffmpeg not found, audio transcoding disabled")
		}
	})
	return ffmpegAvailable
}

// Transcode pipes audio through ffmpeg to produce 16 kHz mono WAV. When
// ffmpeg is unavailable or transcoding fails, the original audio is
// returned unchanged (graceful passthrough).
func Transcode(ctx context.Context, audio []byte) ([]byte, error) {
	if !hasFfmpeg() {
		return audio, nil
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", "pipe:0",
		"-ar", "16000",
		"-ac", "1",
		"-f", "wav",
		"-loglevel", "error",
		"pipe:1",
	)
	cmd.Stdin = bytes.NewReader(audio)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return audio, fmt.Errorf("ffmpeg: %w: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}
