//go:build cgo && sherpa_onnx

package main

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestNativeSpeakerRuntimeSmoke exercises the actual embedding and separation
// C APIs with managed model artifacts. It is opt-in because these models are
// deliberately external resource data, not repository fixtures.
func TestNativeSpeakerRuntimeSmoke(t *testing.T) {
	model := os.Getenv("SHERPA_ONNX_SPEAKER_MODEL")
	separation := os.Getenv("SHERPA_ONNX_SEPARATION_MODEL_DIR")
	wavPath := os.Getenv("SHERPA_ONNX_SPEAKER_TEST_WAV")
	if model == "" || separation == "" || wavPath == "" {
		t.Skip("set SHERPA_ONNX_SPEAKER_MODEL, SHERPA_ONNX_SEPARATION_MODEL_DIR, and SHERPA_ONNX_SPEAKER_TEST_WAV")
	}
	t.Setenv("SHERPA_ONNX_SPEAKER_MODEL", model)
	t.Setenv("SHERPA_ONNX_SEPARATION_MODEL_DIR", separation)
	t.Setenv("SPEAKER_VERIFICATION_PROFILE_DIR", t.TempDir())
	runtime, err := newSpeakerRuntimeFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	native := runtime.(*nativeSpeakerRuntime)
	defer native.Close()
	raw, err := os.ReadFile(wavPath)
	if err != nil {
		t.Fatal(err)
	}
	samples, rate, err := decodeSpeakerAudio(raw)
	if err != nil {
		t.Fatal(err)
	}
	voiced, seconds := trimSpeakerAudio(samples, rate)
	if seconds < nativeSpeakerMinVerify {
		t.Fatalf("test audio voiced duration = %.2fs", seconds)
	}
	embedding, err := native.embed(voiced)
	if err != nil {
		t.Fatal(err)
	}
	if len(embedding) != native.modelDim {
		t.Fatalf("embedding dimension = %d, want %d", len(embedding), native.modelDim)
	}
	cleaned, score, err := native.separate(samples, rate, []nativeSpeakerClip{{Embedding: embedding}})
	if err != nil {
		t.Fatal(err)
	}
	if len(cleaned) == 0 {
		t.Fatal("source separation returned no samples")
	}
	t.Logf("embedding_dim=%d separation_samples=%d selected_score=%.4f", len(embedding), len(cleaned), score)
}

// TestNativeSpeakerHTTPContract exercises the production HTTP boundary with
// real model handles. It is opt-in because the model bundle is external
// resource data, but it deliberately covers the lifecycle that unit tests
// cannot prove: enroll, verify, extract, model mismatch, and deletion.
func TestNativeSpeakerHTTPContract(t *testing.T) {
	if os.Getenv("SHERPA_ONNX_SPEAKER_HTTP_TEST") != "1" {
		t.Skip("set SHERPA_ONNX_SPEAKER_HTTP_TEST=1 to run the native HTTP contract")
	}
	model := os.Getenv("SHERPA_ONNX_SPEAKER_MODEL")
	separation := os.Getenv("SHERPA_ONNX_SEPARATION_MODEL_DIR")
	wavPath := os.Getenv("SHERPA_ONNX_SPEAKER_TEST_WAV")
	if model == "" || separation == "" || wavPath == "" {
		t.Skip("set the native speaker model, separation directory, and test WAV")
	}
	t.Setenv("SPEAKER_VERIFICATION_PROFILE_DIR", t.TempDir())
	runtime, err := newSpeakerRuntimeFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	native := runtime.(*nativeSpeakerRuntime)
	defer native.Close()
	server := httptest.NewServer(native)
	defer server.Close()
	audio, err := os.ReadFile(wavPath)
	if err != nil {
		t.Fatal(err)
	}

	status, _, body := speakerHTTPMultipart(t, http.MethodPost, server.URL+"/v1/profiles", map[string]string{
		"profile_id": "http-contract", "display_name": "HTTP Contract", "label": "native",
	}, audio)
	if status != http.StatusOK {
		t.Fatalf("enroll status=%d body=%s", status, body)
	}

	status, _, body = speakerHTTPMultipart(t, http.MethodPost, server.URL+"/v1/verify", map[string]string{
		"profile_id": "http-contract", "threshold": "0.5",
	}, audio)
	if status != http.StatusOK || !bytes.Contains(body, []byte(`"sufficient":true`)) || bytes.Contains(body, []byte(`"duration_ms":0`)) {
		t.Fatalf("verify status=%d body=%s", status, body)
	}

	status, headers, separated := speakerHTTPMultipart(t, http.MethodPost, server.URL+"/v1/extract", map[string]string{
		"profile_id": "http-contract", "verify": "true",
	}, audio)
	if status != http.StatusOK || len(separated) == 0 || headers.Get("X-Duration-Ms") == "0" {
		t.Fatalf("extract status=%d bytes=%d duration=%q", status, len(separated), headers.Get("X-Duration-Ms"))
	}
	empty := &nativeSpeakerProfile{ID: "empty", ModelName: native.modelName, EmbeddingDim: native.modelDim, SampleRate: nativeSpeakerSampleRate}
	if err := native.saveProfile(empty); err != nil {
		t.Fatal(err)
	}
	status, _, body = speakerHTTPMultipart(t, http.MethodPost, server.URL+"/v1/extract", map[string]string{
		"profile_id": "empty", "verify": "true",
	}, audio)
	if status != http.StatusConflict || !bytes.Contains(body, []byte("no enrollment clips")) {
		t.Fatalf("empty profile extraction status=%d body=%s", status, body)
	}

	legacy := &nativeSpeakerProfile{ID: "legacy", ModelName: "previous/model", EmbeddingDim: native.modelDim, SampleRate: nativeSpeakerSampleRate, Clips: []nativeSpeakerClip{{Embedding: make([]float32, native.modelDim)}}}
	if err := native.saveProfile(legacy); err != nil {
		t.Fatal(err)
	}
	status, _, body = speakerHTTPMultipart(t, http.MethodPost, server.URL+"/v1/verify", map[string]string{
		"profile_id": "legacy", "threshold": "0.5",
	}, audio)
	if status != http.StatusConflict || !bytes.Contains(body, []byte(`"code":"speaker_model_mismatch"`)) || bytes.Contains(body, []byte(`"score"`)) {
		t.Fatalf("mismatch status=%d body=%s", status, body)
	}

	status, _, body = speakerHTTPMultipart(t, http.MethodDelete, server.URL+"/v1/profiles/http-contract", nil, nil)
	if status != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", status, body)
	}
}

func speakerHTTPMultipart(t *testing.T, method, endpoint string, fields map[string]string, audio []byte) (int, http.Header, []byte) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if audio != nil {
		part, err := writer.CreateFormFile("audio", "test.wav")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(audio); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(method, endpoint, &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, resp.Header, result
}
