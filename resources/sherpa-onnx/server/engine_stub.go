//go:build !sherpa_onnx || !cgo

package main

import (
	"fmt"
	"os"
)

func newEngineFromEnv() (TTS, error) {
	_ = os.Getenv("SHERPA_ONNX_MODEL_DIR")
	return nil, fmt.Errorf("sherpa-onnx native TTS is unavailable: this server must be built with cgo and the sherpa_onnx tag")
}

func newStreamingEngineFromEnv() (StreamingSTT, error) {
	return UnsupportedStreaming{}, fmt.Errorf("sherpa-onnx native streaming STT is unavailable: this server must be built with cgo and the sherpa_onnx tag")
}

func newSpeakerRuntimeFromEnv() (SpeakerRuntime, error) {
	return nil, fmt.Errorf("sherpa-onnx native speaker runtime is unavailable: this server must be built with cgo and the sherpa_onnx tag")
}
