//go:build cgo && sherpa_onnx

package main

/*
#cgo linux LDFLAGS: -lsherpa-onnx-c-api -lonnxruntime
#cgo darwin LDFLAGS: -lsherpa-onnx-c-api -lonnxruntime
#cgo windows LDFLAGS: -lsherpa-onnx-c-api -lonnxruntime
#include <stdlib.h>
#include <sherpa-onnx/c-api/c-api.h>
*/
import "C"

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"unsafe"
)

type nativeEngine struct {
	mu  sync.Mutex
	tts *C.SherpaOnnxOfflineTts
}

func newEngineFromEnv() (TTS, error) {
	modelDir := os.Getenv("SHERPA_ONNX_MODEL_DIR")
	if modelDir == "" {
		dataDir := os.Getenv("RESOURCE_DATA_DIR")
		if dataDir == "" {
			return nil, fmt.Errorf("SHERPA_ONNX_MODEL_DIR or RESOURCE_DATA_DIR is required")
		}
		modelDir = filepath.Join(dataDir, "models", "kokoro")
	}
	modelDir, err := filepath.Abs(modelDir)
	if err != nil {
		return nil, fmt.Errorf("resolve model directory: %w", err)
	}
	model := filepath.Join(modelDir, envOr("SHERPA_ONNX_KOKORO_MODEL", "model.int8.onnx"))
	voices := filepath.Join(modelDir, envOr("SHERPA_ONNX_KOKORO_VOICES", "voices.bin"))
	tokens := filepath.Join(modelDir, envOr("SHERPA_ONNX_KOKORO_TOKENS", "tokens.txt"))
	data := filepath.Join(modelDir, envOr("SHERPA_ONNX_KOKORO_DATA_DIR", "espeak-ng-data"))
	lexicon := filepath.Join(modelDir, envOr("SHERPA_ONNX_KOKORO_LEXICON", "lexicon-us-en.txt"))

	modelC := C.CString(model)
	voicesC := C.CString(voices)
	tokensC := C.CString(tokens)
	dataC := C.CString(data)
	lexiconC := C.CString(lexicon)
	providerC := C.CString(envOr("SHERPA_ONNX_PROVIDER", "cpu"))
	defer C.free(unsafe.Pointer(modelC))
	defer C.free(unsafe.Pointer(voicesC))
	defer C.free(unsafe.Pointer(tokensC))
	defer C.free(unsafe.Pointer(dataC))
	defer C.free(unsafe.Pointer(lexiconC))
	defer C.free(unsafe.Pointer(providerC))

	var config C.SherpaOnnxOfflineTtsConfig
	config.model.num_threads = C.int32_t(envInt("SHERPA_ONNX_THREADS", 2))
	config.model.provider = providerC
	config.model.kokoro.model = modelC
	config.model.kokoro.voices = voicesC
	config.model.kokoro.tokens = tokensC
	config.model.kokoro.data_dir = dataC
	config.model.kokoro.lexicon = lexiconC
	config.model.kokoro.lang = C.CString(envOr("SHERPA_ONNX_KOKORO_LANG", "en"))
	defer C.free(unsafe.Pointer(config.model.kokoro.lang))
	config.model.kokoro.length_scale = C.float(1)
	config.max_num_sentences = C.int32_t(1)
	config.silence_scale = C.float(0.2)

	tts := C.SherpaOnnxCreateOfflineTts(&config)
	if tts == nil {
		return nil, fmt.Errorf("sherpa-onnx failed to load Kokoro model %q", model)
	}
	return &nativeEngine{tts: tts}, nil
}

func (e *nativeEngine) Synthesize(ctx context.Context, text string, sid int, speed float32) (Audio, error) {
	if err := ctx.Err(); err != nil {
		return Audio{}, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Audio{}, err
	}
	textC := C.CString(text)
	defer C.free(unsafe.Pointer(textC))
	config := C.SherpaOnnxGenerationConfig{sid: C.int32_t(sid), speed: C.float(speed), silence_scale: C.float(0.2)}
	audio := C.SherpaOnnxOfflineTtsGenerateWithConfig(e.tts, textC, &config, nil, nil)
	if audio == nil {
		return Audio{}, fmt.Errorf("sherpa-onnx failed to synthesize text")
	}
	defer C.SherpaOnnxDestroyOfflineTtsGeneratedAudio(audio)
	if audio.n <= 0 || audio.sample_rate <= 0 {
		return Audio{}, fmt.Errorf("sherpa-onnx returned empty audio")
	}
	samples := make([]float32, int(audio.n))
	copy(samples, unsafe.Slice((*float32)(unsafe.Pointer(audio.samples)), int(audio.n)))
	runtime.KeepAlive(audio)
	return Audio{Samples: samples, SampleRate: int(audio.sample_rate)}, nil
}

func (e *nativeEngine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.tts != nil {
		C.SherpaOnnxDestroyOfflineTts(e.tts)
		e.tts = nil
	}
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err == nil && parsed > 0 {
		return parsed
	}
	return fallback
}
