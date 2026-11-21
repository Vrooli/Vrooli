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
	Status   = capabilityregistry.Status
)

const (
	DependencyScenario      = capabilityregistry.DependencyScenario
	StatusAvailable         = capabilityregistry.StatusAvailable
	StatusUnavailable       = capabilityregistry.StatusUnavailable
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
		Platform:        capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformDegraded, Reason: "optional audio capability depends on the selected provider and host media path"},
	},
}

func NewRegistry() *Registry {
	return capabilityregistry.New(Known, map[string]capabilityregistry.Checker{
		"audio-tools": capabilityregistry.ScenarioStatusChecker{Scenario: "audio-tools"},
	}, 5*time.Second)
}
