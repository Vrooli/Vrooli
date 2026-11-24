package audio

import (
	"bytes"
	"context"
	"encoding/binary"
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
	if got, want := binary.LittleEndian.Uint32(output[4:8]), uint32(len(output)-8); got != want {
		t.Fatalf("RIFF size = %d, want finite file size %d", got, want)
	}
	dataTag := bytes.Index(output, []byte("data"))
	if dataTag < 0 || dataTag+8 > len(output) {
		t.Fatalf("output has no complete data chunk: %x", output[:min(len(output), 64)])
	}
	if got, want := binary.LittleEndian.Uint32(output[dataTag+4:dataTag+8]), uint32(len(output)-dataTag-8); got != want {
		t.Fatalf("data size = %d, want finite payload size %d", got, want)
	}

	generalized, err := TranscodeWithInput(context.Background(), wavHeader, "wav", "wav", 0, 0, 0)
	if err != nil {
		t.Fatalf("TranscodeWithInput: %v", err)
	}
	dataTag = bytes.Index(generalized, []byte("data"))
	if len(generalized) < 12 || string(generalized[:4]) != "RIFF" || binary.LittleEndian.Uint32(generalized[4:8]) != uint32(len(generalized)-8) || dataTag < 0 || dataTag+8 > len(generalized) || binary.LittleEndian.Uint32(generalized[dataTag+4:dataTag+8]) != uint32(len(generalized)-dataTag-8) {
		t.Fatalf("TranscodeWithInput returned non-finite WAV sizes: %x", generalized[:min(len(generalized), 64)])
	}
}

func TestTranscodeWithInput_AdditionalDeclaredFormatsUseRealFFmpeg(t *testing.T) {
	requireRealFFmpeg(t)

	input := sineWAV(t, 0.5, 16000)
	tests := []struct {
		format string
		prefix []byte
		minLen int
	}{
		{format: "webm", prefix: []byte{0x1a, 0x45, 0xdf, 0xa3}},
		{format: "opus", prefix: []byte("OggS")},
		{format: "pcm_s16le", minLen: 16000},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			output, err := TranscodeWithInput(context.Background(), input, "wav", tt.format, 0, 0, 0)
			if err != nil {
				t.Fatalf("TranscodeWithInput(%q): %v", tt.format, err)
			}
			if len(output) < tt.minLen || len(output) < len(tt.prefix) {
				t.Fatalf("%s output too short: got %d bytes, want at least %d", tt.format, len(output), max(tt.minLen, len(tt.prefix)))
			}
			if len(tt.prefix) > 0 && !bytes.HasPrefix(output, tt.prefix) {
				t.Fatalf("%s output prefix = %x, want %x", tt.format, output[:len(tt.prefix)], tt.prefix)
			}
			if tt.format == "pcm_s16le" {
				decoded, err := TranscodeWithInput(context.Background(), output, "pcm_s16le", "wav", 16000, 1, 0)
				if err != nil {
					t.Fatalf("decode PCM S16LE to WAV: %v", err)
				}
				if len(decoded) < 12 || string(decoded[:4]) != "RIFF" || string(decoded[8:12]) != "WAVE" {
					t.Fatalf("PCM S16LE input did not decode to WAV: %x", decoded[:min(len(decoded), 16)])
				}
			}
		})
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
