// Package capabilities declares the optional scenario dependencies exposed by
// the generated scenario's machine-readable capability surface.
package capabilities

import capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"

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

// Known is intentionally empty. Offer Desk owns its graph and does not probe
// optional scenario processes through shell commands; dependency lifecycle is
// the control plane's responsibility.
var Known = []Def{}

func NewRegistry() *Registry {
	return capabilityregistry.New(Known, nil, 0)
}
