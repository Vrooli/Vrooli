package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

// CommitDeps contains dependencies for commit operations.
type CommitDeps struct {
	Git       GitRunner
	RepoDir   string
	Precommit *PrecommitService
	Checks    CommitCheckRecorder
}

func commitValidationFailure(errors []string) *CommitResponse {
	return &CommitResponse{
		Success:          false,
		ValidationErrors: errors,
		Timestamp:        time.Now().UTC(),
	}
}

func validateAmendSafety(ctx context.Context, deps CommitDeps, repoDir string) *CommitResponse {
	status, err := GetRepoStatus(ctx, RepoStatusDeps{
		Git:     deps.Git,
		RepoDir: repoDir,
	})
	if err != nil {
		return nil // caller should return the error via a different path
	}
	if strings.TrimSpace(status.Branch.Upstream) == "" {
		return commitValidationFailure([]string{"upstream not configured; set upstream before amending"})
	}
	if status.Branch.Ahead <= 0 {
		return commitValidationFailure([]string{"last commit already on upstream; amend is blocked"})
	}
	return nil
}

func resolveCommitMessage(ctx context.Context, deps CommitDeps, repoDir string, message string, noEdit bool) string {
	if !noEdit {
		return message
	}
	if lastMessage, err := deps.Git.LastCommitMessage(ctx, repoDir); err == nil {
		lastMessage = strings.TrimSpace(lastMessage)
		if lastMessage != "" {
			return lastMessage
		}
	}
	return message
}

// validateCommitRequest validates the commit message and amend safety.
func validateCommitRequest(ctx context.Context, deps CommitDeps, repoDir string, req CommitRequest, message string) *CommitResponse {
	if message == "" && !req.Amend {
		return commitValidationFailure([]string{"commit message is required"})
	}
	if req.Amend {
		if resp := validateAmendSafety(ctx, deps, repoDir); resp != nil {
			return resp
		}
	}
	if req.ValidateConventional && message != "" {
		if errs := ValidateConventionalCommit(message); len(errs) > 0 {
			return commitValidationFailure(errs)
		}
	}
	return nil
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
	if resp := validateCommitRequest(ctx, deps, repoDir, req, message); resp != nil {
		return resp, nil
	}
	var passedPrecommit *PrecommitRunResult
	if deps.Precommit != nil && !req.SkipPrecommitOnce {
		result, ran, err := deps.Precommit.RunBeforeCommit(ctx, repoDir)
		if err != nil {
			return nil, err
		}
		if ran && result.Status != "passed" {
			return &CommitResponse{
				Success:   false,
				Error:     "precommit failed",
				Precommit: &result,
				Timestamp: time.Now().UTC(),
			}, nil
		}
		if ran {
			passedPrecommit = &result
		}
	}

	noEdit := req.Amend && message == ""
	hash, err := deps.Git.Commit(ctx, repoDir, message, CommitOptions{
		AuthorName:  strings.TrimSpace(req.AuthorName),
		AuthorEmail: strings.TrimSpace(req.AuthorEmail),
		Amend:       req.Amend,
		NoEdit:      noEdit,
		NoVerify:    req.SkipPrecommitOnce,
	})
	if err != nil {
		return &CommitResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now().UTC(),
		}, nil
	}
	if deps.Checks != nil && passedPrecommit != nil {
		if err := deps.Checks.Save(context.Background(), repoDir, hash, commitCheckFromPrecommit(*passedPrecommit)); err != nil {
			log.Printf("save commit check run failed: %v", err)
		}
	}

	return &CommitResponse{
		Success:   true,
		Hash:      hash,
		Message:   resolveCommitMessage(ctx, deps, repoDir, message, noEdit),
		Amended:   req.Amend,
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
