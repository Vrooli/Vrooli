# Architecture Overview

Knowledge Observatory is the control-plane for Vrooli's semantic memory. It indexes scenario documentation, surfaces search/graph/health views, and maintains collection and operational metadata.

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
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐       │
│  │  Search  │ │  Graph   │ │ Metrics  │ │ Collection   │       │
│  │ Service  │ │ Service  │ │Materialzr│ │ Maintenance  │       │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────────┘       │
│       │             │            │             │                 │
│  ┌────▼─────┐ ┌─────▼────┐ ┌────▼─────┐ ┌────▼─────┐          │
│  │ DocHealth│ │ DocSearch│ │ Explorer │ │DeepSearch│           │
│  │ Service  │ │ Service  │ │ + Viewer │ │ + Healing│           │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘           │
│       │             │            │             │                 │
│  ┌────▼─────────────▼────────────▼─────────────▼──────────────┐ │
│  │               Ports (Interface Layer)                      │ │
│  │  VectorStore │ Embedder │ MetadataStore │ JobStore          │ │
│  └──────┬──────────┬───────────────┬─────────────┬────────────┘ │
└─────────┼──────────┼───────────────┼─────────────┼──────────────┘
          │          │               │             │
    ┌─────▼───┐ ┌───▼────┐  ┌──────▼──────┐  ┌───▼──────────────┐
    │ Qdrant  │ │ Ollama │  │ SQLite      │  │ agent-manager    │
    │(vectors)│ │(embed) │  │ (state)     │  │ prompt-manager   │
    └─────────┘ └────────┘  └─────────────┘  └──────────────────┘
```

### System Surfaces
- **API**: REST domain routes plus typed Connect services for revisioned source workflows and documentation health. [CODE: api/server.go] [CODE: api/knowledge_base.go]
- **UI**: Operator dashboard for search, graph, metrics. [CODE: ui/src/main.tsx]
- **CLI**: Thin wrapper over the API for terminal workflows. [CODE: cli/app.go]

## Domain Map

| Domain | Surface(s) | Primary Archetype | Secondary Traits | Source Paths | Notes |
|---|---|---|---|---|---|
| collection-maintenance | API, UI | Service | Mutation, reporting | `path:scenarios/knowledge-observatory/api/collections.go`, `path:scenarios/knowledge-observatory/api/knowledge_maintenance.go`, `path:scenarios/knowledge-observatory/api/internal/ingest`, `path:scenarios/knowledge-observatory/ui/src/surfaces/metrics` | Inspects collection records and provenance, diagnoses collection health, and performs bounded stale-chunk/deduplication maintenance. Qdrant owns vectors; scenario SQLite keeps ingest provenance. |
| search | API, UI, CLI | Reporting / query | Integration/client | `path:scenarios/knowledge-observatory/api/search.go`, `path:scenarios/knowledge-observatory/api/search_surface.go`, `path:scenarios/knowledge-observatory/api/docs_search.go`, `path:scenarios/knowledge-observatory/api/internal/aisearch`, `path:scenarios/knowledge-observatory/api/internal/services/docsearch`, `path:scenarios/knowledge-observatory/ui/src/surfaces/search` | File/text/semantic/unified search over indexed scenario docs and knowledge sources. Search reads these stores; it does not own their durable contents. |
| documentation-indexing | API, CLI | Orchestration | Scheduled/background, integration/client | `path:scenarios/knowledge-observatory/api/search_surface.go`, `path:scenarios/knowledge-observatory/api/internal/aisearch`, `path:scenarios/knowledge-observatory/cli/domains/knowledge/docsearch_cmd.go`, `path:scenarios/knowledge-observatory/.vrooli/search.json` | Reconciles the scenario documentation corpus into the `vrooli-docs` collection; source scenarios remain owners of the indexed documents. |
| knowledge-base | API (Connect-RPC), CLI | Reporting / query | Policy / rules | `path:scenarios/knowledge-observatory/api/knowledge_base.go`, `path:scenarios/knowledge-observatory/api/internal/knowledgebase` | Read-only source search, revision-pinned inspection, and bounded duplicate/backlink review. Review reports explicit scan gaps; it does not establish repository-wide closure or authorize edits. See the [API contract](../reference/api-endpoints.md#knowledge-base-connect-service) and [repository proto schema](../../../../packages/proto/schemas/knowledge-observatory/v1/knowledge_base.proto). |
| deep-search | API, UI | Orchestration | Integration/client, temporal workflow | `path:scenarios/knowledge-observatory/api/docs_deep_search.go`, `path:scenarios/knowledge-observatory/api/internal/services/deepsearch`, `path:scenarios/knowledge-observatory/api/internal/deepsearch`, `path:scenarios/knowledge-observatory/api/internal/adapters/agentmanager` | Agent-backed search jobs with polling; job state is persisted in scenario SQLite. |
| graph | API, UI | Reporting / query | Integration/client | `path:scenarios/knowledge-observatory/api/graph.go`, `path:scenarios/knowledge-observatory/api/internal/services/graph` | Materializes concept graph data from vector similarity. |
| metrics | API, UI | Reporting / query | Scheduled/background work | `path:scenarios/knowledge-observatory/api/metrics.go`, `path:scenarios/knowledge-observatory/api/internal/quality` | Samples Qdrant collections and persists quality metrics and collection rollups in scenario SQLite. |
| documentation-health | API (Connect-RPC), CLI | Policy / rules | Configuration/settings | `path:scenarios/knowledge-observatory/api/handlers/dochealth`, `path:scenarios/knowledge-observatory/api/internal/doccontract`, `path:scenarios/knowledge-observatory/api/internal/doctemplates`, `path:scenarios/knowledge-observatory/api/internal/docvalidation`, `path:scenarios/knowledge-observatory/api/internal/doclogs`, `path:scenarios/knowledge-observatory/api/internal/services/dochealth` | Single source of truth for scenario documentation health. The `KnowledgeObservatoryService.DocHealth` RPC runs structural placement + markdown/mermaid/links/paths + bidirectional refs + command snippets + manifest coverage in one call. Explicit `cli:` marked references and conservative Vrooli-owned fenced shell snippets are delegated to CLI Health for command validation; Knowledge Observatory owns docs policy and reporting, not command catalogs. All callers (knowledge-observatory CLI, test-genie docs phase, CI hooks) consume this surface — no inline validators live elsewhere. Reset is a separate REST endpoint (`api/docs_reset.go`) until its own RPC migration. |
| documentation-operations | API, UI, CLI | Service | Mutation, validation | `path:scenarios/knowledge-observatory/api/docs_explorer.go`, `path:scenarios/knowledge-observatory/api/docs_entry.go`, `path:scenarios/knowledge-observatory/api/docs_viewer.go`, `path:scenarios/knowledge-observatory/api/docs_audit.go`, `path:scenarios/knowledge-observatory/api/docs_reset.go`, `path:scenarios/knowledge-observatory/api/docs_template.go`, `path:scenarios/knowledge-observatory/api/internal/services/explorer`, `path:scenarios/knowledge-observatory/api/internal/services/viewer`, `path:scenarios/knowledge-observatory/cli/domains/docs` | Lists and reads scenario documentation, exposes templates and audit, and supports append/reset workflows. Scenario documentation files remain owned by their source scenarios. File/text search is in `search`; healing is in `documentation-healing`. |
| documentation-healing | API, UI | Orchestration | Integration/client, approvals | `path:scenarios/knowledge-observatory/api/docs_heal.go`, `path:scenarios/knowledge-observatory/api/internal/services/dochealing`, `path:scenarios/knowledge-observatory/api/internal/dochealing` | Coordinates agent-backed documentation fixes and approval workflow; job and approval state is persisted in scenario SQLite. |

## Shared Infrastructure

| Package/Folder | Purpose | Why Not Domain-Owned | Consumers |
|---|---|---|---|
| `path:scenarios/knowledge-observatory/api/server.go` | API runtime, configuration defaults, routing, CORS, dependency wiring. | Cross-cutting bootstrap/runtime layer for all API domains. | All HTTP handlers and services. |
| `path:scenarios/knowledge-observatory/api/internal/ports` | Vector store, embedder, metadata compatibility interface, and documentation access log contracts. | Integration seams are shared by multiple service domains. | Collection maintenance, search, graph, metrics. |
| `path:scenarios/knowledge-observatory/api/internal/adapters` | Concrete adapters for Qdrant, Ollama, agent-manager, and prompt-manager; scenario data uses domain-owned SQLite repositories. | External integrations are reusable infrastructure behind ports. | Service domains through ports/client seams. |
| `path:scenarios/knowledge-observatory/ui/src/shared` | Shared UI controllers, API helpers, selectors, and primitives. | Cross-surface UI mechanics, not one feature's product behavior. | Dashboard, search, graph, metrics, explorer surfaces. |

## Architecture Maturity

| Surface | Level | Evidence | Remaining Drift |
|---|---|---|---|
| API | 3 | Services, ports, and adapters are separated; bootstrap is isolated in `api/server.go`. | Some domain entrypoints remain flat top-level handler files rather than capability-owned handler packages. |
| UI | 3 | Major surfaces are named by user-facing capability and share common primitives/controllers. | Feature boundaries should stay aligned as new panels/routes are added. |
| CLI | 3 | CLI delegates to API endpoints and keeps destructive policy server-side. | Domain command organization should be preserved as commands expand. |
| Docs | 4 | Architecture, seams, and problems now hold architecture memory without a standalone audit report. | Keep domain map current when capabilities are added or retired. |

---

## Collection Maintenance Flow

Knowledge Observatory does not expose the former generic record/document
upsert or ingest-job endpoints. Documentation indexing is handled by the
scenario's configured indexer; this API inspects indexed collections and
provides bounded maintenance operations.

```text
Collection inventory ──► provenance and size signals
         │
         ├──► diagnostics ──► sampled vectors + ingest history
         ├──► prune stale chunks ──► dry-run or bounded deletion
         └──► dedupe content ──► dry-run or bounded deletion
```

Collection vectors remain in Qdrant. Ingest provenance is held in scenario
SQLite and is cleared alongside collection metadata when a collection is
deleted. The active routes and request contracts are in the
[API reference](../reference/api-endpoints.md#collection-inventory).

[CODE: api/collections.go]
[CODE: api/knowledge_maintenance.go]
[CODE: api/internal/ingest/sqlite.go]

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
                         ──► Persist job in scenario SQLite
```

- **Semantic search** logs queries in `search_history` for the activity feed.
- **Deep search** is async — returns a `job_id` and results are polled via `GET /api/v1/docs/search/deep/{job_id}`.

[CODE: api/search.go]
[CODE: api/internal/aisearch/search.go]
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
│ in SQLite    │  in knowledge_relationships
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
   │  Persist to scenario SQLite         │
   │  • quality_metrics table            │
   │  • knowledge_relationships edges    │
   │  • dashboard_metrics view refresh   │
   └─────────────────────────────────────┘
```

[CODE: api/metrics.go]

---

## Documentation Health Flow

Validates scenario documentation layout against the scenario's resolved
documentation contract.

```
Scenario name
       │
       ▼
┌──────────────────┐
│  Resolve scenario│  Filesystem path via VROOLI_SCENARIOS_ROOT
│  root directory  │  + .vrooli/service.json template provenance
└──────┬───────────┘
       │
       ▼
┌──────────────────┐    Contract source:
│  Resolve docs    │    scenario docs/manifest.json when present,
│  contract        │    otherwise template docs/manifest.json
└──────┬───────────┘    with react-vite fallback
       │
       ▼
┌──────────────────┐
│  Validate files  │    Paths, aliases, required docs,
│  against         │    content rules, append-log operations,
│  manifest rules  │    unregistered docs, temporary artifacts
└──────┬───────────┘
       ▼
┌──────────────────┐
│  Generate report │    • health_score (0-1)
│  • missing docs  │    • contract/content findings
│  • extra docs    │    • misplaced docs (wrong path)
│  • warnings      │    • reset/add support from appendLog
└──────┬───────────┘
       │
       ▼
  Health response ──► UI Explorer + Dashboard
```

[CODE: api/handlers/dochealth/handler.go]
[CODE: api/internal/doccontract/manifest.go]
[CODE: api/internal/docvalidation/validation.go]
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
- Job and approval state are tracked in the scenario database.

[CODE: api/docs_heal.go]
[CODE: api/internal/services/dochealing/service.go]
[CODE: api/internal/adapters/agentmanager/dochealing_client.go]
[CODE: api/internal/dochealing/sqlite.go]

---

## Documentation Explorer and content API

Explorer owns scenario selection, the document tree, and the inline code/preview
viewer. The legacy `#/viewer` URL redirects to Explorer; no standalone viewer
page is mounted. The API content and reset endpoints remain available to CLI
workflows. Append-log reset is documented in the [CLI reference](../reference/cli-commands.md#docs-reset).

[CODE: api/docs_explorer.go]
[CODE: api/internal/services/explorer/tree.go]
[CODE: api/docs_viewer.go]
[CODE: api/internal/services/viewer/content.go]
[CODE: ui/src/surfaces/explorer/ExplorerPage.tsx]
[CODE: ui/src/surfaces/explorer/components/DocViewer.tsx]
[CODE: ui/src/shared/controllers/viewerController.ts]
[CODE: ui/src/App.tsx]

---

## Database Schema

The API opens one scenario-scoped SQLite file through the Vrooli storage
resolver. Domain repositories and their DDL are colocated under
`api/internal/<domain>/`; `api/internal/modules/registry.go` assembles the
schema set at startup. The domain-owned schema files are authoritative. The
[data ownership page](DATA.md) summarizes persistence responsibilities without
copying table layouts.

[CODE: api/database.go]
[CODE: api/internal/modules/registry.go]

---

## UI Surface

The route parser and `App` own the hash route table. Collection details are a
child of Metrics; the Viewer URL remains a compatibility redirect into Explorer.

| Route | Page | Purpose |
|-------|------|---------|
| `/#/` | Dashboard | Quick search, health cards, activity feed |
| `/#/search` | Search | 5-mode search workspace |
| `/#/explorer` | Explorer | Scenario browser + doc tree + healing |
| `/#/viewer` | Legacy redirect | Opens Explorer; no separate viewer route |
| `/#/graph` | Graph | Force-directed knowledge graph (D3/canvas) |
| `/#/metrics` | Metrics | Collection health sparklines |
| `/#/collections/<name>` | Collection details | Collection records and provenance |
| `/#/maintenance` | Maintenance | Bounded cleanup queue |

[CODE: ui/src/surfaces/dashboard/DashboardPage.tsx]
[CODE: ui/src/surfaces/search/SearchPage.tsx]
[CODE: ui/src/surfaces/explorer/ExplorerPage.tsx]
[CODE: ui/src/App.tsx]
[CODE: ui/src/shared/controllers/routeController.ts]
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

The dependency roles and failure boundaries are maintained in
[INTEGRATIONS.md](INTEGRATIONS.md). This architecture map focuses on domain
ownership and runtime composition; the integration page avoids a second copy of
the dependency table.

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
