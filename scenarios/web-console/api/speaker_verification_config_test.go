package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	voicev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/voice"
)

func TestLoadSpeakerVerificationConfig_MissingFile(t *testing.T) {
	cfg, err := loadSpeakerVerificationConfig("/nonexistent/path/speaker-verification-config.json")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	defaults := DefaultSpeakerVerificationConfig()
	if cfg.Enabled != defaults.Enabled || cfg.Threshold != defaults.Threshold || cfg.Mode != defaults.Mode {
		t.Fatalf("expected defaults, got %+v", cfg)
	}
}

func TestSpeakerVerificationConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "speaker-verification-config.json")
	cfg := SpeakerVerificationConfig{
		Enabled:                     true,
		ProfileIDs:                  []string{"default"},
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
	if loaded.Enabled != cfg.Enabled || loaded.Threshold != cfg.Threshold ||
		loaded.Mode != cfg.Mode || loaded.RejectBehavior != cfg.RejectBehavior ||
		len(loaded.ProfileIDs) != len(cfg.ProfileIDs) {
		t.Fatalf("loaded config mismatch:\n got %+v\nwant %+v", loaded, cfg)
	}
	for i, id := range cfg.ProfileIDs {
		if loaded.ProfileIDs[i] != id {
			t.Fatalf("profileIDs[%d] = %q, want %q", i, loaded.ProfileIDs[i], id)
		}
	}
}

func TestSpeakerVerificationConfigValidate(t *testing.T) {
	cfg := DefaultSpeakerVerificationConfig()
	cfg.Enabled = true
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing profileIds validation error")
	}
	cfg.ProfileIDs = []string{"default"}
	cfg.Threshold = 1.1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected threshold validation error")
	}
}

func TestHandleGetSpeakerVerificationConfig(t *testing.T) {
	srv := &Server{
		speakerVerificationConfig: DefaultSpeakerVerificationConfig(),
	}
	cfg, err := callGetSpeakerConfig(t, srv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defaults := DefaultSpeakerVerificationConfig()
	if cfg.GetEnabled() != defaults.Enabled || cfg.GetThreshold() != defaults.Threshold || cfg.GetMode() != defaults.Mode {
		t.Fatalf("config mismatch: got %+v", cfg)
	}
}

func TestHandleUpdateSpeakerVerificationConfig(t *testing.T) {
	dir := t.TempDir()
	srv := &Server{
		speakerVerificationConfig:     DefaultSpeakerVerificationConfig(),
		speakerVerificationConfigPath: filepath.Join(dir, "speaker-verification-config.json"),
	}
	cfg, err := callUpdateSpeakerConfig(t, srv, &voicev1.UpdateSpeakerConfigRequest{
		Enabled:           true,
		HasEnabled:        true,
		ProfileIds:        []string{"default"},
		HasProfileIds:     true,
		Threshold:         0.9,
		HasThreshold:      true,
		Mode:              "filter",
		HasMode:           true,
		RejectBehavior:    "drop",
		HasRejectBehavior: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.GetEnabled() || len(cfg.GetProfileIds()) != 1 || cfg.GetProfileIds()[0] != "default" || cfg.GetThreshold() != 0.9 {
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
			ProfileIDs:     []string{"default"},
			Threshold:      0.85,
			Mode:           "filter",
			RejectBehavior: "drop",
		},
		speakerVerification: &SpeakerVerificationResourceClient{
			BaseURL: resource.URL,
			Client:  resource.Client(),
		},
	}

	status, err := callGetSpeakerStatus(t, srv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.GetCapability() != string(StatusAvailable) {
		t.Fatalf("capability = %q, want %q", status.GetCapability(), StatusAvailable)
	}
	if !status.GetResourceReady() {
		t.Fatal("expected resourceReady=true")
	}
	if !status.GetProfileConfigured() || !status.GetProfileExists() {
		t.Fatalf("expected configured existing profile, got %+v", status)
	}
	if status.GetProfileCount() != 1 {
		t.Fatalf("profileCount = %d, want 1", status.GetProfileCount())
	}
	if status.GetInfo() == nil || status.GetInfo().GetBackend() != "nemo-titanet" {
		t.Fatalf("unexpected info: %+v", status.GetInfo())
	}
	if _, err := time.Parse(time.RFC3339, status.GetCheckedAt()); err != nil {
		t.Fatalf("checkedAt is not RFC3339: %q", status.GetCheckedAt())
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

	resp, err := callListSpeakerProfiles(t, srv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetCount() != 1 || len(resp.GetProfiles()) != 1 || resp.GetProfiles()[0].GetId() != "default" {
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

	addToActive := true
	enable := true
	_, err := callEnrollSpeakerProfile(t, srv, &voicev1.EnrollSpeakerProfileRequest{
		Audio:          []byte("fake-audio"),
		ProfileId:      "default",
		DisplayName:    "My Voice",
		AddToActive:    addToActive,
		HasAddToActive: true,
		Enable:         enable,
		HasEnable:      true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := srv.getSpeakerVerificationConfig()
	if !cfg.Enabled || len(cfg.ProfileIDs) != 1 || cfg.ProfileIDs[0] != "default" {
		t.Fatalf("unexpected config after enroll: %+v", cfg)
	}
}

func TestHandleClearSpeakerProfileBinding(t *testing.T) {
	dir := t.TempDir()
	srv := &Server{
		speakerVerificationConfig: SpeakerVerificationConfig{
			Enabled:        true,
			ProfileIDs:     []string{"default", "singing"},
			Threshold:      0.85,
			Mode:           "filter",
			RejectBehavior: "drop",
		},
		speakerVerificationConfigPath: filepath.Join(dir, "speaker-verification-config.json"),
	}

	if _, err := callClearSpeakerProfileBinding(t, srv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := srv.getSpeakerVerificationConfig()
	if cfg.Enabled || len(cfg.ProfileIDs) != 0 {
		t.Fatalf("unexpected config after clear: %+v", cfg)
	}
}
