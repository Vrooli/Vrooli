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
		ID:              "source-ledger",
		Description:     "Source of truth for the append-only journal and all derived semantic memory engines.",
		DependencyKind:  "scenario",
		DependencySlug:  "source-ledger",
		ActionKind:      "scenario_start",
		ActionLabel:     "Start Source Ledger",
		OperatorCommand: "vrooli scenario start source-ledger --json",
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
