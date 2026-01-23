package prompts

import (
	"testing"
)

func TestGenerateUniqueID_NoConflict(t *testing.T) {
	// No existing IDs - should return base ID
	idExists := func(id string) bool {
		return false
	}

	result, err := GenerateUniqueID("New Prompt", idExists)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "new-prompt" {
		t.Errorf("expected 'new-prompt', got '%s'", result)
	}
}

func TestGenerateUniqueID_SingleConflict(t *testing.T) {
	// "new-prompt" exists, so should return "new-prompt-1"
	idExists := func(id string) bool {
		return id == "new-prompt"
	}

	result, err := GenerateUniqueID("New Prompt", idExists)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "new-prompt-1" {
		t.Errorf("expected 'new-prompt-1', got '%s'", result)
	}
}

func TestGenerateUniqueID_MultipleConflicts(t *testing.T) {
	// "new-prompt", "new-prompt-1", "new-prompt-2" exist
	existing := map[string]bool{
		"new-prompt":   true,
		"new-prompt-1": true,
		"new-prompt-2": true,
	}
	idExists := func(id string) bool {
		return existing[id]
	}

	result, err := GenerateUniqueID("New Prompt", idExists)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "new-prompt-3" {
		t.Errorf("expected 'new-prompt-3', got '%s'", result)
	}
}

func TestGenerateUniqueID_EmptySlug(t *testing.T) {
	// Name like "!!!" produces empty slug, should use fallback
	idExists := func(id string) bool {
		return false
	}

	result, err := GenerateUniqueID("!!!", idExists)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != DefaultFallbackPrefix {
		t.Errorf("expected '%s', got '%s'", DefaultFallbackPrefix, result)
	}
}

func TestGenerateUniqueID_EmptySlugWithConflict(t *testing.T) {
	// Empty slug with existing "prompt", should return "prompt-1"
	idExists := func(id string) bool {
		return id == "prompt"
	}

	result, err := GenerateUniqueID("!!!", idExists)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "prompt-1" {
		t.Errorf("expected 'prompt-1', got '%s'", result)
	}
}

func TestGenerateUniqueID_MaxAttemptsExceeded(t *testing.T) {
	// All possible IDs exist (base + 1 through MaxIDSuffixAttempts)
	idExists := func(id string) bool {
		return true // Everything exists
	}

	_, err := GenerateUniqueID("Test", idExists)
	if err == nil {
		t.Error("expected error when max attempts exceeded, got nil")
	}
}

func TestGenerateUniqueID_NameWithNumber(t *testing.T) {
	// Name "New Prompt 1" slugifies to "new-prompt-1"
	// If "new-prompt-1" exists, should return "new-prompt-1-1"
	idExists := func(id string) bool {
		return id == "new-prompt-1"
	}

	result, err := GenerateUniqueID("New Prompt 1", idExists)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "new-prompt-1-1" {
		t.Errorf("expected 'new-prompt-1-1', got '%s'", result)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"New Prompt", "new-prompt"},
		{"hello world", "hello-world"},
		{"Hello_World", "hello-world"},
		{"Test  Prompt", "test-prompt"},
		{"!!!", ""},
		{"Test@#$%Prompt", "testprompt"},
		{"  Spaces  ", "spaces"},
		{"Already-Slugified", "already-slugified"},
		{"123 Numbers", "123-numbers"},
	}

	for _, tc := range tests {
		result := Slugify(tc.input)
		if result != tc.expected {
			t.Errorf("Slugify(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}
