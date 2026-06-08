// Package modules is the single registration point for the scenario's
// Connect-RPC-migrated API domains' static metadata. api/cmd/gen-endpoints
// imports it to enumerate the migrated domains' endpoint descriptors.
//
// ecosystem-manager is mid-migration: most domains still serve hand-registered
// gorilla/mux REST routes (registered the old way in pkg/server) and are NOT
// listed here. Only domains that have moved to Connect-RPC appear below. As
// each domain migrates (see docs/internal/MIGRATION-GUIDE.md) it adds its
// Endpoints to AllEndpoints and its name to MigratedDomains, and removes its
// old REST registration.
//
// MigratedDomains tells gen-endpoints which categories in the existing
// (hand-authored) .vrooli/endpoints.json to DROP — their REST entries are
// superseded by the freshly-generated Connect descriptors. Un-migrated
// domains' REST entries are preserved verbatim so `make endpoints` never
// loses documentation for routes that are still served.
package modules

import (
	"github.com/ecosystem-manager/api/internal/module"

	discoveryH "github.com/ecosystem-manager/api/handlers/discovery"
)

// AllEndpoints returns every Connect-migrated domain's static endpoint
// descriptors in a stable order (domains alphabetically).
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, discoveryH.Endpoints...)
	return out
}

// MigratedDomains returns the set of endpoint categories that have moved to
// Connect-RPC. gen-endpoints drops these categories from the preserved REST
// baseline before appending the regenerated Connect descriptors.
func MigratedDomains() []string {
	return []string{"discovery"}
}
