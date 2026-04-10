// Package domain defines the core domain types for the Agent Inbox scenario.
// This file contains error constructors for chat and message operations.
package domain

import "fmt"

// ErrChatNotFound creates a not-found error for a missing chat.
func ErrChatNotFound(chatID string) *AppError {
	return NewError(ErrCodeChatNotFound, CategoryNotFound,
		"chat not found", ActionVerifyResource).
		WithDetail("chat_id", chatID)
}

// ErrLabelNotFound creates a not-found error for a missing label.
func ErrLabelNotFound(labelID string) *AppError {
	return NewError(ErrCodeLabelNotFound, CategoryNotFound,
		"label not found", ActionVerifyResource).
		WithDetail("label_id", labelID)
}

// ErrNoMessagesInChat creates a validation error when a chat has no messages.
func ErrNoMessagesInChat(chatID string) *AppError {
	return NewError(ErrCodeNoMessagesInChat, CategoryValidation,
		"no messages in chat to process", ActionCorrectInput).
		WithDetail("chat_id", chatID)
}

// Agent mode error constructors
//
// These provide specific, recovery-distinct errors for agent mode operations.
// Each maps to a unique code so the UI can show targeted recovery guidance.

// ErrAgentNotInMode creates a validation error when the chat is not in agent mode.
func ErrAgentNotInMode(chatID string) *AppError {
	return NewError(ErrCodeAgentNotInMode, CategoryValidation,
		"chat is not in agent mode", ActionCorrectInput).
		WithDetail("chat_id", chatID).
		WithDetail("user_message", "This chat is in LLM mode. Switch to agent mode to use this feature.")
}

// ErrAgentNoActiveRun creates a validation error when there is no active agent run.
func ErrAgentNoActiveRun(chatID string) *AppError {
	return NewError(ErrCodeAgentNoActiveRun, CategoryValidation,
		"no active agent run", ActionCorrectInput).
		WithDetail("chat_id", chatID).
		WithDetail("user_message", "No agent is currently running. Start a new agent session first.")
}

// ErrAgentAlreadyActive creates a conflict error when agent mode is already active.
func ErrAgentAlreadyActive(chatID string) *AppError {
	return NewError(ErrCodeAgentAlreadyActive, CategoryConflict,
		"chat already has an active agent run", ActionCorrectInput).
		WithDetail("chat_id", chatID).
		WithDetail("user_message", "An agent is already running in this chat. Stop it first or send a follow-up message.")
}

// ErrAgentRunBusy creates a conflict error when attempting to send a message while the agent run is still active.
func ErrAgentRunBusy(chatID string) *AppError {
	return NewError(ErrCodeAgentRunBusy, CategoryConflict,
		"agent run is still in progress", ActionRetryWithBackoff).
		WithDetail("chat_id", chatID).
		WithDetail("user_message", "The agent is still working. Please wait for it to finish before sending another message.")
}

// ErrAgentManagerUnavailable creates a dependency error when agent-manager is not reachable.
func ErrAgentManagerUnavailable() *AppError {
	return NewError(ErrCodeAgentManagerUnavailable, CategoryDependency,
		"agent-manager service is not available", ActionCheckDependency).
		WithDetail("service", "agent-manager").
		WithDetail("user_message", "The agent-manager service is not running. Please start it with: vrooli scenario start agent-manager")
}

// ErrAgentRunNotFound creates a dependency error when a run ID is not found in agent-manager.
func ErrAgentRunNotFound(runID string) *AppError {
	return NewError(ErrCodeAgentRunNotFound, CategoryDependency,
		"agent run not found in agent-manager", ActionCorrectInput).
		WithDetail("run_id", runID).
		WithDetail("user_message", "The agent run could not be found. It may have expired. Try starting a new agent session.")
}

// ErrAgentProtoParseFailed creates a dependency error when proto response parsing fails.
func ErrAgentProtoParseFailed(operation string, err error) *AppError {
	return NewError(ErrCodeAgentProtoParseFailed, CategoryDependency,
		fmt.Sprintf("failed to parse agent-manager response for %s", operation), ActionEscalate).
		WithCause(err).WithDetail("operation", operation)
}
