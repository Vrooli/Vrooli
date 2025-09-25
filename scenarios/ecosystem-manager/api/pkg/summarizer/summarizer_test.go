package summarizer

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseResult(t *testing.T) {
	result, err := parseResult([]byte(`{"note":"Notes","classification":"full_complete"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Note != "Notes" {
		t.Fatalf("expected note 'Notes', got %q", result.Note)
	}
	if result.Classification != classificationFull {
		t.Fatalf("expected classification %q, got %q", classificationFull, result.Classification)
	}

	res2, err := parseResult([]byte(`{"note":"","classification":"nope"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.Note != "Not sure current status" {
		t.Fatalf("expected fallback note, got %q", res2.Note)
	}
	if res2.Classification != classificationUncertain {
		t.Fatalf("expected fallback classification %q, got %q", classificationUncertain, res2.Classification)
	}
}

func TestParseResultInvalidJSON(t *testing.T) {
	if _, err := parseResult([]byte(`not-json`)); err == nil {
		t.Fatalf("expected error for invalid json")
	}
}

func TestDefaultResult(t *testing.T) {
	def := DefaultResult()
	if def.Note == "" || def.Classification != classificationUncertain {
		t.Fatalf("default result should return uncertain classification")
	}
}

func TestSanitizeLLMJSON(t *testing.T) {
	raw := "```json\n{\n  \"note\": \"Line one\nLine two\",\n  \"classification\": \"partial_progress\"\n}\n```"
	sanitized := sanitizeLLMJSON([]byte(raw))

	if strings.Contains(string(sanitized), "```") {
		t.Fatalf("sanitized payload should not retain code fences: %q", string(sanitized))
	}

	var payload struct {
		Note           string `json:"note"`
		Classification string `json:"classification"`
	}
	if err := json.Unmarshal(sanitized, &payload); err != nil {
		t.Fatalf("expected sanitized JSON to be valid: %v", err)
	}

	if payload.Note != "Line one\nLine two" {
		t.Fatalf("expected newline-preserving note, got %q", payload.Note)
	}
	if payload.Classification != "partial_progress" {
		t.Fatalf("unexpected classification: %q", payload.Classification)
	}

	raw2 := []byte("{\n  \"note\": \"Already pretty good, but could use some additional validation/tidying. Notes from last time:\n- Confirm gateway restarts cleanly\n- Capture failing integration logs\",\n  \"classification\": \"uncertain\"\n}")
	sanitized2 := sanitizeLLMJSON(raw2)
	if !json.Valid(sanitized2) {
		t.Fatalf("expected newline handling to yield valid JSON: %q", string(sanitized2))
	}
}
