// Package rolepolicy owns Agent Manager's portable role and cross-runner
// ordering. Coding-agent resources remain the authority for concrete models.
package rolepolicy

import "agent-manager/internal/domain"

const CurrentSchemaVersion = 1

// Catalog expresses portable intent only. It deliberately has no model
// inventory or native configuration details.
type Catalog struct {
	SchemaVersion int             `json:"schemaVersion"`
	Metadata      Metadata        `json:"metadata"`
	DefaultRole   string          `json:"defaultRole"`
	Roles         map[string]Role `json:"roles"`
}

type Metadata struct {
	CatalogID string `json:"catalogId"`
	UpdatedAt string `json:"updatedAt"`
}

type Role struct {
	Description    string      `json:"description"`
	Intent         string      `json:"intent"`
	OrderingReason string      `json:"orderingReason"`
	Candidates     []Candidate `json:"candidates"`
}

// Candidate selects a resource-owned role on one runner. The resource CLI
// resolves that role to concrete model evidence at run creation.
type Candidate struct {
	Runner       domain.RunnerType `json:"runner"`
	ResourceRole string            `json:"resourceRole"`
}

func (c *Catalog) Clone() *Catalog {
	if c == nil {
		return nil
	}
	clone := &Catalog{
		SchemaVersion: c.SchemaVersion,
		Metadata:      c.Metadata,
		DefaultRole:   c.DefaultRole,
		Roles:         make(map[string]Role, len(c.Roles)),
	}
	for key, role := range c.Roles {
		role.Candidates = append([]Candidate(nil), role.Candidates...)
		clone.Roles[key] = role
	}
	return clone
}
