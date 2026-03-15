// DOC: docs/internal/SEAMS.md#audio-transcoding-seam
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"sync"
)

// transcodeAudio converts audio bytes to 16kHz mono WAV via ffmpeg for optimal
// Whisper accuracy. It is a package-level var to enable test injection (seam).
var transcodeAudio = defaultTranscodeAudio

var (
	ffmpegOnce      sync.Once
	ffmpegAvailable bool
)

// checkFfmpeg lazily checks whether ffmpeg is available on $PATH.
// The result is cached for the process lifetime.
func checkFfmpeg() bool {
	ffmpegOnce.Do(func() {
		_, err := exec.LookPath("ffmpeg")
		ffmpegAvailable = err == nil
		if !ffmpegAvailable {
			log.Printf("voice: ffmpeg not found, audio transcoding disabled")
		}
	})
	return ffmpegAvailable
}

// defaultTranscodeAudio pipes audio through ffmpeg to produce 16kHz mono WAV.
// When ffmpeg is unavailable or transcoding fails, the original audio is
// returned unchanged (graceful passthrough).
func defaultTranscodeAudio(ctx context.Context, audio []byte) ([]byte, error) {
	if !checkFfmpeg() {
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
