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
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

const (
	nativeSpeakerSampleRate       = 16_000
	nativeSpeakerModelName        = "sherpa-onnx/3dspeaker-eres2net-large-sv-zh-cn-16k@v1.13.2"
	nativeSpeakerMinEnroll        = 3.0
	nativeSpeakerMinVerify        = 1.0
	nativeSpeakerDefaultThreshold = 0.5
)

type nativeSpeakerRuntime struct {
	mu          sync.Mutex
	profileMu   sync.Mutex
	extractor   *C.SherpaOnnxSpeakerEmbeddingExtractor
	separator   *C.SherpaOnnxOfflineSourceSeparation
	modelName   string
	profileDir  string
	modelDim    int
	serverStart time.Time
}

type nativeSpeakerClip struct {
	ClipID               string    `json:"clip_id"`
	Label                string    `json:"label"`
	Embedding            []float32 `json:"embedding"`
	VoicedSeconds        float64   `json:"voiced_seconds"`
	AudioSeconds         float64   `json:"audio_seconds"`
	SelfConsistencyScore float64   `json:"self_consistency_score"`
	CreatedAt            string    `json:"created_at"`
}

type nativeSpeakerProfile struct {
	ID           string              `json:"id"`
	DisplayName  string              `json:"display_name"`
	Notes        string              `json:"notes"`
	ModelName    string              `json:"model_name"`
	EmbeddingDim int                 `json:"embedding_dim"`
	SampleRate   int                 `json:"sample_rate"`
	Clips        []nativeSpeakerClip `json:"clips"`
	CreatedAt    string              `json:"created_at"`
	UpdatedAt    string              `json:"updated_at"`
}

func newSpeakerRuntimeFromEnv() (SpeakerRuntime, error) {
	dataDir := os.Getenv("RESOURCE_DATA_DIR")
	modelPath := os.Getenv("SHERPA_ONNX_SPEAKER_MODEL")
	separatorDir := os.Getenv("SHERPA_ONNX_SEPARATION_MODEL_DIR")
	profileDir := os.Getenv("SPEAKER_VERIFICATION_PROFILE_DIR")
	if dataDir != "" {
		if modelPath == "" {
			modelPath = filepath.Join(dataDir, "models", "speaker", "3dspeaker_speech_eres2net_large_sv_zh-cn_3dspeaker_16k.onnx")
		}
		if separatorDir == "" {
			separatorDir = filepath.Join(dataDir, "models", "separation", "sherpa-onnx-spleeter-2stems-fp16")
		}
		if profileDir == "" {
			profileDir = filepath.Join(dataDir, "profiles")
		}
	}
	if modelPath == "" || separatorDir == "" || profileDir == "" {
		return nil, fmt.Errorf("speaker model, separation model, and profile directory must be configured")
	}
	for label, path := range map[string]string{
		"speaker model":                  modelPath,
		"separation vocals model":        filepath.Join(separatorDir, "vocals.fp16.onnx"),
		"separation accompaniment model": filepath.Join(separatorDir, "accompaniment.fp16.onnx"),
	} {
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("%s %q is unavailable: %w", label, path, err)
		}
	}
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return nil, fmt.Errorf("create speaker profile directory: %w", err)
	}

	modelC := C.CString(modelPath)
	providerC := C.CString(envOr("SHERPA_ONNX_PROVIDER", "cpu"))
	var embedConfig C.SherpaOnnxSpeakerEmbeddingExtractorConfig
	embedConfig.model = modelC
	embedConfig.num_threads = C.int32_t(envInt("SHERPA_ONNX_THREADS", 2))
	embedConfig.provider = providerC
	extractor := C.SherpaOnnxCreateSpeakerEmbeddingExtractor(&embedConfig)
	C.free(unsafe.Pointer(modelC))
	C.free(unsafe.Pointer(providerC))
	if extractor == nil {
		return nil, fmt.Errorf("sherpa-onnx failed to load speaker embedding model %q", modelPath)
	}

	vocalsC := C.CString(filepath.Join(separatorDir, "vocals.fp16.onnx"))
	accompanimentC := C.CString(filepath.Join(separatorDir, "accompaniment.fp16.onnx"))
	separatorProviderC := C.CString(envOr("SHERPA_ONNX_PROVIDER", "cpu"))
	var separatorConfig C.SherpaOnnxOfflineSourceSeparationConfig
	separatorConfig.model.spleeter.vocals = vocalsC
	separatorConfig.model.spleeter.accompaniment = accompanimentC
	separatorConfig.model.num_threads = C.int32_t(envInt("SHERPA_ONNX_THREADS", 2))
	separatorConfig.model.provider = separatorProviderC
	separator := C.SherpaOnnxCreateOfflineSourceSeparation(&separatorConfig)
	C.free(unsafe.Pointer(vocalsC))
	C.free(unsafe.Pointer(accompanimentC))
	C.free(unsafe.Pointer(separatorProviderC))
	if separator == nil {
		C.SherpaOnnxDestroySpeakerEmbeddingExtractor(extractor)
		return nil, fmt.Errorf("sherpa-onnx failed to load source separation models")
	}

	runtime := &nativeSpeakerRuntime{
		extractor:   extractor,
		separator:   separator,
		modelName:   envOr("SHERPA_ONNX_SPEAKER_MODEL_NAME", nativeSpeakerModelName),
		profileDir:  profileDir,
		modelDim:    int(C.SherpaOnnxSpeakerEmbeddingExtractorDim(extractor)),
		serverStart: time.Now(),
	}
	if runtime.modelDim <= 0 {
		runtime.Close()
		return nil, fmt.Errorf("sherpa-onnx speaker model returned invalid embedding dimension")
	}
	return runtime, nil
}

func (s *nativeSpeakerRuntime) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.separator != nil {
		C.SherpaOnnxDestroyOfflineSourceSeparation(s.separator)
		s.separator = nil
	}
	if s.extractor != nil {
		C.SherpaOnnxDestroySpeakerEmbeddingExtractor(s.extractor)
		s.extractor = nil
	}
}

func (s *nativeSpeakerRuntime) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/ready":
		s.ready(w, r)
	case r.URL.Path == "/v1/info":
		s.info(w, r)
	case r.URL.Path == "/v1/profiles" && r.Method == http.MethodGet:
		s.listProfiles(w)
	case r.URL.Path == "/v1/profiles" && r.Method == http.MethodPost:
		s.enroll(w, r)
	case strings.HasPrefix(r.URL.Path, "/v1/profiles/"):
		s.profileRoute(w, r)
	case r.URL.Path == "/v1/verify" && r.Method == http.MethodPost:
		s.verify(w, r)
	case r.URL.Path == "/v1/extract" && r.Method == http.MethodPost:
		s.extract(w, r)
	default:
		s.writeError(w, http.StatusNotFound, "not found", "")
	}
}

func (s *nativeSpeakerRuntime) ready(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "model_loaded": s.extractor != nil, "profile_store_ok": true, "temp_dir_ok": true, "speaker_model_loaded": s.extractor != nil, "separation_model_loaded": s.separator != nil})
}

func (s *nativeSpeakerRuntime) info(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{
		"backend": "sherpa-onnx", "model": s.modelName, "device": envOr("SHERPA_ONNX_PROVIDER", "cpu"),
		"sample_rate": nativeSpeakerSampleRate, "version": "native-v1", "embedding_dim": s.modelDim,
		"score_agg": "max", "extraction_model": "spleeter-2stems-fp16", "extraction_sample_rate": C.SherpaOnnxOfflineSourceSeparationGetOutputSampleRate(s.separator),
	})
}

func (s *nativeSpeakerRuntime) profileRoute(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "v1" || parts[1] != "profiles" {
		s.writeError(w, http.StatusNotFound, "not found", "")
		return
	}
	id := parts[2]
	if len(parts) == 3 && r.Method == http.MethodGet {
		profile, err := s.loadProfile(id)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error(), "")
			return
		}
		if profile == nil {
			s.writeError(w, http.StatusNotFound, "profile not found", "")
			return
		}
		s.writeJSON(w, http.StatusOK, s.publicProfile(profile, true))
		return
	}
	if len(parts) == 3 && r.Method == http.MethodDelete {
		if !safeProfileID(id) {
			s.writeError(w, http.StatusBadRequest, "invalid profile id", "")
			return
		}
		s.profileMu.Lock()
		err := os.Remove(s.profilePath(id))
		s.profileMu.Unlock()
		if os.IsNotExist(err) {
			s.writeError(w, http.StatusNotFound, "profile not found", "")
			return
		}
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error(), "")
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	if len(parts) == 4 && parts[3] == "clips" && r.Method == http.MethodGet {
		profile, err := s.loadProfile(id)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error(), "")
			return
		}
		if profile == nil {
			s.writeError(w, http.StatusNotFound, "profile not found", "")
			return
		}
		clips := make([]any, 0, len(profile.Clips))
		for _, clip := range profile.Clips {
			clips = append(clips, s.publicClip(clip))
		}
		s.writeJSON(w, http.StatusOK, map[string]any{"profile_id": id, "clips": clips, "count": len(clips)})
		return
	}
	if len(parts) == 5 && parts[3] == "clips" && r.Method == http.MethodDelete {
		s.deleteClip(w, id, parts[4])
		return
	}
	s.writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
}

func (s *nativeSpeakerRuntime) listProfiles(w http.ResponseWriter) {
	entries, err := os.ReadDir(s.profileDir)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	profiles := make([]any, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		profile, err := s.loadProfile(strings.TrimSuffix(entry.Name(), ".json"))
		if err == nil && profile != nil {
			profiles = append(profiles, s.publicProfile(profile, false))
		}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"profiles": profiles, "count": len(profiles)})
}

func (s *nativeSpeakerRuntime) enroll(w http.ResponseWriter, r *http.Request) {
	fields, audio, err := readMultipartAudio(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	samples, sampleRate, err := decodeSpeakerAudio(audio)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	voiced, voicedSeconds := trimSpeakerAudio(samples, sampleRate)
	if voicedSeconds < nativeSpeakerMinEnroll {
		s.writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "insufficient voiced audio", "voiced_seconds": voicedSeconds, "audio_seconds": float64(len(samples)) / float64(sampleRate), "min_voiced_seconds": nativeSpeakerMinEnroll})
		return
	}
	embedding, err := s.embed(voiced)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	id := strings.TrimSpace(fields["profile_id"])
	if id == "" {
		id = fmt.Sprintf("%x", time.Now().UnixNano())
	}
	if !safeProfileID(id) {
		s.writeError(w, http.StatusBadRequest, "invalid profile id", "")
		return
	}
	profile, err := s.loadProfile(id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	if profile == nil {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		profile = &nativeSpeakerProfile{ID: id, DisplayName: fields["display_name"], Notes: fields["notes"], ModelName: s.modelName, EmbeddingDim: s.modelDim, SampleRate: nativeSpeakerSampleRate, CreatedAt: now, UpdatedAt: now}
	} else if profile.ModelName != s.modelName || profile.EmbeddingDim != s.modelDim {
		s.writeError(w, http.StatusConflict, "profile requires re-enrollment under the active speaker model", "speaker_model_mismatch")
		return
	}
	selfScore := -1.0
	selfLabel, selfID := "", ""
	if len(profile.Clips) > 0 {
		selfScore, selfLabel, selfID = bestSpeakerMatch(embedding, profile.Clips)
	}
	clip := nativeSpeakerClip{ClipID: fmt.Sprintf("%x", time.Now().UnixNano()), Label: strings.TrimSpace(fields["label"]), Embedding: embedding, VoicedSeconds: voicedSeconds, AudioSeconds: float64(len(samples)) / float64(sampleRate), SelfConsistencyScore: selfScore, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	profile.Clips = append(profile.Clips, clip)
	profile.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := s.saveProfile(profile); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"profile_id": id, "clip_id": clip.ClipID, "label": clip.Label, "voiced_seconds": voicedSeconds, "audio_seconds": clip.AudioSeconds, "clip_count": len(profile.Clips), "total_voiced_seconds": totalVoiced(profile.Clips), "embedding_dim": s.modelDim, "sample_rate": nativeSpeakerSampleRate, "model_name": s.modelName, "vad_model": "energy", "self_consistency_score": selfScore, "self_consistency_threshold": nativeSpeakerDefaultThreshold, "self_consistency_warning": selfScore >= 0 && selfScore < nativeSpeakerDefaultThreshold, "self_consistency_best_clip_label": selfLabel, "self_consistency_best_clip_id": selfID, "created_at": profile.CreatedAt})
}

func (s *nativeSpeakerRuntime) verify(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	fields, audio, err := readMultipartAudio(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	profile, err := s.loadProfile(strings.TrimSpace(fields["profile_id"]))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	if profile == nil {
		s.writeError(w, http.StatusNotFound, "profile not found", "")
		return
	}
	if profile.ModelName != s.modelName || profile.EmbeddingDim != s.modelDim {
		s.writeError(w, http.StatusConflict, "profile requires re-enrollment under the active speaker model", "speaker_model_mismatch")
		return
	}
	threshold := nativeSpeakerDefaultThreshold
	if raw := strings.TrimSpace(fields["threshold"]); raw != "" {
		if value, parseErr := strconv.ParseFloat(raw, 64); parseErr == nil {
			threshold = value
		}
	}
	samples, sampleRate, err := decodeSpeakerAudio(audio)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	voiced, voicedSeconds := trimSpeakerAudio(samples, sampleRate)
	if voicedSeconds < nativeSpeakerMinVerify {
		s.writeJSON(w, http.StatusOK, map[string]any{"profile_id": profile.ID, "matched": false, "score": 0, "threshold": threshold, "sufficient": false, "voiced_seconds": voicedSeconds, "audio_seconds": float64(len(samples)) / float64(sampleRate), "duration_ms": elapsedMilliseconds(started), "backend": "sherpa-onnx", "model": s.modelName, "score_agg": "max", "vad_model": "energy", "n_clips": len(profile.Clips), "best_clip_label": "", "best_clip_id": "", "best_clip_score": 0})
		return
	}
	embedding, err := s.embed(voiced)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	score, label, clipID := bestSpeakerMatch(embedding, profile.Clips)
	if score < 0 {
		score = 0
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"profile_id": profile.ID, "matched": score >= threshold, "score": score, "threshold": threshold, "sufficient": true, "voiced_seconds": voicedSeconds, "audio_seconds": float64(len(samples)) / float64(sampleRate), "duration_ms": elapsedMilliseconds(started), "backend": "sherpa-onnx", "model": s.modelName, "score_agg": "max", "vad_model": "energy", "n_clips": len(profile.Clips), "best_clip_label": label, "best_clip_id": clipID, "best_clip_score": score})
}

func (s *nativeSpeakerRuntime) extract(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	fields, audio, err := readMultipartAudio(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	profile, err := s.loadProfile(strings.TrimSpace(fields["profile_id"]))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	if profile == nil {
		s.writeError(w, http.StatusNotFound, "profile not found", "")
		return
	}
	if profile.ModelName != s.modelName || profile.EmbeddingDim != s.modelDim {
		s.writeError(w, http.StatusConflict, "profile requires re-enrollment under the active speaker model", "speaker_model_mismatch")
		return
	}
	if len(profile.Clips) == 0 {
		s.writeError(w, http.StatusConflict, "profile has no enrollment clips", "")
		return
	}
	samples, sampleRate, err := decodeSpeakerAudio(audio)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	cleaned, score, err := s.separate(samples, sampleRate, profile.Clips)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	pcm := speakerPCM16(cleaned)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Speaker-Score", fmt.Sprintf("%.6f", score))
	w.Header().Set("X-Speaker-Matched", strconv.FormatBool(score >= nativeSpeakerDefaultThreshold))
	w.Header().Set("X-Duration-Ms", fmt.Sprintf("%.3f", elapsedMilliseconds(started)))
	w.Header().Set("X-Audio-Seconds", fmt.Sprintf("%.3f", float64(len(cleaned))/nativeSpeakerSampleRate))
	_, _ = w.Write(pcm)
}

func elapsedMilliseconds(started time.Time) float64 {
	return float64(time.Since(started).Microseconds()) / 1000
}

func (s *nativeSpeakerRuntime) deleteClip(w http.ResponseWriter, id, clipID string) {
	profile, err := s.loadProfile(id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	if profile == nil {
		s.writeError(w, http.StatusNotFound, "profile not found", "")
		return
	}
	remaining := profile.Clips[:0]
	removed := false
	for _, clip := range profile.Clips {
		if clip.ClipID == clipID {
			removed = true
		} else {
			remaining = append(remaining, clip)
		}
	}
	if !removed {
		s.writeError(w, http.StatusNotFound, "clip not found", "")
		return
	}
	if len(remaining) == 0 {
		_ = os.Remove(s.profilePath(id))
		s.writeJSON(w, http.StatusOK, map[string]any{"profile_id": id, "clip_id": clipID, "deleted_profile": true, "clip_count": 0, "total_voiced_seconds": 0})
		return
	}
	profile.Clips = remaining
	profile.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := s.saveProfile(profile); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"profile_id": id, "clip_id": clipID, "deleted_profile": false, "clip_count": len(remaining), "total_voiced_seconds": totalVoiced(remaining)})
}

func (s *nativeSpeakerRuntime) embed(samples []float32) ([]float32, error) {
	if len(samples) == 0 {
		return nil, fmt.Errorf("audio is empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	stream := C.SherpaOnnxSpeakerEmbeddingExtractorCreateStream(s.extractor)
	if stream == nil {
		return nil, fmt.Errorf("create speaker embedding stream")
	}
	defer C.SherpaOnnxDestroyOnlineStream(stream)
	C.SherpaOnnxOnlineStreamAcceptWaveform(stream, C.int32_t(nativeSpeakerSampleRate), (*C.float)(unsafe.Pointer(&samples[0])), C.int32_t(len(samples)))
	C.SherpaOnnxOnlineStreamInputFinished(stream)
	if C.SherpaOnnxSpeakerEmbeddingExtractorIsReady(s.extractor, stream) == 0 {
		return nil, fmt.Errorf("speaker embedding input is too short")
	}
	vector := C.SherpaOnnxSpeakerEmbeddingExtractorComputeEmbedding(s.extractor, stream)
	if vector == nil {
		return nil, fmt.Errorf("speaker embedding failed")
	}
	defer C.SherpaOnnxSpeakerEmbeddingExtractorDestroyEmbedding(vector)
	values := unsafe.Slice(vector, s.modelDim)
	return copyCFloat(values), nil
}

func (s *nativeSpeakerRuntime) separate(samples []float32, sampleRate int, clips []nativeSpeakerClip) ([]float32, float64, error) {
	if len(samples) == 0 {
		return nil, 0, fmt.Errorf("audio is empty")
	}
	resampled := resampleSpeaker(samples, sampleRate, int(C.SherpaOnnxOfflineSourceSeparationGetOutputSampleRate(s.separator)))
	s.mu.Lock()
	defer s.mu.Unlock()
	inputSamples := C.malloc(C.size_t(len(resampled)*2) * C.size_t(unsafe.Sizeof(C.float(0))))
	if inputSamples == nil {
		return nil, 0, fmt.Errorf("allocate source separation input")
	}
	defer C.free(inputSamples)
	inputValues := unsafe.Slice((*C.float)(inputSamples), len(resampled)*2)
	for i, value := range resampled {
		inputValues[i] = C.float(value)
		inputValues[len(resampled)+i] = C.float(value)
	}
	inputPointers := C.malloc(C.size_t(unsafe.Sizeof(uintptr(0))) * 2)
	if inputPointers == nil {
		return nil, 0, fmt.Errorf("allocate source separation channel pointers")
	}
	defer C.free(inputPointers)
	channels := unsafe.Slice((**C.float)(inputPointers), 2)
	channels[0] = (*C.float)(inputSamples)
	channels[1] = (*C.float)(unsafe.Add(unsafe.Pointer(inputSamples), uintptr(len(resampled))*unsafe.Sizeof(C.float(0))))
	output := C.SherpaOnnxOfflineSourceSeparationProcess(s.separator, (**C.float)(inputPointers), 2, C.int32_t(len(resampled)), C.int32_t(C.SherpaOnnxOfflineSourceSeparationGetOutputSampleRate(s.separator)))
	if output == nil {
		return nil, 0, fmt.Errorf("source separation failed")
	}
	defer C.SherpaOnnxDestroySourceSeparationOutput(output)
	stems := unsafe.Slice(output.stems, int(output.num_stems))
	bestScore := -1.0
	var best []float32
	for _, stem := range stems {
		if stem.num_channels <= 0 || stem.n <= 0 {
			continue
		}
		channels := unsafe.Slice(stem.samples, int(stem.num_channels))
		values := unsafe.Slice(channels[0], int(stem.n))
		candidate := copyCFloat(values)
		candidate = resampleSpeaker(candidate, int(output.sample_rate), nativeSpeakerSampleRate)
		embedding, err := s.embedLocked(candidate)
		if err != nil {
			continue
		}
		score, _, _ := bestSpeakerMatch(embedding, clips)
		if score > bestScore {
			bestScore, best = score, candidate
		}
	}
	if best == nil {
		return samples, 0, nil
	}
	return best, bestScore, nil
}

func (s *nativeSpeakerRuntime) embedLocked(samples []float32) ([]float32, error) {
	stream := C.SherpaOnnxSpeakerEmbeddingExtractorCreateStream(s.extractor)
	if stream == nil {
		return nil, fmt.Errorf("create speaker embedding stream")
	}
	defer C.SherpaOnnxDestroyOnlineStream(stream)
	C.SherpaOnnxOnlineStreamAcceptWaveform(stream, C.int32_t(nativeSpeakerSampleRate), (*C.float)(unsafe.Pointer(&samples[0])), C.int32_t(len(samples)))
	C.SherpaOnnxOnlineStreamInputFinished(stream)
	if C.SherpaOnnxSpeakerEmbeddingExtractorIsReady(s.extractor, stream) == 0 {
		return nil, fmt.Errorf("speaker embedding input is too short")
	}
	vector := C.SherpaOnnxSpeakerEmbeddingExtractorComputeEmbedding(s.extractor, stream)
	if vector == nil {
		return nil, fmt.Errorf("speaker embedding failed")
	}
	defer C.SherpaOnnxSpeakerEmbeddingExtractorDestroyEmbedding(vector)
	return copyCFloat(unsafe.Slice(vector, s.modelDim)), nil
}

func (s *nativeSpeakerRuntime) profilePath(id string) string {
	return filepath.Join(s.profileDir, id+".json")
}
func (s *nativeSpeakerRuntime) loadProfile(id string) (*nativeSpeakerProfile, error) {
	if !safeProfileID(id) {
		return nil, nil
	}
	s.profileMu.Lock()
	defer s.profileMu.Unlock()
	data, err := os.ReadFile(s.profilePath(id))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var profile nativeSpeakerProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}
func (s *nativeSpeakerRuntime) saveProfile(profile *nativeSpeakerProfile) error {
	s.profileMu.Lock()
	defer s.profileMu.Unlock()
	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	tmp := s.profilePath(profile.ID) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.profilePath(profile.ID))
}
func (s *nativeSpeakerRuntime) publicProfile(p *nativeSpeakerProfile, clips bool) map[string]any {
	out := map[string]any{"id": p.ID, "display_name": p.DisplayName, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt, "model_name": p.ModelName, "embedding_dim": p.EmbeddingDim, "sample_rate": p.SampleRate, "clip_count": len(p.Clips), "total_voiced_seconds": totalVoiced(p.Clips), "notes": p.Notes}
	if clips {
		public := make([]any, 0, len(p.Clips))
		for _, c := range p.Clips {
			public = append(public, s.publicClip(c))
		}
		out["clips"] = public
	}
	return out
}
func (s *nativeSpeakerRuntime) publicClip(c nativeSpeakerClip) map[string]any {
	return map[string]any{"clip_id": c.ClipID, "label": c.Label, "voiced_seconds": c.VoicedSeconds, "audio_seconds": c.AudioSeconds, "self_consistency_score": c.SelfConsistencyScore, "vad_model": "energy", "created_at": c.CreatedAt, "embedding_dim": len(c.Embedding)}
}
func (s *nativeSpeakerRuntime) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func (s *nativeSpeakerRuntime) writeError(w http.ResponseWriter, status int, message, code string) {
	value := map[string]any{"error": message}
	if code != "" {
		value["code"] = code
	}
	s.writeJSON(w, status, value)
}

func readMultipartAudio(r *http.Request) (map[string]string, []byte, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil, nil, fmt.Errorf("parse multipart form: %w", err)
	}
	fields := make(map[string]string)
	for key, values := range r.MultipartForm.Value {
		if len(values) > 0 {
			fields[key] = values[0]
		}
	}
	file, _, err := r.FormFile("audio")
	if err != nil {
		return nil, nil, fmt.Errorf("audio upload is required: %w", err)
	}
	defer file.Close()
	audio, err := io.ReadAll(io.LimitReader(file, 32<<20))
	if err != nil {
		return nil, nil, err
	}
	if len(audio) == 0 {
		return nil, nil, fmt.Errorf("empty audio upload")
	}
	return fields, audio, nil
}

func decodeSpeakerAudio(raw []byte) ([]float32, int, error) {
	if len(raw) >= 12 && string(raw[:4]) == "RIFF" && string(raw[8:12]) == "WAVE" {
		pos := 12
		channels, rate, bits := 0, 0, 0
		var data []byte
		for pos+8 <= len(raw) {
			kind := string(raw[pos : pos+4])
			size := int(binary.LittleEndian.Uint32(raw[pos+4 : pos+8]))
			pos += 8
			if size < 0 || pos+size > len(raw) {
				return nil, 0, fmt.Errorf("invalid WAV chunk")
			}
			chunk := raw[pos : pos+size]
			pos += size
			if size%2 != 0 {
				pos++
			}
			switch kind {
			case "fmt ":
				if len(chunk) < 16 {
					return nil, 0, fmt.Errorf("invalid WAV format")
				}
				if binary.LittleEndian.Uint16(chunk[0:2]) != 1 {
					return nil, 0, fmt.Errorf("WAV must be PCM")
				}
				channels = int(binary.LittleEndian.Uint16(chunk[2:4]))
				rate = int(binary.LittleEndian.Uint32(chunk[4:8]))
				bits = int(binary.LittleEndian.Uint16(chunk[14:16]))
			case "data":
				data = chunk
			}
		}
		if channels <= 0 || rate <= 0 || bits != 16 || len(data)%2 != 0 {
			return nil, 0, fmt.Errorf("WAV must be signed-16 PCM")
		}
		return resampleSpeaker(pcmBytesToFloats(data, channels), rate, nativeSpeakerSampleRate), nativeSpeakerSampleRate, nil
	}
	if len(raw)%2 != 0 {
		return nil, 0, fmt.Errorf("raw PCM has odd byte length")
	}
	return pcmBytesToFloats(raw, 1), nativeSpeakerSampleRate, nil
}
func pcmBytesToFloats(data []byte, channels int) []float32 {
	frames := len(data) / (2 * channels)
	out := make([]float32, frames)
	for i := 0; i < frames; i++ {
		var total float32
		for c := 0; c < channels; c++ {
			total += float32(int16(binary.LittleEndian.Uint16(data[(i*channels+c)*2:]))) / 32768
		}
		out[i] = total / float32(channels)
	}
	return out
}
func trimSpeakerAudio(samples []float32, rate int) ([]float32, float64) {
	if len(samples) == 0 {
		return nil, 0
	}
	frame := maxInt(1, rate/50)
	peak := float32(0)
	for _, v := range samples {
		if float32(math.Abs(float64(v))) > peak {
			peak = float32(math.Abs(float64(v)))
		}
	}
	if peak < 1e-4 {
		return nil, 0
	}
	threshold := peak * 0.08
	first, last := -1, -1
	for start := 0; start < len(samples); start += frame {
		end := minInt(len(samples), start+frame)
		var sum float64
		for _, v := range samples[start:end] {
			sum += float64(v * v)
		}
		rms := float32(math.Sqrt(sum / float64(end-start)))
		if rms > threshold {
			if first < 0 {
				first = start
			}
			last = end
		}
	}
	if first < 0 {
		return nil, 0
	}
	return samples[first:last], float64(last-first) / float64(rate)
}
func bestSpeakerMatch(embedding []float32, clips []nativeSpeakerClip) (float64, string, string) {
	best := -1.0
	label, id := "", ""
	for _, c := range clips {
		if len(c.Embedding) != len(embedding) {
			continue
		}
		var dot, aa, bb float64
		for i, v := range embedding {
			dot += float64(v * c.Embedding[i])
			aa += float64(v * v)
			bb += float64(c.Embedding[i] * c.Embedding[i])
		}
		if aa == 0 || bb == 0 {
			continue
		}
		score := dot / math.Sqrt(aa*bb)
		if score > best {
			best, label, id = score, c.Label, c.ClipID
		}
	}
	return best, label, id
}
func totalVoiced(clips []nativeSpeakerClip) float64 {
	total := 0.0
	for _, c := range clips {
		total += c.VoicedSeconds
	}
	return total
}
func resampleSpeaker(samples []float32, from, to int) []float32 {
	if from <= 0 || to <= 0 || from == to {
		return samples
	}
	n := int(math.Round(float64(len(samples)) * float64(to) / float64(from)))
	if n < 1 {
		return nil
	}
	out := make([]float32, n)
	scale := float64(from) / float64(to)
	for i := range out {
		pos := float64(i) * scale
		left := int(pos)
		if left >= len(samples)-1 {
			out[i] = samples[len(samples)-1]
			continue
		}
		frac := float32(pos - float64(left))
		out[i] = samples[left]*(1-frac) + samples[left+1]*frac
	}
	return out
}
func speakerPCM16(samples []float32) []byte {
	out := make([]byte, len(samples)*2)
	for i, v := range samples {
		if v > 1 {
			v = 1
		}
		if v < -1 {
			v = -1
		}
		binary.LittleEndian.PutUint16(out[i*2:], uint16(int16(v*32767)))
	}
	return out
}
func safeProfileID(id string) bool {
	if id == "" || len(id) > 128 || id == "." || id == ".." {
		return false
	}
	for _, r := range id {
		if !(r == '-' || r == '_' || r == '.' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func copyCFloat(values []C.float) []float32 {
	out := make([]float32, len(values))
	for i, value := range values {
		out[i] = float32(value)
	}
	return out
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
