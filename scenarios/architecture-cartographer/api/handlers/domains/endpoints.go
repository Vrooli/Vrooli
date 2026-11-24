package domains

import (
	"github.com/vrooli/api-core/module"

	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains/domains_v1connect"
)

// Endpoints describes the domains owner's Connect-RPC and HTTP routes. The
// Connect paths use generated procedure constants; the scoped zone-classifier
// route is an additive JSON endpoint.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "domains.classify-zones",
		Path:        classifyZonesPath,
		Method:      "POST",
		Summary:     "Classify selected scenario paths into architecture zones",
		Description: "Classifies up to 128 existing paths from one exact scenario using its derived domain map and configured zone rules. It returns each classification basis and leaves unclassified paths UNKNOWN; it does not inventory files or assert runtime wiring.",
		Category:    "domains",
	},
	{
		ID:          "domains.extract",
		Path:        domains_v1connect.DomainsServiceExtractDomainsProcedure,
		Method:      "POST",
		Summary:     "Derive the domain map for a scenario",
		Description: "Walks the domain-extraction ladder (DOMAINS.md → api/internal folders → cli groups) and returns the canonical derived domain map with per-source provenance.",
		Category:    "domains",
	},
	{
		ID:          "domains.show",
		Path:        domains_v1connect.DomainsServiceGetDomainMapProcedure,
		Method:      "POST",
		Summary:     "Show the derived domain map",
		Description: "Returns the derived domain map for a scenario (re-derived per call; deterministic).",
		Category:    "domains",
	},
	{
		ID:          "domains.convergence",
		Path:        domains_v1connect.DomainsServiceConvergenceReportProcedure,
		Method:      "POST",
		Summary:     "Report cross-surface domain convergence",
		Description: "Reports where a scenario's surfaces (DOMAINS.md, api folders, cli groups, ui features) disagree on the domain set. Advisory, never a hard gate.",
		Category:    "domains",
	},
	{
		ID:          "domains.draft",
		Path:        domains_v1connect.DomainsServiceDraftDomainsProcedure,
		Method:      "POST",
		Summary:     "Draft a DOMAINS.md inventory",
		Description: "Proposes a docs/concepts/DOMAINS.md inventory from extracted evidence without writing authority to disk.",
		Category:    "domains",
	},
}
