// DOC: docs/concepts/ARCHITECTURE.md#domain-modules
// DOC: docs/reference/data-model.md#skill-connections
// DOC: docs/concepts/GLOSSARY.md
// DOC: docs/internal/SEAMS.md#change-axes
//
// Package skill defines the Skill Connection Management domain.
//
// # Purpose
//
// Skill connections link prompt-manager steer skills to reference scenarios,
// enabling programmatic validation of skill expectations against known-good code.
// Each connection tracks:
//   - Which skill is connected to which reference
//   - Version/hash at connection time for drift detection
//   - Structural expectations and CLI assertions per connection
//
// # Why This Domain Exists
//
// Steer skills are markdown files that guide AI agents. Without explicit connections
// to reference scenarios, there's no way to:
//   - Know which skills have been validated against which references
//   - Detect when skill content changes (drift)
//   - Validate that skill expectations match reference reality
//
// # Domain Boundaries
//
// This domain handles:
//   - Creating/deleting skill-reference connections
//   - Storing connection metadata (version, hash)
//   - Drift detection (comparing stored vs current version)
//
// This domain does NOT:
//   - Manage skills themselves (prompt-manager owns skills)
//   - Run validations (see: validation domain)
//   - Generate reports (see: report domain)
//
// [REQ:REQ-P0-003] Prompt-Manager Skill Connection Store
package skill

import (
	"time"
)

// Connection represents a link between a prompt-manager skill and a reference scenario.
//
// A Connection stores the skill's version and content hash at connection time,
// enabling drift detection when the skill content changes in prompt-manager.
//
// # Fields
//
//   - ID: UUID for database relationships
//   - ReferenceID: UUID of the connected reference scenario
//   - SkillID: Identifier of the skill in prompt-manager (e.g., "api-steer", "cli-steer")
//   - SkillVersion: Version number from prompt-manager at connection time
//   - SkillContentHash: SHA-256 hash of skill content at connection time
//   - ConnectedAt: Timestamp when the connection was created
//   - UpdatedAt: Timestamp when the connection was last modified
//
// # Invariants
//
//   - (ReferenceID, SkillID) must be unique - a skill can only be connected once per reference
//   - ReferenceID must reference an existing reference scenario
//   - SkillID must be a valid prompt-manager skill identifier
//
// [REQ:REQ-P0-003] Prompt-Manager Skill Connection Store
type Connection struct {
	ID               string    `json:"id"`
	ReferenceID      string    `json:"reference_id"`
	SkillID          string    `json:"skill_id"`
	SkillVersion     string    `json:"skill_version,omitempty"`
	SkillContentHash string    `json:"skill_content_hash,omitempty"`
	ConnectedAt      time.Time `json:"connected_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ConnectInput contains the required fields for creating a new skill connection.
type ConnectInput struct {
	ReferenceID      string `json:"reference_id"`
	SkillID          string `json:"skill_id"`
	SkillVersion     string `json:"skill_version,omitempty"`
	SkillContentHash string `json:"skill_content_hash,omitempty"`
}

// UpdateInput contains the mutable fields for updating a skill connection.
// Used when refreshing version/hash from prompt-manager.
type UpdateInput struct {
	SkillVersion     *string `json:"skill_version,omitempty"`
	SkillContentHash *string `json:"skill_content_hash,omitempty"`
}

// ListOptions contains filtering and pagination options for listing connections.
type ListOptions struct {
	ReferenceID string `json:"reference_id,omitempty"`
	SkillID     string `json:"skill_id,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
}

// DriftStatus represents whether a skill connection has drifted from its stored version.
type DriftStatus struct {
	ConnectionID     string `json:"connection_id"`
	SkillID          string `json:"skill_id"`
	StoredVersion    string `json:"stored_version"`
	StoredHash       string `json:"stored_hash"`
	CurrentVersion   string `json:"current_version"`
	CurrentHash      string `json:"current_hash"`
	HasDrifted       bool   `json:"has_drifted"`
	VersionChanged   bool   `json:"version_changed"`
	ContentChanged   bool   `json:"content_changed"`
}
