// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md
// Package testutil provides test fixtures and factories for creating test data.
package testutil

import (
	"time"

	"development-toolchain-validator/domain/reference"
	"development-toolchain-validator/domain/skill"
)

// ReferenceFactory creates Reference test fixtures with builder pattern.
type ReferenceFactory struct {
	ref reference.Reference
}

// NewReferenceFactory creates a new factory with default valid values.
func NewReferenceFactory() *ReferenceFactory {
	return &ReferenceFactory{
		ref: reference.Reference{
			ID:          "00000000-0000-0000-0000-000000000001",
			Slug:        "test-reference",
			Name:        "Test Reference",
			Template:    "react-vite",
			Path:        "/tmp/test-scenarios/test-reference",
			Description: "A test reference scenario",
			CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

// WithID sets a custom ID.
func (f *ReferenceFactory) WithID(id string) *ReferenceFactory {
	f.ref.ID = id
	return f
}

// WithSlug sets a custom slug.
func (f *ReferenceFactory) WithSlug(slug string) *ReferenceFactory {
	f.ref.Slug = slug
	return f
}

// WithName sets a custom name.
func (f *ReferenceFactory) WithName(name string) *ReferenceFactory {
	f.ref.Name = name
	return f
}

// WithTemplate sets a custom template.
func (f *ReferenceFactory) WithTemplate(template string) *ReferenceFactory {
	f.ref.Template = template
	return f
}

// WithPath sets a custom path.
func (f *ReferenceFactory) WithPath(path string) *ReferenceFactory {
	f.ref.Path = path
	return f
}

// WithDescription sets a custom description.
func (f *ReferenceFactory) WithDescription(desc string) *ReferenceFactory {
	f.ref.Description = desc
	return f
}

// Build returns the configured Reference.
func (f *ReferenceFactory) Build() *reference.Reference {
	// Return a copy to prevent accidental mutation
	result := f.ref
	return &result
}

// CreateInputFactory creates CreateInput test fixtures.
type CreateInputFactory struct {
	input reference.CreateInput
}

// NewCreateInputFactory creates a factory with valid default values.
func NewCreateInputFactory() *CreateInputFactory {
	return &CreateInputFactory{
		input: reference.CreateInput{
			Slug:        "test-reference",
			Name:        "Test Reference",
			Template:    "react-vite",
			Path:        "/tmp/test-scenarios/test-reference",
			Description: "A test reference scenario",
		},
	}
}

// WithSlug sets a custom slug.
func (f *CreateInputFactory) WithSlug(slug string) *CreateInputFactory {
	f.input.Slug = slug
	return f
}

// WithName sets a custom name.
func (f *CreateInputFactory) WithName(name string) *CreateInputFactory {
	f.input.Name = name
	return f
}

// WithTemplate sets a custom template.
func (f *CreateInputFactory) WithTemplate(template string) *CreateInputFactory {
	f.input.Template = template
	return f
}

// WithPath sets a custom path.
func (f *CreateInputFactory) WithPath(path string) *CreateInputFactory {
	f.input.Path = path
	return f
}

// WithDescription sets a custom description.
func (f *CreateInputFactory) WithDescription(desc string) *CreateInputFactory {
	f.input.Description = desc
	return f
}

// Build returns the configured CreateInput.
func (f *CreateInputFactory) Build() reference.CreateInput {
	return f.input
}

// ============================================================================
// Skill Connection Factories
// ============================================================================

// ConnectionFactory creates Connection test fixtures with builder pattern.
type ConnectionFactory struct {
	conn skill.Connection
}

// NewConnectionFactory creates a new factory with default valid values.
func NewConnectionFactory() *ConnectionFactory {
	return &ConnectionFactory{
		conn: skill.Connection{
			ID:               "00000000-0000-0000-0000-000000000010",
			ReferenceID:      "00000000-0000-0000-0000-000000000001",
			SkillID:          "test-skill",
			SkillVersion:     "1.0.0",
			SkillContentHash: "abc123hash",
			ConnectedAt:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

// WithID sets a custom ID.
func (f *ConnectionFactory) WithID(id string) *ConnectionFactory {
	f.conn.ID = id
	return f
}

// WithReferenceID sets a custom reference ID.
func (f *ConnectionFactory) WithReferenceID(refID string) *ConnectionFactory {
	f.conn.ReferenceID = refID
	return f
}

// WithSkillID sets a custom skill ID.
func (f *ConnectionFactory) WithSkillID(skillID string) *ConnectionFactory {
	f.conn.SkillID = skillID
	return f
}

// WithSkillVersion sets a custom skill version.
func (f *ConnectionFactory) WithSkillVersion(version string) *ConnectionFactory {
	f.conn.SkillVersion = version
	return f
}

// WithSkillContentHash sets a custom skill content hash.
func (f *ConnectionFactory) WithSkillContentHash(hash string) *ConnectionFactory {
	f.conn.SkillContentHash = hash
	return f
}

// Build returns the configured Connection.
func (f *ConnectionFactory) Build() *skill.Connection {
	// Return a copy to prevent accidental mutation
	result := f.conn
	return &result
}

// ConnectInputFactory creates ConnectInput test fixtures.
type ConnectInputFactory struct {
	input skill.ConnectInput
}

// NewConnectInputFactory creates a factory with valid default values.
func NewConnectInputFactory() *ConnectInputFactory {
	return &ConnectInputFactory{
		input: skill.ConnectInput{
			ReferenceID:      "00000000-0000-0000-0000-000000000001",
			SkillID:          "test-skill",
			SkillVersion:     "1.0.0",
			SkillContentHash: "abc123hash",
		},
	}
}

// WithReferenceID sets a custom reference ID.
func (f *ConnectInputFactory) WithReferenceID(refID string) *ConnectInputFactory {
	f.input.ReferenceID = refID
	return f
}

// WithSkillID sets a custom skill ID.
func (f *ConnectInputFactory) WithSkillID(skillID string) *ConnectInputFactory {
	f.input.SkillID = skillID
	return f
}

// WithSkillVersion sets a custom skill version.
func (f *ConnectInputFactory) WithSkillVersion(version string) *ConnectInputFactory {
	f.input.SkillVersion = version
	return f
}

// WithSkillContentHash sets a custom skill content hash.
func (f *ConnectInputFactory) WithSkillContentHash(hash string) *ConnectInputFactory {
	f.input.SkillContentHash = hash
	return f
}

// Build returns the configured ConnectInput.
func (f *ConnectInputFactory) Build() skill.ConnectInput {
	return f.input
}
