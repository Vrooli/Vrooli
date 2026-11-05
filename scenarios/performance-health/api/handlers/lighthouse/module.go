package lighthouse

import (
	"log"

	internallh "performance-health/internal/lighthouse"
	"performance-health/internal/module"

	"github.com/gorilla/mux"
	lighthousev1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/lighthouse"
	lighthouseconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/lighthouse/lighthouse_v1connect"
)

// ProtoFile is the FileDescriptor backing the Connect-mounted LighthouseService.
var ProtoFile = lighthousev1.File_performance_health_v1_lighthouse_lighthouse_proto

// Module mounts the LighthouseService backed by the real Lighthouse CLI runner
// (its own Chrome — NOT via BAS). It scores a scenario's configured pages
// against per-page thresholds from .vrooli/lighthouse.json, silently skipping
// when the CLI is absent or there is no resolvable UI URL.
func Module(logger *log.Logger, repoRoot string) module.Module {
	svc := internallh.NewService(&internallh.CLIRunner{RepoRoot: repoRoot})
	handler := NewHandler(svc, logger)
	path, connectHandler := lighthouseconnect.NewLighthouseServiceHandler(handler)
	return module.Module{
		Name: "lighthouse",
		Mount: func(r *mux.Router) {
			r.PathPrefix(path).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the empty schema: lighthouse owns no database tables.
func Schema() string { return "" }

// Endpoints is the static endpoint metadata for codegen and the parity test.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "lighthouse_run_lighthouse",
		Path:        lighthouseconnect.LighthouseServiceRunLighthouseProcedure,
		Method:      "POST",
		Summary:     "Score a scenario's UI with Lighthouse",
		Description: "Wraps the Lighthouse CLI to score a scenario's pages against per-page thresholds; silently skips when the CLI is absent or there is no UI URL.",
		Category:    "lighthouse",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "outcome": "LighthouseOutcome", "pages": "array<PageScore>", "reason": "string"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario is missing"}, {Status: 500, Code: "internal", Description: "Lighthouse run failure"}},
	},
}
