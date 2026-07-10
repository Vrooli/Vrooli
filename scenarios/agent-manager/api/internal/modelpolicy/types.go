// Package modelpolicy owns agent-manager's declared model inventory and
// execution-policy contract. Runner codecs own process mechanics and measured
// capabilities; they do not own curated model identifiers.
package modelpolicy

import "agent-manager/internal/domain"

const CurrentSchemaVersion = 1

type PolicyIntent string

const (
	PolicyIntentCheap PolicyIntent = "cheap"
	PolicyIntentFast  PolicyIntent = "fast"
	PolicyIntentSmart PolicyIntent = "smart"
)

func (i PolicyIntent) IsValid() bool {
	switch i {
	case PolicyIntentCheap, PolicyIntentFast, PolicyIntentSmart:
		return true
	default:
		return false
	}
}

type SelectionType string

const (
	SelectionTypeModel         SelectionType = "model"
	SelectionTypeRunnerDefault SelectionType = "runner_default"
)

type Catalog struct {
	SchemaVersion int                             `json:"schemaVersion"`
	Metadata      Metadata                        `json:"metadata"`
	DefaultPolicy string                          `json:"defaultPolicy"`
	Runners       map[domain.RunnerType]Inventory `json:"runners"`
	Policies      map[string]Policy               `json:"policies"`
}

type Metadata struct {
	CatalogID string   `json:"catalogId"`
	UpdatedAt string   `json:"updatedAt"`
	Sources   []Source `json:"sources"`
}

type Source struct {
	Name       string `json:"name"`
	Reference  string `json:"reference"`
	VerifiedAt string `json:"verifiedAt"`
}

type Inventory struct {
	Models                []Model  `json:"models"`
	SupportsRunnerDefault bool     `json:"supportsRunnerDefault"`
	DynamicModelPrefixes  []string `json:"dynamicModelPrefixes,omitempty"`
}

type Model struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type Policy struct {
	Intent     PolicyIntent `json:"intent"`
	Candidates []Candidate  `json:"candidates"`
}

type Candidate struct {
	Runner    domain.RunnerType `json:"runner"`
	Selection Selection         `json:"selection"`
}

type Selection struct {
	Type  SelectionType `json:"type"`
	Model string        `json:"model,omitempty"`
}

func (c *Catalog) Clone() *Catalog {
	if c == nil {
		return nil
	}
	clone := &Catalog{
		SchemaVersion: c.SchemaVersion,
		Metadata: Metadata{
			CatalogID: c.Metadata.CatalogID,
			UpdatedAt: c.Metadata.UpdatedAt,
			Sources:   append([]Source(nil), c.Metadata.Sources...),
		},
		DefaultPolicy: c.DefaultPolicy,
		Runners:       make(map[domain.RunnerType]Inventory, len(c.Runners)),
		Policies:      make(map[string]Policy, len(c.Policies)),
	}
	for runnerType, inventory := range c.Runners {
		clone.Runners[runnerType] = Inventory{
			Models:                append([]Model(nil), inventory.Models...),
			SupportsRunnerDefault: inventory.SupportsRunnerDefault,
			DynamicModelPrefixes:  append([]string(nil), inventory.DynamicModelPrefixes...),
		}
	}
	for name, policy := range c.Policies {
		clone.Policies[name] = Policy{
			Intent:     policy.Intent,
			Candidates: append([]Candidate(nil), policy.Candidates...),
		}
	}
	return clone
}

func (c *Catalog) ModelIDs(runnerType domain.RunnerType) []string {
	if c == nil {
		return nil
	}
	inventory, ok := c.Runners[runnerType]
	if !ok {
		return nil
	}
	ids := make([]string, 0, len(inventory.Models))
	for _, model := range inventory.Models {
		ids = append(ids, model.ID)
	}
	return ids
}
