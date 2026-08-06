// Package capabilities declares the optional scenario dependencies exposed by
// the generated scenario's machine-readable capability surface.
package capabilities

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
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
	},
	{
		ID: "vrooli-events", Name: "Vrooli Events",
		Description:    "Optional typed telemetry bus for governed program lifecycle events.",
		DependencyKind: capabilityregistry.DependencyScenario, DependencySlug: "vrooli-events",
		ActionKind: ActionKindScenarioStart, ActionLabel: "Start Vrooli Events",
		OperatorCommand: "vrooli scenario start vrooli-events --json",
		Features:        []string{"program-telemetry"},
	},
}

type ScenarioChecker struct{ Scenario string }


func (c ScenarioChecker) Check(context.Context) (capabilityregistry.Status, string) {
	scenario := c.Scenario
	if scenario == "" {
		scenario = "audio-tools"
	}
	output, err := exec.Command("vrooli", "scenario", "status", scenario, "--json").Output()
	if err != nil {
		return capabilityregistry.StatusUnavailable, scenario + " is unavailable; start it with the operator action"
	}
	var payload struct {
		Scenario struct {
			Status string `json:"status"`
		} `json:"scenario"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return capabilityregistry.StatusUnavailable, "audio-tools status was not valid JSON"
	}
	if strings.EqualFold(payload.Scenario.Status, "running") || strings.EqualFold(payload.Scenario.Status, "healthy") {
		return capabilityregistry.StatusAvailable, scenario + " is healthy"
	}
	return capabilityregistry.StatusUnavailable, scenario + " is not running; start it with the operator action"
}

func NewRegistry() *Registry {
	return capabilityregistry.New(Known, map[string]capabilityregistry.Checker{
		"audio-tools":   ScenarioChecker{Scenario: "audio-tools"},
		"vrooli-events": ScenarioChecker{Scenario: "vrooli-events"},
	}, 5*time.Second)
}
