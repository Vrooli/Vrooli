# Architecture Overview

Knowledge Observatory is the control-plane for Vrooli's semantic memory. It ingests knowledge into Qdrant, surfaces search/graph/health views, and persists metadata in Postgres.

## System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         UI (React + Vite)                       │
│  ┌───────────┐ ┌────────┐ ┌──────────┐ ┌───────┐ ┌─────────┐  │
│  │ Dashboard  │ │ Search │ │ Explorer │ │ Graph │ │ Metrics │  │
│  │  (home)   │ │(5 mode)│ │(doc tree)│ │(force)│ │(health) │  │
│  └─────┬─────┘ └───┬────┘ └────┬─────┘ └──┬────┘ └────┬────┘  │
│        └────────────┴───────────┴──────────┴───────────┘        │
│                          REST calls                             │
└──────────────────────────────┬──────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────┐
│                        Go REST API                              │
│                                                                 │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐           │
│  │  Ingest  │ │  Search  │ │  Graph   │ │ Metrics  │           │
│  │ Service  │ │ Service  │ │ Service  │ │Materialzr│           │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘           │
│       │             │            │             │                 │
│  ┌────▼─────┐ ┌─────▼────┐ ┌────▼─────┐ ┌────▼─────┐          │
│  │ DocHealth│ │ DocSearch│ │DeepSearch│ │DocHealing│           │
│  │ Service  │ │ Service  │ │ Service  │ │ Service  │           │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘           │
│       │             │            │             │                 │
│  ┌────▼─────────────▼────────────▼─────────────▼──────────────┐ │
│  │               Ports (Interface Layer)                      │ │
│  │  VectorStore │ Embedder │ MetadataStore │ JobStore          │ │
│  └──────┬──────────┬───────────────┬─────────────┬────────────┘ │
└─────────┼──────────┼───────────────┼─────────────┼──────────────┘
          │          │               │             │
    ┌─────▼───┐ ┌───▼────┐  ┌──────▼──────┐  ┌───▼──────────────┐
    │ Qdrant  │ │ Ollama │  │ PostgreSQL  │  │ agent-manager    │
    │(vectors)│ │(embed) │  │ (metadata)  │  │ prompt-manager   │
    └─────────┘ └────────┘  └─────────────┘  └──────────────────┘
```

### System Surfaces
- **API**: Canonical write path + query surface. [CODE: api/server.go]
- **UI**: Operator dashboard for search, graph, metrics. [CODE: ui/src/main.tsx]
- **CLI**: Thin wrapper over the API for terminal workflows. [CODE: cli/app.go]

---

## Ingest Flow

Records and documents enter through the API and are embedded, stored, and tracked.

```
Document / Text
       │
       ▼
┌──────────────┐
│  Validate &  │  Namespace, visibility, defaults
│  Normalize   │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│    Chunk     │  Default: 1200 chars, 150 overlap
│  (documents) │  Max: 5000 chunks per document
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Embed via   │  768-dim vectors (nomic-embed-text)
│   Ollama     │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Upsert into │  Creates collection if needed
│    Qdrant    │  Idempotent via external_id_map
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   Persist    │  metadata + ingest_history
│  in Postgres │  content_hash (SHA256)
└──────────────┘
```

**Sync path**: `POST /api/v1/knowledge/records/upsert` (single record) or `POST /api/v1/knowledge/documents/ingest` (chunked).
**Async path**: `POST /api/v1/ingest/jobs` enqueues a job; a background runner polls at 2s intervals.

[CODE: api/ingest.go]
[CODE: api/document_ingest.go]
[CODE: api/internal/services/ingest/service.go]
[CODE: api/internal/services/ingest/chunking.go]
[CODE: api/internal/services/ingestjobs/runner.go]

---

## Search Flow

```
Query ─┬─► Semantic ──► Embed query ──► Qdrant similarity search
       │                                 ──► Filter by namespace/visibility/tags
       │                                 ──► Rank by score + threshold
       │
       ├─► File ──────► Glob pattern match against filenames
       │
       ├─► Text ──────► Regex search across file contents
       │
       ├─► Unified ───► Blend file + text + semantic results
       │                with combined ranking
       │
       └─► Deep ──────► Spawn AI agent via agent-manager
                         ──► Agent uses documentation-search skill
                         ──► Poll run events (2s interval)
                         ──► Parse structured JSON (Ollama fallback)
                         ──► Persist job in Postgres
```

- **Semantic search** logs queries in `search_history` for the activity feed.
- **Deep search** is async — returns a `job_id` and results are polled via `GET /api/v1/docs/search/deep/{job_id}`.

[CODE: api/search.go]
[CODE: api/internal/services/search/service.go]
[CODE: api/docs_search.go]
[CODE: api/internal/services/docsearch/service.go]
[CODE: api/docs_deep_search.go]
[CODE: api/internal/services/deepsearch/service.go]

---

## Knowledge Graph Flow

```
"ollama" (center concept)
       │
       ▼
┌──────────────┐
│ Embed center │
│   concept    │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Find nearest │  Seed nodes from Qdrant
│   vectors    │
└──────┬───────┘
       │
       ▼                     ┌──────────┐
┌──────────────┐   depth>0   │  Expand   │
│ Build edges  │ ──────────► │ neighbors │ ──► repeat
│  (cosine     │             │ for each  │
│  similarity) │             │   node    │
└──────┬───────┘             └──────────┘
       │
       ▼
┌──────────────┐
│ Materialize  │  Store relationship edges
│ in Postgres  │  in knowledge_relationships
└──────┬───────┘
       │
       ▼
  { nodes, edges }  ──► UI force-directed graph (D3/canvas)
```

- Nodes represent concepts (from nearest vectors).
- Edges carry a `weight` (cosine similarity) and `relationship: "semantic_similarity"`.
- Depth traversal expands outward from the center concept.

[CODE: api/graph.go]
[CODE: api/internal/services/graph/service.go]

---

## Quality Metrics Materializer

A background loop runs every 5 minutes, sampling vectors and computing health scores.

```
  ┌─────────────────────────────────────────────────┐
  │           Materializer (every 5 min)            │
  └────────────────────┬────────────────────────────┘
                       │
          ┌────────────┼────────────┐
          ▼            ▼            ▼
   ┌────────────┐ ┌─────────┐ ┌──────────┐
   │  Sample    │ │ Sample  │ │ Sample   │  ... per collection
   │ vectors    │ │ vectors │ │ vectors  │
   └─────┬──────┘ └────┬────┘ └────┬─────┘
         │              │           │
         ▼              ▼           ▼
   ┌─────────────────────────────────────┐
   │  Calculate per-collection metrics:  │
   │                                     │
   │  Coherence  = avg pairwise cosine   │
   │               similarity            │
   │  Freshness  = exponential decay     │
   │               (30-day half-life)    │
   │  Redundancy = % pairs with          │
   │               similarity > 0.95     │
   │  Coverage   = configurable          │
   │               (default 0.70)        │
   └─────────────────┬──────────────────┘
                     │
                     ▼
   ┌─────────────────────────────────────┐
   │  Persist to Postgres                │
   │  • quality_metrics table            │
   │  • knowledge_relationships edges    │
   │  • dashboard_metrics view refresh   │
   └─────────────────────────────────────┘
```

[CODE: api/metrics.go]

---

## Documentation Health Flow

Validates scenario documentation layout against 15 canonical doc types.

```
Scenario name
       │
       ▼
┌──────────────────┐
│  Resolve scenario│  Filesystem path via VROOLI_SCENARIOS_ROOT
│  root directory  │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐    Known doc types:
│  Validate layout │    README, PROBLEMS, PROGRESS, SEAMS,
│  against         │    INVARIANTS, ASSUMPTIONS, ERROR-SEMANTICS,
│  docschema       │    SECURITY-POSTURE, TEMPORAL-FLOWS,
│  standards       │    COHERENCE-NOTES, EXPERIENCE-AUDIT,
└──────┬───────────┘    QUICKSTART, ARCHITECTURE, GLOSSARY,
       │                PRD, manifest.json
       ▼
┌──────────────────┐
│  Generate report │    • health_score (0-1)
│  • missing docs  │    • misplaced docs (wrong path)
│  • extra docs    │    • warnings + auto-fix hints
│  • warnings      │
└──────┬───────────┘
       │
       ▼
  Health response ──► UI Explorer + Dashboard
```

[CODE: api/docs_health.go]
[CODE: api/internal/docschema/validation.go]
[CODE: api/internal/docschema/types.go]
[CODE: api/internal/services/dochealth/service.go]

---

## Documentation Healing Flow (Agent-Powered)

```
Scenario name ──► Validate current doc health
                       │
                       ▼
              ┌────────────────────┐
              │  Spawn agent-      │  Sandboxed run
              │  manager run with  │  documentation-health skill
              │  doc-health skill  │  from prompt-manager
              └────────┬───────────┘
                       │
                       ▼
              ┌────────────────────┐
              │  Agent proposes    │  File creates, moves,
              │  changes (diff)    │  content updates
              └────────┬───────────┘
                       │
                       ▼
              ┌────────────────────┐
              │  Estimate          │  Projected health_after
              │  projected health  │  vs health_before
              └────────┬───────────┘
                       │
            ┌──────────┴──────────┐
            ▼                     ▼
    ┌──────────────┐     ┌──────────────┐
    │   Approve    │     │   Reject     │
    │  (apply to   │     │  (discard)   │
    │  filesystem) │     │              │
    └──────────────┘     └──────────────┘
```

- Job status: `running` → `needs_review` → `approved`/`rejected`
- Tracked in `doc_heal_jobs` Postgres table.

[CODE: api/docs_heal.go]
[CODE: api/internal/services/dochealing/service.go]
[CODE: api/internal/adapters/agentmanager/dochealing_client.go]
[CODE: api/internal/adapters/dochealingstore/postgres.go]

---

## Documentation Explorer & Viewer

```
┌─────────────────┐         ┌───────────────────┐
│  Explorer Page  │         │   Viewer Page     │
│                 │         │                   │
│  Scenario list  │ click   │  Markdown render  │
│  with doc stats ├────────►│  + Mermaid        │
│  + health score │         │  + syntax hilite  │
│                 │         │                   │
│  Doc tree with  │         │  Reset support    │
│  type hints &   │         │  (PROBLEMS/       │
│  warnings       │         │   PROGRESS)       │
└─────────────────┘         └───────────────────┘
```

[CODE: api/docs_explorer.go]
[CODE: api/internal/services/explorer/tree.go]
[CODE: api/docs_viewer.go]
[CODE: api/internal/services/viewer/content.go]
[CODE: ui/src/surfaces/explorer/ExplorerPage.tsx]
[CODE: ui/src/surfaces/viewer/ViewerPage.tsx]

---

## Database Schema

```
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ quality_metrics  │  │knowledge_metadata│  │  ingest_history  │
│ ────────────── │  │ ────────────── │  │ ────────────── │
│ collection       │  │ record_id        │  │ record_id        │
│ coherence  0-1   │  │ namespace        │  │ content_hash     │
│ freshness  0-1   │  │ visibility       │  │ namespace        │
│ redundancy 0-1   │  │ tags[]           │  │ source           │
│ coverage   0-1   │  │ source           │  │ chunks_created   │
│ avg_quality (gen)│  │ created_at       │  │ created_at       │
└──────────────────┘  └──────────────────┘  └──────────────────┘

┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│   ingest_jobs    │  │deep_search_jobs  │  │  doc_heal_jobs   │
│ ────────────── │  │ ────────────── │  │ ────────────── │
│ status (pending/ │  │ query            │  │ scenario         │
│  running/done)   │  │ scope            │  │ agent_run_id     │
│ total_chunks     │  │ agent_run_id     │  │ proposed_diff    │
│ processed_chunks │  │ results (jsonb)  │  │ health_before    │
│ error_message    │  │ status           │  │ health_after     │
└──────────────────┘  └──────────────────┘  │ status           │
                                            └──────────────────┘

┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ external_id_map  │  │knowledge_relns   │  │ collection_stats │
│ ────────────── │  │ ────────────── │  │ ────────────── │
│ namespace        │  │ source_id        │  │ collection       │
│ external_id      │  │ target_id        │  │ total_points     │
│ record_id        │  │ weight           │  │ total_collections│
│ (unique pair)    │  │ relationship     │  │ last_refreshed   │
└──────────────────┘  └──────────────────┘  └──────────────────┘

┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│  search_history  │  │     alerts       │  │user_preferences  │
│ ────────────── │  │ ────────────── │  │ ────────────── │
│ query (FTS idx)  │  │ severity         │  │ user_id          │
│ collection       │  │ message          │  │ preferences      │
│ result_count     │  │ category         │  │  (jsonb)         │
│ took_ms          │  │ acknowledged     │  │                  │
└──────────────────┘  └──────────────────┘  └──────────────────┘

                 dashboard_metrics (materialized view)
```

[CODE: initialization/postgres/schema.sql]

---

## UI Surface

The dashboard, search, graph, and metrics pages are routed by hash.

| Route | Page | Purpose |
|-------|------|---------|
| `/#/dashboard` | Dashboard | Quick search, health cards, activity feed |
| `/#/search` | Search | 5-mode search workspace |
| `/#/explorer` | Explorer | Scenario browser + doc tree + healing |
| `/#/viewer` | Viewer | Markdown preview + reset controls |
| `/#/graph` | Graph | Force-directed knowledge graph (D3/canvas) |
| `/#/metrics` | Metrics | Collection health sparklines |

[CODE: ui/src/surfaces/dashboard/DashboardPage.tsx]
[CODE: ui/src/surfaces/search/SearchPage.tsx]
[CODE: ui/src/surfaces/explorer/ExplorerPage.tsx]
[CODE: ui/src/surfaces/viewer/ViewerPage.tsx]
[CODE: ui/src/surfaces/graph/GraphPage.tsx]
[CODE: ui/src/surfaces/metrics/MetricsPage.tsx]

### Unified Search Workspace
A single search surface exposes semantic, file, text, unified, and deep search modes. Mode selection persists per session and supports quick-search handoff from the dashboard.

[CODE: ui/src/shared/components/SearchModeSelector.tsx]
[CODE: ui/src/surfaces/search/DocSearchPanelContainer.tsx]

### Dashboard Intelligence Console
Quick search, documentation health, scenario coverage, and activity feed are surfaced in one control panel. Activity feed is backed by lightweight local storage tracking for searches, healing jobs, and doc resets.

[CODE: ui/src/surfaces/dashboard/components/QuickSearchPanel.tsx]
[CODE: ui/src/surfaces/dashboard/components/ActivityFeed.tsx]
[CODE: ui/src/shared/lib/activityStore.ts]

---

## Integrations

```
┌──────────────────────────────────────┐
│          Knowledge Observatory       │
│                API                   │
└──┬──────────┬──────────┬──────────┬──┘
   │          │          │          │
   ▼          ▼          ▼          ▼
┌──────┐  ┌──────┐  ┌────────┐  ┌──────────────┐
│Qdrant│  │Ollama│  │Postgres│  │agent-manager │
│      │  │      │  │        │  │prompt-manager│
└──────┘  └──────┘  └────────┘  └──────────────┘
 vectors   embed     metadata    AI agent runs
 search    768-dim   jobs        skill retrieval
 upsert    nomic-    metrics     deep search
 delete    embed-    history     doc healing
           text      relations
```

- **Qdrant**: Vector storage + similarity search. [CODE: api/internal/adapters/vectorstore/qdrant.go]
- **Ollama**: Embedding + structured output coercion. [CODE: api/internal/adapters/embedder/ollama.go]
- **Postgres**: Metadata, job state, quality metrics. [CODE: api/internal/adapters/metadatastore/postgres.go]
- **agent-manager**: Spawns agents for deep search + doc healing. [CODE: api/internal/adapters/agentmanager/client.go]
- **prompt-manager**: Provides skills (documentation-search, documentation-health). [CODE: api/internal/adapters/promptmanager/client.go]

---

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
