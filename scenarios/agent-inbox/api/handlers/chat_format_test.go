package handlers

import (
	"strings"
	"testing"
	"time"

	"agent-inbox/domain"
)

// TestFormatMarkdown_SystemMessage verifies system messages are handled.
func TestFormatMarkdown_SystemMessage(t *testing.T) {
	now := time.Now()
	chat := &domain.Chat{
		ID:        "chat-1",
		Name:      "System Chat",
		Model:     "gpt-4",
		CreatedAt: now,
		UpdatedAt: now,
	}

	messages := []domain.Message{
		{
			ID:        "msg-1",
			Role:      domain.RoleSystem,
			Content:   "You are a helpful assistant.",
			CreatedAt: now,
		},
	}

	result := formatMarkdown(chat, messages)

	if !strings.Contains(result, "## System") {
		t.Error("expected markdown to contain '## System'")
	}
	if !strings.Contains(result, "You are a helpful assistant.") {
		t.Error("expected markdown to contain system message content")
	}
}

// TestFormatPlainText verifies the formatPlainText function.
func TestFormatPlainText(t *testing.T) {
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
			CreatedAt: now.Add(time.Second),
		},
	}

	result := formatPlainText(chat, messages)

	// Check title
	if !strings.Contains(result, "Test Chat") {
		t.Error("expected plain text to contain 'Test Chat'")
	}

	// Check model info
	if !strings.Contains(result, "Model: gpt-4") {
		t.Error("expected plain text to contain model info")
	}

	// Check timestamps
	if !strings.Contains(result, "[10:30:00] User:") {
		t.Error("expected plain text to contain user timestamp")
	}
	if !strings.Contains(result, "[10:30:01] Assistant:") {
		t.Error("expected plain text to contain assistant timestamp")
	}

	// Check content
	if !strings.Contains(result, "Hello") {
		t.Error("expected plain text to contain user message")
	}
	if !strings.Contains(result, "Hi there!") {
		t.Error("expected plain text to contain assistant message")
	}
}

// TestFormatPlainText_AllRoles verifies all role types are handled.
func TestFormatPlainText_AllRoles(t *testing.T) {
	now := time.Now()
	chat := &domain.Chat{
		ID:        "chat-1",
		Name:      "All Roles",
		Model:     "gpt-4",
		CreatedAt: now,
		UpdatedAt: now,
	}

	messages := []domain.Message{
		{Role: domain.RoleSystem, Content: "System msg", CreatedAt: now},
		{Role: domain.RoleUser, Content: "User msg", CreatedAt: now},
		{Role: domain.RoleAssistant, Content: "Assistant msg", CreatedAt: now},
		{Role: domain.RoleTool, Content: "Tool msg", CreatedAt: now},
	}

	result := formatPlainText(chat, messages)

	if !strings.Contains(result, "System:") {
		t.Error("expected 'System:' in plain text")
	}
	if !strings.Contains(result, "User:") {
		t.Error("expected 'User:' in plain text")
	}
	if !strings.Contains(result, "Assistant:") {
		t.Error("expected 'Assistant:' in plain text")
	}
	if !strings.Contains(result, "Tool:") {
		t.Error("expected 'Tool:' in plain text")
	}
}

// TestFormatMarkdown_EmptyMessages verifies handling of empty message list.
func TestFormatMarkdown_EmptyMessages(t *testing.T) {
	now := time.Now()
	chat := &domain.Chat{
		ID:        "chat-1",
		Name:      "Empty Chat",
		Model:     "gpt-4",
		CreatedAt: now,
		UpdatedAt: now,
	}

	result := formatMarkdown(chat, []domain.Message{})

	// Should still have header
	if !strings.Contains(result, "# Empty Chat") {
		t.Error("expected markdown to contain chat name")
	}
	// Should have metadata
	if !strings.Contains(result, "**Model:** gpt-4") {
		t.Error("expected markdown to contain model info")
	}
}

// TestFormatPlainText_EmptyMessages verifies handling of empty message list.
func TestFormatPlainText_EmptyMessages(t *testing.T) {
	now := time.Now()
	chat := &domain.Chat{
		ID:        "chat-1",
		Name:      "Empty Chat",
		Model:     "gpt-4",
		CreatedAt: now,
		UpdatedAt: now,
	}

	result := formatPlainText(chat, []domain.Message{})

	// Should still have header
	if !strings.Contains(result, "Empty Chat") {
		t.Error("expected plain text to contain chat name")
	}
	// Should have metadata
	if !strings.Contains(result, "Model: gpt-4") {
		t.Error("expected plain text to contain model info")
	}
}
