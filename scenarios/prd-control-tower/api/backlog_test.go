package main

import (
	"strings"
	"testing"
)

func TestParseBacklogInput(t *testing.T) {
	raw := `1. Scenario intake idea
- [Resource] shared cache idea
•  another scenario concept

[scenario] Already typed`
	entries := parseBacklogInput(raw, "scenario")
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	if entries[0].EntityType != EntityTypeScenario || entries[0].IdeaText != "Scenario intake idea" {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}

	if entries[1].EntityType != EntityTypeResource {
		t.Errorf("expected second entry to be resource, got %s", entries[1].EntityType)
	}

	if entries[2].SuggestedName == "" {
		t.Error("expected slug to be generated for third entry")
	}

	if entries[3].IdeaText != "Already typed" {
		t.Errorf("expected bracket prefix to be trimmed, got %q", entries[3].IdeaText)
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       string
		wantPrefix string
	}{
		{name: "simple", input: "Simple Name", want: "simple-name"},
		{name: "fallback", input: "!!!", wantPrefix: "scenario-"},
		{name: "collapse-spaces", input: "Multiple   Spaces", want: "multiple-spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateSlug(tt.input)
			if tt.wantPrefix != "" {
				if !strings.HasPrefix(got, tt.wantPrefix) {
					t.Fatalf("generateSlug(%q) = %q, expected prefix %q", tt.input, got, tt.wantPrefix)
				}
				return
			}

			if got != tt.want {
				t.Fatalf("generateSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStripLeadingNumber(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1. Test", "Test"},
		{"12) Value", "Value"},
		{"NoNumber", "NoNumber"},
	}

	for _, tt := range tests {
		if got := stripLeadingNumber(tt.input); got != tt.want {
			t.Errorf("stripLeadingNumber(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
