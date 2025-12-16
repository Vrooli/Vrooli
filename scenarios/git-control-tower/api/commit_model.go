package main

import "time"

// CommitRequest contains the parameters for creating a commit.
// [REQ:GCT-OT-P0-005] Commit composition API
type CommitRequest struct {
	// Message is the commit message (required).
	// Should follow conventional commit format if ValidateConventional is true.
	Message string `json:"message"`

	// ValidateConventional enables conventional commit message validation.
	// When true, the message must match the format: type(scope): description
	ValidateConventional bool `json:"validate_conventional,omitempty"`
}

// CommitResponse contains the result of a commit operation.
type CommitResponse struct {
	// Success indicates if the commit was created.
	Success bool `json:"success"`

	// Hash is the short commit hash (OID) of the new commit.
	Hash string `json:"hash,omitempty"`

	// Message is the commit message that was used.
	Message string `json:"message,omitempty"`

	// ValidationErrors contains any commit message validation errors.
	ValidationErrors []string `json:"validation_errors,omitempty"`

	// Error contains the error message if Success is false.
	Error string `json:"error,omitempty"`

	// Timestamp is when the operation completed.
	Timestamp time.Time `json:"timestamp"`
}

// ConventionalCommitTypes are the valid prefixes for conventional commits.
// Based on https://www.conventionalcommits.org/
var ConventionalCommitTypes = []string{
	"feat",     // New feature
	"fix",      // Bug fix
	"docs",     // Documentation only
	"style",    // Formatting, whitespace (no code change)
	"refactor", // Code change that neither fixes a bug nor adds a feature
	"perf",     // Performance improvement
	"test",     // Adding/updating tests
	"build",   // Changes to build system or dependencies
	"ci",       // CI configuration changes
	"chore",    // Other changes that don't modify src or test files
	"revert",   // Reverts a previous commit
}
