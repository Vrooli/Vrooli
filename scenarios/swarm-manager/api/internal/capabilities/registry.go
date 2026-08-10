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

var Metadata = RegistryMetadata{Platform: capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformSupported, Reason: "Swarm orchestration and durable work records are host-neutral; provider constraints are declared by their manifests."}}

// Known contains optional capabilities only. service.json is the source of
// truth for declared resource and scenario dependencies.
var Known = []Def{}

// NewRegistry is the single registry construction seam used by future API
// capability handlers. Concrete HTTP checkers remain scenario-owned.
func NewRegistry(checker Checker) *Registry {
	checkers := make(map[string]Checker, len(Known))
	for _, def := range Known {
		if def.DependencyKind == capabilityregistry.DependencyResource {
			checkers[def.ID] = ResourceChecker{Slug: def.DependencySlug}
		} else {
			checkers[def.ID] = ScenarioChecker{Slug: def.DependencySlug}
		}
	}
	if checker != nil {
		checkers["audio-tools"] = checker
	}
	return capabilityregistry.New(Known, checkers, 30*time.Second)
}

type StaticChecker struct{ Available bool }

func (c StaticChecker) Check(context.Context) (Status, string) {
	if c.Available {
		return StatusAvailable, "audio-tools is reachable"
	}
	return StatusUnavailable, "audio-tools is unavailable; use the scenario start action"
}

// ScenarioChecker is the runtime reachability seam for declared scenario
// dependencies. It deliberately uses the control plane rather than reaching
// into another scenario's process or private API.
type ScenarioChecker struct {
	Slug string
	Run  func(context.Context, string, ...string) ([]byte, error)
}

type ResourceChecker struct {
	Slug string
	Run  func(context.Context, string, ...string) ([]byte, error)
}

func (c ResourceChecker) Check(ctx context.Context) (Status, string) {
	slug := strings.TrimSpace(c.Slug)
	if slug == "" {
		return StatusUnavailable, "resource slug is not configured"
	}
	run := c.Run
	if run == nil {
		run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).Output()
		}
	}
	out, err := run(ctx, "vrooli", "resource", "status", slug, "--json")
	if err != nil {
		return StatusUnavailable, "resource status unavailable; use the operator action"
	}
	body := strings.ToLower(string(out))
	if strings.Contains(body, `"healthy"`) || strings.Contains(body, `"running"`) {
		return StatusAvailable, "resource is healthy"
	}
	return StatusUnavailable, "resource is installed but unhealthy"
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
