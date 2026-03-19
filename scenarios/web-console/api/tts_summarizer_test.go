package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTTSSummarizer_Summarize(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("expected /api/chat, got %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		messages := body["messages"].([]any)
		sysMsg := messages[0].(map[string]any)
		// Verify system prompt contains the level-specific text
		if !strings.Contains(sysMsg["content"].(string), "Condense") {
			t.Errorf("expected light-level prompt, got %q", sysMsg["content"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"content": "summarized output"},
		})
	}))
	defer ts.Close()

	s := NewTTSSummarizer(ts.URL)
	result, err := s.Summarize(context.Background(), "long text input", "test-model", "light")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "summarized output" {
		t.Errorf("expected %q, got %q", "summarized output", result)
	}
}

func TestTTSSummarizer_UnknownLevel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		messages := body["messages"].([]any)
		sysMsg := messages[0].(map[string]any)
		// Unknown level should fall back to moderate
		if !strings.Contains(sysMsg["content"].(string), "Summarize") {
			t.Errorf("expected moderate-level prompt for unknown level, got %q", sysMsg["content"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"content": "ok"},
		})
	}))
	defer ts.Close()

	s := NewTTSSummarizer(ts.URL)
	_, err := s.Summarize(context.Background(), "text", "model", "unknown_level")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTTSSummarizer_EachLevel(t *testing.T) {
	levels := map[string]string{
		"light":    "Condense",
		"moderate": "Summarize",
		"heavy":    "brief spoken summary",
	}
	for level, expected := range levels {
		t.Run(level, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				messages := body["messages"].([]any)
				sysMsg := messages[0].(map[string]any)
				if !strings.Contains(sysMsg["content"].(string), expected) {
					t.Errorf("level %s: expected prompt containing %q, got %q", level, expected, sysMsg["content"])
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"message": map[string]string{"content": "ok"},
				})
			}))
			defer ts.Close()

			s := NewTTSSummarizer(ts.URL)
			_, err := s.Summarize(context.Background(), "text", "model", level)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTTSSummarizer_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer ts.Close()

	s := NewTTSSummarizer(ts.URL)
	_, err := s.Summarize(context.Background(), "text", "model", "moderate")
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestTTSSummarizer_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"content": "too late"},
		})
	}))
	defer ts.Close()

	s := NewTTSSummarizer(ts.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := s.Summarize(ctx, "text", "model", "moderate")
	if err == nil {
		t.Error("expected error for timeout")
	}
}
