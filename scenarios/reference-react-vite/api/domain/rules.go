// Package domain provides shared business rules and validation constants.
// This file serves as the single source of truth for domain-level decisions.
//
// ╔════════════════════════════════════════════════════════════════════════════╗
// ║  BUSINESS RULES - Read before modifying                                    ║
// ║                                                                            ║
// ║  These rules define the core business constraints. Changes here affect:    ║
// ║  - Input validation in domain factory functions                            ║
// ║  - API error messages and responses                                        ║
// ║  - Database constraints (must stay in sync with schema.sql)                ║
// ║                                                                            ║
// ║  When changing rules:                                                      ║
// ║  1. Update the constant here                                               ║
// ║  2. Update corresponding domain package validation                         ║
// ║  3. Update database schema if storing the value                            ║
// ║  4. Update tests to verify new boundaries                                  ║
// ╚════════════════════════════════════════════════════════════════════════════╝
//
// DOC: docs/internal/SEAMS.md#decision-points
// DOC: docs/reference/data-model.md
package domain

// =============================================================================
// VALIDATION LIMITS
// These define the boundaries for user-provided data.
// =============================================================================

// ValidationLimits groups all max-length constraints for input validation.
// These are the authoritative values that domain packages should reference.
type ValidationLimits struct {
	// TaskTitleMaxLength is the maximum characters allowed for task titles.
	TaskTitleMaxLength int
	// ProjectNameMaxLength is the maximum characters allowed for project names.
	ProjectNameMaxLength int
	// NoteContentMaxLength is the maximum characters allowed for note content.
	NoteContentMaxLength int
}

// DefaultValidationLimits returns the default validation limits.
// These are designed for typical use cases and match the database schema.
func DefaultValidationLimits() ValidationLimits {
	return ValidationLimits{
		TaskTitleMaxLength:   255, // VARCHAR(255) in schema.sql
		ProjectNameMaxLength: 100, // VARCHAR(100) in schema.sql
		NoteContentMaxLength: 10000, // TEXT with application limit
	}
}

// =============================================================================
// DEFAULT VALUES
// These define the initial/fallback values for optional fields.
// =============================================================================

// TaskDefaults groups default values applied when creating new tasks.
type TaskDefaults struct {
	// Status is the initial status for new tasks.
	// Rationale: Tasks start as work-to-do, not work-in-progress.
	Status string
	// Priority is used when no priority is specified.
	// Rationale: Medium priority as a safe default (not urgent, not ignored).
	Priority int
}

// DefaultTaskDefaults returns the standard defaults for new tasks.
func DefaultTaskDefaults() TaskDefaults {
	return TaskDefaults{
		Status:   "pending",
		Priority: 2, // Medium
	}
}

// ProjectDefaults groups default values applied when creating new projects.
type ProjectDefaults struct {
	// Status is the initial status for new projects.
	// Rationale: Projects are active when created (ready to receive tasks).
	Status string
}

// DefaultProjectDefaults returns the standard defaults for new projects.
func DefaultProjectDefaults() ProjectDefaults {
	return ProjectDefaults{
		Status: "active",
	}
}

// =============================================================================
// STATUS DEFINITIONS
// These enumerate the allowed status values for each entity type.
// =============================================================================

// TaskStatuses defines the allowed lifecycle states for tasks.
// The lifecycle is: pending -> in_progress -> completed -> archived
// All transitions are currently allowed (no state machine enforcement).
//
// DESIGN DECISION: No transition rules are enforced currently.
// If you need to add transition rules, consider:
// 1. Adding a ValidateTransition(from, to Status) error function
// 2. Documenting the transition matrix below
// 3. Updating ApplyUpdate to call ValidateTransition
var TaskStatuses = struct {
	Pending    string // Task not yet started
	InProgress string // Task actively being worked on
	Completed  string // Task finished successfully
	Archived   string // Task removed from active view
}{
	Pending:    "pending",
	InProgress: "in_progress",
	Completed:  "completed",
	Archived:   "archived",
}

// ProjectStatuses defines the allowed lifecycle states for projects.
// The lifecycle is: active -> paused/complete -> archived
var ProjectStatuses = struct {
	Active   string // Project accepting new tasks
	Paused   string // Project temporarily on hold
	Complete string // Project finished, no new work
	Archived string // Project removed from active view
}{
	Active:   "active",
	Paused:   "paused",
	Complete: "complete",
	Archived: "archived",
}

// =============================================================================
// PRIORITY DEFINITIONS
// These define the priority levels and their semantics.
// =============================================================================

// PriorityLevels defines the available priority levels for tasks.
// Higher numbers = higher priority (1=low, 2=medium, 3=high).
var PriorityLevels = struct {
	Low    int // Can wait, not time-sensitive
	Medium int // Normal priority, default value
	High   int // Urgent, should be addressed first
	Min    int // Minimum valid priority
	Max    int // Maximum valid priority
}{
	Low:    1,
	Medium: 2,
	High:   3,
	Min:    1,
	Max:    3,
}

// IsPriorityValid checks if a priority value is within the allowed range.
func IsPriorityValid(p int) bool {
	return p >= PriorityLevels.Min && p <= PriorityLevels.Max
}

// =============================================================================
// COLOR VALIDATION
// These define the rules for project color codes.
// =============================================================================

// ColorFormat describes the expected format for color values.
// Projects use hex color codes for UI consistency.
const ColorFormat = "#RRGGBB"

// IsValidHexColor checks if a string is a valid hex color code.
// Empty strings are valid (no color assigned).
// Valid format: #RRGGBB (7 characters, # prefix, hex digits)
func IsValidHexColor(color string) bool {
	if color == "" {
		return true
	}
	if len(color) != 7 || color[0] != '#' {
		return false
	}
	for _, c := range color[1:] {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}
