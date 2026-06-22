package recommendation

import (
	"context"
	"testing"

	"agent-manager/internal/adapters/capacity"
	"agent-manager/internal/domain"
)

// fakeBroker records claim/release calls for the extractor wiring test.
type fakeBroker struct {
	claimed    string
	released   bool
	claimedID  string
	releasedID string
}

func (f *fakeBroker) Claim(_ context.Context, ownerID string, _ int64) (capacity.Lease, error) {
	f.claimed = ownerID
	f.claimedID = "clm-test"
	return capacity.Lease{ClaimID: "clm-test"}, nil
}

func (f *fakeBroker) Release(_ context.Context, claimID string) {
	f.released = true
	f.releasedID = claimID
}

// The extractor claims an op-scoped ollama capacity claim around the generate
// and releases it (even though resource-ollama is absent in the test env and
// extraction fails gracefully).
func TestExtractClaimsCapacityAroundGenerate(t *testing.T) {
	fb := &fakeBroker{}
	e := NewOllamaExtractor().WithCapacity(fb)

	// Pre-cancel so the resource-ollama exec fails fast (the capacity claim is
	// taken before the generate, so the assertions hold without a live model).
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := e.Extract(ctx, domain.ExtractionRequest{InvestigationText: "do a thing"})
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if res == nil {
		t.Fatal("Extract() returned nil result")
	}
	if fb.claimed != "ollama:agent-manager-extract" {
		t.Errorf("claimed owner = %q, want ollama:agent-manager-extract", fb.claimed)
	}
	if !fb.released || fb.releasedID != "clm-test" {
		t.Errorf("release not called with claim id (released=%v id=%q)", fb.released, fb.releasedID)
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON",
			input:    `{"categories": [{"name": "Test", "recommendations": [{"text": "Do something"}]}]}`,
			expected: `{"categories": [{"name": "Test", "recommendations": [{"text": "Do something"}]}]}`,
		},
		{
			name:     "JSON in markdown code block",
			input:    "```json\n{\"categories\": [{\"name\": \"Test\", \"recommendations\": [{\"text\": \"Do something\"}]}]}\n```",
			expected: `{"categories": [{"name": "Test", "recommendations": [{"text": "Do something"}]}]}`,
		},
		{
			name:     "JSON in plain code block",
			input:    "```\n{\"categories\": [{\"name\": \"Test\", \"recommendations\": [{\"text\": \"Do something\"}]}]}\n```",
			expected: `{"categories": [{"name": "Test", "recommendations": [{"text": "Do something"}]}]}`,
		},
		{
			name:     "JSON with leading text",
			input:    "Here is the JSON:\n{\"categories\": []}",
			expected: `{"categories": []}`,
		},
		{
			name:  "JSON with trailing text",
			input: "{\"categories\": []}\nThat's the output.",
			// Fixed: Now properly isolates the JSON object
			expected: `{"categories": []}`,
		},
		{
			name:     "nested objects",
			input:    `{"categories": [{"name": "Test", "recommendations": [{"text": "Do {something} weird"}]}]}`,
			expected: `{"categories": [{"name": "Test", "recommendations": [{"text": "Do {something} weird"}]}]}`,
		},
		{
			name:     "braces inside strings are ignored",
			input:    `{"text": "this { has } braces inside"}`,
			expected: `{"text": "this { has } braces inside"}`,
		},
		{
			name:     "escaped quotes in strings",
			input:    `{"text": "say \"hello\" and \"goodbye\""}`,
			expected: `{"text": "say \"hello\" and \"goodbye\""}`,
		},
		{
			name:     "complex nested with braces in strings",
			input:    `{"cats": [{"name": "Code {Review}", "recs": [{"text": "Check { and } matching"}]}]}`,
			expected: `{"cats": [{"name": "Code {Review}", "recs": [{"text": "Check { and } matching"}]}]}`,
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "no JSON",
			input:    "This is just plain text with no JSON",
			expected: "",
		},
		{
			name:     "whitespace around JSON",
			input:    "   \n\n  {\"categories\": []}  \n\n  ",
			expected: `{"categories": []}`,
		},
		{
			name:     "multiple code blocks - takes first",
			input:    "```json\n{\"first\": true}\n```\n\n```json\n{\"second\": true}\n```",
			expected: `{"first": true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			if result != tt.expected {
				t.Errorf("extractJSON(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractJSON_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldMatch bool
		note        string
	}{
		{
			name:        "unclosed brace",
			input:       `{"categories": [`,
			shouldMatch: false, // Fixed: Now returns empty for unclosed JSON
			note:        "Returns empty for unclosed JSON objects",
		},
		{
			name:        "array instead of object - finds inner object",
			input:       `[{"name": "Test"}]`,
			shouldMatch: true, // Finds the inner object {"name": "Test"}
			note:        "Finds the first { and matches to corresponding }",
		},
		{
			name:        "empty object",
			input:       `{}`,
			shouldMatch: true,
		},
		{
			name:        "deeply nested",
			input:       `{"a": {"b": {"c": {"d": {}}}}}`,
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			hasMatch := result != ""
			if hasMatch != tt.shouldMatch {
				t.Errorf("extractJSON(%q) hasMatch=%v, want %v (result=%q) - %s",
					tt.input, hasMatch, tt.shouldMatch, result, tt.note)
			}
		})
	}
}

func TestBuildPrompt(t *testing.T) {
	extractor := NewOllamaExtractor()
	prompt := extractor.buildPrompt("Test investigation output")

	// Verify the prompt contains key elements
	if !contains(prompt, "investigation output") {
		t.Error("prompt should contain input text")
	}
	if !contains(prompt, "JSON") {
		t.Error("prompt should mention JSON output format")
	}
	if !contains(prompt, "categories") {
		t.Error("prompt should describe the expected structure")
	}
	if !contains(prompt, "recommendations") {
		t.Error("prompt should describe the expected structure")
	}
}

func TestSanitizeJSONString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "valid JSON - no changes",
			input:    `{"text": "hello world"}`,
			expected: `{"text": "hello world"}`,
		},
		{
			name:     "already escaped newline",
			input:    `{"text": "line1\nline2"}`,
			expected: `{"text": "line1\nline2"}`,
		},
		{
			name:     "unescaped newline in string",
			input:    "{\"text\": \"line1\nline2\"}",
			expected: `{"text": "line1\nline2"}`,
		},
		{
			name:     "unescaped tab in string",
			input:    "{\"text\": \"col1\tcol2\"}",
			expected: `{"text": "col1\tcol2"}`,
		},
		{
			name:     "unescaped carriage return in string",
			input:    "{\"text\": \"line1\rline2\"}",
			expected: `{"text": "line1\rline2"}`,
		},
		{
			name:     "newline between JSON elements is preserved",
			input:    "{\n\"text\": \"hello\"\n}",
			expected: "{\n\"text\": \"hello\"\n}",
		},
		{
			name:     "multiple unescaped newlines in string",
			input:    "{\"text\": \"line1\nline2\nline3\"}",
			expected: `{"text": "line1\nline2\nline3"}`,
		},
		{
			name:     "mixed escaped and unescaped",
			input:    "{\"text\": \"escaped\\nnewline and unescaped\nnewline\"}",
			expected: `{"text": "escaped\nnewline and unescaped\nnewline"}`,
		},
		{
			name:     "nested objects with unescaped newlines",
			input:    "{\"cat\": {\"recs\": [{\"text\": \"line1\nline2\"}]}}",
			expected: `{"cat": {"recs": [{"text": "line1\nline2"}]}}`,
		},
		{
			name:     "escaped quote in string",
			input:    `{"text": "say \"hello\""}`,
			expected: `{"text": "say \"hello\""}`,
		},
		{
			name:     "escaped backslash in string",
			input:    `{"text": "path\\file"}`,
			expected: `{"text": "path\\file"}`,
		},
		{
			name:     "complex real-world example",
			input:    "{\"categories\": [{\"name\": \"Code Changes\", \"recommendations\": [{\"text\": \"Update the function to:\n1. Add error handling\n2. Add logging\"}]}]}",
			expected: `{"categories": [{"name": "Code Changes", "recommendations": [{"text": "Update the function to:\n1. Add error handling\n2. Add logging"}]}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeJSONString(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeJSONString() = %q, want %q", result, tt.expected)
			}
		})
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
