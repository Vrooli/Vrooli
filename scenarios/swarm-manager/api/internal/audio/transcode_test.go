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

	// Generate a tiny valid WAV as input (1 sample of silence).
	// WAV header for 16kHz mono 16-bit PCM, 2 bytes of data.
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

func TestTranscode_InvalidInput(t *testing.T) {
	if !ffmpegOnPath() {
		t.Skip("ffmpeg not available")
	}

	garbage := []byte("not valid audio at all")
	output, err := Transcode(context.Background(), garbage)
	if err == nil {
		t.Error("expected error for invalid audio input")
	}
	if &output[0] != &garbage[0] {
		t.Error("expected original audio returned on error (graceful fallback)")
	}
}

func TestTranscode_ContextCanceled(t *testing.T) {
	if !ffmpegOnPath() {
		t.Skip("ffmpeg not available")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	input := []byte("some audio")
	output, err := Transcode(ctx, input)
	if err == nil {
		t.Error("expected error for canceled context")
	}
	if &output[0] != &input[0] {
		t.Error("expected original audio returned on context cancellation")
	}
}
