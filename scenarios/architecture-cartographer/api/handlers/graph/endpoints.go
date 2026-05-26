package graph

import (
	"architecture-cartographer/internal/module"

	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph/graph_v1connect"
)

// Endpoints describes the graph domain's Connect-RPC routes.
//
// Only graph.extract carries a CLIMapping: it is the single graph RPC
// wired into the CLI today. GetGraphSnapshot, ListGraphSnapshots,
// ClearGraphSnapshots, and ExportGraph are intentionally NOT exposed as
// CLI commands yet — they are documented in cli/manifest.json::omitted
// with reasons. The parity gate (cmd/gen-endpoints) enforces the
// invariant that a CLIMapping is present iff a registered CLI command
// mirrors it, so these stay CLIMapping-free until their commands land.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "graph.extract",
		Path:        graph_v1connect.GraphServiceExtractGraphProcedure,
		Method:      "POST",
		Summary:     "Extract and persist a graph snapshot",
		Description: "Delegates to the language code-graph adapter(s), normalizes, dedupes, and persists the resulting snapshot.",
		Category:    "graph",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart graph extract"},
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
}
