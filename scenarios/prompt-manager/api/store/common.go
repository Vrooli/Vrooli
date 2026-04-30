// Package store provides the refactored storage layer for prompt-manager.
// It implements per-entity file storage with normalized relations and generated indexes.
//
// Key concepts:
// - Skills: Reusable guidance documents stored as skill.json + SKILL.md per skill
// - Agents: Persistent identities with team memberships and markdown-based skill references
// - Teams: Shared intent and org structure
// - Relations: Normalized team-member relationships
// - Indexes: Generated from entity files, never hand-edited
//
// DOC: docs/concepts/STORAGE.md
package store

import (
	"time"
)

// Entity types
const (
	KindSkill      = "skill"
	KindVariant    = "variant"
	KindExperiment = "experiment"
	KindAgent      = "agent"
	KindTeam       = "team"
	KindTeamRoles  = "team-roles"
	KindOrgChart   = "org-chart"
	KindTeamInbox  = "team-inbox"
	KindTeamMember = "team-member"
	KindTopic      = "topic"
	KindAction     = "action"
)

// Experiment status values
const (
	ExperimentStatusDraft     = "draft"
	ExperimentStatusRunning   = "running"
	ExperimentStatusConcluded = "concluded"
)

// ControlVariantID is the implicit variant representing the original SKILL.md content.
const ControlVariantID = "control"

// Status values for skills
const (
	StatusActive   = "active"
	StatusDraft    = "draft"
	StatusArchived = "archived"
)

// Agent status values
const (
	AgentStatusActive    = "active"
	AgentStatusInactive  = "inactive"
	AgentStatusSuspended = "suspended"
)

// Team member status values
const (
	MemberStatusActive   = "active"
	MemberStatusInactive = "inactive"
	MemberStatusPending  = "pending"
)

// CurrentSchemaVersion is the current schema version for all entities
const CurrentSchemaVersion = 1

// BaseEntity contains common fields for all entities
type BaseEntity struct {
	Kind          string `json:"kind"`
	SchemaVersion int    `json:"schemaVersion"`
}

// Timestamps contains common timestamp fields
type Timestamps struct {
	Revision  int    `json:"revision"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// NewTimestamps creates timestamps for a new entity
func NewTimestamps() Timestamps {
	now := time.Now().UTC().Format(time.RFC3339)
	return Timestamps{
		Revision:  1,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// UpdateTimestamp updates the timestamp and increments revision
func (t *Timestamps) UpdateTimestamp() {
	t.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	t.Revision++
}

// Capability represents a required capability for a skill
type Capability struct {
	CapabilityID string   `json:"capabilityId"`
	Verbs        []string `json:"verbs"`
}

// SkillRequires contains capability requirements for a skill
type SkillRequires struct {
	Capabilities []Capability `json:"capabilities,omitempty"`
}
