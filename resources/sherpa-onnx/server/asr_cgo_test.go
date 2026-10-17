//go:build cgo && sherpa_onnx

package main

import (
	"encoding/binary"
	"os"
	"testing"
)

// TestNativeStreamingModelSmoke is intentionally a short model-load/forward
// contract check. It is opt-in because model data is a managed resource
// artifact, not a repository fixture.
func TestNativeStreamingModelSmoke(t *testing.T) {
	modelDir := os.Getenv("SHERPA_ONNX_STREAMING_MODEL_DIR")
	wavPath := os.Getenv("SHERPA_ONNX_STREAMING_TEST_WAV")
	if modelDir == "" || wavPath == "" {
		t.Skip("set SHERPA_ONNX_STREAMING_MODEL_DIR and SHERPA_ONNX_STREAMING_TEST_WAV")
	}
	t.Setenv("SHERPA_ONNX_STREAMING_MODEL_DIR", modelDir)
	engine, err := newStreamingEngineFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	stream, err := engine.NewStream()
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	wav, err := os.ReadFile(wavPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(wav) < 44 || string(wav[:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		t.Fatal("test input is not a RIFF/WAVE file")
	}
	if binary.LittleEndian.Uint16(wav[34:36]) != 16 || binary.LittleEndian.Uint16(wav[22:24]) != 1 {
		t.Fatal("test input must be mono signed-16 PCM")
	}
	var text string
	for offset := 44; offset < len(wav); offset += 3200 {
		end := offset + 3200
		if end > len(wav) {
			end = len(wav)
		}
		if (end-offset)%2 != 0 {
			end--
		}
		events, err := stream.AcceptPCM(wav[offset:end])
		if err != nil {
			t.Fatal(err)
		}
		for _, event := range events {
			if event.Text != "" {
				text = event.Text
			}
		}
	}
	events, err := stream.Finish()
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Text != "" {
			text = event.Text
		}
	}
	if text == "" {
		t.Fatal("streaming model produced no text")
	}
	t.Logf("recognized text = %q", text)
}
