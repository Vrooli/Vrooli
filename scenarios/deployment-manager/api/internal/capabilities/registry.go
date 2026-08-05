// Package capabilities is the source-of-truth registry for deployment-manager's
// cross-scenario dependencies. The dependency analyzer reads this registry when
// it validates the service manifest, while the control plane remains the only
// process allowed to start or inspect another scenario.
package capabilities

import (
	"context"
	"os/exec"
	"strings"
)

type Definition struct {
	ID              string
	Description     string
	DependencyKind  string
	DependencySlug  string
	ActionKind      string
	ActionLabel     string
	OperatorCommand string
}

var Known = []Definition{
	{
		ID:              "scenario-dependency-analyzer",
		Description:     "Dependency graph facts used for deployment analysis and target fitness.",
		DependencyKind:  "scenario",
		DependencySlug:  "scenario-dependency-analyzer",
		ActionKind:      "scenario_start",
		ActionLabel:     "Start Scenario Dependency Analyzer",
		OperatorCommand: "vrooli scenario start scenario-dependency-analyzer --json",
	},
	{
		ID:              "secrets-manager",
		Description:     "Secret classification and template support for deployment planning and bundle assembly.",
		DependencyKind:  "scenario",
		DependencySlug:  "secrets-manager",
		ActionKind:      "scenario_start",
		ActionLabel:     "Start Secrets Manager",
		OperatorCommand: "vrooli scenario start secrets-manager --json",
	},
	{
		ID:              "swarm-manager",
		Description:     "Optional backlog and review task integration for approved migration work.",
		DependencyKind:  "scenario",
		DependencySlug:  "swarm-manager",
		ActionKind:      "scenario_start",
		ActionLabel:     "Start Swarm Manager",
		OperatorCommand: "vrooli scenario start swarm-manager --json",
	},
}

type Status string

const (
	StatusAvailable   Status = "available"
	StatusUnavailable Status = "unavailable"
)

// Checker is the narrow reachability seam used by dependency-health and by
// future capability endpoints. It deliberately invokes the control plane's
// public CLI rather than importing another scenario's implementation.
type Checker interface {
	Check(context.Context) (Status, string)
}

type ScenarioChecker struct {
	Slug string
}

func (c ScenarioChecker) Check(ctx context.Context) (Status, string) {
	slug := strings.TrimSpace(c.Slug)
	if slug == "" {
		return StatusUnavailable, "scenario slug is not configured"
	}
	out, err := exec.CommandContext(ctx, "vrooli", "scenario", "status", slug, "--json").Output()
	if err != nil {
		return StatusUnavailable, "scenario status unavailable; use the operator start action"
	}
	body := strings.ToLower(string(out))
	if strings.Contains(body, "healthy") || strings.Contains(body, "running") {
		return StatusAvailable, "scenario is healthy"
	}
	return StatusUnavailable, "scenario is installed but stopped"
}
