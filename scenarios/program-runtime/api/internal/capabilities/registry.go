// Package capabilities declares the optional scenario dependencies exposed by
// the generated scenario's machine-readable capability surface.
package capabilities

import (
	"time"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

type (
	Def      = capabilityregistry.Def
	Registry = capabilityregistry.Registry
)

const (
	ActionKindScenarioStart = capabilityregistry.ActionKindScenarioStart
)

var Known = []Def{
	{
		ID: "audio-tools", Name: "Audio Tools",
		Description:    "Optional shared voice input and audio output for this scenario.",
		DependencyKind: capabilityregistry.DependencyScenario, DependencySlug: "audio-tools",
		ActionKind: ActionKindScenarioStart, ActionLabel: "Start Audio Tools",
		OperatorCommand: "vrooli scenario start audio-tools --json",
		Features:        []string{"voice-input", "voice-output"},
		Platform:        capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformDegraded, Reason: "voice capability depends on the declared audio provider and host media path"},
	},
}

func NewRegistry() *Registry {
	checkers := make(map[string]capabilityregistry.Checker)
	for _, def := range Known {
		if def.DependencyKind == capabilityregistry.DependencyScenario && def.DependencySlug != "" {
			checkers[def.ID] = capabilityregistry.ScenarioStatusChecker{Scenario: def.DependencySlug}
		}
	}
	return capabilityregistry.New(Known, checkers, 5*time.Second)
}
