package graph

import (
	"architecture-cartographer/internal/module"

	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph/graph_v1connect"
)

// Endpoints describes the graph domain's Connect-RPC routes.
//
// Only graph.extract is bound to the CLI today. GetGraphSnapshot,
// ListGraphSnapshots, ClearGraphSnapshots, and ExportGraph are intentionally
// NOT exposed as CLI commands yet — they are documented in
// cli/manifest.json::omitted with reasons. The gen-endpoints coverage gate
// enforces that each Connect procedure is either bound or omitted.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "graph.extract",
		Path:        graph_v1connect.GraphServiceExtractGraphProcedure,
		Method:      "POST",
		Summary:     "Extract and persist a graph snapshot",
		Description: "Delegates to the language code-graph adapter(s), normalizes, dedupes, and persists the resulting snapshot.",
		Category:    "graph",
	},
	{
		ID:          "graph.get",
		Path:        graph_v1connect.GraphServiceGetGraphSnapshotProcedure,
		Method:      "POST",
		Summary:     "Get a persisted snapshot by id",
		Description: "Returns the canonical-form GraphSnapshot for the supplied id.",
		Category:    "graph",
	},
	{
		ID:          "graph.list",
		Path:        graph_v1connect.GraphServiceListGraphSnapshotsProcedure,
		Method:      "POST",
		Summary:     "List persisted snapshots",
		Description: "Cursor-paginated list of persisted snapshots, optionally filtered by scenario.",
		Category:    "graph",
	},
	{
		ID:          "graph.clear",
		Path:        graph_v1connect.GraphServiceClearGraphSnapshotsProcedure,
		Method:      "POST",
		Summary:     "Clear cached snapshots",
		Description: "Removes cached snapshots for the scenario. Honors X-Dry-Run.",
		Category:    "graph",
	},
	{
		ID:          "graph.export",
		Path:        graph_v1connect.GraphServiceExportGraphProcedure,
		Method:      "POST",
		Summary:     "Export snapshot bytes",
		Description: "Returns the persisted snapshot serialized as canonical-form JSON bytes for offline analysis or fixture authoring.",
		Category:    "graph",
	},
	{
		ID:          "zones.show",
		Path:        graph_v1connect.GraphServiceGetZoneMapProcedure,
		Method:      "POST",
		Summary:     "Show the template-derived zone map",
		Description: "Classifies graph packages into manifest-backed code-layout zones and returns layering cross-check violations.",
		Category:    "zones",
	},
	{
		ID:          "slice.show",
		Path:        graph_v1connect.GraphServiceGetSliceProcedure,
		Method:      "POST",
		Summary:     "Show the implementation slice for one domain",
		Description: "Returns proto, handler, internal, CLI, and UI rungs with per-rung evidence for a derived domain.",
		Category:    "slice",
	},
	{
		ID:          "archetype.infer",
		Path:        graph_v1connect.GraphServiceInferArchetypeProcedure,
		Method:      "POST",
		Summary:     "Infer and converge each domain's archetype (Q20)",
		Description: "Infers each domain's archetype from graph signals and converges it against the declared DOMAINS.md value, reporting drift without overriding the declared archetype.",
		Category:    "archetype",
	},
}
