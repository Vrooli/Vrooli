package domains

import (
	"architecture-cartographer/internal/module"

	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains/domains_v1connect"
)

// Endpoints describes the domains domain's Connect-RPC routes. Paths
// reference the generated *Procedure constants so validateTransport
// recognizes them as Connect routes.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "domains.extract",
		Path:        domains_v1connect.DomainsServiceExtractDomainsProcedure,
		Method:      "POST",
		Summary:     "Derive the domain map for a scenario",
		Description: "Walks the domain-extraction ladder (DOMAINS.md → api/internal folders → cli groups) and returns the canonical derived domain map with per-source provenance.",
		Category:    "domains",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart domains extract"},
	},
	{
		ID:          "domains.show",
		Path:        domains_v1connect.DomainsServiceGetDomainMapProcedure,
		Method:      "POST",
		Summary:     "Show the derived domain map",
		Description: "Returns the derived domain map for a scenario (re-derived per call; deterministic).",
		Category:    "domains",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart domains show"},
	},
	{
		ID:          "domains.convergence",
		Path:        domains_v1connect.DomainsServiceConvergenceReportProcedure,
		Method:      "POST",
		Summary:     "Report cross-surface domain convergence",
		Description: "Reports where a scenario's surfaces (DOMAINS.md, api folders, cli groups, ui features) disagree on the domain set. Advisory, never a hard gate.",
		Category:    "domains",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart domains convergence"},
	},
}
