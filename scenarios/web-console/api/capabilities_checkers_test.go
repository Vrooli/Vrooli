package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResourceChecker_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	checker := &ResourceChecker{URL: srv.URL}
	status, msg := checker.Check(context.Background())

	if status != StatusAvailable {
		t.Errorf("status = %q, want %q", status, StatusAvailable)
	}
	if msg != "resource is healthy" {
		t.Errorf("message = %q, want %q", msg, "resource is healthy")
	}
}

func TestResourceChecker_Redirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer srv.Close()

	checker := &ResourceChecker{
		URL:    srv.URL,
		Client: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}
	status, msg := checker.Check(context.Background())

	if status != StatusAvailable {
		t.Errorf("status = %q, want %q", status, StatusAvailable)
	}
	if msg != "resource is healthy" {
		t.Errorf("message = %q, want %q", msg, "resource is healthy")
	}
}

func TestResourceChecker_Unavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	checker := &ResourceChecker{URL: srv.URL}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "resource returned unexpected status" {
		t.Errorf("message = %q, want %q", msg, "resource returned unexpected status")
	}
}

func TestResourceChecker_ConnectionRefused(t *testing.T) {
	// Start and immediately close a server to get a port that refuses connections
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	checker := &ResourceChecker{URL: url}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "resource is not responding" {
		t.Errorf("message = %q, want %q", msg, "resource is not responding")
	}
}

// --- WhisperChecker tests ---

// fakeWhisperServer creates a mock Whisper server with configurable behavior.
func fakeWhisperServer(t *testing.T, asrStatus int, asrBody any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
		if r.URL.Path == "/asr" {
			// Consume body to unblock pipe writer
			_, _ = io.Copy(io.Discard, r.Body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(asrStatus)
			if asrBody != nil {
				_ = json.NewEncoder(w).Encode(asrBody)
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestWhisperChecker_HealthyAndTranscribes(t *testing.T) {
	srv := fakeWhisperServer(t, http.StatusOK, map[string]string{"text": ""})
	defer srv.Close()

	checker := &WhisperChecker{BaseURL: srv.URL}
	status, msg := checker.Check(context.Background())

	if status != StatusAvailable {
		t.Errorf("status = %q, want %q", status, StatusAvailable)
	}
	if msg != "resource is healthy and transcription verified" {
		t.Errorf("message = %q, want %q", msg, "resource is healthy and transcription verified")
	}
}

func TestWhisperChecker_LiveButTranscriptionFails(t *testing.T) {
	srv := fakeWhisperServer(t, http.StatusInternalServerError, nil)
	defer srv.Close()

	checker := &WhisperChecker{BaseURL: srv.URL}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "transcription endpoint returned non-200 status" {
		t.Errorf("message = %q, want %q", msg, "transcription endpoint returned non-200 status")
	}
}

func TestWhisperChecker_LiveButInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Consume body
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	checker := &WhisperChecker{BaseURL: srv.URL}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "transcription response is not valid JSON" {
		t.Errorf("message = %q, want %q", msg, "transcription response is not valid JSON")
	}
}

func TestWhisperChecker_ConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	checker := &WhisperChecker{BaseURL: url}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "resource is not responding" {
		t.Errorf("message = %q, want %q", msg, "resource is not responding")
	}
}

func TestGenerateSilentWAV(t *testing.T) {
	wav := generateSilentWAV()

	// Basic WAV header validation
	if len(wav) < 44 {
		t.Fatalf("WAV too short: %d bytes", len(wav))
	}
	if string(wav[:4]) != "RIFF" {
		t.Errorf("missing RIFF header")
	}
	if string(wav[8:12]) != "WAVE" {
		t.Errorf("missing WAVE marker")
	}
	if string(wav[12:16]) != "fmt " {
		t.Errorf("missing fmt chunk")
	}
}
