// Package domain defines the core domain entities for agent-manager.
//
// This file contains VALIDATION LOGIC for domain entities.
// All input validation is centralized here for consistency and testability.
//
// DESIGN PRINCIPLES:
// - Validate at system boundaries (API handlers, before persistence)
// - Return structured ValidationErrors for consistent client handling
// - Make validation rules explicit and testable
// - Distinguish between creation validation and general validation
//
// INVARIANTS ENFORCED BY VALIDATION:
// - Names are non-empty and reasonably sized
// - IDs are valid UUIDs when required
// - Status/State values are within valid enums
// - Conflicting settings are detected (allow/deny overlap)
// - Numeric ranges are sensible (no negative timeouts, etc.)

package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// STATUS/STATE VALIDITY HELPERS
// =============================================================================
// These helpers ensure enum values are valid. They are used by validation
// logic and decision helpers throughout the codebase.

// IsValid returns whether this is a valid TaskStatus.
func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusQueued, TaskStatusRunning, TaskStatusNeedsReview,
		TaskStatusApproved, TaskStatusRejected, TaskStatusFailed, TaskStatusCancelled:
		return true
	default:
		return false
	}
}

// IsTerminal returns whether this is a terminal status (no further transitions allowed).
func (s TaskStatus) IsTerminal() bool {
	switch s {
	case TaskStatusApproved, TaskStatusRejected, TaskStatusFailed, TaskStatusCancelled:
		return true
	default:
		return false
	}
}

// IsValid returns whether this is a valid RunStatus.
func (s RunStatus) IsValid() bool {
	switch s {
	case RunStatusPending, RunStatusStarting, RunStatusRunning,
		RunStatusNeedsReview, RunStatusComplete, RunStatusFailed, RunStatusCancelled:
		return true
	default:
		return false
	}
}

// IsTerminal returns whether this is a terminal status (no further transitions allowed).
func (s RunStatus) IsTerminal() bool {
	switch s {
	case RunStatusComplete, RunStatusFailed, RunStatusCancelled:
		return true
	default:
		return false
	}
}

// IsActive returns whether this status indicates an actively processing run.
func (s RunStatus) IsActive() bool {
	switch s {
	case RunStatusPending, RunStatusStarting, RunStatusRunning:
		return true
	default:
		return false
	}
}

// IsValid returns whether this is a valid ApprovalState.
func (s ApprovalState) IsValid() bool {
	switch s {
	case ApprovalStateNone, ApprovalStatePending, ApprovalStatePartiallyApproved,
		ApprovalStateApproved, ApprovalStateRejected:
		return true
	default:
		return false
	}
}

// IsValid returns whether this is a valid RunPhase.
func (p RunPhase) IsValid() bool {
	switch p {
	case RunPhaseQueued, RunPhaseInitializing, RunPhaseSandboxCreating,
		RunPhaseRunnerAcquiring, RunPhaseExecuting, RunPhaseCollectingResults,
		RunPhaseAwaitingReview, RunPhaseApplying, RunPhaseCleaningUp, RunPhaseCompleted:
		return true
	default:
		return false
	}
}

// IsValid returns whether this is a valid RunEventType.
func (t RunEventType) IsValid() bool {
	switch t {
	case EventTypeLog, EventTypeMessage, EventTypeToolCall, EventTypeToolResult,
		EventTypeStatus, EventTypeMetric, EventTypeArtifact, EventTypeError:
		return true
	default:
		return false
	}
}

// IsValid returns whether this is a valid IdempotencyStatus.
func (s IdempotencyStatus) IsValid() bool {
	switch s {
	case IdempotencyStatusPending, IdempotencyStatusComplete, IdempotencyStatusFailed:
		return true
	default:
		return false
	}
}

// =============================================================================
// AGENT PROFILE VALIDATION
// =============================================================================

// Validate checks if an AgentProfile is valid for creation/update.
// Returns nil if valid, or a ValidationError describing the problem.
//
// INVARIANTS ENFORCED:
// - Name is required and ≤255 characters
// - RunnerType is one of the supported types
// - MaxTurns is non-negative (0 = unlimited)
// - Timeout is non-negative (0 = use default)
// - AllowedTools and DeniedTools don't overlap
// - AllowedPaths and DeniedPaths don't overlap
func (p *AgentProfile) Validate() error {
	// Name is required
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return NewValidationErrorWithHint("name", "field is required",
			"Provide a descriptive name for this agent profile")
	}
	if len(name) > 255 {
		return NewValidationError("name", "must be 255 characters or less")
	}

	// Description has a reasonable limit
	if len(p.Description) > 4096 {
		return NewValidationError("description", "must be 4096 characters or less")
	}

	// RunnerType must be valid
	if !p.RunnerType.IsValid() {
		return NewValidationErrorWithHint("runnerType", "invalid runner type",
			"valid types: claude-code, codex, opencode")
	}

	// MaxTurns must be non-negative
	if p.MaxTurns < 0 {
		return NewValidationError("maxTurns", "cannot be negative")
	}

	// Timeout must be non-negative
	if p.Timeout < 0 {
		return NewValidationError("timeout", "cannot be negative")
	}

	// AllowedTools and DeniedTools should not overlap
	if hasStringOverlap(p.AllowedTools, p.DeniedTools) {
		return NewValidationError("allowedTools/deniedTools",
			"same tool cannot be both allowed and denied")
	}

	// AllowedPaths and DeniedPaths should not overlap
	if hasStringOverlap(p.AllowedPaths, p.DeniedPaths) {
		return NewValidationError("allowedPaths/deniedPaths",
			"same path cannot be both allowed and denied")
	}

	return nil
}

// =============================================================================
// TASK VALIDATION
// =============================================================================

// Validate checks if a Task is valid for creation/update.
//
// INVARIANTS ENFORCED:
// - Title is required and ≤255 characters
// - ScopePath is valid (no path traversal)
// - ContextAttachments are individually valid
func (t *Task) Validate() error {
	// Title is required
	title := strings.TrimSpace(t.Title)
	if title == "" {
		return NewValidationErrorWithHint("title", "field is required",
			"Provide a descriptive title for this task")
	}
	if len(title) > 255 {
		return NewValidationError("title", "must be 255 characters or less")
	}

	// Description has a reasonable limit
	if len(t.Description) > 16384 {
		return NewValidationError("description", "must be 16384 characters or less")
	}

	// ScopePath is required
	scopePath := strings.TrimSpace(t.ScopePath)
	if scopePath == "" {
		return NewValidationErrorWithHint("scopePath", "field is required",
			"use '.' or '/' for repository root")
	}

	// Validate scope path doesn't escape (basic check)
	if strings.Contains(t.ScopePath, "..") {
		return NewValidationErrorWithHint("scopePath", "cannot contain '..'",
			"path traversal is not allowed")
	}

	// Validate context attachments
	for i, att := range t.ContextAttachments {
		if err := att.Validate(); err != nil {
			// Wrap with index for better error messages
			if ve, ok := err.(*ValidationError); ok {
				return &ValidationError{
					Field:   "contextAttachments[" + string(rune('0'+i)) + "]." + ve.Field,
					Message: ve.Message,
					Hint:    ve.Hint,
				}
			}
			return err
		}
	}

	return nil
}

// =============================================================================
// RUN VALIDATION
// =============================================================================

// Validate checks if a Run is valid. This is the general validation method
// called by handlers.
//
// INVARIANTS ENFORCED:
// - TaskID is a valid non-nil UUID
// - AgentProfileID is a valid non-nil UUID
// - RunMode is valid
// - Status is valid (if set)
// - Phase is valid (if set)
// - ProgressPercent is 0-100
func (r *Run) Validate() error {
	// TaskID is required
	if r.TaskID == uuid.Nil {
		return NewValidationError("taskId", "field is required")
	}

	// AgentProfileID is required
	if r.AgentProfileID == uuid.Nil {
		return NewValidationError("agentProfileId", "field is required")
	}

	// RunMode must be valid
	if r.RunMode != RunModeSandboxed && r.RunMode != RunModeInPlace {
		return NewValidationErrorWithHint("runMode", "invalid run mode",
			"valid modes: sandboxed, in_place")
	}

	// Status must be valid if set
	if r.Status != "" && !r.Status.IsValid() {
		return NewValidationError("status", "invalid status value")
	}

	// Phase must be valid if set
	if r.Phase != "" && !r.Phase.IsValid() {
		return NewValidationError("phase", "invalid phase value")
	}

	// ProgressPercent must be 0-100
	if r.ProgressPercent < 0 || r.ProgressPercent > 100 {
		return NewValidationError("progressPercent", "must be between 0 and 100")
	}

	// ApprovalState must be valid if set
	if r.ApprovalState != "" && !r.ApprovalState.IsValid() {
		return NewValidationError("approvalState", "invalid approval state value")
	}

	return nil
}

// ValidateForCreation checks if a Run has valid initial state.
// This is stricter than Validate() as it checks creation-specific constraints.
func (r *Run) ValidateForCreation() error {
	// First run general validation
	if err := r.Validate(); err != nil {
		return err
	}

	// Initial status should be pending or starting
	if r.Status != "" && r.Status != RunStatusPending {
		return NewValidationError("status",
			"new runs must start in pending status")
	}

	// Initial phase should be queued
	if r.Phase != "" && r.Phase != RunPhaseQueued {
		return NewValidationError("phase",
			"new runs must start in queued phase")
	}

	// ApprovalState should be none for new runs
	if r.ApprovalState != "" && r.ApprovalState != ApprovalStateNone {
		return NewValidationError("approvalState",
			"new runs must start with no approval state")
	}

	return nil
}

// =============================================================================
// POLICY VALIDATION
// =============================================================================

// Validate checks if a Policy is valid for creation/update.
//
// INVARIANTS ENFORCED:
// - Name is required
// - Priority is non-negative
// - Rules are internally consistent
func (p *Policy) Validate() error {
	// Name is required
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return NewValidationErrorWithHint("name", "field is required",
			"Provide a descriptive name for this policy")
	}
	if len(name) > 255 {
		return NewValidationError("name", "must be 255 characters or less")
	}

	// Priority must be non-negative
	if p.Priority < 0 {
		return NewValidationError("priority", "cannot be negative")
	}

	// Priority should be reasonable
	if p.Priority > 10000 {
		return NewValidationError("priority", "must be 10000 or less")
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
		return NewValidationError("rules.maxConcurrentRuns",
			"must be at least 1 if specified")
	}

	if r.MaxConcurrentPerScope != nil && *r.MaxConcurrentPerScope < 1 {
		return NewValidationError("rules.maxConcurrentPerScope",
			"must be at least 1 if specified")
	}

	// Resource limits must be positive if set
	if r.MaxFilesChanged != nil && *r.MaxFilesChanged < 1 {
		return NewValidationError("rules.maxFilesChanged",
			"must be at least 1 if specified")
	}

	if r.MaxTotalSizeBytes != nil && *r.MaxTotalSizeBytes < 1 {
		return NewValidationError("rules.maxTotalSizeBytes",
			"must be at least 1 if specified")
	}

	if r.MaxExecutionTimeMs != nil && *r.MaxExecutionTimeMs < 1000 {
		return NewValidationError("rules.maxExecutionTimeMs",
			"must be at least 1000ms (1 second) if specified")
	}

	// Validate runner lists don't conflict
	if hasRunnerOverlap(r.AllowedRunners, r.DeniedRunners) {
		return NewValidationError("rules.allowedRunners/deniedRunners",
			"runner appears in both allowed and denied lists")
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
		return NewValidationErrorWithHint("type", "invalid attachment type",
			"valid types: file, link, note")
	}

	switch c.Type {
	case "file":
		if strings.TrimSpace(c.Path) == "" {
			return NewValidationError("path", "required for file attachments")
		}
	case "link":
		if strings.TrimSpace(c.URL) == "" {
			return NewValidationError("url", "required for link attachments")
		}
	case "note":
		if strings.TrimSpace(c.Content) == "" {
			return NewValidationError("content", "required for note attachments")
		}
	}

	return nil
}

// =============================================================================
// RUN EVENT VALIDATION
// =============================================================================

// Validate checks that a RunEvent is valid.
func (e *RunEvent) Validate() error {
	if e.RunID == uuid.Nil {
		return NewValidationError("runId", "field is required")
	}

	if e.Sequence < 0 {
		return NewValidationError("sequence", "must be non-negative")
	}

	if !e.EventType.IsValid() {
		return NewValidationError("eventType", "invalid event type")
	}

	if e.Timestamp.IsZero() {
		return NewValidationError("timestamp", "field is required")
	}

	// Data payload must match event type if present
	if e.Data != nil && e.Data.EventType() != e.EventType {
		return NewValidationError("data",
			"payload type does not match event type")
	}

	return nil
}

// =============================================================================
// CHECKPOINT VALIDATION
// =============================================================================

// Validate checks that a RunCheckpoint is valid.
func (c *RunCheckpoint) Validate() error {
	if c.RunID == uuid.Nil {
		return NewValidationError("runId", "field is required")
	}

	if !c.Phase.IsValid() {
		return NewValidationError("phase", "invalid phase value")
	}

	if c.StepWithinPhase < 0 {
		return NewValidationError("stepWithinPhase", "must be non-negative")
	}

	if c.LastEventSequence < 0 {
		return NewValidationError("lastEventSequence", "must be non-negative")
	}

	if c.RetryCount < 0 {
		return NewValidationError("retryCount", "must be non-negative")
	}

	return nil
}

// =============================================================================
// SCOPE LOCK VALIDATION
// =============================================================================

// Validate checks that a ScopeLock is valid.
func (l *ScopeLock) Validate() error {
	if l.RunID == uuid.Nil {
		return NewValidationError("runId", "field is required")
	}

	if strings.TrimSpace(l.ScopePath) == "" {
		return NewValidationError("scopePath", "field is required")
	}

	if l.AcquiredAt.IsZero() {
		return NewValidationError("acquiredAt", "field is required")
	}

	if l.ExpiresAt.IsZero() {
		return NewValidationError("expiresAt", "field is required")
	}

	if l.ExpiresAt.Before(l.AcquiredAt) {
		return NewValidationError("expiresAt", "must be after acquiredAt")
	}

	return nil
}

// IsExpired returns whether this lock has expired.
func (l *ScopeLock) IsExpired() bool {
	return time.Now().After(l.ExpiresAt)
}

// =============================================================================
// IDEMPOTENCY RECORD VALIDATION
// =============================================================================

// Validate checks that an IdempotencyRecord is valid.
func (r *IdempotencyRecord) Validate() error {
	if strings.TrimSpace(r.Key) == "" {
		return NewValidationError("key", "field is required")
	}

	if !r.Status.IsValid() {
		return NewValidationError("status", "invalid status value")
	}

	if r.CreatedAt.IsZero() {
		return NewValidationError("createdAt", "field is required")
	}

	if r.ExpiresAt.IsZero() {
		return NewValidationError("expiresAt", "field is required")
	}

	if r.ExpiresAt.Before(r.CreatedAt) {
		return NewValidationError("expiresAt", "must be after createdAt")
	}

	return nil
}

// =============================================================================
// VALIDATION HELPERS
// =============================================================================

// hasStringOverlap checks if two string slices have any common elements.
func hasStringOverlap(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}

	set := make(map[string]bool, len(a))
	for _, v := range a {
		set[v] = true
	}
	for _, v := range b {
		if set[v] {
			return true
		}
	}
	return false
}

// hasRunnerOverlap checks if two runner type slices have any common elements.
func hasRunnerOverlap(a, b []RunnerType) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}

	set := make(map[RunnerType]bool, len(a))
	for _, v := range a {
		set[v] = true
	}
	for _, v := range b {
		if set[v] {
			return true
		}
	}
	return false
}
