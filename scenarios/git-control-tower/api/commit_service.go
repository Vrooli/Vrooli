package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// CommitDeps contains dependencies for commit operations.
type CommitDeps struct {
	Git     GitRunner
	RepoDir string
}

// CreateCommit creates a new git commit with the given message.
// [REQ:GCT-OT-P0-005] Commit composition API
func CreateCommit(ctx context.Context, deps CommitDeps, req CommitRequest) (*CommitResponse, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		return &CommitResponse{
			Success:          false,
			ValidationErrors: []string{"commit message is required"},
			Timestamp:        time.Now().UTC(),
		}, nil
	}

	// DECISION BOUNDARY: Validate conventional commit format if requested
	if req.ValidateConventional {
		validationErrors := ValidateConventionalCommit(message)
		if len(validationErrors) > 0 {
			return &CommitResponse{
				Success:          false,
				ValidationErrors: validationErrors,
				Timestamp:        time.Now().UTC(),
			}, nil
		}
	}

	// Attempt to create the commit
	hash, err := deps.Git.Commit(ctx, repoDir, message)
	if err != nil {
		return &CommitResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now().UTC(),
		}, nil
	}

	return &CommitResponse{
		Success:   true,
		Hash:      hash,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}, nil
}

// conventionalCommitRegex matches the conventional commit format:
// type(scope): description  OR  type: description
// Examples:
//
//	feat(auth): add login endpoint
//	fix: resolve null pointer
//	docs(readme): update installation steps
var conventionalCommitRegex = regexp.MustCompile(`^([a-z]+)(\([a-z0-9-]+\))?!?: .+$`)

// ValidateConventionalCommit checks if a message follows conventional commit format.
// Returns a list of validation errors, or empty slice if valid.
//
// DECISION BOUNDARY: This defines what constitutes a valid conventional commit.
// Based on https://www.conventionalcommits.org/en/v1.0.0/
func ValidateConventionalCommit(message string) []string {
	var errors []string

	// Check basic format
	if !conventionalCommitRegex.MatchString(message) {
		errors = append(errors, "message must match format: type(scope): description or type: description")
		return errors
	}

	// Extract the type (everything before the first ( or :)
	typePart := message
	if idx := strings.Index(message, "("); idx != -1 {
		typePart = message[:idx]
	} else if idx := strings.Index(message, ":"); idx != -1 {
		typePart = message[:idx]
	}
	// Remove trailing ! for breaking changes
	typePart = strings.TrimSuffix(typePart, "!")

	// DECISION BOUNDARY: Check if type is valid
	validType := false
	for _, t := range ConventionalCommitTypes {
		if typePart == t {
			validType = true
			break
		}
	}
	if !validType {
		errors = append(errors, fmt.Sprintf("invalid type %q; valid types are: %s",
			typePart, strings.Join(ConventionalCommitTypes, ", ")))
	}

	return errors
}
