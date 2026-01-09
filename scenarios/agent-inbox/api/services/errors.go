// Package services contains business logic orchestration.
// This file defines sentinel errors for the service layer.
package services

import "errors"

// Sentinel errors for completion service operations.
// These enable callers to use errors.Is() for type-safe error handling.
var (
	// ErrChatNotFound indicates the requested chat does not exist.
	ErrChatNotFound = errors.New("chat not found")

	// ErrNoMessages indicates a chat has no messages for completion.
	ErrNoMessages = errors.New("no messages in chat")

	// ErrDatabaseError indicates a database operation failed.
	ErrDatabaseError = errors.New("database error")

	// ErrMessagesFailed indicates failure to retrieve messages.
	ErrMessagesFailed = errors.New("failed to get messages")
)
