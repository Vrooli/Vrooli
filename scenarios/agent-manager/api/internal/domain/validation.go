// Package domain defines the core domain entities for agent-manager.
//
// This file contains VALIDATION LOGIC for domain entities.
// All input validation is centralized here for consistency and testability.

package domain

import (
	"strings"
)

// =============================================================================
// AGENT PROFILE VALIDATION
// =============================================================================

// Validate checks if an AgentProfile is valid for creation/update.
// Returns nil if valid, or a ValidationError describing the problem.
func (p *AgentProfile) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return &ValidationError{
			Field:   "name",
			Message: "profile name is required",
		}
	}

	if !p.RunnerType.IsValid() {
		return &ValidationError{
			Field:   "runnerType",
			Message: "invalid runner type",
			Hint:    "valid types: claude-code, codex, opencode",
		}
	}

	if p.MaxTurns < 0 {
		return &ValidationError{
			Field:   "maxTurns",
			Message: "maxTurns cannot be negative",
		}
	}

	if p.Timeout < 0 {
		return &ValidationError{
			Field:   "timeout",
			Message: "timeout cannot be negative",
		}
	}

	return nil
}

// =============================================================================
// TASK VALIDATION
// =============================================================================

// Validate checks if a Task is valid for creation/update.
func (t *Task) Validate() error {
	if strings.TrimSpace(t.Title) == "" {
		return &ValidationError{
			Field:   "title",
			Message: "task title is required",
		}
	}

	if strings.TrimSpace(t.ScopePath) == "" {
		return &ValidationError{
			Field:   "scopePath",
			Message: "scope path is required",
			Hint:    "use '.' or '/' for repository root",
		}
	}

	// Validate scope path doesn't escape (basic check)
	if strings.Contains(t.ScopePath, "..") {
		return &ValidationError{
			Field:   "scopePath",
			Message: "scope path cannot contain '..'",
			Hint:    "path traversal is not allowed",
		}
	}

	return nil
}

// =============================================================================
// RUN VALIDATION
// =============================================================================

// ValidateForCreation checks if a Run has valid initial state.
func (r *Run) ValidateForCreation() error {
	if r.TaskID.String() == "00000000-0000-0000-0000-000000000000" {
		return &ValidationError{
			Field:   "taskId",
			Message: "task ID is required",
		}
	}

	if r.AgentProfileID.String() == "00000000-0000-0000-0000-000000000000" {
		return &ValidationError{
			Field:   "agentProfileId",
			Message: "agent profile ID is required",
		}
	}

	if r.RunMode != RunModeSandboxed && r.RunMode != RunModeInPlace {
		return &ValidationError{
			Field:   "runMode",
			Message: "invalid run mode",
			Hint:    "valid modes: sandboxed, in_place",
		}
	}

	return nil
}

// =============================================================================
// POLICY VALIDATION
// =============================================================================

// Validate checks if a Policy is valid for creation/update.
func (p *Policy) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return &ValidationError{
			Field:   "name",
			Message: "policy name is required",
		}
	}

	if p.Priority < 0 {
		return &ValidationError{
			Field:   "priority",
			Message: "priority cannot be negative",
		}
	}

	// Validate rules
	if err := p.Rules.Validate(); err != nil {
		return err
	}

	return nil
}

// Validate checks if PolicyRules are internally consistent.
func (r *PolicyRules) Validate() error {
	// Concurrency limits must be positive if set
	if r.MaxConcurrentRuns != nil && *r.MaxConcurrentRuns < 1 {
		return &ValidationError{
			Field:   "rules.maxConcurrentRuns",
			Message: "must be at least 1 if specified",
		}
	}

	if r.MaxConcurrentPerScope != nil && *r.MaxConcurrentPerScope < 1 {
		return &ValidationError{
			Field:   "rules.maxConcurrentPerScope",
			Message: "must be at least 1 if specified",
		}
	}

	// Resource limits must be positive if set
	if r.MaxFilesChanged != nil && *r.MaxFilesChanged < 1 {
		return &ValidationError{
			Field:   "rules.maxFilesChanged",
			Message: "must be at least 1 if specified",
		}
	}

	if r.MaxTotalSizeBytes != nil && *r.MaxTotalSizeBytes < 1 {
		return &ValidationError{
			Field:   "rules.maxTotalSizeBytes",
			Message: "must be at least 1 if specified",
		}
	}

	if r.MaxExecutionTimeMs != nil && *r.MaxExecutionTimeMs < 1000 {
		return &ValidationError{
			Field:   "rules.maxExecutionTimeMs",
			Message: "must be at least 1000ms (1 second) if specified",
		}
	}

	// Validate runner lists don't conflict
	if len(r.AllowedRunners) > 0 && len(r.DeniedRunners) > 0 {
		for _, allowed := range r.AllowedRunners {
			for _, denied := range r.DeniedRunners {
				if allowed == denied {
					return &ValidationError{
						Field:   "rules.allowedRunners",
						Message: "runner appears in both allowed and denied lists",
						Hint:    string(allowed) + " is in both lists",
					}
				}
			}
		}
	}

	return nil
}

// =============================================================================
// CONTEXT ATTACHMENT VALIDATION
// =============================================================================

// Validate checks if a ContextAttachment is valid.
func (c *ContextAttachment) Validate() error {
	validTypes := map[string]bool{"file": true, "link": true, "note": true}
	if !validTypes[c.Type] {
		return &ValidationError{
			Field:   "contextAttachment.type",
			Message: "invalid attachment type",
			Hint:    "valid types: file, link, note",
		}
	}

	switch c.Type {
	case "file":
		if strings.TrimSpace(c.Path) == "" {
			return &ValidationError{
				Field:   "contextAttachment.path",
				Message: "file attachment requires path",
			}
		}
	case "link":
		if strings.TrimSpace(c.URL) == "" {
			return &ValidationError{
				Field:   "contextAttachment.url",
				Message: "link attachment requires url",
			}
		}
	case "note":
		if strings.TrimSpace(c.Content) == "" {
			return &ValidationError{
				Field:   "contextAttachment.content",
				Message: "note attachment requires content",
			}
		}
	}

	return nil
}
