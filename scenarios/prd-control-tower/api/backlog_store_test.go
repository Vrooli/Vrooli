package main

import (
	"testing"
)

// [REQ:PCT-BACKLOG-INTAKE] Test nullIfEmpty returns nil for empty strings
func TestNullIfEmpty(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
	}{
		{
			name:    "empty string",
			input:   "",
			wantNil: true,
		},
		{
			name:    "only spaces",
			input:   "   ",
			wantNil: true,
		},
		{
			name:    "only tabs",
			input:   "\t\t",
			wantNil: true,
		},
		{
			name:    "mixed whitespace",
			input:   " \t \n ",
			wantNil: true,
		},
		{
			name:    "non-empty string",
			input:   "hello",
			wantNil: false,
		},
		{
			name:    "string with surrounding spaces",
			input:   "  hello  ",
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullIfEmpty(tt.input)
			if tt.wantNil {
				if result != nil {
					t.Errorf("nullIfEmpty(%q) = %v, want nil", tt.input, result)
				}
			} else {
				if result == nil {
					t.Errorf("nullIfEmpty(%q) = nil, want non-nil", tt.input)
				}
			}
		})
	}
}

// [REQ:PCT-BACKLOG-INTAKE] Test generateSlug truncates long input
func TestGenerateSlug_LongInput(t *testing.T) {
	longInput := "this is a very long input that should be truncated to a maximum of eighty characters in the slug output so we dont have overly long file names"
	result := generateSlug(longInput)
	if len(result) > 80 {
		t.Errorf("generateSlug() result length = %d, want <= 80", len(result))
	}
}

// [REQ:PCT-BACKLOG-INTAKE] Test buildBacklogDraftContent generates proper PRD template
func TestBuildBacklogDraftContent(t *testing.T) {
	ideaText := "Build a new inventory management system"
	entityName := "inventory-manager"

	content := buildBacklogDraftContent(ideaText, entityName)

	// Check for required PRD sections
	expectedSections := []string{
		"# Product Requirements Document (PRD)",
		"## 🎯 Overview",
		"## 🎯 Operational Targets",
		"### 🔴 P0 – Must ship for viability",
		"### 🟠 P1 – Should have post-launch",
		"### 🟢 P2 – Future / expansion",
		"## 🧱 Tech Direction Snapshot",
		"## 🤝 Dependencies & Launch Plan",
		"## 🎨 UX & Branding",
		"## 📎 Appendix",
		"inventory manager", // Display name from entityName
		ideaText,            // Original idea text should appear
	}

	for _, section := range expectedSections {
		if !contains(content, section) {
			t.Errorf("buildBacklogDraftContent() missing expected section: %q", section)
		}
	}

	// Check for operational target templates
	if !contains(content, "[T.001]") {
		t.Error("buildBacklogDraftContent() should include P0 target template")
	}
	if !contains(content, "[T.101]") {
		t.Error("buildBacklogDraftContent() should include P1 target template")
	}
	if !contains(content, "[T.201]") {
		t.Error("buildBacklogDraftContent() should include P2 target template")
	}
}

// [REQ:PCT-BACKLOG-INTAKE] Test buildBacklogDraftContent handles entity names with special chars
func TestBuildBacklogDraftContent_SpecialChars(t *testing.T) {
	ideaText := "Test idea"
	entityName := "my-super-cool-feature"

	content := buildBacklogDraftContent(ideaText, entityName)

	// Should convert hyphens/underscores to spaces for display name
	if !contains(content, "my super cool feature") {
		t.Error("buildBacklogDraftContent() should convert entity name to display format")
	}
}

// [REQ:PCT-BACKLOG-INTAKE] Test parseBacklogInput handles mixed type prefixes
func TestParseBacklogInput_MixedTypes(t *testing.T) {
	input := `[scenario] Build user dashboard
[resource] Install authentication service
Build another feature without prefix`

	results := parseBacklogInput(input, EntityTypeScenario)

	if len(results) != 3 {
		t.Fatalf("parseBacklogInput() returned %d entries, want 3", len(results))
	}

	if results[0].EntityType != EntityTypeScenario {
		t.Errorf("Entry 0 EntityType = %q, want %q", results[0].EntityType, EntityTypeScenario)
	}
	if results[0].IdeaText != "Build user dashboard" {
		t.Errorf("Entry 0 IdeaText = %q, want %q", results[0].IdeaText, "Build user dashboard")
	}

	if results[1].EntityType != EntityTypeResource {
		t.Errorf("Entry 1 EntityType = %q, want %q", results[1].EntityType, EntityTypeResource)
	}
	if results[1].IdeaText != "Install authentication service" {
		t.Errorf("Entry 1 IdeaText = %q, want %q", results[1].IdeaText, "Install authentication service")
	}

	if results[2].EntityType != EntityTypeScenario {
		t.Errorf("Entry 2 EntityType = %q, want %q (should use default)", results[2].EntityType, EntityTypeScenario)
	}
}
