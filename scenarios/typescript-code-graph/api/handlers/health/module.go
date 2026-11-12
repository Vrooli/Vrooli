package health

import (
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/module"
)

// Module returns the health domain's contribution to the API: the
// /health and /api/v1/health routes plus the descriptor the codegen
// reads. The single handler is mounted at both paths — /health is the
// probe convention infrastructure (LB, Kubernetes) reaches for;
// /api/v1/health is what API clients use so they only have to know
// one base path.
func Module(pinger apidb.Pinger, service, version string, sidecar ...SidecarStatusProvider) module.Module {
	var sp SidecarStatusProvider
	if len(sidecar) > 0 {
		sp = sidecar[0]
	}
	h := NewHandler(Deps{Pinger: pinger, Service: service, Version: version, Sidecar: sp})
	return module.Health(h, Endpoints)
}

// Schema returns "" — health is stateless, no tables to own. The
// modules registry includes this re-export anyway so adding a stateful
// domain later is a uniform "create the file, return the SQL" pattern
// instead of "remember to also add a Schema() function."
func Schema() string { return "" }
