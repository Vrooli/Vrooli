# Seams & Architecture Boundaries

## Last Updated
2026-11-21

## Integration Seams
- **Vector store seam**: `ports.VectorStore` enables Qdrant substitution for testing. [CODE: api/internal/ports/ports.go]
- **Embedder seam**: `ports.Embedder` isolates embedding provider. [CODE: api/internal/ports/ports.go]
- **Metadata compatibility seam**: `ports.MetadataStore` fans calls into separate scenario-database repositories for metadata, ingest history, search history, quality metrics, and graph edges. [CODE: api/internal/ports/ports.go] [CODE: api/internal/modules/stores.go]
- **Agent job-store seams**: deep-search and documentation-healing each own a repository backed by the scenario database; there is no shared `ports.JobStore` or generic ingest-job queue. [CODE: api/internal/deepsearch/sqlite.go] [CODE: api/internal/dochealing/sqlite.go]
- **Scenario filesystem seam**: doc health service encapsulates filesystem access to scenario docs. [CODE: api/internal/services/dochealth/service.go]
- **External link probe seam**: `dochealth.Doer` (HTTP client interface) lets tests substitute a fake transport for external link validation; production wires `*http.Client` with timeout from staticConfig. [CODE: api/internal/services/dochealth/httpc.go] [CODE: api/internal/services/dochealth/links.go]
- **Markdown walk root seam**: validators accept a scenario root path; no `os.Getwd` inside validator code, so tests can drive any directory layout. [CODE: api/internal/services/dochealth/walk.go]
- **DocHealth Connect-RPC seam**: `KnowledgeObservatoryService.DocHealth` is the canonical documentation-health contract. See [the architecture domain map](../concepts/ARCHITECTURE.md#domain-map) for owner responsibilities and [the API reference](../reference/api-endpoints.md#documentation-health) for request details and the retired legacy REST route. REST reset remains separate. The proto schema lives at repository `packages/proto/schemas/knowledge-observatory/v1/api.proto`.
- **Mermaid parser seam**: `DiagramValidator` batches Mermaid blocks through the exact-pinned Node sidecar. A parser outage falls back to conservative classification but emits `mermaid_unverified` for every affected block; it never silently passes diagrams. `ValidateMarkdownDiagrams` exposes the same parser over Connect for plan authoring and CLI callers. [CODE: api/internal/services/dochealth/mermaid_sidecar.go] [CODE: tools/mermaid-lint/lint.mjs]
- **Documentation maturity seam**: `.vrooli/test-genie.json` owns the scenario's docs-health capability taxonomy and finding mappings. The handler converts native `DocHealthResult` findings into the shared `common.v1.MaturityAssessment`, keeping `assessment.local` for legacy consumers and `assessment.capabilities[]` for capability-aware Test Genie and CLI reports. [CODE: .vrooli/test-genie.json] [CODE: api/handlers/dochealth/handler.go]
- **Documentation search seam**: docsearch service owns file/text/unified search over documentation roots. [CODE: api/internal/services/docsearch/service.go]
- **Documentation explorer seam**: explorer service builds scenario doc trees with warnings. [CODE: api/internal/services/explorer/tree.go]
- **Documentation viewer seam**: viewer service owns safe document loading + reset operations. [CODE: api/internal/services/viewer/service.go]
- **Documentation deep search seam**: deepsearch service orchestrates agent-powered doc discovery + job tracking. [CODE: api/internal/services/deepsearch/service.go]
- **Deep search job store seam**: SQLite repository persists deep-search job state. [CODE: api/internal/deepsearch/sqlite.go]
- **Agent-manager seam**: deep search and doc-healing adapters use the generated AgentManagerService Connect client for profile reconciliation, task/run creation and reads, run events, and diff approval/rejection. Base URL injection supports real Connect transport tests; scenario discovery remains the production resolver. Typed `AuthorizationApprovalPending` details become `agentmanager.ErrAuthorizationApprovalPending` before create-run error redaction. [CODE: api/internal/adapters/agentmanager/client.go] [CODE: api/internal/adapters/agentmanager/connect_client_test.go]
- **Documentation healing seam**: dochealing service coordinates agent-manager runs, diff review, and approvals. [CODE: api/internal/services/dochealing/service.go]
- **Doc healing job store seam**: SQLite repository persists diff, job, and approval state. [CODE: api/internal/dochealing/sqlite.go]
- **Doc healing agent seam**: doc healing agent client maps diff and approval/rejection operations over the generated Agent Manager Connect contract. [CODE: api/internal/adapters/agentmanager/dochealing_client.go]
- **Prompt-manager seam**: prompt-manager client retrieves skill content for deep search. [CODE: api/internal/adapters/promptmanager/client.go]
- **Activity feed seam**: UI-local activity store isolates search/healing event tracking. [CODE: ui/src/shared/lib/activityStore.ts]
- **Search intent seam**: dashboard quick search hands off intent to search workspace. [CODE: ui/src/shared/controllers/searchIntent.ts]
- **CLI maintenance seam**: collection diagnostics, stale-chunk pruning, and deduplication are thin wrappers over API endpoints with dry-run defaults; destructive policy remains in server handlers. Search and reindex commands use the same API surface. [CODE: cli/domains/knowledge/register.go] [CODE: api/knowledge_maintenance.go] [CODE: cli/domains/knowledge/docsearch_cmd.go]

## Responsibility Zones
- **Entry/presentation**: HTTP handlers and UI pages. [CODE: api/server.go] [CODE: ui/src/surfaces/dashboard/DashboardPage.tsx]
- **CLI entry/presentation**: CLI command parsing and output formatting delegates behavior to API endpoints. [CODE: cli/app.go]
- **Coordination/orchestration**: server wiring + service construction. [CODE: api/server.go]
- **Domain rules**: documentation search/indexing, graph calculation, and collection maintenance. [CODE: api/internal/services/docsearch/service.go] [CODE: api/internal/aisearch/search.go] [CODE: api/internal/services/graph/service.go]
- **Documentation contract seam**: doccontract, doctemplates, docvalidation, and doclogs interpret scenario manifest contracts instead of hardcoding layout rules. [CODE: api/internal/doccontract/manifest.go] [CODE: api/internal/doctemplates/resolver.go] [CODE: api/internal/docvalidation/validation.go] [CODE: api/internal/doclogs/logs.go]
- **Documentation health API**: Connect-RPC handler + service. Its scope and callers are maintained in the [architecture domain map](../concepts/ARCHITECTURE.md#domain-map); the separate REST reset implementation is `api/docs_reset.go`. [CODE: api/handlers/dochealth/handler.go] [CODE: api/internal/services/dochealth/service.go]
- **Documentation search API**: handlers + docsearch service for file/text/unified search. [CODE: api/docs_search.go] [CODE: api/internal/services/docsearch/service.go]
- **Documentation explorer API**: handlers + explorer service for scenario listing and doc tree. [CODE: api/docs_explorer.go] [CODE: api/docs_search.go] [CODE: api/internal/services/explorer/tree.go]
- **Documentation viewer API**: handlers + viewer service for content and reset. [CODE: api/docs_viewer.go] [CODE: api/internal/services/viewer/content.go]
- **Documentation deep search API**: handlers + deep search service for agent-backed search. [CODE: api/docs_deep_search.go] [CODE: api/internal/services/deepsearch/service.go]
- **Documentation healing API**: handlers + healing service for agent-backed fixes. [CODE: api/docs_heal.go] [CODE: api/internal/services/dochealing/service.go]
- **Integrations/infrastructure**: Qdrant and Ollama clients sit behind adapters; domain repositories use the scenario database. [CODE: api/internal/adapters/vectorstore/qdrant.go] [CODE: api/internal/adapters/embedder/ollama.go] [CODE: api/internal/metadata/sqlite.go]
- **Cross-cutting concerns**: CORS, logging, health checks. [CODE: api/server.go]

## Decision Points
- **Visibility normalization**: enforced values (`private/shared/global`). [CODE: api/knowledge_schema.go]
- **Search defaults**: limit/threshold normalization. [CODE: api/search.go]
- **Indexing strategy**: source manifests and the configured reconciler define which documentation enters the derived search index. [CODE: api/internal/aisearch/indexer.go] [CODE: .vrooli/search.json]

## Change Axes
- Primary change axis: API contracts for search, indexing, collection maintenance, and documentation health.
- Secondary change axis: UI surface composition and data presentation.

## Observability Surface
- Health endpoint for dependency visibility. [CODE: api/server.go]
- Metrics endpoint for quality telemetry. [CODE: api/metrics.go]

## Architecture Clarity Notes
- API wiring is centralized in `api/server.go`.
- Domain services are isolated under `api/internal/services`.

## Architecture Alignment Notes

| Area | Drift | Decision | Follow-up |
|---|---|---|---|
| API bootstrap | `api/main.go` previously mixed lifecycle boot, server wiring, handlers, and integration config access. | Keep `api/main.go` minimal and keep runtime wiring/config defaults in `api/server.go`. | Preserve split as new handlers/services are added. |
| Integration config | Qdrant/Ollama/resource CLI defaults were implicit at call sites. | Centralize integration defaults and wrappers in server/runtime wiring, then expose concrete dependencies through ports/adapters. | Keep new external integrations behind ports or narrow clients. |
| Resource-qdrant execution | Calls could block indefinitely without request deadlines. | Use the centralized default-timeout wrapper for resource CLI calls. | Move timeout policy into a dedicated integration adapter if more resource CLI calls appear. |

## Exploration Log
- 2025-12-16: Split API bootstrap from server wiring; durable notes now live in `docs/concepts/ARCHITECTURE.md`, this file, and `docs/internal/PROBLEMS.md`.
