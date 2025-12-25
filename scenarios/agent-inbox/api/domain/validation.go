// Package domain defines the core domain types for the Agent Inbox scenario.
// This file contains validation logic for domain types.
package domain

// ValidationResult represents the outcome of a validation check.
// Using a structured result makes validation decisions explicit and testable.
type ValidationResult struct {
	Valid   bool
	Message string
}

// OK returns a successful validation result.
func OK() ValidationResult {
	return ValidationResult{Valid: true}
}

// Invalid returns a failed validation result with the given message.
func Invalid(message string) ValidationResult {
	return ValidationResult{Valid: false, Message: message}
}

// Message Validation

// ValidateMessageInput validates the inputs for creating a message.
// Decision boundary: Is this message input valid?
//
// Rules:
//   - Role and content are required
//   - Role must be a valid message role
//   - Tool messages must have a tool_call_id
func ValidateMessageInput(role, content, toolCallID string) ValidationResult {
	if role == "" {
		return Invalid("role is required")
	}
	if content == "" {
		return Invalid("content is required")
	}
	if !IsValidRole(role) {
		return Invalid("role must be 'user', 'assistant', 'system', or 'tool'")
	}
	if role == RoleTool && toolCallID == "" {
		return Invalid("tool_call_id is required for tool messages")
	}
	return OK()
}

// Chat Validation

// ValidateChatCreate validates inputs for creating a chat.
func ValidateChatCreate(name, model, viewMode string) ValidationResult {
	if viewMode != "" && !IsValidViewMode(viewMode) {
		return Invalid("view_mode must be 'bubble' or 'terminal'")
	}
	return OK()
}

// ValidateChatUpdate validates inputs for updating a chat.
func ValidateChatUpdate(name, model *string) ValidationResult {
	if name == nil && model == nil {
		return Invalid("no fields to update")
	}
	return OK()
}

// Label Validation

// ValidateLabelCreate validates inputs for creating a label.
func ValidateLabelCreate(name, color string) ValidationResult {
	if name == "" {
		return Invalid("name is required")
	}
	if color == "" {
		return Invalid("color is required")
	}
	return OK()
}

// ValidateLabelUpdate validates inputs for updating a label.
func ValidateLabelUpdate(name, color *string) ValidationResult {
	if name == nil && color == nil {
		return Invalid("no fields to update")
	}
	return OK()
}
