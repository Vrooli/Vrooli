package handlers

import (
	"agent-inbox/domain"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Helper Function Tests
// =============================================================================

// TestParseInt verifies the parseInt helper function.
func TestParseInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"zero", "0", 0, false},
		{"positive", "42", 42, false},
		{"negative", "-10", -10, false},
		{"large number", "1000000", 1000000, false},
		{"empty string", "", 0, true},
		{"invalid string", "abc", 0, true},
		{"float string", "3.14", 0, true},
		{"mixed", "123abc", 0, true},
		{"whitespace", " 5 ", 5, false}, // JSON unmarshal handles whitespace
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseInt(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("parseInt(%q) expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseInt(%q) unexpected error: %v", tc.input, err)
				return
			}
			if got != tc.want {
				t.Errorf("parseInt(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

// TestSanitizeFilename verifies the sanitizeFilename helper function.
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "test", "test"},
		{"with spaces", "my chat", "my chat"},
		{"with slash", "test/file", "test_file"},
		{"with backslash", "test\\file", "test_file"},
		{"with colon", "test:file", "test_file"},
		{"with asterisk", "test*file", "test_file"},
		{"with question", "test?file", "test_file"},
		{"with quotes", `test"file`, "test_file"},
		{"with less than", "test<file", "test_file"},
		{"with greater than", "test>file", "test_file"},
		{"with pipe", "test|file", "test_file"},
		{"multiple special", "a/b\\c:d*e?f\"g<h>i|j", "a_b_c_d_e_f_g_h_i_j"},
		{"empty", "", "chat"},
		{"whitespace only", "   ", "chat"},
		{"long name", strings.Repeat("a", 100), strings.Repeat("a", 50)},
		{"trimmed", "  test  ", "test"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeFilename(tc.input)
			if got != tc.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// =============================================================================
// Format Function Tests
// =============================================================================

// TestFormatMarkdown verifies the formatMarkdown function.
func TestFormatMarkdown(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	chat := &domain.Chat{
		ID:        "chat-1",
		Name:      "Test Chat",
		Model:     "gpt-4",
		CreatedAt: now,
		UpdatedAt: now,
	}

	messages := []domain.Message{
		{
			ID:        "msg-1",
			Role:      domain.RoleUser,
			Content:   "Hello",
			CreatedAt: now,
		},
		{
			ID:        "msg-2",
			Role:      domain.RoleAssistant,
			Content:   "Hi there!",
			Model:     "gpt-4",
			CreatedAt: now.Add(time.Second),
		},
	}

	result := formatMarkdown(chat, messages)

	// Check title
	if !strings.Contains(result, "# Test Chat") {
		t.Error("expected markdown to contain '# Test Chat'")
	}

	// Check model info
	if !strings.Contains(result, "**Model:** gpt-4") {
		t.Error("expected markdown to contain model info")
	}

	// Check user message header
	if !strings.Contains(result, "## User") {
		t.Error("expected markdown to contain '## User'")
	}

	// Check assistant message header with model
	if !strings.Contains(result, "## Assistant (gpt-4)") {
		t.Error("expected markdown to contain '## Assistant (gpt-4)'")
	}

	// Check content
	if !strings.Contains(result, "Hello") {
		t.Error("expected markdown to contain user message")
	}
	if !strings.Contains(result, "Hi there!") {
		t.Error("expected markdown to contain assistant message")
	}
}

// TestFormatMarkdown_WithToolCalls verifies tool calls are included.
func TestFormatMarkdown_WithToolCalls(t *testing.T) {
	now := time.Now()
	chat := &domain.Chat{
		ID:        "chat-1",
		Name:      "Tool Chat",
		Model:     "gpt-4",
		CreatedAt: now,
		UpdatedAt: now,
	}

	tc := domain.ToolCall{
		ID: "tc-1",
	}
	tc.Function.Name = "search"
	tc.Function.Arguments = `{"query": "test"}`

	messages := []domain.Message{
		{
			ID:        "msg-1",
			Role:      domain.RoleAssistant,
			Content:   "Let me search for that.",
			ToolCalls: []domain.ToolCall{tc},
			CreatedAt: now,
		},
		{
			ID:         "msg-2",
			Role:       domain.RoleTool,
			Content:    "Found 5 results",
			ToolCallID: "tc-1",
			CreatedAt:  now.Add(time.Second),
		},
	}

	result := formatMarkdown(chat, messages)

	// Check tool calls section
	if !strings.Contains(result, "**Tool Calls:**") {
		t.Error("expected markdown to contain '**Tool Calls:**'")
	}

	// Check tool name
	if !strings.Contains(result, "`search`") {
		t.Error("expected markdown to contain tool name")
	}

	// Check tool arguments
	if !strings.Contains(result, `{"query": "test"}`) {
		t.Error("expected markdown to contain tool arguments")
	}

	// Check tool response header
	if !strings.Contains(result, "## Tool Response") {
		t.Error("expected markdown to contain '## Tool Response'")
	}
}
