package capabilities

import (
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

// NewRegistry exposes the empty optional-capability catalog. service.json is
// the source of truth for scenario dependencies.
func NewRegistry() *Registry {
	return capabilityregistry.New(Known, nil, 30*time.Second)
}
