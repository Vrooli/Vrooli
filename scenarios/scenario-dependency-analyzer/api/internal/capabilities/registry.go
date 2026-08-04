package capabilities

import "context"

// Status is the source-level integration contract used by dependency health
// when the analyzer or one of its declared providers is not running.
type Status string

const (
	StatusAvailable   Status = "available"
	StatusUnavailable Status = "unavailable"
)

type Checker interface {
	Check(context.Context) (Status, string)
}

type Def struct {
	ID              string
	Description     string
	DependencyKind  string
	DependencySlug  string
	ActionKind      string
	ActionLabel     string
	OperatorCommand string
}

var Known = []Def{
	{
		ID: "proto-health", Description: "Batch protobuf surface facts for interface graph attribution.",
		DependencyKind: "scenario", DependencySlug: "proto-health",
		ActionKind: "scenario_start", ActionLabel: "Start Proto Health",
		OperatorCommand: "vrooli scenario start proto-health --json",
	},
	{
		ID: "code-facts", Description: "Batch import facts for cross-scenario usage evidence.",
		DependencyKind: "scenario", DependencySlug: "code-facts",
		ActionKind: "scenario_start", ActionLabel: "Start Code Facts",
		OperatorCommand: "vrooli scenario start code-facts --json",
	},
}

type StaticChecker struct{ Available bool }

func (c StaticChecker) Check(context.Context) (Status, string) {
	if c.Available {
		return StatusAvailable, "dependency analysis provider is reachable"
	}
	return StatusUnavailable, "dependency analysis provider is unavailable; rerun after recovery"
}
