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
	"fmt"
	"sort"
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
		RunStatusNeedsReview, RunStatusParked, RunStatusComplete, RunStatusFailed, RunStatusCancelled, RunStatusUnknown:
		return true
	default:
		return false
	}
}

// IsTerminal returns whether this is a terminal status (no further transitions allowed).
// parked is intentionally NOT terminal: a parked run resumes (wake) or is
// cancelled; it is a resting-but-resumable state like needs_review.
func (s RunStatus) IsTerminal() bool {
	switch s {
	case RunStatusComplete, RunStatusFailed, RunStatusCancelled, RunStatusUnknown:
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

// IsValid returns whether this is a valid RunFinalizationStatus.
func (s RunFinalizationStatus) IsValid() bool {
	switch s {
	case RunFinalizationStatusNone, RunFinalizationStatusPending, RunFinalizationStatusRunning,
		RunFinalizationStatusSucceeded, RunFinalizationStatusFailed, RunFinalizationStatusSkipped:
		return true
	default:
		return false
	}
}

// IsFailed returns whether post-run sandbox finalization failed.
func (s RunFinalizationStatus) IsFailed() bool {
	return s == RunFinalizationStatusFailed
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
	case EventTypeLog, EventTypeMessage, EventTypeMessageDeleted, EventTypeToolCall, EventTypeToolResult,
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

	// ProfileKey is optional but must be non-empty if provided
	if p.ProfileKey != "" {
		key := strings.TrimSpace(p.ProfileKey)
		if key == "" {
			return NewValidationError("profileKey", "cannot be empty when provided")
		}
		if len(key) > 255 {
			return NewValidationError("profileKey", "must be 255 characters or less")
		}
	}

	if strings.TrimSpace(p.RoleRef) == "" {
		return NewValidationErrorWithHint("roleRef", "field is required",
			"Select a portable role from the active role-policy catalog")
	}

	// MaxTurns must be non-negative
	if p.MaxTurns < 0 {
		return NewValidationError("maxTurns", "cannot be negative")
	}

	// Timeout must be non-negative
	if p.Timeout < 0 {
		return NewValidationError("timeout", "cannot be negative")
	}
	if !p.Effort.IsValid() {
		return NewValidationErrorWithHint("effort", "invalid effort", "valid values: low, medium, high, xhigh, max")
	}

	if err := validateCanonicalTools("allowedTools", p.AllowedTools); err != nil {
		return err
	}
	if err := validateCanonicalTools("deniedTools", p.DeniedTools); err != nil {
		return err
	}
	if !p.ToolRestrictionPolicy.IsValid() {
		return NewValidationErrorWithHint("toolRestrictionPolicy", "invalid tool restriction policy", "valid values: enforced, advisory")
	}

	// AllowedTools and DeniedTools should not overlap.
	if hasStringOverlap(p.AllowedTools, p.DeniedTools) {
		return NewValidationError("allowedTools/deniedTools",
			"same tool cannot be both allowed and denied")
	}

	// AllowedPaths and DeniedPaths should not overlap
	if hasStringOverlap(p.AllowedPaths, p.DeniedPaths) {
		return NewValidationError("allowedPaths/deniedPaths",
			"same path cannot be both allowed and denied")
	}

	if !p.NetworkAccess.IsValid() {
		return NewValidationErrorWithHint("networkAccess", "invalid network access level",
			"valid values: none, localhost, full")
	}

	if err := validateSandboxConfig(p.SandboxConfig); err != nil {
		return err
	}

	if err := validateExtraFlagsStructure(p.ExtraFlags); err != nil {
		return err
	}

	return nil
}

func validateCanonicalTools(field string, tools []string) error {
	for _, tool := range tools {
		name := strings.TrimSpace(tool)
		if CanonicalTool(name).IsValid() {
			continue
		}
		return NewValidationErrorWithHint(field, fmt.Sprintf("unknown canonical tool %q", tool),
			"Use a canonical tool name; nearest match: "+nearestCanonicalTool(name))
	}
	return nil
}

// ValidateCanonicalToolList validates a resolved run's allowlist without
// requiring profile-only fields such as name and roleRef.
func ValidateCanonicalToolList(field string, tools []string) error {
	return validateCanonicalTools(field, tools)
}

func nearestCanonicalTool(name string) string {
	candidates := CanonicalTools()
	sort.Slice(candidates, func(i, j int) bool {
		left, right := levenshteinDistance(name, string(candidates[i])), levenshteinDistance(name, string(candidates[j]))
		if left == right {
			return candidates[i] < candidates[j]
		}
		return left < right
	})
	return string(candidates[0])
}

func levenshteinDistance(left, right string) int {
	previous := make([]int, len(right)+1)
	for j := range previous {
		previous[j] = j
	}
	for i, leftRune := range left {
		current := make([]int, len(right)+1)
		current[0] = i + 1
		for j, rightRune := range right {
			cost := 0
			if leftRune != rightRune {
				cost = 1
			}
			current[j+1] = minInt(current[j]+1, previous[j+1]+1, previous[j]+cost)
		}
		previous = current
	}
	return previous[len(right)]
}

func minInt(values ...int) int {
	min := values[0]
	for _, value := range values[1:] {
		if value < min {
			min = value
		}
	}
	return min
}

func validateSandboxConfig(cfg *SandboxConfig) error {
	if cfg == nil {
		return nil
	}
	if cfg.Acceptance.Mode != "" && cfg.Acceptance.Mode != "allowlist" {
		return NewValidationError("sandboxConfig.acceptance.mode", "invalid acceptance mode")
	}
	if cfg.Lifecycle.TTL < 0 {
		return NewValidationError("sandboxConfig.lifecycle.ttl", "cannot be negative")
	}
	if cfg.Lifecycle.IdleTimeout < 0 {
		return NewValidationError("sandboxConfig.lifecycle.idleTimeout", "cannot be negative")
	}
	if err := validateSandboxMode(cfg.Mode); err != nil {
		return err
	}
	if cfg.NetworkMode != "" && !cfg.NetworkMode.IsValid() {
		return NewValidationErrorWithHint(
			"sandboxConfig.networkMode",
			"invalid network mode",
			"valid values: none, localhost, full",
		)
	}
	return nil
}

// validateSandboxMode rejects unknown SandboxMode values. All recognised
// modes (unspecified, off, tracking, protected) are accepted at the
// validation layer; whether protected mode actually launches through
// the sandbox or falls back to the host depends on runtime configuration
// of the runner's SandboxLauncherFactory (see runnercore.NewRunner /
// SetSandboxLauncherFactory).
//
// SandboxModeOff is the explicit "no sandbox" choice; it is the only
// Mode that produces RunModeInPlace via DeriveRunMode.
func validateSandboxMode(mode SandboxMode) error {
	switch mode {
	case SandboxModeUnspecified, SandboxModeOff, SandboxModeTracking, SandboxModeProtected:
		return nil
	default:
		return NewValidationErrorWithHint(
			"sandboxConfig.mode",
			"invalid sandbox mode",
			"valid values: off, tracking (default), protected",
		)
	}
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

	// Description has a reasonable limit (64KB accommodates large agent prompts)
	if len(t.Description) > 65536 {
		return NewValidationError("description", "must be 65536 characters or less")
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
	seenKeys := make(map[string]bool)
	for i, att := range t.ContextAttachments {
		if err := att.Validate(); err != nil {
			// Wrap with index for better error messages
			if ve, ok := err.(*ValidationError); ok {
				return &ValidationError{
					Field:   fmt.Sprintf("contextAttachments[%d].%s", i, ve.Field),
					Message: ve.Message,
					Hint:    ve.Hint,
				}
			}
			return err
		}

		// Check for duplicate keys
		if att.Key != "" {
			if seenKeys[att.Key] {
				return &ValidationError{
					Field:   fmt.Sprintf("contextAttachments[%d].key", i),
					Message: "duplicate key: " + att.Key,
					Hint:    "each context attachment must have a unique key",
				}
			}
			seenKeys[att.Key] = true
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

	// Either AgentProfileID or ResolvedConfig is required
	hasProfile := r.AgentProfileID != nil && *r.AgentProfileID != uuid.Nil
	hasConfig := r.ResolvedConfig != nil
	if !hasProfile && !hasConfig {
		return NewValidationErrorWithHint("agentProfileId", "either agentProfileId or inline config is required",
			"provide agentProfileId or inline config fields (runnerType, maxTurns, etc.)")
	}

	// RunMode must be valid
	if r.RunMode != RunModeSandboxed && r.RunMode != RunModeInPlace {
		return NewValidationErrorWithHint("runMode", "invalid run mode",
			"valid modes: sandboxed, in_place")
	}

	// Interactive execution mode is only available for non-protected (in-place)
	// runs. See ValidateInteractiveRunMode.
	if err := ValidateInteractiveRunMode(r.ExecutionMode, r.InteractiveSandboxMode()); err != nil {
		return err
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

// ValidateInteractiveRunMode validates only the execution-mode vocabulary.
// Feasibility of an execution-mode/sandbox combination is a runner capability
// and profile policy concern, not a hard-coded domain prohibition.
func ValidateInteractiveRunMode(execMode ExecutionMode, sandboxMode SandboxMode) error {
	_ = sandboxMode
	return nil
}

// InteractiveSandboxMode returns the resolved sandbox policy that governs an
// interactive launch. Legacy sandboxed rows are conservatively Protected.
func (r *Run) InteractiveSandboxMode() SandboxMode {
	if r.SandboxConfig != nil {
		return r.SandboxConfig.Mode
	}
	if r.ResolvedConfig != nil && r.ResolvedConfig.SandboxConfig != nil {
		return r.ResolvedConfig.SandboxConfig.Mode
	}
	if r.RunMode == RunModeSandboxed {
		return SandboxModeProtected
	}
	return SandboxModeOff
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
	validTypes := map[string]bool{"file": true, "link": true, "note": true, "image": true}
	if !validTypes[c.Type] {
		return NewValidationErrorWithHint("type", "invalid attachment type",
			"valid types: file, link, note, image")
	}

	// Key validation: optional but must be valid format if provided
	if c.Key != "" {
		key := strings.TrimSpace(c.Key)
		if key == "" {
			return NewValidationError("key", "cannot be whitespace-only")
		}
		if len(key) > 128 {
			return NewValidationError("key", "must be 128 characters or less")
		}
		if !isValidContextKey(key) {
			return NewValidationErrorWithHint("key", "invalid key format",
				"use lowercase alphanumeric with hyphens or underscores (e.g., 'error-logs')")
		}
	}

	// Tags validation
	if len(c.Tags) > 10 {
		return NewValidationError("tags", "cannot have more than 10 tags")
	}
	for i, tag := range c.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return NewValidationError("tags", "tag cannot be empty")
		}
		if len(tag) > 64 {
			return NewValidationError("tags", fmt.Sprintf("tag[%d] must be 64 characters or less", i))
		}
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
	case "image":
		if strings.TrimSpace(c.AttachmentID) == "" {
			return NewValidationError("attachment_id", "required for image attachments")
		}
	}

	return nil
}

// isValidContextKey checks if a key follows the allowed format:
// lowercase letters, numbers, hyphens, and underscores only.
func isValidContextKey(key string) bool {
	for _, r := range key {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
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

// validateExtraFlagsStructure performs structural validation on extra flags.
// It checks runner type validity, flag count limits, flag syntax, and shell meta characters.
// It does NOT validate flags against runner allowlists (that's the runner layer's job).
func validateExtraFlagsStructure(flags RunnerExtraFlags) error {
	for rt, flagList := range flags {
		if !rt.IsValid() {
			return NewValidationError("extraFlags", "invalid runner type key: "+string(rt))
		}
		if len(flagList) > 20 {
			return NewValidationError("extraFlags",
				fmt.Sprintf("too many flags for runner %s (max 20)", rt))
		}
		for i, flag := range flagList {
			if strings.TrimSpace(flag) == "" {
				return NewValidationError("extraFlags",
					fmt.Sprintf("empty flag at index %d for runner %s", i, rt))
			}
			if !strings.HasPrefix(flag, "-") {
				return NewValidationError("extraFlags",
					fmt.Sprintf("flag %q must start with '-'", flag))
			}
			if containsShellMeta(flag) {
				return NewValidationError("extraFlags",
					fmt.Sprintf("flag %q contains disallowed characters", flag))
			}
		}
	}
	return nil
}

// containsShellMeta returns true if the string contains shell metacharacters
// that could enable command injection.
func containsShellMeta(s string) bool {
	for _, c := range s {
		switch c {
		case '|', '&', ';', '$', '`', '(', ')', '{', '}', '<', '>', '\n', '\r':
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
