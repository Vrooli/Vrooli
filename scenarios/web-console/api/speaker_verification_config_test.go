package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadSpeakerVerificationConfig_MissingFile(t *testing.T) {
	cfg, err := loadSpeakerVerificationConfig("/nonexistent/path/speaker-verification-config.json")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cfg != DefaultSpeakerVerificationConfig() {
		t.Fatalf("expected defaults, got %+v", cfg)
	}
}

func TestSpeakerVerificationConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "speaker-verification-config.json")
	cfg := SpeakerVerificationConfig{
		Enabled:                     true,
		ProfileID:                   "default",
		Threshold:                   0.9,
		Mode:                        "filter",
		RejectBehavior:              "drop",
		FallbackWithoutVerification: false,
	}
	if err := saveSpeakerVerificationConfig(path, cfg); err != nil {
		t.Fatalf("saveSpeakerVerificationConfig error = %v", err)
	}
	loaded, err := loadSpeakerVerificationConfig(path)
	if err != nil {
		t.Fatalf("loadSpeakerVerificationConfig error = %v", err)
	}
	if loaded != cfg {
		t.Fatalf("loaded config mismatch:\n got %+v\nwant %+v", loaded, cfg)
	}
}

func TestSpeakerVerificationConfigValidate(t *testing.T) {
	cfg := DefaultSpeakerVerificationConfig()
	cfg.Enabled = true
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing profileId validation error")
	}
	cfg.ProfileID = "default"
	cfg.Threshold = 1.1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected threshold validation error")
	}
}

func TestHandleGetSpeakerVerificationConfig(t *testing.T) {
	srv := &Server{
		speakerVerificationConfig: DefaultSpeakerVerificationConfig(),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/voice/speaker/config", nil)
	rec := httptest.NewRecorder()
	srv.handleGetSpeakerVerificationConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var cfg SpeakerVerificationConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if cfg != DefaultSpeakerVerificationConfig() {
		t.Fatalf("config mismatch: got %+v", cfg)
	}
}

func TestHandleUpdateSpeakerVerificationConfig(t *testing.T) {
	dir := t.TempDir()
	srv := &Server{
		speakerVerificationConfig:     DefaultSpeakerVerificationConfig(),
		speakerVerificationConfigPath: filepath.Join(dir, "speaker-verification-config.json"),
	}
	body := `{"enabled":true,"profileId":"default","threshold":0.9,"mode":"filter","rejectBehavior":"drop"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/voice/speaker/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleUpdateSpeakerVerificationConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}
	var cfg SpeakerVerificationConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !cfg.Enabled || cfg.ProfileID != "default" || cfg.Threshold != 0.9 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestHandleGetSpeakerVerificationStatus(t *testing.T) {
	resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ready":
			writeJSON(w, http.StatusOK, SpeakerVerificationResourceReady{
				Status:         "ready",
				ModelLoaded:    true,
				ProfileStoreOK: true,
				TempDirOK:      true,
			})
		case "/v1/profiles":
			writeJSON(w, http.StatusOK, SpeakerVerificationProfileList{
				Count: 1,
				Profiles: []SpeakerVerificationProfile{
					{ID: "default", DisplayName: "Default Voice"},
				},
			})
		case "/v1/info":
			writeJSON(w, http.StatusOK, SpeakerVerificationResourceInfo{
				Backend:      "nemo-titanet",
				Model:        "titanet_large",
				Device:       "cpu",
				SampleRate:   16000,
				Version:      "test",
				EmbeddingDim: 192,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer resource.Close()

	checker := &fakeChecker{status: StatusAvailable, message: "resource is healthy"}
	reg := NewCapabilityRegistry(
		knownCapabilities,
		map[string]StatusChecker{"speaker-verification": checker},
		0,
	)
	reg.SetLivenessCheckers(map[string]StatusChecker{"speaker-verification": checker})

	srv := &Server{
		capabilities: reg,
		speakerVerificationConfig: SpeakerVerificationConfig{
			Enabled:        true,
			ProfileID:      "default",
			Threshold:      0.85,
			Mode:           "filter",
			RejectBehavior: "drop",
		},
		speakerVerification: &SpeakerVerificationResourceClient{
			BaseURL: resource.URL,
			Client:  resource.Client(),
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/voice/speaker/status", nil)
	rec := httptest.NewRecorder()
	srv.handleGetSpeakerVerificationStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}

	var status SpeakerVerificationStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if status.Capability != string(StatusAvailable) {
		t.Fatalf("capability = %q, want %q", status.Capability, StatusAvailable)
	}
	if !status.ResourceReady {
		t.Fatal("expected resourceReady=true")
	}
	if !status.ProfileConfigured || !status.ProfileExists {
		t.Fatalf("expected configured existing profile, got %+v", status)
	}
	if status.ProfileCount != 1 {
		t.Fatalf("profileCount = %d, want 1", status.ProfileCount)
	}
	if status.Info == nil || status.Info.Backend != "nemo-titanet" {
		t.Fatalf("unexpected info: %+v", status.Info)
	}
	if _, err := time.Parse(time.RFC3339, status.CheckedAt); err != nil {
		t.Fatalf("checkedAt is not RFC3339: %q", status.CheckedAt)
	}
}

func TestHandleGetSpeakerVerificationProfiles(t *testing.T) {
	resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/profiles" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, SpeakerVerificationProfileList{
			Count: 1,
			Profiles: []SpeakerVerificationProfile{
				{ID: "default", DisplayName: "My Voice"},
			},
		})
	}))
	defer resource.Close()

	srv := &Server{
		speakerVerification: &SpeakerVerificationResourceClient{
			BaseURL: resource.URL,
			Client:  resource.Client(),
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/voice/speaker/profiles", nil)
	rec := httptest.NewRecorder()
	srv.handleGetSpeakerVerificationProfiles(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}
	var resp SpeakerVerificationProfilesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Count != 1 || len(resp.Profiles) != 1 || resp.Profiles[0].ID != "default" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestHandleEnrollSpeakerProfile(t *testing.T) {
	resource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/profiles" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if got := r.FormValue("profile_id"); got != "default" {
			t.Fatalf("profile_id = %q, want default", got)
		}
		file, _, err := r.FormFile("audio")
		if err != nil {
			t.Fatalf("missing audio field: %v", err)
		}
		_, _ = io.ReadAll(file)
		_ = file.Close()
		writeJSON(w, http.StatusOK, SpeakerVerificationEnrollmentResponse{
			ProfileID:              "default",
			DisplayName:            "My Voice",
			EmbeddingDim:           192,
			SampleRate:             16000,
			EnrollmentAudioSeconds: 6.2,
			ModelName:              "titanet_large",
			CreatedAt:              time.Now().UTC().Format(time.RFC3339),
		})
	}))
	defer resource.Close()

	dir := t.TempDir()
	srv := &Server{
		speakerVerificationConfig:     DefaultSpeakerVerificationConfig(),
		speakerVerificationConfigPath: filepath.Join(dir, "speaker-verification-config.json"),
		speakerVerification: &SpeakerVerificationResourceClient{
			BaseURL: resource.URL,
			Client:  resource.Client(),
		},
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("audio_file", "voice.webm")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write([]byte("fake-audio")); err != nil {
		t.Fatalf("write audio: %v", err)
	}
	_ = writer.WriteField("profileId", "default")
	_ = writer.WriteField("displayName", "My Voice")
	_ = writer.WriteField("setActive", "true")
	_ = writer.WriteField("enable", "true")
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/speaker/enroll", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	srv.handleEnrollSpeakerProfile(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}

	cfg := srv.getSpeakerVerificationConfig()
	if !cfg.Enabled || cfg.ProfileID != "default" {
		t.Fatalf("unexpected config after enroll: %+v", cfg)
	}
}

func TestHandleClearSpeakerProfileBinding(t *testing.T) {
	dir := t.TempDir()
	srv := &Server{
		speakerVerificationConfig: SpeakerVerificationConfig{
			Enabled:        true,
			ProfileID:      "default",
			Threshold:      0.85,
			Mode:           "filter",
			RejectBehavior: "drop",
		},
		speakerVerificationConfigPath: filepath.Join(dir, "speaker-verification-config.json"),
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/voice/speaker/profile", nil)
	rec := httptest.NewRecorder()
	srv.handleClearSpeakerProfileBinding(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}

	cfg := srv.getSpeakerVerificationConfig()
	if cfg.Enabled || cfg.ProfileID != "" {
		t.Fatalf("unexpected config after clear: %+v", cfg)
	}
}
