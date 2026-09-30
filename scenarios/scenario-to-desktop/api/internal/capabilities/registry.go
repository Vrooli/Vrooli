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

type RegistryMetadata struct {
	Platform capabilityregistry.PlatformVerdict `json:"platform"`
}

var Metadata = RegistryMetadata{Platform: capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformDegraded, Reason: "Desktop generation is host-neutral, but bundled artifacts and remote validation depend on the selected host OS and delivery path."}}

// Known contains optional capabilities only. service.json is the source of
// truth for declared dependencies.
var Known = []Def{}

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
