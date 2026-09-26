package bootstrap_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"audio-tools/internal/bootstrap"
)

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

func TestLoadNativeSherpaEndpointFeedsSpeakerAndTTS(t *testing.T) {
	t.Setenv("SHERPA_ONNX_URL", "")
	t.Setenv("AUDIO_SHERPA_URL", "http://sherpa-native:8880")

	env := bootstrap.Load()
	if env.SherpaURL != "http://sherpa-native:8880" {
		t.Fatalf("native sherpa endpoint was not loaded: sherpa=%q", env.SherpaURL)
	}
}
