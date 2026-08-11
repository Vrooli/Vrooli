// Package capabilities exposes react-component-library's declared scenario
// integrations through the shared capability-registry contract.
package capabilities

import (
	"context"
	"os/exec"
	"strings"
	"time"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

type (
	DependencyKind = capabilityregistry.DependencyKind
	Status         = capabilityregistry.Status
	ActionKind     = capabilityregistry.ActionKind
	Def            = capabilityregistry.Def
	State          = capabilityregistry.State
	Checker        = capabilityregistry.Checker
	Registry       = capabilityregistry.Registry
)

type RegistryMetadata struct {
	Platform capabilityregistry.PlatformVerdict `json:"platform"`
}

var Metadata = RegistryMetadata{Platform: capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformSupported, Reason: "The component catalog and direct library APIs are host-neutral; optional integrations are declared in service.json."}}

const (
	DependencyScenario      = capabilityregistry.DependencyScenario
	StatusAvailable         = capabilityregistry.StatusAvailable
	StatusUnavailable       = capabilityregistry.StatusUnavailable
	ActionKindScenarioStart = capabilityregistry.ActionKindScenarioStart
)

var Known = []Def{}

func NewRegistry() *Registry {
	return capabilityregistry.New(Known, map[string]Checker{
		"agent-manager": ScenarioChecker{Slug: "agent-manager"},
	}, 30*time.Second)
}

// ScenarioChecker asks the control plane for lifecycle status rather than
// reaching into another scenario's private API.
type ScenarioChecker struct{ Slug string }

func (c ScenarioChecker) Check(ctx context.Context) (Status, string) {
	slug := strings.TrimSpace(c.Slug)
	if slug == "" {
		return StatusUnavailable, "scenario slug is not configured; use the start action"
	}
	out, err := exec.CommandContext(ctx, "vrooli", "scenario", "status", slug, "--json").Output()
	if err != nil {
		return StatusUnavailable, "scenario status unavailable; use the start action"
	}
	body := strings.ToLower(string(out))
	if strings.Contains(body, `"healthy"`) || strings.Contains(body, `"running"`) {
		return StatusAvailable, "scenario is healthy"
	}
	return StatusUnavailable, "scenario is installed but stopped; use the start action"
}
