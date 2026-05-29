package audit

import (
	"architecture-cartographer/internal/module"

	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit/audit_v1connect"
)

// Endpoints describes the audit domain's Connect-RPC routes.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "audit.run",
		Path:        audit_v1connect.AuditServiceRunProcedure,
		Method:      "POST",
		Summary:     "Run a CI-shaped drift audit",
		Description: "Orchestrates graph extract (if needed), domains derivation, and conflicts detection; applies severity / type filters; returns a deterministic summary and exit code mapping.",
		Category:    "audit",
		CLIMapping:  &module.CLIMapping{Command: "architecture-cartographer audit run"},
	},
	{
		ID:          "audit.run-all",
		Path:        audit_v1connect.AuditServiceRunAllProcedure,
		Method:      "POST",
		Summary:     "Sweep every discoverable scenario",
		Description: "Walks scenarios/*/.vrooli/service.json, runs Audit on each, and returns per-scenario reports plus a totals rollup. Honors include_scenarios / exclude_scenarios filters.",
		Category:    "audit",
		CLIMapping:  &module.CLIMapping{Command: "architecture-cartographer audit run-all"},
	},
}
