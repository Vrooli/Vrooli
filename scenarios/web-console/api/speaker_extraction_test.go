package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	intvoice "web-console/internal/voice"
)

// ---------------------------------------------------------------------------
// Go client Extract() tests
// ---------------------------------------------------------------------------

func fakeWAVBytes(n int) []byte {
	// Minimal valid WAV header + n zero bytes of PCM data.
	header := []byte("RIFF")
	header = append(header, byte(36+n), byte((36+n)>>8), byte((36+n)>>16), byte((36+n)>>24))
	header = append(header, []byte("WAVEfmt ")...)
	header = append(header, 16, 0, 0, 0)      // fmt chunk size
	header = append(header, 1, 0)             // PCM format
	header = append(header, 1, 0)             // mono
	header = append(header, 0x80, 0x3E, 0, 0) // 16000 Hz
	header = append(header, 0, 0x7D, 0, 0)    // byte rate (32000)
	header = append(header, 2, 0)             // block align
	header = append(header, 16, 0)            // bits per sample
	header = append(header, []byte("data")...)
	header = append(header, byte(n), byte(n>>8), byte(n>>16), byte(n>>24))
	header = append(header, make([]byte, n)...)
	return header
}

func TestExtract_Success(t *testing.T) {
	wavData := fakeWAVBytes(3200) // 0.1s of 16kHz 16-bit audio

	resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/extract" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.FormValue("profile_id"); got != "default" {
			t.Errorf("profile_id = %q, want default", got)
		}
		if got := r.FormValue("verify"); got != "true" {
			t.Errorf("verify = %q, want true", got)
		}

		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("X-Speaker-Score", "0.92")
		w.Header().Set("X-Speaker-Matched", "true")
		w.Header().Set("X-Duration-Ms", "1234")
		w.Header().Set("X-Audio-Seconds", "2.5")
		_, _ = w.Write(wavData)
	}))
	defer resource.Close()

	client := &intvoice.SpeakerClient{
		BaseURL: resource.URL,
		Client:  resource.Client(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.Extract(ctx, []byte("fake-audio"), "default", true)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if len(result.Audio) != len(wavData) {
		t.Errorf("audio length = %d, want %d", len(result.Audio), len(wavData))
	}
	if result.Score < 0.91 || result.Score > 0.93 {
		t.Errorf("score = %.4f, want ~0.92", result.Score)
	}
	if !result.Matched {
		t.Error("expected matched=true")
	}
	if result.DurationMs < 1233 || result.DurationMs > 1235 {
		t.Errorf("durationMs = %.1f, want ~1234", result.DurationMs)
	}
	if result.AudioSeconds < 2.4 || result.AudioSeconds > 2.6 {
		t.Errorf("audioSeconds = %.2f, want ~2.5", result.AudioSeconds)
	}
}

func TestExtract_ProfileNotFound(t *testing.T) {
	resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"detail": "Profile not found"})
	}))
	defer resource.Close()

	client := &intvoice.SpeakerClient{
		BaseURL: resource.URL,
		Client:  resource.Client(),
	}

	ctx := context.Background()
	_, err := client.Extract(ctx, []byte("audio"), "nonexistent", true)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if got := err.Error(); !contains(got, "404") {
		t.Errorf("error = %q, want to contain '404'", got)
	}
}

func TestExtract_Timeout(t *testing.T) {
	resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer resource.Close()

	client := &intvoice.SpeakerClient{
		BaseURL: resource.URL,
		Client:  &http.Client{Timeout: 100 * time.Millisecond},
	}

	ctx := context.Background()
	_, err := client.Extract(ctx, []byte("audio"), "default", true)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// intvoice.ExtractTargetSpeaker() gate function tests
// ---------------------------------------------------------------------------

func TestExtractTargetSpeaker_Enabled(t *testing.T) {
	wavData := fakeWAVBytes(3200)

	resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Speaker-Score", "0.85")
		w.Header().Set("X-Speaker-Matched", "true")
		w.Header().Set("X-Duration-Ms", "500")
		w.Header().Set("X-Audio-Seconds", "2.0")
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write(wavData)
	}))
	defer resource.Close()

	cfg := intvoice.SpeakerConfig{
		Enabled:           true,
		ExtractionEnabled: true,
		ProfileIDs:        []string{"default"},
		Threshold:         0.35,
		Mode:              "filter",
		RejectBehavior:    "drop",
	}
	client := &intvoice.SpeakerClient{
		BaseURL: resource.URL,
		Client:  resource.Client(),
	}

	ctx := context.Background()
	audio, decision := intvoice.ExtractTargetSpeaker(ctx, cfg, client, []byte("original-audio"))

	if !decision.Allowed {
		t.Fatal("expected allowed=true")
	}
	if !decision.Extracted {
		t.Fatal("expected extracted=true")
	}
	if !decision.Matched {
		t.Fatal("expected matched=true")
	}
	if len(audio) != len(wavData) {
		t.Errorf("audio length = %d, want %d (extracted WAV)", len(audio), len(wavData))
	}
}

func TestExtractTargetSpeaker_Disabled(t *testing.T) {
	cfg := intvoice.SpeakerConfig{
		Enabled:                     true,
		ExtractionEnabled:           false,
		ProfileIDs:                  []string{"default"},
		Threshold:                   0.35,
		Mode:                        "filter",
		RejectBehavior:              "drop",
		FallbackWithoutVerification: true,
	}

	ctx := context.Background()
	originalAudio := []byte("original-audio")
	audio, decision := intvoice.ExtractTargetSpeaker(ctx, cfg, nil, originalAudio)

	if decision.Extracted {
		t.Fatal("expected extracted=false when TSE disabled")
	}
	if string(audio) != string(originalAudio) {
		t.Error("expected original audio returned when TSE disabled")
	}
}

func TestExtractTargetSpeaker_ClientError(t *testing.T) {
	// Resource returns 500 to simulate extraction failure.
	resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/extract" {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "internal error")
			return
		}
		// Verify endpoint for fallback
		if r.URL.Path == "/v1/verify" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(intvoice.SpeakerVerifyResult{
				ProfileID:  "default",
				Matched:    true,
				Score:      0.9,
				Threshold:  0.35,
				DurationMs: 100,
				Backend:    "nemo-titanet",
				Model:      "titanet_large",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer resource.Close()

	cfg := intvoice.SpeakerConfig{
		Enabled:           true,
		ExtractionEnabled: true,
		ProfileIDs:        []string{"default"},
		Threshold:         0.35,
		Mode:              "filter",
		RejectBehavior:    "drop",
	}
	client := &intvoice.SpeakerClient{
		BaseURL: resource.URL,
		Client:  resource.Client(),
	}

	ctx := context.Background()
	originalAudio := []byte("original-audio")
	audio, decision := intvoice.ExtractTargetSpeaker(ctx, cfg, client, originalAudio)

	// Should fall back to verification-only and return original audio
	if decision.Extracted {
		t.Fatal("expected extracted=false on client error fallback")
	}
	if !decision.Allowed {
		t.Fatal("expected allowed=true after fallback to verify-only (matched=true)")
	}
	if string(audio) != string(originalAudio) {
		t.Error("expected original audio returned on extraction failure")
	}
}

// ---------------------------------------------------------------------------
// Config round-trip with ExtractionEnabled
// ---------------------------------------------------------------------------

func TestSpeakerVerificationConfig_ExtractionEnabled_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "speaker-verification-config.json")
	cfg := intvoice.SpeakerConfig{
		Enabled:           true,
		ProfileIDs:        []string{"default"},
		Threshold:         0.35,
		Mode:              "filter",
		RejectBehavior:    "drop",
		ExtractionEnabled: true,
	}
	if err := intvoice.SaveSpeakerConfig(path, cfg); err != nil {
		t.Fatalf("save error = %v", err)
	}
	loaded, err := intvoice.LoadSpeakerConfig(path)
	if err != nil {
		t.Fatalf("load error = %v", err)
	}
	if !loaded.ExtractionEnabled {
		t.Fatal("expected extractionEnabled=true after round-trip")
	}
}

func TestSpeakerVerificationConfigPatch_ExtractionEnabled(t *testing.T) {
	base := intvoice.DefaultSpeakerConfig()
	if base.ExtractionEnabled {
		t.Fatal("default should have extractionEnabled=false")
	}

	enabled := true
	patch := intvoice.SpeakerConfigPatch{ExtractionEnabled: &enabled}
	updated := patch.Apply(base)
	if !updated.ExtractionEnabled {
		t.Fatal("patch should set extractionEnabled=true")
	}
}
