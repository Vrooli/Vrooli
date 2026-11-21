package audio

import (
	"context"
	"os/exec"
	"testing"
)

func ffmpegOnPath() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func TestTranscode_Success(t *testing.T) {
	if !ffmpegOnPath() {
		t.Skip("ffmpeg not available")
	}

	// Tiny valid WAV: 16 kHz mono 16-bit PCM, 1 sample of silence.
	wavHeader := []byte{
		'R', 'I', 'F', 'F',
		38, 0, 0, 0,
		'W', 'A', 'V', 'E',
		'f', 'm', 't', ' ',
		16, 0, 0, 0,
		1, 0,
		1, 0,
		0x80, 0x3E, 0, 0,
		0, 0x7D, 0, 0,
		2, 0,
		16, 0,
		'd', 'a', 't', 'a',
		2, 0, 0, 0,
		0, 0,
	}

	output, err := Transcode(context.Background(), wavHeader)
	if err != nil {
		t.Fatalf("transcode error: %v", err)
	}
	if len(output) < 4 {
		t.Fatalf("output too short: %d bytes", len(output))
	}
	if string(output[:4]) != "RIFF" {
		t.Errorf("output does not start with RIFF magic, got %q", string(output[:4]))
	}
}

// TestTranscode_InvalidInput verifies the greenfield contract: invalid
// audio surfaces as an error, never as a silent passthrough.
func TestTranscode_InvalidInput(t *testing.T) {
	if !ffmpegOnPath() {
		t.Skip("ffmpeg not available")
	}
	_, err := Transcode(context.Background(), []byte("not valid audio at all"))
	if err == nil {
		t.Fatal("expected error for invalid audio input")
	}
}

func TestTranscode_ContextCanceled(t *testing.T) {
	if !ffmpegOnPath() {
		t.Skip("ffmpeg not available")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Transcode(ctx, []byte("some audio"))
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}
