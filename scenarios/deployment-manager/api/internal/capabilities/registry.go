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

type PlatformVerdict struct {
	Support string `json:"support,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// RegistryMetadata describes deployment planning itself. Manifest
// dependencies remain in service.json and are not repeated in Known.
type RegistryMetadata struct {
	Platform PlatformVerdict `json:"platform"`
}

var Metadata = RegistryMetadata{Platform: PlatformVerdict{Support: "supported", Reason: "Deployment planning is host-neutral; the resolver evaluates the selected delivery tier and host OS."}}

type Definition struct {
	ID              string
	Description     string
	DependencyKind  string
	DependencySlug  string
	ActionKind      string
	ActionLabel     string
	OperatorCommand string
	Platform        PlatformVerdict
}

// Known contains optional capabilities only. service.json is the source of
// truth for declared dependencies.
var Known = []Definition{}

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
