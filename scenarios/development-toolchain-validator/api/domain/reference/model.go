// DOC: docs/concepts/ARCHITECTURE.md#domain-modules
// DOC: docs/reference/data-model.md#references
// DOC: docs/concepts/GLOSSARY.md
// DOC: docs/internal/SEAMS.md#change-axes
//
// Package reference defines the Reference Scenario Registry domain.
//
// # Purpose
//
// Reference scenarios serve as known-good ground truth implementations
// for validating steer skill interoperability and development tooling.
// They are actual, working scenarios in the scenarios/ directory that:
//
//   - Pass all validation tools (scenario-auditor, test-genie, completeness-scoring)
//   - Implement patterns expected by steer skills correctly
//   - Serve as baselines for detecting tooling regressions
//
// # Why This Domain Exists
//
// Steer skills are markdown files that guide AI agents. There's no way to
// programmatically verify whether a skill's guidance is correct without
// running it against real code. References provide that real code baseline.
//
// When a development tool gives incorrect results on a reference scenario
// (false positive violations, failing tests, low scores), the tool is wrong.
// When a skill's expectations don't match reference reality, the skill is wrong.
//
// # Domain Boundaries
//
// This domain handles CRUD for reference registrations only. It does NOT:
//   - Connect skills to references (see: skill domain - planned)
//   - Run validations (see: validation domain - planned)
//   - Generate reports (see: report domain - planned)
//   - Modify reference scenario code (read-only validation principle)
package reference

import (
	"time"
)

// Reference represents a registered reference scenario in the validation system.
//
// A Reference points to an existing scenario directory and tracks metadata
// about which template it was created from. This enables template-aware
// validation and skill connection.
//
// # Fields
//
//   - ID: UUID for database relationships
//   - Slug: Human-friendly identifier for CLI/API (e.g., "reference-react-vite")
//   - Name: Display name for UI (e.g., "React Vite Reference")
//   - Template: Template type for validation rules (e.g., "react-vite", "landing-page-react-vite")
//   - Path: Absolute filesystem path to the scenario directory
//   - Description: Optional human-readable context
//
// # Invariants
//
//   - Slug must be unique across all references
//   - Path must point to an existing directory at creation time
//   - Path is stored as absolute path for consistent resolution
//
// [REQ:P0-001] Reference Scenario Database Schema
type Reference struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Template    string    `json:"template"`
	Path        string    `json:"path"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateInput contains the required fields for creating a new reference.
type CreateInput struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Template    string `json:"template"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
}

// UpdateInput contains the mutable fields for updating a reference.
type UpdateInput struct {
	Name        *string `json:"name,omitempty"`
	Template    *string `json:"template,omitempty"`
	Path        *string `json:"path,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ListOptions contains filtering and pagination options for listing references.
type ListOptions struct {
	Template string `json:"template,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}
