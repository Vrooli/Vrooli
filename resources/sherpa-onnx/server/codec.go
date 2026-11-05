package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
)

// Encoder is the format boundary between sherpa-onnx's float PCM output and
// the existing Kokoro response-format contract. Keeping it as a seam makes
// the HTTP contract testable without requiring ffmpeg in unit-test images.
type Encoder interface {
	Encode(context.Context, Audio, string) ([]byte, string, error)
}

type ffmpegEncoder struct {
	command   string
	available bool
}

type encoderReadiness interface {
	Ready() bool
}

func newFFmpegEncoder() Encoder {
	command := os.Getenv("SHERPA_ONNX_FFMPEG")
	if command == "" {
		command = "ffmpeg"
	}
	_, err := exec.LookPath(command)
	return ffmpegEncoder{command: command, available: err == nil}
}

func (e ffmpegEncoder) Ready() bool { return e.available }

func (e ffmpegEncoder) Encode(ctx context.Context, audio Audio, format string) ([]byte, string, error) {
	if format == "wav" {
		wav, err := encodeWAV(audio)
		return wav, "audio/wav", err
	}
	if !e.available {
		return nil, "", fmt.Errorf("encode %s: ffmpeg executable %q is unavailable", format, e.command)
	}
	codecArgs, contentType, ok := codecArgsFor(format)
	if !ok {
		return nil, "", fmt.Errorf("unsupported response format %q", format)
	}
	pcm := make([]byte, len(audio.Samples)*4)
	for i, sample := range audio.Samples {
		binary.LittleEndian.PutUint32(pcm[i*4:], math.Float32bits(sample))
	}
	args := []string{
		"-y", "-loglevel", "error",
		"-f", "f32le", "-ar", fmt.Sprint(audio.SampleRate), "-ac", "1",
		"-i", "pipe:0",
	}
	args = append(args, codecArgs...)
	args = append(args, "pipe:1")
	cmd := exec.CommandContext(ctx, e.command, args...)
	cmd.Stdin = bytes.NewReader(pcm)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return nil, "", fmt.Errorf("encode %s: %w: %s", format, err, stderr.String())
		}
		return nil, "", fmt.Errorf("encode %s: %w", format, err)
	}
	if len(out) == 0 {
		return nil, "", fmt.Errorf("encode %s returned empty audio", format)
	}
	return out, contentType, nil
}

func codecArgsFor(format string) ([]string, string, bool) {
	switch format {
	case "mp3":
		return []string{"-f", "mp3"}, "audio/mpeg", true
	case "opus":
		return []string{"-c:a", "libopus", "-f", "opus"}, "audio/ogg", true
	case "flac":
		return []string{"-f", "flac"}, "audio/flac", true
	default:
		return nil, "", false
	}
}
