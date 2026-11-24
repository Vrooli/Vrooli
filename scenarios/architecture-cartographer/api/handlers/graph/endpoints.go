package graph

import (
	"github.com/vrooli/api-core/module"

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
		ID:          "graph.retention.preview",
		Path:        graph_v1connect.GraphServicePreviewSnapshotRetentionProcedure,
		Method:      "POST",
		Summary:     "Preview reclaimable snapshot storage",
		Description: "Reports snapshot payload bytes and rows beyond the retention floor, per scenario, without deleting anything.",
		Category:    "graph",
	},
	{
		ID:          "graph.retention.apply",
		Path:        graph_v1connect.GraphServiceApplySnapshotRetentionProcedure,
		Method:      "POST",
		Summary:     "Prune snapshots beyond the retention floor",
		Description: "Keeps the newest N snapshots per scenario and returns freed database pages to the filesystem. Requires explicit confirmation.",
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
		ID:          "slice.search",
		Path:        "/api/v1/slices/search",
		Method:      "POST",
		Summary:     "Search one scenario's implementation slices",
		Description: "Ranks domain slices inside one explicit scenario scope and returns rung counts, source paths, layer edges, and the existing DERIVED/PARTIAL attestation. Uses cached graph evidence only.",
		Category:    "slice",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"query": "string (required)", "scope": "scenario:<slug> (required)", "limit": "integer (default 5, max 5)", "snapshot_id": "string (optional; must belong to scope)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"results": "array<{id,title,score,text,path,paths,scenario,domain,snapshot_id,slice,evidence_bounds}>", "evidence_bounds": "per-rung total/returned/omitted counts for files, symbols and evidence plus layer edges; slice arrays are capped at four items each", "degradations": "array<Degradation>",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Query, scenario scope, limit, or snapshot scope is invalid"},
			{Status: 503, Code: "unavailable", Description: "Domain map or requested graph snapshot is unavailable"},
		},
	},
	{
		ID: "docs-code.drift.search", Path: "/api/v1/doc-code-drift/search", Method: "POST",
		Summary:     "Compare documented API paths with source candidates",
		Description: "Compares authored Source Paths in one high-confidence DOMAINS.md with bounded Go source, then links supported Gorilla literal Methods chains and statically identified net/http ServeMux registrations (typed parameter/receiver, DefaultServeMux, or package-level Handle*) through handler → direct internal-package call → target declaration chains. Generated Connect, custom routers, dynamic routes, duplicate chains, low-confidence authored authority, parser failures, and scan caps remain UNKNOWN/PARTIAL; direct path presence does not establish semantic or runtime conformance.", Category: "docs-code",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"query": "string (required)", "scope": "scenario:<slug> (required; one scenario)", "limit": "integer (default 10, max 10)"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"results": "array<DocCodeDrift {api_status, api?: array<DriftEvidence>}>", "authority": "string", "scope": "string", "degradations": "array<string>"}},
		Errors:   []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Query, scenario scope, or result limit is invalid"}, {Status: 503, Code: "unavailable", Description: "Scenario domain map or bounded source comparison unavailable"}},
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
