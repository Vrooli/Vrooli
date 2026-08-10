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

const (
	DependencyScenario      = capabilityregistry.DependencyScenario
	ActionKindScenarioStart = capabilityregistry.ActionKindScenarioStart
	StatusAvailable         = capabilityregistry.StatusAvailable
	StatusUnavailable       = capabilityregistry.StatusUnavailable
)

// Known is the public integration contract for scenario-to-desktop's declared
// scenario dependencies. It is also the source fallback used when the API is
// not running, so keep each declared dependency represented here.
var Known = []Def{
	{
		ID:              "agent-manager",
		Name:            "Agent Manager",
		Description:     "Agent orchestration for pipeline investigations.",
		DependencyKind:  DependencyScenario,
		DependencySlug:  "agent-manager",
		ActionKind:      ActionKindScenarioStart,
		ActionLabel:     "Start Agent Manager",
		OperatorCommand: "vrooli scenario start agent-manager --json",
	},
	{
		ID:              "deployment-manager",
		Name:            "Deployment Manager",
		Description:     "Bundle manifest generation for bundled desktop runtimes.",
		DependencyKind:  DependencyScenario,
		DependencySlug:  "deployment-manager",
		ActionKind:      ActionKindScenarioStart,
		ActionLabel:     "Start Deployment Manager",
		OperatorCommand: "vrooli scenario start deployment-manager --json",
	},
	{
		ID:              "vrooli-bridge",
		Name:            "Vrooli Bridge",
		Description:     "Trusted dispatch and target identity for optional remote desktop validation.",
		DependencyKind:  DependencyScenario,
		DependencySlug:  "vrooli-bridge",
		ActionKind:      ActionKindScenarioStart,
		ActionLabel:     "Start Vrooli Bridge",
		OperatorCommand: "vrooli scenario start vrooli-bridge --json",
	},
}

// NewRegistry is the single construction seam for the API capability
// description. The control plane is the authority for dependency reachability;
// this package does not reach into another scenario's process or private API.
func NewRegistry() *Registry {
	checkers := make(map[string]Checker, len(Known))
	for _, def := range Known {
		checkers[def.ID] = ScenarioChecker{Slug: def.DependencySlug}
	}
	return capabilityregistry.New(Known, checkers, 30*time.Second)
}

type ScenarioChecker struct {
	Slug string
	Run  func(context.Context, string, ...string) ([]byte, error)
}

func (c ScenarioChecker) Check(ctx context.Context) (Status, string) {
	slug := strings.TrimSpace(c.Slug)
	if slug == "" {
		return StatusUnavailable, "scenario slug is not configured"
	}
	run := c.Run
	if run == nil {
		run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).Output()
		}
	}
	out, err := run(ctx, "vrooli", "scenario", "status", slug, "--json")
	if err != nil {
		return StatusUnavailable, "scenario status unavailable; use the start action"
	}
	body := strings.ToLower(string(out))
	if strings.Contains(body, `"healthy"`) || strings.Contains(body, `"running"`) {
		return StatusAvailable, "scenario is healthy"
	}
	return StatusUnavailable, "scenario is installed but stopped"
}
