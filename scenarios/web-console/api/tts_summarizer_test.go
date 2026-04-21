package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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
		if !strings.Contains(sysMsg["content"].(string), "Tighten") {
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
		if !strings.Contains(sysMsg["content"].(string), "Rewrite") {
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
		"light":    "Tighten",
		"moderate": "Rewrite",
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

func TestStripThinkTags(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"no tags", "hello world", "hello world"},
		{"with think block", "<think>\nreasoning here\n</think>\nactual answer", "actual answer"},
		{"unclosed tag", "<think>partial reasoning", ""},
		{"multiple blocks", "<think>a</think>first<think>b</think>second", "firstsecond"},
		{"empty think", "<think></think>answer", "answer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripThinkTags(tt.input)
			if got != tt.want {
				t.Errorf("stripThinkTags(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTTSSummarizer_StripsThinkTags(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{
				"content": "<think>\nlong reasoning\n</think>\nThe quick summary.",
			},
		})
	}))
	defer ts.Close()

	s := NewTTSSummarizer(ts.URL)
	result, err := s.Summarize(context.Background(), "text", "qwen3:1.7b", "moderate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "The quick summary." {
		t.Errorf("expected think tags stripped, got %q", result)
	}
}

func TestTTSSummarizer_SendsTokenAndTemperatureOptions(t *testing.T) {
	cases := []struct {
		level     string
		inputText string
	}{
		{"light", strings.Repeat("word ", 400)},
		{"moderate", strings.Repeat("word ", 400)},
		{"heavy", strings.Repeat("word ", 400)},
		{"moderate", "tiny"}, // exercises the floor path
	}
	for _, tc := range cases {
		t.Run(tc.level+"/"+strconv.Itoa(len(tc.inputText)), func(t *testing.T) {
			var captured map[string]any
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewDecoder(r.Body).Decode(&captured)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"message": map[string]string{"content": "ok"},
				})
			}))
			defer ts.Close()

			s := NewTTSSummarizer(ts.URL)
			if _, err := s.Summarize(context.Background(), tc.inputText, "m", tc.level); err != nil {
				t.Fatalf("summarize: %v", err)
			}

			opts, ok := captured["options"].(map[string]any)
			if !ok {
				t.Fatalf("expected options in request body, got %#v", captured)
			}
			temp, _ := opts["temperature"].(float64)
			if temp != 0.2 {
				t.Errorf("expected temperature=0.2, got %v", opts["temperature"])
			}
			numPredict, _ := opts["num_predict"].(float64)
			want := summarizeTokenBudget(tc.level, len(tc.inputText))
			if int(numPredict) != want {
				t.Errorf("num_predict: got %v, want %d", opts["num_predict"], want)
			}
			if numPredict <= 0 {
				t.Errorf("num_predict must be positive, got %v", opts["num_predict"])
			}
		})
	}
}

func TestSummarizeTokenBudget_LevelShape(t *testing.T) {
	// heavy is a constant budget — independent of input.
	if got := summarizeTokenBudget("heavy", 100); got != 120 {
		t.Errorf("heavy budget: got %d, want 120", got)
	}
	if got := summarizeTokenBudget("heavy", 100000); got != 120 {
		t.Errorf("heavy budget with large input: got %d, want 120", got)
	}

	// moderate scales with input above the floor.
	smallModerate := summarizeTokenBudget("moderate", 10)
	if smallModerate != 60 {
		t.Errorf("moderate floor: got %d, want 60", smallModerate)
	}
	largeModerate := summarizeTokenBudget("moderate", 10000)
	if largeModerate <= smallModerate {
		t.Errorf("moderate budget should scale: got %d <= floor %d", largeModerate, smallModerate)
	}

	// light is looser than moderate for the same input.
	inputChars := 10000
	if summarizeTokenBudget("light", inputChars) <= summarizeTokenBudget("moderate", inputChars) {
		t.Errorf("light budget should exceed moderate for the same input")
	}

	// Unknown levels fall through to moderate.
	if summarizeTokenBudget("unknown", inputChars) != summarizeTokenBudget("moderate", inputChars) {
		t.Errorf("unknown level should default to moderate budget")
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
