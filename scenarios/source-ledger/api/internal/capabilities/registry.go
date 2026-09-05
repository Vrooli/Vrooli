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
		Platform:        capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformDegraded, Reason: "optional audio capability depends on the selected provider and host media path"},
	},
}

type ScenarioChecker struct{ Slug string }

func (c ScenarioChecker) Check(context.Context) (capabilityregistry.Status, string) {
	output, err := exec.Command("vrooli", "scenario", "status", c.Slug, "--json").Output()
	if err != nil {
		return capabilityregistry.StatusUnavailable, c.Slug + " is unavailable; start it with the operator action"
	}
	var payload struct {
		Scenario struct {
			Status string `json:"status"`
		} `json:"scenario"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return capabilityregistry.StatusUnavailable, c.Slug + " status was not valid JSON"
	}
	if strings.EqualFold(payload.Scenario.Status, "running") || strings.EqualFold(payload.Scenario.Status, "healthy") {
		return capabilityregistry.StatusAvailable, c.Slug + " is healthy"
	}
	return capabilityregistry.StatusUnavailable, c.Slug + " is not running; start it with the operator action"
}

func NewRegistry() *Registry {
	return capabilityregistry.New(Known, map[string]capabilityregistry.Checker{
		"ai-gateway":  ScenarioChecker{Slug: "ai-gateway"},
		"search-hub":  ScenarioChecker{Slug: "search-hub"},
		"audio-tools": ScenarioChecker{Slug: "audio-tools"},
	}, 5*time.Second)
}
