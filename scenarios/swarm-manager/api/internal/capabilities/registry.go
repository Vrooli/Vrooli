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

var Known = []Def{
	{ID: "agent-manager", Name: "Agent Manager", Description: "Sandboxed agent execution for swarm-manager dispatch.", DependencyKind: DependencyScenario, DependencySlug: "agent-manager", ActionKind: capabilityregistry.ActionKindScenarioStart, ActionLabel: "Start Agent Manager", OperatorCommand: "vrooli scenario start agent-manager --json"},
	{ID: "audio-tools", Name: "Audio Tools", Description: "Speech input and audio output for swarm-manager composer and assistant sessions.", DependencyKind: DependencyScenario, DependencySlug: "audio-tools", Features: []string{"voice-input", "voice-output"}, ActionKind: capabilityregistry.ActionKindScenarioStart, ActionLabel: "Start Audio Tools", OperatorCommand: "vrooli scenario start audio-tools --json"},
	{ID: "git-control-tower", Name: "Git Control Tower", Description: "Repository baseline and change-control operations for swarm-manager.", DependencyKind: DependencyScenario, DependencySlug: "git-control-tower", ActionKind: capabilityregistry.ActionKindScenarioStart, ActionLabel: "Start Git Control Tower", OperatorCommand: "vrooli scenario start git-control-tower --json"},
	{ID: "knowledge-observatory", Name: "Knowledge Observatory", Description: "Durable knowledge and observability support for swarm-manager.", DependencyKind: DependencyScenario, DependencySlug: "knowledge-observatory", ActionKind: capabilityregistry.ActionKindScenarioStart, ActionLabel: "Start Knowledge Observatory", OperatorCommand: "vrooli scenario start knowledge-observatory --json"},
	{ID: "plan-manager", Name: "Plan Manager", Description: "Plan execution and evidence tracking for swarm-manager.", DependencyKind: DependencyScenario, DependencySlug: "plan-manager", ActionKind: capabilityregistry.ActionKindScenarioStart, ActionLabel: "Start Plan Manager", OperatorCommand: "vrooli scenario start plan-manager --json"},
	{ID: "prompt-manager", Name: "Prompt Manager", Description: "Prompt and skill discovery support for swarm-manager.", DependencyKind: DependencyScenario, DependencySlug: "prompt-manager", ActionKind: capabilityregistry.ActionKindScenarioStart, ActionLabel: "Start Prompt Manager", OperatorCommand: "vrooli scenario start prompt-manager --json"},
	{ID: "scenario-completeness-scoring", Name: "Scenario Completeness Scoring", Description: "Completeness scoring support for swarm-manager work products.", DependencyKind: DependencyScenario, DependencySlug: "scenario-completeness-scoring", ActionKind: capabilityregistry.ActionKindScenarioStart, ActionLabel: "Start Completeness Scoring", OperatorCommand: "vrooli scenario start scenario-completeness-scoring --json"},
	{ID: "test-genie", Name: "Test Genie", Description: "Server-owned scenario validation and evidence collection.", DependencyKind: DependencyScenario, DependencySlug: "test-genie", ActionKind: capabilityregistry.ActionKindScenarioStart, ActionLabel: "Start Test Genie", OperatorCommand: "vrooli scenario start test-genie --json"},
	{ID: "visited-tracker", Name: "Visited Tracker", Description: "Visited-state tracking support for swarm-manager navigation.", DependencyKind: DependencyScenario, DependencySlug: "visited-tracker", ActionKind: capabilityregistry.ActionKindScenarioStart, ActionLabel: "Start Visited Tracker", OperatorCommand: "vrooli scenario start visited-tracker --json"},
	{ID: "ollama", Name: "Ollama", Description: "Local model inference for swarm-manager assistance.", DependencyKind: capabilityregistry.DependencyResource, DependencySlug: "ollama", ActionKind: capabilityregistry.ActionKindOwnerGuidance, ActionLabel: "Review Ollama", OperatorCommand: "vrooli resource status ollama --json"},
	{ID: "qdrant", Name: "Qdrant", Description: "Vector search storage for swarm-manager knowledge features.", DependencyKind: capabilityregistry.DependencyResource, DependencySlug: "qdrant", ActionKind: capabilityregistry.ActionKindOwnerGuidance, ActionLabel: "Review Qdrant", OperatorCommand: "vrooli resource status qdrant --json"},
}

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
