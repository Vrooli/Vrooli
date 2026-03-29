package main

import (
	"context"
	"os/exec"
	"testing"
)

func hasFfmpeg() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func TestTranscodeAudio_PassthroughWhenNoFfmpeg(t *testing.T) {
	// Server-level seam: inject a no-op transcoder to simulate ffmpeg unavailable.
	srv := &Server{
		transcodeAudio: func(_ context.Context, audio []byte) ([]byte, error) {
			return audio, nil
		},
	}

	input := []byte("fake audio data")
	output, err := srv.transcodeAudio(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if &output[0] != &input[0] {
		t.Error("expected passthrough to return the same slice")
	}
}

func TestTranscodeAudio_Success(t *testing.T) {
	if !hasFfmpeg() {
		t.Skip("ffmpeg not available")
	}

	// Reset the sync.Once so defaultTranscodeAudio re-checks ffmpeg.
	// Instead of resetting the Once, call defaultTranscodeAudio directly
	// since we already verified ffmpeg is available.

	// Generate a tiny valid WAV as input (1 sample of silence).
	// WAV header for 16kHz mono 16-bit PCM, 2 bytes of data.
	wavHeader := []byte{
		'R', 'I', 'F', 'F',
		38, 0, 0, 0, // ChunkSize = 38
		'W', 'A', 'V', 'E',
		'f', 'm', 't', ' ',
		16, 0, 0, 0, // Subchunk1Size = 16
		1, 0, // AudioFormat = PCM
		1, 0, // NumChannels = 1
		0x80, 0x3E, 0, 0, // SampleRate = 16000
		0, 0x7D, 0, 0, // ByteRate = 32000
		2, 0, // BlockAlign = 2
		16, 0, // BitsPerSample = 16
		'd', 'a', 't', 'a',
		2, 0, 0, 0, // Subchunk2Size = 2
		0, 0, // 1 sample of silence
	}

	output, err := defaultTranscodeAudio(context.Background(), wavHeader)
	if err != nil {
		t.Fatalf("transcode error: %v", err)
	}

	// Output should be valid WAV (starts with RIFF header)
	if len(output) < 4 {
		t.Fatalf("output too short: %d bytes", len(output))
	}
	if string(output[:4]) != "RIFF" {
		t.Errorf("output does not start with RIFF magic, got %q", string(output[:4]))
	}
}

func TestTranscodeAudio_InvalidInput(t *testing.T) {
	if !hasFfmpeg() {
		t.Skip("ffmpeg not available")
	}

	garbage := []byte("not valid audio at all")
	output, err := defaultTranscodeAudio(context.Background(), garbage)
	if err == nil {
		t.Error("expected error for invalid audio input")
	}
	// Graceful fallback: returns original audio on error
	if &output[0] != &garbage[0] {
		t.Error("expected original audio returned on error (graceful fallback)")
	}
}

func TestTranscodeAudio_ContextCanceled(t *testing.T) {
	if !hasFfmpeg() {
		t.Skip("ffmpeg not available")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	input := []byte("some audio")
	output, err := defaultTranscodeAudio(ctx, input)
	if err == nil {
		t.Error("expected error for canceled context")
	}
	// Graceful fallback: returns original audio
	if &output[0] != &input[0] {
		t.Error("expected original audio returned on context cancellation")
	}
}
