package bootstrap_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	"audio-tools/internal/bootstrap"
	"audio-tools/internal/modules"
	inttts "audio-tools/internal/tts"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
)

func testPCM16WAV() []byte {
	data := []byte{0, 0}
	wav := make([]byte, 44+len(data))
	copy(wav[0:4], "RIFF")
	binary.LittleEndian.PutUint32(wav[4:8], uint32(len(wav)-8))
	copy(wav[8:16], "WAVEfmt ")
	binary.LittleEndian.PutUint32(wav[16:20], 16)
	binary.LittleEndian.PutUint16(wav[20:22], 1)
	binary.LittleEndian.PutUint16(wav[22:24], 1)
	binary.LittleEndian.PutUint32(wav[24:28], 24000)
	binary.LittleEndian.PutUint32(wav[28:32], 48000)
	binary.LittleEndian.PutUint16(wav[32:34], 2)
	binary.LittleEndian.PutUint16(wav[34:36], 16)
	copy(wav[36:40], "data")
	binary.LittleEndian.PutUint32(wav[40:44], uint32(len(data)))
	copy(wav[44:], data)
	return wav
}

func wavDataBytes(wav []byte) (uint32, error) {
	if len(wav) < 12 || string(wav[:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		return 0, fmt.Errorf("not a RIFF/WAVE file")
	}
	for offset := 12; offset+8 <= len(wav); {
		tag := string(wav[offset : offset+4])
		size := binary.LittleEndian.Uint32(wav[offset+4 : offset+8])
		if tag == "data" {
			if uint64(size) > uint64(len(wav)-offset-8) {
				return 0, fmt.Errorf("data chunk exceeds file")
			}
			return size, nil
		}
		next := offset + 8 + int(size) + int(size&1)
		if size == ^uint32(0) || next <= offset || next > len(wav) {
			return 0, fmt.Errorf("invalid %q chunk size", tag)
		}
		offset = next
	}
	return 0, fmt.Errorf("WAVE data chunk is missing")
}

func TestBuild_EndToEnd(t *testing.T) {
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(whisper.Close)
	kokoro := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(kokoro.Close)
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ollama.Close)

	dir := t.TempDir()
	t.Setenv("AUDIO_WHISPER_URL", whisper.URL)
	t.Setenv("AUDIO_SHERPA_URL", kokoro.URL)
	t.Setenv("AUDIO_OLLAMA_URL", ollama.URL)
	t.Setenv("VROOLI_STORAGE_ROOT", dir)
	t.Setenv("AUDIO_TOOLS_DB_KEY_PATH", filepath.Join(dir, "byok.key"))

	srv, cleanup, err := bootstrap.Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if srv == nil {
		t.Fatal("Build returned nil server")
	}
	if cleanup == nil {
		t.Fatal("Build returned nil cleanup")
	}
	t.Cleanup(func() {
		if err := cleanup(); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /health = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
}

func TestBuildSherpaTTSBackendUsesSherpaEndpointAndCacheBoundary(t *testing.T) {
	var requestedSyntheses int
	sherpa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/healthz"):
			_, _ = w.Write([]byte(`{"status":"ok","tts_model_loaded":true,"encoder_ready":true,"streaming_model_loaded":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/audio/voices":
			_, _ = w.Write([]byte(`["af_heart","af_bella"]`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/audio/speech":
			body, _ := io.ReadAll(r.Body)
			if strings.Contains(string(body), `"input":"integration"`) {
				requestedSyntheses++
				if !strings.Contains(string(body), `"voice":"af_bella"`) {
					t.Errorf("speech request did not preserve resolved voice: %s", body)
				}
			}
			w.Header().Set("Content-Type", "audio/wav")
			_, _ = w.Write([]byte("sherpa-audio"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(sherpa.Close)
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(whisper.Close)
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(ollama.Close)

	dir := t.TempDir()
	t.Setenv("AUDIO_TTS_BACKEND", "sherpa-onnx")
	t.Setenv("AUDIO_KOKORO_URL", "http://127.0.0.1:1")
	t.Setenv("AUDIO_SHERPA_URL", sherpa.URL)
	t.Setenv("AUDIO_WHISPER_URL", whisper.URL)
	t.Setenv("AUDIO_OLLAMA_URL", ollama.URL)
	t.Setenv("VROOLI_STORAGE_ROOT", dir)
	t.Setenv("AUDIO_TOOLS_DB_KEY_PATH", filepath.Join(dir, "byok.key"))

	_, deps, cleanup, err := bootstrap.BuildWithDeps(context.Background())
	if err != nil {
		t.Fatalf("BuildWithDeps: %v", err)
	}
	t.Cleanup(func() {
		if err := cleanup(); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	if !deps.TTS.LocalBackendReady(context.Background()) {
		t.Fatal("sherpa-backed local TTS should pass its synthesis readiness probe")
	}
	voices, err := deps.TTS.ListVoices(context.Background())
	if err != nil {
		t.Fatalf("sherpa-backed voice listing: %v", err)
	}
	if len(voices) != 2 || voices[1].ID != "af_bella" {
		t.Fatalf("sherpa-backed voices = %#v, want sherpa voice IDs", voices)
	}
	out, err := deps.TTS.Synthesize(context.Background(), inttts.SynthesizeInput{
		Input: "integration", Voice: "af_bella", ResponseFormat: "wav", Speed: 1,
		EventID: "sherpa-event", Version: "v1",
	})
	if err != nil {
		t.Fatalf("sherpa-backed synthesis: %v", err)
	}
	if string(out.Audio) != "sherpa-audio" || out.ContentType != "audio/wav" {
		t.Fatalf("unexpected synthesis result: content_type=%q bytes=%q", out.ContentType, out.Audio)
	}
	if requestedSyntheses != 1 {
		t.Fatalf("requested sherpa syntheses = %d, want 1", requestedSyntheses)
	}
	if _, ok := deps.Cache.Get(inttts.CacheKey{EventID: "sherpa-event", Backend: "sherpa-onnx:kokoro", Voice: "af_bella", Speed: 1, ResponseFormat: "wav", Version: "v1"}); !ok {
		t.Fatal("Sherpa synthesis was not cached in its runtime-specific output-format namespace")
	}
	if _, ok := deps.Cache.Get(inttts.CacheKey{EventID: "sherpa-event", Backend: "kokoro", Voice: "af_bella", Speed: 1, ResponseFormat: "wav", Version: "v1"}); ok {
		t.Fatal("Sherpa synthesis leaked into the standalone Kokoro runtime namespace")
	}
}

func TestBuildSherpaTTSProductConnectRouteAndEventCache(t *testing.T) {
	mp3 := []byte("sherpa-mp3")
	var speechRequest map[string]any
	var syntheses atomic.Int32
	sherpa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/healthz"):
			_, _ = w.Write([]byte(`{"status":"ok","tts_model_loaded":true,"encoder_ready":true,"streaming_model_loaded":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/audio/voices":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/audio/speech":
			syntheses.Add(1)
			if err := json.NewDecoder(r.Body).Decode(&speechRequest); err != nil {
				t.Error(err)
			}
			w.Header().Set("Content-Type", "audio/mpeg")
			_, _ = w.Write(mp3)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(sherpa.Close)
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(whisper.Close)
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(ollama.Close)

	dir := t.TempDir()
	t.Setenv("AUDIO_TTS_BACKEND", "sherpa-onnx")
	t.Setenv("AUDIO_KOKORO_URL", "http://127.0.0.1:1")
	t.Setenv("AUDIO_SHERPA_URL", sherpa.URL)
	t.Setenv("AUDIO_WHISPER_URL", whisper.URL)
	t.Setenv("AUDIO_OLLAMA_URL", ollama.URL)
	t.Setenv("VROOLI_STORAGE_ROOT", dir)
	t.Setenv("AUDIO_TOOLS_DB_KEY_PATH", filepath.Join(dir, "byok.key"))
	srv, _, cleanup, err := bootstrap.BuildWithDeps(context.Background())
	if err != nil {
		t.Fatalf("BuildWithDeps: %v", err)
	}
	t.Cleanup(func() {
		if err := cleanup(); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	product := httptest.NewServer(srv.Handler())
	t.Cleanup(product.Close)
	client := ttsconnect.NewTTSServiceClient(http.DefaultClient, product.URL)
	request := connect.NewRequest(&ttsv1.SynthesizeRequest{
		Text: "through the sherpa product route", Voice: "voice.feminine.warm", Speed: 1,
		ResponseFormat: commonv1.ResponseFormat_RESPONSE_FORMAT_MP3, EventId: "sherpa-product-event", Version: "v4", ChunkIndex: 3,
	})
	request.Header().Set("X-Audio-Operation-ID", "sherpa-product-operation")
	response, err := client.Synthesize(context.Background(), request)
	if err != nil {
		t.Fatalf("product route Synthesize: %v", err)
	}
	if speechRequest["model"] != "kokoro" || speechRequest["voice"] != "af_bella" || speechRequest["input"] != "through the sherpa product route" || speechRequest["response_format"] != "mp3" {
		t.Fatalf("Sherpa runtime request = %#v", speechRequest)
	}
	if syntheses.Load() != 1 {
		t.Fatalf("Sherpa syntheses = %d, want 1", syntheses.Load())
	}
	if response.Msg.GetProviderId() != "kokoro-tts" || response.Msg.GetModelId() != "kokoro-82m-v1.0-sherpa-int8" || response.Msg.GetVoiceUsed() != "af_bella" {
		t.Fatalf("product response identity = provider %q model %q voice %q", response.Msg.GetProviderId(), response.Msg.GetModelId(), response.Msg.GetVoiceUsed())
	}
	if response.Msg.GetContentType() != "audio/mpeg" || !bytes.Equal(response.Msg.GetAudio(), mp3) {
		t.Fatalf("unexpected MP3 response: type=%q bytes=%x", response.Msg.GetContentType(), response.Msg.GetAudio())
	}
	cache, err := client.GetCache(context.Background(), connect.NewRequest(&ttsv1.GetCacheRequest{
		EventId: "sherpa-product-event", Voice: "voice.feminine.warm", Speed: 1,
		ResponseFormat: commonv1.ResponseFormat_RESPONSE_FORMAT_MP3, Version: "v4", ChunkIndex: 3,
	}))
	if err != nil {
		t.Fatalf("product route GetCache: %v", err)
	}
	if !cache.Msg.GetHit() || cache.Msg.GetContentType() != "audio/mpeg" || !bytes.Equal(cache.Msg.GetAudio(), mp3) {
		t.Fatalf("event cache did not preserve Sherpa MP3: hit=%v type=%q bytes=%x", cache.Msg.GetHit(), cache.Msg.GetContentType(), cache.Msg.GetAudio())
	}
	if syntheses.Load() != 1 {
		t.Fatalf("cache lookup triggered synthesis: count=%d", syntheses.Load())
	}
}

func TestBuildKaleTTSBackendUsesDedicatedRuntimeModelAndCacheNamespace(t *testing.T) {
	var speechRequest map[string]any
	kale := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			_, _ = w.Write([]byte(`{"status":"ok","model_loaded":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/audio/speech":
			if err := json.NewDecoder(r.Body).Decode(&speechRequest); err != nil {
				t.Error(err)
			}
			w.Header().Set("Content-Type", "audio/wav")
			_, _ = w.Write([]byte("kale-audio"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(kale.Close)
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(whisper.Close)
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(ollama.Close)

	dir := t.TempDir()
	t.Setenv("AUDIO_TTS_BACKEND", "kale-onnx")
	t.Setenv("AUDIO_KALE_URL", kale.URL)
	t.Setenv("AUDIO_WHISPER_URL", whisper.URL)
	t.Setenv("AUDIO_OLLAMA_URL", ollama.URL)
	t.Setenv("VROOLI_STORAGE_ROOT", dir)
	t.Setenv("AUDIO_TOOLS_DB_KEY_PATH", filepath.Join(dir, "byok.key"))
	_, deps, cleanup, err := bootstrap.BuildWithDeps(context.Background())
	if err != nil {
		t.Fatalf("BuildWithDeps: %v", err)
	}
	t.Cleanup(func() {
		if err := cleanup(); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	if !deps.TTS.LocalBackendReady(context.Background()) {
		t.Fatal("Kale TTS should pass the runtime health probe")
	}
	out, err := deps.TTS.Synthesize(context.Background(), inttts.SynthesizeInput{Input: "small local model", Voice: "kenzie", ResponseFormat: "wav", Speed: 1, EventID: "kale-event"})
	if err != nil {
		t.Fatalf("Kale synthesis: %v", err)
	}
	if string(out.Audio) != "kale-audio" || out.ContentType != "audio/wav" {
		t.Fatalf("result = %q %q", out.Audio, out.ContentType)
	}
	if speechRequest["model"] != "kale-tts-1-nano" || speechRequest["voice"] != "kenzie" {
		t.Fatalf("Kale speech request = %#v", speechRequest)
	}
	if _, ok := deps.Cache.Get(inttts.CacheKey{EventID: "kale-event", Backend: "kale-tts-1-nano", Voice: "kenzie", Speed: 1, ResponseFormat: "wav", Version: "active"}); !ok {
		t.Fatal("Kale synthesis was not stored in its backend-specific cache namespace")
	}
}

func TestBuildKaleTTSProductConnectRouteAndCache(t *testing.T) {
	wav := testPCM16WAV()
	var speechRequest map[string]any
	var syntheses int
	kale := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			_, _ = w.Write([]byte(`{"status":"ok","model_loaded":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/audio/speech":
			syntheses++
			if err := json.NewDecoder(r.Body).Decode(&speechRequest); err != nil {
				t.Error(err)
			}
			w.Header().Set("Content-Type", "audio/wav")
			_, _ = w.Write(wav)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(kale.Close)
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(whisper.Close)
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(ollama.Close)

	dir := t.TempDir()
	t.Setenv("AUDIO_TTS_BACKEND", "kale-onnx")
	t.Setenv("AUDIO_KALE_URL", kale.URL)
	t.Setenv("AUDIO_WHISPER_URL", whisper.URL)
	t.Setenv("AUDIO_OLLAMA_URL", ollama.URL)
	t.Setenv("VROOLI_STORAGE_ROOT", dir)
	t.Setenv("AUDIO_TOOLS_DB_KEY_PATH", filepath.Join(dir, "byok.key"))
	srv, _, cleanup, err := bootstrap.BuildWithDeps(context.Background())
	if err != nil {
		t.Fatalf("BuildWithDeps: %v", err)
	}
	t.Cleanup(func() {
		if err := cleanup(); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	product := httptest.NewServer(srv.Handler())
	t.Cleanup(product.Close)
	client := ttsconnect.NewTTSServiceClient(http.DefaultClient, product.URL)
	request := connect.NewRequest(&ttsv1.SynthesizeRequest{
		Text: "route through the selected provider", Voice: "voice.feminine.warm", Speed: 1,
		ResponseFormat: commonv1.ResponseFormat_RESPONSE_FORMAT_WAV, EventId: "kale-product-event", Version: "v7", ChunkIndex: 2,
	})
	request.Header().Set("X-Audio-Operation-ID", "kale-product-operation")
	response, err := client.Synthesize(context.Background(), request)
	if err != nil {
		t.Fatalf("product route Synthesize: %v", err)
	}
	if speechRequest["model"] != "kale-tts-1-nano" || speechRequest["voice"] != "kenzie" || speechRequest["input"] != "route through the selected provider" {
		t.Fatalf("Kale runtime request = %#v", speechRequest)
	}
	if syntheses != 1 {
		t.Fatalf("Kale syntheses = %d, want 1", syntheses)
	}
	if response.Msg.GetProviderId() != "kale-tts" || response.Msg.GetModelId() != "kale-tts-1-nano" || response.Msg.GetVoiceUsed() != "kenzie" {
		t.Fatalf("product response identity = provider %q model %q voice %q", response.Msg.GetProviderId(), response.Msg.GetModelId(), response.Msg.GetVoiceUsed())
	}
	gotWAV := response.Msg.GetAudio()
	if response.Msg.GetContentType() != "audio/wav" || !bytes.HasPrefix(gotWAV, []byte("RIFF")) || string(gotWAV[8:12]) != "WAVE" || !bytes.Equal(gotWAV, wav) {
		t.Fatalf("product response is not the expected WAV: content_type=%q bytes=%x", response.Msg.GetContentType(), gotWAV)
	}
	cache, err := client.GetCache(context.Background(), connect.NewRequest(&ttsv1.GetCacheRequest{
		EventId: "kale-product-event", Voice: "voice.feminine.warm", Speed: 1,
		ResponseFormat: commonv1.ResponseFormat_RESPONSE_FORMAT_WAV, Version: "v7", ChunkIndex: 2,
	}))
	if err != nil {
		t.Fatalf("product route GetCache: %v", err)
	}
	if !cache.Msg.GetHit() || cache.Msg.GetContentType() != "audio/wav" || !bytes.Equal(cache.Msg.GetAudio(), wav) {
		t.Fatalf("cache response did not preserve synthesized WAV: hit=%v type=%q bytes=%x", cache.Msg.GetHit(), cache.Msg.GetContentType(), cache.Msg.GetAudio())
	}
	if syntheses != 1 {
		t.Fatalf("cache lookup triggered synthesis: count=%d", syntheses)
	}
}

func TestBuildAutoTTSPrefersConditionalKaleAndFallsBackToKokoroOnUnavailable(t *testing.T) {
	wav := testPCM16WAV()
	var kaleSyntheses atomic.Int32
	kale := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			_, _ = w.Write([]byte(`{"status":"ok","model_loaded":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/audio/speech":
			if kaleSyntheses.Add(1) == 1 {
				http.Error(w, "runtime is restarting", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "audio/wav")
			_, _ = w.Write(wav)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(kale.Close)
	kokoro := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/audio/voices":
			w.WriteHeader(http.StatusOK)
		case "/v1/audio/speech":
			w.Header().Set("Content-Type", "audio/wav")
			_, _ = w.Write(wav)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(kokoro.Close)
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(whisper.Close)
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(ollama.Close)

	dir := t.TempDir()
	eventID := fmt.Sprintf("auto-fallback-%d", time.Now().UnixNano())
	t.Setenv("AUDIO_TTS_BACKEND", "auto")
	t.Setenv("AUDIO_KALE_URL", kale.URL)
	t.Setenv("AUDIO_KOKORO_URL", kokoro.URL)
	t.Setenv("AUDIO_WHISPER_URL", whisper.URL)
	t.Setenv("AUDIO_OLLAMA_URL", ollama.URL)
	t.Setenv("VROOLI_STORAGE_ROOT", dir)
	t.Setenv("AUDIO_TOOLS_DB_KEY_PATH", filepath.Join(dir, "byok.key"))
	srv, _, cleanup, err := bootstrap.BuildWithDeps(context.Background())
	if err != nil {
		t.Fatalf("BuildWithDeps: %v", err)
	}
	t.Cleanup(func() {
		if err := cleanup(); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	product := httptest.NewServer(srv.Handler())
	t.Cleanup(product.Close)
	client := ttsconnect.NewTTSServiceClient(http.DefaultClient, product.URL)

	first, err := client.Synthesize(context.Background(), connect.NewRequest(&ttsv1.SynthesizeRequest{
		Text: "fall back after a runtime drop", Voice: "voice.feminine.warm", Speed: 1,
		ResponseFormat: commonv1.ResponseFormat_RESPONSE_FORMAT_WAV, EventId: eventID,
	}))
	if err != nil {
		t.Fatalf("fallback Synthesize: %v", err)
	}
	if first.Msg.GetProviderId() != "kokoro-tts" || first.Msg.GetModelId() != "kokoro-82m-v1.0" || first.Msg.GetVoiceUsed() != "af_bella" {
		t.Fatalf("fallback identity = provider %q model %q voice %q", first.Msg.GetProviderId(), first.Msg.GetModelId(), first.Msg.GetVoiceUsed())
	}
	cached, err := client.GetCache(context.Background(), connect.NewRequest(&ttsv1.GetCacheRequest{
		EventId: eventID, Voice: "voice.feminine.warm", Speed: 1,
		ResponseFormat: commonv1.ResponseFormat_RESPONSE_FORMAT_WAV,
	}))
	if err != nil || !cached.Msg.GetHit() || !bytes.Equal(cached.Msg.GetAudio(), wav) {
		t.Fatalf("fallback cache lookup: response=%v err=%v", cached, err)
	}

	second, err := client.Synthesize(context.Background(), connect.NewRequest(&ttsv1.SynthesizeRequest{
		Text: "prefer Kale after recovery", Voice: "voice.feminine.warm", Speed: 1,
		ResponseFormat: commonv1.ResponseFormat_RESPONSE_FORMAT_WAV,
	}))
	if err != nil {
		t.Fatalf("recovered Synthesize: %v", err)
	}
	if second.Msg.GetProviderId() != "kale-tts" || second.Msg.GetModelId() != "kale-tts-1-nano" || second.Msg.GetVoiceUsed() != "kenzie" {
		t.Fatalf("second identity = provider %q model %q voice %q", second.Msg.GetProviderId(), second.Msg.GetModelId(), second.Msg.GetVoiceUsed())
	}
	if kaleSyntheses.Load() != 2 {
		t.Fatalf("Kale synthesis attempts = %d, want initial unavailable attempt and recovered retry", kaleSyntheses.Load())
	}
}

func TestBuildKaleTTSLiveProductRoute(t *testing.T) {
	liveURL := strings.TrimSpace(os.Getenv("AUDIO_KALE_URL"))
	if liveURL == "" {
		t.Skip("set AUDIO_KALE_URL to run the opt-in managed Kale product integration")
	}
	dir := t.TempDir()
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(whisper.Close)
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(ollama.Close)
	// Exercise declared conditional-auto ordering through the isolated product
	// composition, not an explicit backend pin.
	t.Setenv("AUDIO_TTS_BACKEND", "auto")
	t.Setenv("AUDIO_KALE_URL", liveURL)
	t.Setenv("AUDIO_WHISPER_URL", whisper.URL)
	t.Setenv("AUDIO_OLLAMA_URL", ollama.URL)
	t.Setenv("VROOLI_STORAGE_ROOT", dir)
	t.Setenv("AUDIO_TOOLS_DB_KEY_PATH", filepath.Join(dir, "byok.key"))

	srv, _, cleanup, err := bootstrap.BuildWithDeps(context.Background())
	if err != nil {
		t.Fatalf("BuildWithDeps: %v", err)
	}
	t.Cleanup(func() {
		if err := cleanup(); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	product := httptest.NewServer(srv.Handler())
	t.Cleanup(product.Close)
	client := ttsconnect.NewTTSServiceClient(http.DefaultClient, product.URL)
	started := time.Now()
	response, err := client.Synthesize(context.Background(), connect.NewRequest(&ttsv1.SynthesizeRequest{
		Text: "This is a short Kale integration check.", Voice: "voice.feminine.warm", Speed: 1.25,
		ResponseFormat: commonv1.ResponseFormat_RESPONSE_FORMAT_WAV,
		EventId:        "kale-live-product-check", Version: "integration-v1",
	}))
	wallTime := time.Since(started)
	if err != nil {
		t.Fatalf("live product route Synthesize: %v", err)
	}
	audio := response.Msg.GetAudio()
	if response.Msg.GetProviderId() != "kale-tts" || response.Msg.GetModelId() != "kale-tts-1-nano" || response.Msg.GetVoiceUsed() != "kenzie" {
		t.Fatalf("live response identity = provider %q model %q voice %q", response.Msg.GetProviderId(), response.Msg.GetModelId(), response.Msg.GetVoiceUsed())
	}
	if response.Msg.GetContentType() != "audio/wav" || len(audio) < 46 || string(audio[:4]) != "RIFF" || string(audio[8:12]) != "WAVE" || int(binary.LittleEndian.Uint32(audio[4:8]))+8 != len(audio) {
		t.Fatalf("live result is not a complete WAV: content_type=%q bytes=%d header=%x", response.Msg.GetContentType(), len(audio), audio[:min(len(audio), 16)])
	}
	byteRate := binary.LittleEndian.Uint32(audio[28:32])
	if byteRate == 0 {
		t.Fatal("live WAV has zero byte rate")
	}
	dataBytes, err := wavDataBytes(audio)
	if err != nil {
		t.Fatalf("live WAV data chunk: %v", err)
	}
	cacheReq := connect.NewRequest(&ttsv1.GetCacheRequest{
		EventId: "kale-live-product-check", Voice: "voice.feminine.warm", Speed: 1.25,
		ResponseFormat: commonv1.ResponseFormat_RESPONSE_FORMAT_WAV, Version: "integration-v1",
	})
	for i := 0; i < 2; i++ {
		cached, cacheErr := client.GetCache(context.Background(), cacheReq)
		if cacheErr != nil {
			t.Fatalf("live product route GetCache #%d: %v", i+1, cacheErr)
		}
		if !cached.Msg.GetHit() || cached.Msg.GetContentType() != "audio/wav" || !bytes.Equal(cached.Msg.GetAudio(), audio) {
			t.Fatalf("cache lookup #%d did not preserve live WAV: hit=%v type=%q bytes=%d", i+1, cached.Msg.GetHit(), cached.Msg.GetContentType(), len(cached.Msg.GetAudio()))
		}
	}
	t.Logf("Kale live product receipt: model=%s provider=%s voice=%s bytes=%d duration_ms=%d wall_ms=%d cache_repeat_hits=2", response.Msg.GetModelId(), response.Msg.GetProviderId(), response.Msg.GetVoiceUsed(), len(audio), dataBytes*1000/byteRate, wallTime.Milliseconds())
	for i, canonicalVoice := range inttts.CanonicalVoiceIDs {
		voice := inttts.LocalKaleMapping[canonicalVoice]
		voiceStarted := time.Now()
		voiceResponse, voiceErr := client.Synthesize(context.Background(), connect.NewRequest(&ttsv1.SynthesizeRequest{
			Text: "Kale warm voice route smoke.", Voice: canonicalVoice, Speed: 1,
			ResponseFormat: commonv1.ResponseFormat_RESPONSE_FORMAT_WAV,
			EventId:        fmt.Sprintf("kale-voice-smoke-%d", i), Version: "integration-v1",
		}))
		if voiceErr != nil {
			t.Fatalf("live product route voice %q: %v", canonicalVoice, voiceErr)
		}
		voiceAudio := voiceResponse.Msg.GetAudio()
		if voiceResponse.Msg.GetProviderId() != "kale-tts" || voiceResponse.Msg.GetModelId() != "kale-tts-1-nano" || voiceResponse.Msg.GetVoiceUsed() != voice {
			t.Fatalf("live voice identity for %q = provider %q model %q voice %q; want Kale Nano voice %q", canonicalVoice, voiceResponse.Msg.GetProviderId(), voiceResponse.Msg.GetModelId(), voiceResponse.Msg.GetVoiceUsed(), voice)
		}
		if voiceResponse.Msg.GetContentType() != "audio/wav" || len(voiceAudio) < 46 || string(voiceAudio[:4]) != "RIFF" || string(voiceAudio[8:12]) != "WAVE" || int(binary.LittleEndian.Uint32(voiceAudio[4:8]))+8 != len(voiceAudio) {
			t.Fatalf("live voice output for %q is not a complete WAV: content_type=%q bytes=%d", canonicalVoice, voiceResponse.Msg.GetContentType(), len(voiceAudio))
		}
		voiceDataBytes, dataErr := wavDataBytes(voiceAudio)
		if dataErr != nil {
			t.Fatalf("live voice output data chunk for %q: %v", canonicalVoice, dataErr)
		}
		voiceByteRate := binary.LittleEndian.Uint32(voiceAudio[28:32])
		if voiceByteRate == 0 {
			t.Fatalf("live voice output for %q has zero byte rate", canonicalVoice)
		}
		t.Logf("Kale warm voice receipt: canonical=%s backend=%s bytes=%d duration_ms=%d wall_ms=%d", canonicalVoice, voice, len(voiceAudio), voiceDataBytes*1000/voiceByteRate, time.Since(voiceStarted).Milliseconds())
	}
}

func TestLoadNativeSherpaEndpointFeedsSpeakerAndTTS(t *testing.T) {
	t.Setenv("SHERPA_ONNX_URL", "")
	t.Setenv("AUDIO_SHERPA_URL", "http://sherpa-native:8880")
	t.Setenv("AUDIO_TTS_BACKEND", "sherpa-onnx")
	t.Setenv("VROOLI_VARIANT", "live")

	env := bootstrap.Load()
	if env.SherpaURL != "http://sherpa-native:8880" {
		t.Fatalf("native sherpa endpoint was not loaded: sherpa=%q", env.SherpaURL)
	}
	if env.TTSBackend != "sherpa-onnx" {
		t.Fatalf("sherpa TTS backend was not loaded: backend=%q", env.TTSBackend)
	}
}

func TestLoadLabAPIRequiresSupervisorLabVariant(t *testing.T) {
	t.Setenv("AUDIO_ENABLE_LAB_API", "true")
	t.Setenv("VROOLI_VARIANT", "live")
	if env := bootstrap.Load(); env.EnableLabAPI {
		t.Fatal("caller AUDIO_ENABLE_LAB_API must not enable lab routes")
	}
	t.Setenv("VROOLI_VARIANT", "shadow")
	if env := bootstrap.Load(); env.EnableLabAPI {
		t.Fatal("non-lab variants must not enable lab routes")
	}
	t.Setenv("VROOLI_VARIANT", "lab")
	if env := bootstrap.Load(); env.EnableLabAPI != modules.IsLabBuild() {
		t.Fatalf("lab runtime selection = %v, build supports lab = %v", env.EnableLabAPI, modules.IsLabBuild())
	}
}

func TestBuildRejectsUnknownTTSBackendBeforeOpeningStores(t *testing.T) {
	t.Setenv("AUDIO_TTS_BACKEND", "typo-backend")
	_, _, err := bootstrap.Build(context.Background())
	if err == nil || !strings.Contains(err.Error(), `unsupported AUDIO_TTS_BACKEND "typo-backend"`) {
		t.Fatalf("Build error = %v, want unsupported backend validation", err)
	}
}
