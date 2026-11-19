// Package capabilities declares the scenario dependencies exposed by the
// generated scenario's machine-readable capability surface.
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
		ID: "ai-gateway", Name: "AI Gateway",
		Description:    "Provides governed classification, embedding, and summary inference for source-ledger enrichment and compaction.",
		DependencyKind: capabilityregistry.DependencyScenario, DependencySlug: "ai-gateway",
		ActionKind: ActionKindScenarioStart, ActionLabel: "Start AI Gateway",
		OperatorCommand: "vrooli scenario start ai-gateway --json",
		Criticality:     capabilityregistry.CriticalityRequired,
		Enabled:         true,
		Required:        true,
		StartupPolicy:   "try_start",
		Platform:        capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformDegraded, Reason: "The append-only ledger remains durable, while enrichment and compaction pause until ai-gateway is available."},
	},
	{
		ID: "search-hub", Name: "Search Hub",
		Description:    "Optional federated search registration for source-ledger corpus discovery.",
		DependencyKind: capabilityregistry.DependencyScenario, DependencySlug: "search-hub",
		ActionKind: ActionKindScenarioStart, ActionLabel: "Start Search Hub",
		OperatorCommand: "vrooli scenario start search-hub --json",
		Enabled:         true,
		Required:        false,
		StartupPolicy:   "try_start",
		Features:        []string{"federated-search"},
		Platform:        capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformDegraded, Reason: "Source-ledger recall remains available; only external search registration is unavailable."},
	},
}

func NewRegistry() *Registry {
	return capabilityregistry.New(Known, map[string]capabilityregistry.Checker{
		"ai-gateway": capabilityregistry.ScenarioStatusChecker{Scenario: "ai-gateway"},
		"search-hub": capabilityregistry.ScenarioStatusChecker{Scenario: "search-hub"},
	}, 5*time.Second)
}
