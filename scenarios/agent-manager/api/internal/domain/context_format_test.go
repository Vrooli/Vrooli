package domain

import (
	"strings"
	"testing"
)

func TestFormatContextForPrompt_Empty(t *testing.T) {
	result := FormatContextForPrompt(nil)
	if result != "" {
		t.Errorf("expected empty string for nil attachments, got %q", result)
	}

	result = FormatContextForPrompt([]ContextAttachment{})
	if result != "" {
		t.Errorf("expected empty string for empty attachments, got %q", result)
	}
}

func TestFormatContextForPrompt_FileWithKeyAndTags(t *testing.T) {
	attachments := []ContextAttachment{
		{
			Type: "file",
			Key:  "error-logs",
			Tags: []string{"logs", "debug"},
			Path: "/var/log/app.log",
		},
	}

	result := FormatContextForPrompt(attachments)

	expectedContains := []string{
		`<context`,
		`key="error-logs"`,
		`type="file"`,
		`tags="logs,debug"`,
		`path="/var/log/app.log"`,
		`</context>`,
	}

	for _, expected := range expectedContains {
		if !strings.Contains(result, expected) {
			t.Errorf("expected result to contain %q, got:\n%s", expected, result)
		}
	}
}

func TestFormatContextForPrompt_NoteWithContent(t *testing.T) {
	attachments := []ContextAttachment{
		{
			Type:    "note",
			Key:     "deployment-notes",
			Content: "Deploy to staging first",
		},
	}

	result := FormatContextForPrompt(attachments)

	expectedContains := []string{
		`key="deployment-notes"`,
		`type="note"`,
		`Deploy to staging first`,
	}

	for _, expected := range expectedContains {
		if !strings.Contains(result, expected) {
			t.Errorf("expected result to contain %q, got:\n%s", expected, result)
		}
	}
}

func TestFormatContextForPrompt_LinkWithURL(t *testing.T) {
	attachments := []ContextAttachment{
		{
			Type:    "link",
			Key:     "docs",
			URL:     "https://example.com/docs",
			Content: "Reference documentation",
			Label:   "API Docs",
		},
	}

	result := FormatContextForPrompt(attachments)

	expectedContains := []string{
		`key="docs"`,
		`type="link"`,
		`url="https://example.com/docs"`,
		`label="API Docs"`,
		`Reference documentation`,
	}

	for _, expected := range expectedContains {
		if !strings.Contains(result, expected) {
			t.Errorf("expected result to contain %q, got:\n%s", expected, result)
		}
	}
}

func TestFormatContextForPrompt_MultipleAttachments(t *testing.T) {
	attachments := []ContextAttachment{
		{
			Type: "file",
			Key:  "logs",
			Path: "/var/log/app.log",
		},
		{
			Type:    "note",
			Key:     "instructions",
			Content: "Check error messages",
		},
	}

	result := FormatContextForPrompt(attachments)

	// Should contain both attachments
	if !strings.Contains(result, `key="logs"`) {
		t.Errorf("expected result to contain first attachment key")
	}
	if !strings.Contains(result, `key="instructions"`) {
		t.Errorf("expected result to contain second attachment key")
	}

	// Should have two </context> closing tags
	count := strings.Count(result, "</context>")
	if count != 2 {
		t.Errorf("expected 2 closing context tags, got %d", count)
	}
}

func TestFormatContextForPrompt_XMLEscaping(t *testing.T) {
	attachments := []ContextAttachment{
		{
			Type:    "note",
			Key:     "special-chars",
			Label:   `Test "quotes" & <brackets>`,
			Content: "Normal content",
		},
	}

	result := FormatContextForPrompt(attachments)

	// Should escape XML special characters in attributes
	if strings.Contains(result, `"quotes"`) {
		t.Errorf("quotes should be escaped in attributes")
	}
	if strings.Contains(result, `& `) && !strings.Contains(result, `&amp;`) {
		t.Errorf("ampersands should be escaped in attributes")
	}
	if strings.Contains(result, `<brackets>`) {
		t.Errorf("angle brackets should be escaped in attributes")
	}
}

func TestBuildPromptWithContext_EmptyAttachments(t *testing.T) {
	basePrompt := "Fix the authentication bug."

	result := BuildPromptWithContext(basePrompt, nil)
	if result != basePrompt {
		t.Errorf("expected unchanged prompt for nil attachments, got %q", result)
	}

	result = BuildPromptWithContext(basePrompt, []ContextAttachment{})
	if result != basePrompt {
		t.Errorf("expected unchanged prompt for empty attachments, got %q", result)
	}
}

func TestBuildPromptWithContext_WithAttachments(t *testing.T) {
	basePrompt := "Fix the authentication bug in the login flow."
	attachments := []ContextAttachment{
		{
			Type:    "note",
			Key:     "requirements",
			Content: "Must support OAuth2",
		},
	}

	result := BuildPromptWithContext(basePrompt, attachments)

	// Should start with base prompt
	if !strings.HasPrefix(result, basePrompt) {
		t.Errorf("result should start with base prompt, got:\n%s", result)
	}

	// Should contain context attachment
	if !strings.Contains(result, `key="requirements"`) {
		t.Errorf("result should contain context attachment key")
	}

	if !strings.Contains(result, "Must support OAuth2") {
		t.Errorf("result should contain context attachment content")
	}
}

func TestFormatContextForPrompt_NoKeyOrTags(t *testing.T) {
	attachments := []ContextAttachment{
		{
			Type:    "note",
			Content: "Simple note without key or tags",
		},
	}

	result := FormatContextForPrompt(attachments)

	// Should have type but not key or tags attributes
	if !strings.Contains(result, `type="note"`) {
		t.Errorf("expected type attribute")
	}
	if strings.Contains(result, `key="`) {
		t.Errorf("should not have key attribute when key is empty")
	}
	if strings.Contains(result, `tags="`) {
		t.Errorf("should not have tags attribute when tags is empty")
	}
}

func TestFormatContextForPrompt_FileWithoutContent(t *testing.T) {
	attachments := []ContextAttachment{
		{
			Type: "file",
			Path: "/path/to/file.txt",
		},
	}

	result := FormatContextForPrompt(attachments)

	// Should have placeholder content
	if !strings.Contains(result, "[File:") {
		t.Errorf("expected file placeholder when content is empty")
	}
	if !strings.Contains(result, "/path/to/file.txt") {
		t.Errorf("expected file path in placeholder")
	}
}

func TestFormatContextForPrompt_LinkWithoutContent(t *testing.T) {
	attachments := []ContextAttachment{
		{
			Type: "link",
			URL:  "https://example.com",
		},
	}

	result := FormatContextForPrompt(attachments)

	// Should have placeholder content
	if !strings.Contains(result, "[Link:") {
		t.Errorf("expected link placeholder when content is empty")
	}
	if !strings.Contains(result, "https://example.com") {
		t.Errorf("expected URL in placeholder")
	}
}

// =============================================================================
// BuildSplitPrompt Tests
// =============================================================================

func TestBuildSplitPrompt_NoAttachments(t *testing.T) {
	// Without attachments, falls back to legacy behavior:
	// everything in user message, system prompt empty.
	sys, user := BuildSplitPrompt("do the thing", nil, "")
	if sys != "" {
		t.Errorf("expected empty system prompt, got %q", sys)
	}
	if user != "do the thing" {
		t.Errorf("expected user message = task description, got %q", user)
	}
}

func TestBuildSplitPrompt_NoAttachments_WithOverride(t *testing.T) {
	sys, user := BuildSplitPrompt("original", nil, "override")
	if sys != "" {
		t.Errorf("expected empty system prompt, got %q", sys)
	}
	if user != "override" {
		t.Errorf("expected user message = override prompt, got %q", user)
	}
}

func TestBuildSplitPrompt_WithAttachments(t *testing.T) {
	attachments := []ContextAttachment{
		{
			Type:     "note",
			Key:      "run-data",
			Content:  "the run failed at step 3",
			Format:   "text",
			Priority: "high",
		},
	}

	sys, user := BuildSplitPrompt("investigate this run", attachments, "")

	if sys != "investigate this run" {
		t.Errorf("expected system prompt = task description, got %q", sys)
	}
	if !strings.Contains(user, "the run failed at step 3") {
		t.Errorf("expected user message to contain attachment content, got %q", user)
	}
	if !strings.Contains(user, `key="run-data"`) {
		t.Errorf("expected user message to contain context tags, got %q", user)
	}
	// System prompt should NOT contain attachment data
	if strings.Contains(sys, "run-data") {
		t.Errorf("system prompt should not contain attachment data")
	}
}

func TestBuildSplitPrompt_WithAttachments_OverrideBecomesSystem(t *testing.T) {
	attachments := []ContextAttachment{
		{
			Type:    "note",
			Key:     "ctx",
			Content: "some context",
			Format:  "text",
		},
	}

	sys, user := BuildSplitPrompt("original task", attachments, "custom override")

	if sys != "custom override" {
		t.Errorf("expected system prompt = override, got %q", sys)
	}
	if !strings.Contains(user, "some context") {
		t.Errorf("expected user message to contain context, got %q", user)
	}
}

func TestBuildSplitPrompt_AttachmentPriorityOrder(t *testing.T) {
	attachments := []ContextAttachment{
		{
			Type:     "note",
			Key:      "low-pri",
			Content:  "low priority data",
			Format:   "text",
			Priority: "low",
		},
		{
			Type:     "note",
			Key:      "high-pri",
			Content:  "high priority data",
			Format:   "text",
			Priority: "high",
		},
	}

	_, user := BuildSplitPrompt("instructions", attachments, "")

	highIdx := strings.Index(user, "high priority data")
	lowIdx := strings.Index(user, "low priority data")
	if highIdx < 0 || lowIdx < 0 {
		t.Fatalf("expected both attachments in user message, got %q", user)
	}
	if highIdx > lowIdx {
		t.Errorf("expected high-priority attachment before low-priority, high=%d low=%d", highIdx, lowIdx)
	}
}
