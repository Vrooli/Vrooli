package main

import (
	"strings"
	"testing"
)

// [REQ:PCT-FUNC-005][REQ:PCT-AI-GENERATE] AI generation - Test action prompt building
func TestBuildActionPrompt(t *testing.T) {
	tests := []struct {
		name          string
		action        string
		selectedText  string
		wantContains  []string
		wantNotEmpty  bool
		checkCallback func(string) error
	}{
		{
			name:         "improve action",
			action:       "improve",
			selectedText: "this text needs improvement",
			wantContains: []string{
				"this text needs improvement",
				"professional",
				"clear",
				"actionable",
				"IMPORTANT:",
			},
			wantNotEmpty: true,
		},
		{
			name:         "expand action",
			action:       "expand",
			selectedText: "brief text to expand",
			wantContains: []string{
				"brief text to expand",
				"detail",
				"examples",
				"context",
				"IMPORTANT:",
			},
			wantNotEmpty: true,
		},
		{
			name:         "simplify action",
			action:       "simplify",
			selectedText: "complex technical jargon to simplify",
			wantContains: []string{
				"complex technical jargon to simplify",
				"concise",
				"easier to understand",
				"jargon",
				"IMPORTANT:",
			},
			wantNotEmpty: true,
		},
		{
			name:         "grammar action",
			action:       "grammar",
			selectedText: "text with potental erors",
			wantContains: []string{
				"text with potental erors",
				"grammar",
				"spelling",
				"formatting",
				"IMPORTANT:",
			},
			wantNotEmpty: true,
		},
		{
			name:         "technical action",
			action:       "technical",
			selectedText: "make this more technical",
			wantContains: []string{
				"make this more technical",
				"technical",
				"precise",
				"IMPORTANT:",
			},
			wantNotEmpty: true,
		},
		{
			name:         "unknown action defaults to improve",
			action:       "unknown_action",
			selectedText: "some text",
			wantContains: []string{
				"some text",
				"professional",
				"IMPORTANT:",
			},
			wantNotEmpty: true,
		},
		{
			name:         "empty action defaults to improve",
			action:       "",
			selectedText: "some text",
			wantContains: []string{
				"some text",
				"professional",
				"IMPORTANT:",
			},
			wantNotEmpty: true,
		},
		{
			name:         "markdown preservation instruction in improve",
			action:       "improve",
			selectedText: "# Heading\n- List item",
			wantContains: []string{
				"markdown",
				"formatting",
			},
			wantNotEmpty: true,
		},
		{
			name:         "all actions include IMPORTANT directive",
			action:       "expand",
			selectedText: "test",
			wantContains: []string{
				"IMPORTANT:",
				"Return ONLY",
			},
			wantNotEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildActionPrompt(tt.action, tt.selectedText)

			if tt.wantNotEmpty && result == "" {
				t.Errorf("buildActionPrompt(%q, %q) returned empty string, want non-empty",
					tt.action, tt.selectedText)
			}

			if !tt.wantNotEmpty && result != "" {
				t.Errorf("buildActionPrompt(%q, %q) returned %q, want empty string",
					tt.action, tt.selectedText, result)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("buildActionPrompt(%q, %q) result missing %q\nGot: %s",
						tt.action, tt.selectedText, want, result)
				}
			}

			if tt.checkCallback != nil {
				if err := tt.checkCallback(result); err != nil {
					t.Errorf("buildActionPrompt(%q, %q) callback failed: %v",
						tt.action, tt.selectedText, err)
				}
			}
		})
	}
}

// [REQ:PCT-FUNC-005][REQ:PCT-AI-GENERATE] AI generation - Test all action types
func TestBuildActionPrompt_AllActionsSupported(t *testing.T) {
	supportedActions := []string{"improve", "expand", "simplify", "grammar", "technical"}
	selectedText := "Test text for verification"

	for _, action := range supportedActions {
		t.Run(action, func(t *testing.T) {
			result := buildActionPrompt(action, selectedText)
			if result == "" {
				t.Errorf("buildActionPrompt(%q, %q) returned empty, want non-empty prompt",
					action, selectedText)
			}
			if !strings.Contains(result, selectedText) {
				t.Errorf("buildActionPrompt(%q, %q) result doesn't include selected text",
					action, selectedText)
			}
			if !strings.Contains(result, "IMPORTANT:") {
				t.Errorf("buildActionPrompt(%q, %q) result missing IMPORTANT directive",
					action, selectedText)
			}
		})
	}
}

// [REQ:PCT-AI-GENERATE] Verify prompt format consistency
func TestBuildActionPrompt_FormatConsistency(t *testing.T) {
	actions := []string{"improve", "expand", "simplify", "grammar", "technical"}
	selectedText := "Sample text"

	for _, action := range actions {
		t.Run(action, func(t *testing.T) {
			result := buildActionPrompt(action, selectedText)

			// All prompts should include the selected text
			if !strings.Contains(result, selectedText) {
				t.Errorf("Action %q prompt missing selected text", action)
			}

			// All prompts should have Requirements section
			if !strings.Contains(result, "Requirements:") {
				t.Errorf("Action %q prompt missing Requirements section", action)
			}

			// All prompts should have IMPORTANT directive
			if !strings.Contains(result, "IMPORTANT:") {
				t.Errorf("Action %q prompt missing IMPORTANT directive", action)
			}

			// All prompts should mention markdown
			if !strings.Contains(result, "markdown") && action != "unknown" {
				t.Errorf("Action %q prompt should mention markdown formatting", action)
			}

			// All prompts should instruct to return only the result
			if !strings.Contains(result, "Return ONLY") && !strings.Contains(result, "Return only") {
				t.Errorf("Action %q prompt should instruct to return only the result", action)
			}
		})
	}
}

// [REQ:PCT-AI-GENERATE] Test special characters and markdown in selected text
func TestBuildActionPrompt_SpecialCharacters(t *testing.T) {
	specialTexts := []struct {
		name string
		text string
	}{
		{
			name: "markdown heading",
			text: "# Main Heading\n## Subheading",
		},
		{
			name: "code blocks",
			text: "```go\nfunc main() {}\n```",
		},
		{
			name: "special characters",
			text: "Text with <html> & special $chars!",
		},
		{
			name: "quotes",
			text: `"Quoted text" and 'single quotes'`,
		},
		{
			name: "newlines",
			text: "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range specialTexts {
		t.Run(tt.name, func(t *testing.T) {
			result := buildActionPrompt("improve", tt.text)
			if !strings.Contains(result, tt.text) {
				t.Errorf("buildActionPrompt should preserve special text: %q", tt.text)
			}
		})
	}
}
