package skills

import (
	"testing"
)

func TestGenerateUniqueID_NoConflict(t *testing.T) {
	// No existing IDs - should return base ID
	idExists := func(id string) bool {
		return false
	}

	result, err := GenerateUniqueID("New Skill", idExists)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "new-skill" {
		t.Errorf("expected 'new-skill', got '%s'", result)
	}
}

func TestGenerateUniqueID_SingleConflict(t *testing.T) {
	// "new-skill" exists, so should return "new-skill-1"
	idExists := func(id string) bool {
		return id == "new-skill"
	}

	result, err := GenerateUniqueID("New Skill", idExists)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "new-skill-1" {
		t.Errorf("expected 'new-skill-1', got '%s'", result)
	}
}

func TestGenerateUniqueID_MultipleConflicts(t *testing.T) {
	// "new-skill", "new-skill-1", "new-skill-2" exist
	existing := map[string]bool{
		"new-skill":   true,
		"new-skill-1": true,
		"new-skill-2": true,
	}
	idExists := func(id string) bool {
		return existing[id]
	}

	result, err := GenerateUniqueID("New Skill", idExists)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "new-skill-3" {
		t.Errorf("expected 'new-skill-3', got '%s'", result)
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
	// Empty slug with existing "skill", should return "skill-1"
	idExists := func(id string) bool {
		return id == "skill"
	}

	result, err := GenerateUniqueID("!!!", idExists)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "skill-1" {
		t.Errorf("expected 'skill-1', got '%s'", result)
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
	// Name "New Skill 1" slugifies to "new-skill-1"
	// If "new-skill-1" exists, should return "new-skill-1-1"
	idExists := func(id string) bool {
		return id == "new-skill-1"
	}

	result, err := GenerateUniqueID("New Skill 1", idExists)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "new-skill-1-1" {
		t.Errorf("expected 'new-skill-1-1', got '%s'", result)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"New Skill", "new-skill"},
		{"hello world", "hello-world"},
		{"Hello_World", "hello-world"},
		{"Test  Skill", "test-skill"},
		{"!!!", ""},
		{"Test@#$%Skill", "testskill"},
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
