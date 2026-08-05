// Package capabilities is the source-of-truth integration contract for the
// scenario dependencies declared by vrooli-memory. The dependency analyzer
// reads this package when the API is unavailable; the descriptions stay
// deliberately transport-neutral so they can be reused by an API surface.
package capabilities

import "context"

type Definition struct {
	ID              string
	Description     string
	DependencyKind  string
	DependencySlug  string
	ActionKind      string
	ActionLabel     string
	OperatorCommand string
}

type Checker interface {
	Check(context.Context) (string, string)
}

var Known = []Definition{
	{
		ID:              "ai-gateway",
		Description:     "Governed inference gateway for classification, facet derivation, embeddings, and compaction summaries.",
		DependencyKind:  "scenario",
		DependencySlug:  "ai-gateway",
		ActionKind:      "scenario_start",
		ActionLabel:     "Start AI Gateway",
		OperatorCommand: "vrooli scenario start ai-gateway --json",
	},
	{
		ID:              "search-hub",
		Description:     "Optional federated semantic-search provider for cross-corpus memory discovery.",
		DependencyKind:  "scenario",
		DependencySlug:  "search-hub",
		ActionKind:      "scenario_start",
		ActionLabel:     "Start Search Hub",
		OperatorCommand: "vrooli scenario start search-hub --json",
	},
	{
		ID:              "vrooli-events",
		Description:     "Optional runtime receipt correlation source for durable memory entries.",
		DependencyKind:  "scenario",
		DependencySlug:  "vrooli-events",
		ActionKind:      "scenario_start",
		ActionLabel:     "Start Vrooli Events",
		OperatorCommand: "vrooli scenario start vrooli-events --json",
	},
	{
		ID:              "swarm-manager",
		Description:     "Optional work-record source for importing durable agent execution history into shared memory.",
		DependencyKind:  "scenario",
		DependencySlug:  "swarm-manager",
		ActionKind:      "scenario_start",
		ActionLabel:     "Start Swarm Manager",
		OperatorCommand: "vrooli scenario start swarm-manager --json",
	},
}

type ScenarioChecker struct{ Slug string }

func (c ScenarioChecker) Check(context.Context) (string, string) {
	if c.Slug == "" {
		return "unavailable", "scenario slug is not configured"
	}
	return "unknown", "scenario status is resolved by the control plane"
}
