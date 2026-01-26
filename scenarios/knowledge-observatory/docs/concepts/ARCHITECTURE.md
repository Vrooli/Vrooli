# Architecture Overview

Knowledge Observatory is the control-plane for Vrooli’s semantic memory. It ingests knowledge into Qdrant, surfaces search/graph/health views, and persists metadata in Postgres.

## System Surfaces
- **API**: Canonical write path + query surface. [CODE: api/server.go]
- **UI**: Operator dashboard for search, graph, metrics. [CODE: ui/src/main.tsx]
- **CLI**: Thin wrapper over the API for terminal workflows. [CODE: cli/knowledge-observatory]

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

## API Runtime
The API is the lifecycle-managed entrypoint with configuration + routing.

[CODE: api/main.go]
[CODE: api/server.go]

## UI Surface
The dashboard, search, graph, and metrics pages are routed by hash.

[CODE: ui/src/surfaces/dashboard/DashboardPage.tsx]
[CODE: ui/src/surfaces/search/SearchPage.tsx]
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

See also: [DOC: PRD.md#operational-targets]
