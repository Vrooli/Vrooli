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
		CLIMapping:  &module.CLIMapping{Command: "arch-cart audit run"},
	},
}
