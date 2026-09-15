// Package capabilities declares the scenario dependencies exposed by this
// scenario's machine-readable capability surface.
//
// Every entry below mirrors a dependency declared in
// `.vrooli/service.json` under `dependencies.scenarios`. Keep the two in
// step: a capability the manifest does not declare is a capability the
// lifecycle will never start, and a manifest dependency with no entry here
// is invisible to the console and to an agent reading
// `/api/v1/capabilities/describe`.
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

// Known mirrors `dependencies.scenarios` in `.vrooli/service.json`. The
// Criticality and Required fields carry the manifest's `required` value, and
// StartupPolicy carries its `startup_policy`, so a reader of the describe
// endpoint sees the same contract the lifecycle reads.
var Known = []Def{
	{
		ID: "landing-page-business-suite", Name: "Business Suite",
		Description:    "Holds the credit reservation and settlement surface. A reservation is obtained before any provider is called and settled when the instance is destroyed.",
		DependencyKind: capabilityregistry.DependencyScenario, DependencySlug: "landing-page-business-suite",
		ActionKind: ActionKindScenarioStart, ActionLabel: "Start Business Suite",
		OperatorCommand: "vrooli scenario start landing-page-business-suite --json",
		Criticality:     capabilityregistry.CriticalityRequired,
		Enabled:         true,
		Required:        true,
		StartupPolicy:   "must_start",
		Platform:        capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformSupported},
	},
	{
		ID: "vrooli-bridge", Name: "Vrooli Bridge",
		Description:    "Holds node identity, pairing and first touch. Enrollment queues and retries while it is unavailable; the instance is still created, metered and expiring.",
		DependencyKind: capabilityregistry.DependencyScenario, DependencySlug: "vrooli-bridge",
		ActionKind: ActionKindScenarioStart, ActionLabel: "Start Vrooli Bridge",
		OperatorCommand: "vrooli scenario start vrooli-bridge --json",
		Criticality:     capabilityregistry.CriticalityRequired,
		Enabled:         true,
		Required:        true,
		StartupPolicy:   "try_start",
		Platform:        capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformDegraded, Reason: "enrollment degrades to queued and retried while bridge is unavailable"},
	},
	{
		ID: "treasury", Name: "Treasury",
		Description:    "Bounds what an agent may spend on the agent-initiated capacity path. Not consulted for operator-initiated or customer-initiated requests.",
		DependencyKind: capabilityregistry.DependencyScenario, DependencySlug: "treasury",
		ActionKind: ActionKindScenarioStart, ActionLabel: "Start Treasury",
		OperatorCommand: "vrooli scenario start treasury --json",
		Criticality:     capabilityregistry.CriticalityOptional,
		Enabled:         true,
		StartupPolicy:   "ignore",
		Platform:        capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformDegraded, Reason: "agent-initiated provisioning refuses while the mandate cannot be checked; operator-initiated provisioning continues"},
	},
	{
		ID: "offer-desk", Name: "Offer Desk",
		Description:    "Holds the sellable definition of provisioned capacity. Read at publishing time only; no runtime path depends on it.",
		DependencyKind: capabilityregistry.DependencyScenario, DependencySlug: "offer-desk",
		ActionKind: ActionKindScenarioStart, ActionLabel: "Start Offer Desk",
		OperatorCommand: "vrooli scenario start offer-desk --json",
		Criticality:     capabilityregistry.CriticalityOptional,
		Enabled:         true,
		StartupPolicy:   "ignore",
		Platform:        capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformSupported},
	},
}

// ScenarioChecker probes one scenario dependency through the control plane.
// The zero value probes nothing and reports unavailable, which is the honest
// answer for a checker that was never given a slug.
type ScenarioChecker struct {
	Slug string
}

func (c ScenarioChecker) Check(context.Context) (capabilityregistry.Status, string) {
	if c.Slug == "" {
		return capabilityregistry.StatusUnavailable, "no scenario slug was configured for this checker"
	}
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

// NewRegistry pairs every declared scenario capability with a checker. A
// declaration with no checker reports nothing, which reads as healthy rather
// than as unknown, so the map is derived from Known instead of hand-listed.
func NewRegistry() *Registry {
	checkers := make(map[string]capabilityregistry.Checker, len(Known))
	for _, def := range Known {
		if def.DependencyKind != capabilityregistry.DependencyScenario || def.DependencySlug == "" {
			continue
		}
		checkers[def.ID] = ScenarioChecker{Slug: def.DependencySlug}
	}
	return capabilityregistry.New(Known, checkers, 5*time.Second)
}
