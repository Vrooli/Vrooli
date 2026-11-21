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

// NewRegistry exposes the current empty optional-capability catalog.
func NewRegistry() *Registry {
	return capabilityregistry.New(Known, nil, 30*time.Second)
}
