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
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"unsafe"
)

type nativeStreamingEngine struct {
	mu         sync.Mutex
	recognizer *C.SherpaOnnxOnlineRecognizer
	punct      *C.SherpaOnnxOfflinePunctuation
}

type nativeSTTStream struct {
	engine            *nativeStreamingEngine
	stream            *C.SherpaOnnxOnlineStream
	totalSamples      int64
	segmentStart      int64
	lastText          string
	closed            bool
	maxSegmentSamples int64
}

func newStreamingEngineFromEnv() (StreamingSTT, error) {
	modelDir := os.Getenv("SHERPA_ONNX_STREAMING_MODEL_DIR")
	if modelDir == "" {
		dataDir := os.Getenv("RESOURCE_DATA_DIR")
		if dataDir == "" {
			return nil, fmt.Errorf("SHERPA_ONNX_STREAMING_MODEL_DIR or RESOURCE_DATA_DIR is required")
		}
		modelDir = filepath.Join(dataDir, "models", "asr", "sherpa-onnx-streaming-zipformer-en-2023-06-26")
	}
	modelDir, err := filepath.Abs(modelDir)
	if err != nil {
		return nil, fmt.Errorf("resolve streaming model directory: %w", err)
	}
	path := func(name, fallback string) string {
		if value := os.Getenv(name); value != "" {
			return value
		}
		return filepath.Join(modelDir, fallback)
	}
	encoder := path("SHERPA_ONNX_STREAMING_ENCODER", "encoder-epoch-99-avg-1-chunk-16-left-128.onnx")
	decoder := path("SHERPA_ONNX_STREAMING_DECODER", "decoder-epoch-99-avg-1-chunk-16-left-128.onnx")
	joiner := path("SHERPA_ONNX_STREAMING_JOINER", "joiner-epoch-99-avg-1-chunk-16-left-128.onnx")
	tokens := path("SHERPA_ONNX_STREAMING_TOKENS", "tokens.txt")
	for label, value := range map[string]string{"encoder": encoder, "decoder": decoder, "joiner": joiner, "tokens": tokens} {
		if _, err := os.Stat(value); err != nil {
			return nil, fmt.Errorf("streaming %s %q is unavailable: %w", label, value, err)
		}
	}

	encoderC := C.CString(encoder)
	decoderC := C.CString(decoder)
	joinerC := C.CString(joiner)
	tokensC := C.CString(tokens)
	providerC := C.CString(envOr("SHERPA_ONNX_PROVIDER", "cpu"))
	decodingC := C.CString(envOr("SHERPA_ONNX_STREAMING_DECODING", "greedy_search"))
	modelTypeC := C.CString(envOr("SHERPA_ONNX_STREAMING_MODEL_TYPE", "zipformer2"))
	defer C.free(unsafe.Pointer(encoderC))
	defer C.free(unsafe.Pointer(decoderC))
	defer C.free(unsafe.Pointer(joinerC))
	defer C.free(unsafe.Pointer(tokensC))
	defer C.free(unsafe.Pointer(providerC))
	defer C.free(unsafe.Pointer(decodingC))
	defer C.free(unsafe.Pointer(modelTypeC))

	var config C.SherpaOnnxOnlineRecognizerConfig
	config.feat_config.sample_rate = C.int32_t(streamingSampleRate)
	config.feat_config.feature_dim = C.int32_t(80)
	config.model_config.transducer.encoder = encoderC
	config.model_config.transducer.decoder = decoderC
	config.model_config.transducer.joiner = joinerC
	config.model_config.tokens = tokensC
	config.model_config.provider = providerC
	config.model_config.num_threads = C.int32_t(envInt("SHERPA_ONNX_THREADS", 2))
	config.model_config.model_type = modelTypeC
	config.decoding_method = decodingC
	config.enable_endpoint = 1
	config.rule1_min_trailing_silence = C.float(2.4)
	config.rule2_min_trailing_silence = C.float(1.2)
	config.rule3_min_utterance_length = C.float(20)

	recognizer := C.SherpaOnnxCreateOnlineRecognizer(&config)
	if recognizer == nil {
		return nil, fmt.Errorf("sherpa-onnx failed to load streaming model %q", modelDir)
	}
	punctDir := os.Getenv("SHERPA_ONNX_PUNCTUATION_MODEL_DIR")
	if punctDir == "" {
		dataDir := os.Getenv("RESOURCE_DATA_DIR")
		if dataDir == "" {
			C.SherpaOnnxDestroyOnlineRecognizer(recognizer)
			return nil, fmt.Errorf("SHERPA_ONNX_PUNCTUATION_MODEL_DIR or RESOURCE_DATA_DIR is required")
		}
		punctDir = filepath.Join(dataDir, "models", "punctuation", "sherpa-onnx-punct-ct-transformer-zh-en-vocab272727-2024-04-12-int8")
	}
	punctModel := filepath.Join(punctDir, "model.int8.onnx")
	if _, err := os.Stat(punctModel); err != nil {
		C.SherpaOnnxDestroyOnlineRecognizer(recognizer)
		return nil, fmt.Errorf("punctuation model %q is unavailable: %w", punctModel, err)
	}
	punctModelC := C.CString(punctModel)
	providerPunctC := C.CString(envOr("SHERPA_ONNX_PROVIDER", "cpu"))
	var punctConfig C.SherpaOnnxOfflinePunctuationConfig
	punctConfig.model.ct_transformer = punctModelC
	punctConfig.model.num_threads = C.int32_t(envInt("SHERPA_ONNX_THREADS", 2))
	punctConfig.model.provider = providerPunctC
	punct := C.SherpaOnnxCreateOfflinePunctuation(&punctConfig)
	C.free(unsafe.Pointer(punctModelC))
	C.free(unsafe.Pointer(providerPunctC))
	if punct == nil {
		C.SherpaOnnxDestroyOnlineRecognizer(recognizer)
		return nil, fmt.Errorf("sherpa-onnx failed to load punctuation model %q", punctModel)
	}
	return &nativeStreamingEngine{recognizer: recognizer, punct: punct}, nil
}

func (e *nativeStreamingEngine) NewStream() (STTStream, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.recognizer == nil {
		return nil, fmt.Errorf("streaming recognizer is closed")
	}
	stream := C.SherpaOnnxCreateOnlineStream(e.recognizer)
	if stream == nil {
		return nil, fmt.Errorf("sherpa-onnx failed to create streaming session")
	}
	maxSeconds := envInt("SHERPA_ONNX_STREAMING_MAX_SEGMENT_SECONDS", 15)
	return &nativeSTTStream{engine: e, stream: stream, maxSegmentSamples: int64(maxSeconds) * streamingSampleRate}, nil
}

func (e *nativeStreamingEngine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.recognizer != nil {
		if e.punct != nil {
			C.SherpaOnnxDestroyOfflinePunctuation(e.punct)
			e.punct = nil
		}
		C.SherpaOnnxDestroyOnlineRecognizer(e.recognizer)
		e.recognizer = nil
	}
}

func (e *nativeStreamingEngine) punctuate(text string) (string, error) {
	if e.punct == nil || text == "" {
		return text, nil
	}
	textC := C.CString(text)
	defer C.free(unsafe.Pointer(textC))
	result := C.SherpaOfflinePunctuationAddPunct(e.punct, textC)
	if result == nil {
		return "", fmt.Errorf("sherpa-onnx punctuation returned no text")
	}
	punctuated := C.GoString(result)
	C.SherpaOfflinePunctuationFreeText(result)
	return punctuated, nil
}

func (s *nativeSTTStream) AcceptPCM(pcm []byte) ([]STTEvent, error) {
	if s.closed {
		return nil, fmt.Errorf("streaming session is closed")
	}
	if len(pcm)%2 != 0 {
		return nil, fmt.Errorf("PCM frame has odd byte length")
	}
	if len(pcm) == 0 {
		return nil, nil
	}
	samples := make([]float32, len(pcm)/2)
	for i := range samples {
		samples[i] = float32(int16(binary.LittleEndian.Uint16(pcm[i*2:]))) / 32768
	}
	s.engine.mu.Lock()
	defer s.engine.mu.Unlock()
	if s.engine.recognizer == nil || s.stream == nil {
		return nil, fmt.Errorf("streaming recognizer is closed")
	}
	C.SherpaOnnxOnlineStreamAcceptWaveform(s.stream, C.int32_t(streamingSampleRate), (*C.float)(unsafe.Pointer(&samples[0])), C.int32_t(len(samples)))
	s.totalSamples += int64(len(samples))
	return s.collectLocked(false)
}

func (s *nativeSTTStream) Finish() ([]STTEvent, error) {
	if s.closed {
		return nil, nil
	}
	s.engine.mu.Lock()
	defer s.engine.mu.Unlock()
	if s.engine.recognizer == nil || s.stream == nil {
		return nil, fmt.Errorf("streaming recognizer is closed")
	}
	C.SherpaOnnxOnlineStreamInputFinished(s.stream)
	return s.collectLocked(true)
}

func (s *nativeSTTStream) collectLocked(finished bool) ([]STTEvent, error) {
	for C.SherpaOnnxIsOnlineStreamReady(s.engine.recognizer, s.stream) != 0 {
		C.SherpaOnnxDecodeOnlineStream(s.engine.recognizer, s.stream)
	}
	result := C.SherpaOnnxGetOnlineStreamResult(s.engine.recognizer, s.stream)
	if result == nil {
		return nil, fmt.Errorf("sherpa-onnx returned no streaming result")
	}
	text := C.GoString(result.text)
	C.SherpaOnnxDestroyOnlineRecognizerResult(result)
	endpoint := C.SherpaOnnxOnlineStreamIsEndpoint(s.engine.recognizer, s.stream) != 0
	force := s.maxSegmentSamples > 0 && s.totalSamples-s.segmentStart >= s.maxSegmentSamples
	if text == "" {
		if finished || endpoint || force {
			s.resetLocked()
		}
		return nil, nil
	}
	if !finished && !endpoint && !force {
		if text == s.lastText {
			return nil, nil
		}
		s.lastText = text
		return []STTEvent{{Text: text, StartSample: s.segmentStart, EndSample: s.totalSamples}}, nil
	}
	event := STTEvent{Text: text, Final: true, StartSample: s.segmentStart, EndSample: s.totalSamples}
	punctuated, err := s.engine.punctuate(text)
	if err != nil {
		return nil, err
	}
	event.Text = restoreCapitalization(normalizePunctuation(punctuated))
	s.resetLocked()
	return []STTEvent{event}, nil
}

func (s *nativeSTTStream) resetLocked() {
	C.SherpaOnnxOnlineStreamReset(s.engine.recognizer, s.stream)
	s.segmentStart = s.totalSamples
	s.lastText = ""
}

func (s *nativeSTTStream) Close() {
	if s.closed {
		return
	}
	s.engine.mu.Lock()
	defer s.engine.mu.Unlock()
	if s.stream != nil {
		C.SherpaOnnxDestroyOnlineStream(s.stream)
		s.stream = nil
	}
	s.closed = true
}
