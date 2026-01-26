# Architecture Overview

Knowledge Observatory is the control-plane for Vrooli's semantic memory. It ingests knowledge into Qdrant, surfaces search/graph/health views, and persists metadata in Postgres.

## System Surfaces
- **API**: Canonical write path + query surface. [CODE: api/server.go]
- **UI**: Operator dashboard for search, graph, metrics. [CODE: ui/src/main.tsx]
- **CLI**: Thin wrapper over the API for terminal workflows. [CODE: cli/app.go]

## Data Flow (high level)

```
Inputs (CLI/UI/HTTP)
        |
        v
Knowledge Observatory API
  |   |    |
  v   v    v
Ollama  Qdrant  Postgres
(embed) (vectors) (metadata/metrics/jobs)
```

## Ingest Flow
1. **Validate & normalize** incoming data (namespace, visibility, defaults).
2. **Embed** content via Ollama (if configured).
3. **Upsert** vectors into Qdrant (ensures collection exists).
4. **Persist metadata** and ingest history in Postgres.

[CODE: api/ingest.go]
[CODE: api/document_ingest.go]
[CODE: api/internal/services/ingest/service.go]

## Search Flow
1. **Embed** the query.
2. **Search** across Qdrant collections.
3. **Normalize** results for API/UI/CLI.

[CODE: api/search.go]
[CODE: api/internal/services/search/service.go]

## Graph Flow
1. **Embed** the center concept.
2. **Seed** nearest vectors.
3. **Optionally expand** depth with additional neighbor searches.

[CODE: api/graph.go]
[CODE: api/internal/services/graph/service.go]

## Health and Metrics Flow
- Sample vectors to compute coherence/freshness/redundancy/coverage.
- Materialize metrics and relationships into Postgres for UI display.

[CODE: api/metrics.go]

## Documentation Health Flow
1. Resolve scenario root for filesystem access.
2. Validate doc layout against docschema standards.
3. Optionally reset PROBLEMS/PROGRESS with retention rules.

[CODE: api/docs_health.go]
[CODE: api/internal/docschema/validation.go]
[CODE: api/internal/docschema/reset.go]
[CODE: api/internal/services/dochealth/service.go]

## Documentation Search Flow
1. Resolve scope (global/scenario/path) to documentation roots.
2. Glob file names for direct discovery.
3. Run full-text regex search for matching lines.
4. Optionally blend semantic search results into unified ranking.

[CODE: api/docs_search.go]
[CODE: api/internal/services/docsearch/service.go]
[CODE: api/internal/services/docsearch/glob.go]
[CODE: api/internal/services/docsearch/text.go]
[CODE: api/internal/services/docsearch/unified.go]

## Documentation Deep Search Flow
1. Resolve scope to a filesystem root (global/scenario/path).
2. Spawn an agent-manager run with the documentation-search skill.
3. Poll run events for progress and final output.
4. Parse structured JSON results (with optional Ollama coercion).
5. Persist job status/results in Postgres for retrieval.

[CODE: api/docs_deep_search.go]
[CODE: api/internal/services/deepsearch/service.go]
[CODE: api/internal/adapters/agentmanager/deepsearch_client.go]
[CODE: api/internal/adapters/deepsearchstore/postgres.go]

## Documentation Explorer Flow
1. List scenarios with documentation stats.
2. Build scenario doc tree with doc type hints and health warnings.
3. Surface per-scenario health details for missing/misplaced docs.

[CODE: api/docs_explorer.go]
[CODE: api/docs_health.go]
[CODE: api/internal/services/explorer/tree.go]
[CODE: api/internal/services/dochealth/service.go]

## Documentation Viewer Flow
1. Resolve repo-relative path with filesystem safeguards.
2. Read content + metadata (size, modified time, doc type).
3. Optionally apply reset rules for PROBLEMS/PROGRESS docs.
4. Render code/preview modes with markdown + mermaid support.

[CODE: api/docs_viewer.go]
[CODE: api/internal/services/viewer/content.go]
[CODE: api/internal/services/viewer/reset.go]
[CODE: ui/src/surfaces/viewer/ViewerPage.tsx]

## API Runtime
The API is the lifecycle-managed entrypoint with configuration + routing.

[CODE: api/main.go]
[CODE: api/server.go]

## UI Surface
The dashboard, search, graph, and metrics pages are routed by hash.

[CODE: ui/src/surfaces/dashboard/DashboardPage.tsx]
[CODE: ui/src/surfaces/search/SearchPage.tsx]
[CODE: ui/src/surfaces/explorer/ExplorerPage.tsx]
[CODE: ui/src/surfaces/viewer/ViewerPage.tsx]
[CODE: ui/src/surfaces/graph/GraphPage.tsx]
[CODE: ui/src/surfaces/metrics/MetricsPage.tsx]

## Integrations
- **Qdrant** vector operations. [CODE: api/internal/adapters/vectorstore/qdrant.go]
- **Ollama** embedding provider. [CODE: api/internal/adapters/embedder/ollama.go]
- **Postgres** metadata and job storage. [CODE: api/internal/adapters/metadatastore/postgres.go]

## Operational Targets Mapping
- Semantic search: [REQ: OT-P0-001]
- Quality metrics: [REQ: OT-P0-002]
- Graph access: [REQ: OT-P0-003]
- API endpoints: [REQ: OT-P0-004]
- CLI commands: [REQ: OT-P0-005]
- UI dashboard: [REQ: OT-P0-006]

## Future Targets
- Timeline visualization: [REQ: OT-P1-001]
- Bulk operations: [REQ: OT-P1-002]
- Scenario contribution tracking: [REQ: OT-P1-003]
- Semantic diffing: [REQ: OT-P1-004]
- Coverage gap analysis: [REQ: OT-P1-005]
- 3D graph visualization: [REQ: OT-P2-001]
- AI recommendations: [REQ: OT-P2-002]
- Export/import bundles: [REQ: OT-P2-003]
- Advanced metadata filtering: [REQ: OT-P2-004]

See also: [DOC: PRD.md#operational-targets]
