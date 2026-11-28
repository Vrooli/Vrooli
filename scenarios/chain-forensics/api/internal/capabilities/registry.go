// Package capabilities exposes the service-manifest-backed capability registry.
package capabilities

import (
	"github.com/vrooli/api-core/discovery"
	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
	capabilitydefaults "github.com/vrooli/vrooli/packages/capability-registry-go/defaults"
)

type Registry = capabilityregistry.Registry

func NewRegistry() *Registry {
	return capabilitydefaults.MustNewManifestRegistry(discovery.ResolveScenarioURLDefaultNoCLI)
}
