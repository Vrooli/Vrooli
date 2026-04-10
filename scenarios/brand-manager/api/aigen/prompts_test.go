package aigen

import (
	"strings"
	"testing"
)

// TestColorPrompt verifies color prompt construction. [REQ:BM-REQ-AI-TEXT]
func TestColorPrompt(t *testing.T) {
	prompt := ColorPrompt("Acme", "A cool product", "Use blue tones")
	if !strings.Contains(prompt, "Acme") {
		t.Error("prompt should contain brand name")
	}
	if !strings.Contains(prompt, "A cool product") {
		t.Error("prompt should contain description")
	}
	if !strings.Contains(prompt, "Use blue tones") {
		t.Error("prompt should contain notes")
	}
	if !strings.Contains(prompt, "WCAG AA") {
		t.Error("prompt should mention WCAG AA contrast")
	}
	if !strings.Contains(prompt, "primary") {
		t.Error("prompt should mention primary color")
	}
}

// TestTypographyPrompt verifies typography prompt. [REQ:BM-REQ-AI-TEXT]
func TestTypographyPrompt(t *testing.T) {
	prompt := TypographyPrompt("Acme", "", "")
	if !strings.Contains(prompt, "Acme") {
		t.Error("prompt should contain brand name")
	}
	if !strings.Contains(prompt, "heading_font") {
		t.Error("prompt should mention heading_font")
	}
}

// TestVoicePrompt verifies voice prompt. [REQ:BM-REQ-AI-TEXT]
func TestVoicePrompt(t *testing.T) {
	prompt := VoicePrompt("Acme", "Widgets", "")
	if !strings.Contains(prompt, "tone") {
		t.Error("prompt should mention tone")
	}
	if !strings.Contains(prompt, "keywords") {
		t.Error("prompt should mention keywords")
	}
}

// TestLogoPrompt verifies logo prompt. [REQ:BM-REQ-AI-IMAGE]
func TestLogoPrompt(t *testing.T) {
	prompt := LogoPrompt("Acme", "SaaS platform", "#3366cc")
	if !strings.Contains(prompt, "Acme") {
		t.Error("prompt should contain brand name")
	}
	if !strings.Contains(prompt, "#3366cc") {
		t.Error("prompt should contain primary color")
	}
	if !strings.Contains(prompt, "logo") {
		t.Error("prompt should mention logo")
	}
}

// TestFaviconPrompt verifies favicon prompt. [REQ:BM-REQ-AI-IMAGE]
func TestFaviconPrompt(t *testing.T) {
	prompt := FaviconPrompt("Acme", "#ff0000")
	if !strings.Contains(prompt, "favicon") {
		t.Error("prompt should mention favicon")
	}
	if !strings.Contains(prompt, "#ff0000") {
		t.Error("prompt should contain color")
	}
}

// TestPromptEmptyOptionalFields verifies prompts handle empty fields. [REQ:BM-REQ-AI-TEXT]
func TestPromptEmptyOptionalFields(t *testing.T) {
	prompt := ColorPrompt("X", "", "")
	if strings.Contains(prompt, "Description:") {
		t.Error("should not include Description label when empty")
	}
	if strings.Contains(prompt, "Notes:") {
		t.Error("should not include Notes label when empty")
	}
}
