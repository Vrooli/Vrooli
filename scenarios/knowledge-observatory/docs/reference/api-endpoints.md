# API Endpoints

Base URL: `http://localhost:${API_PORT}` (API port is assigned by the scenario lifecycle).
Resolve it with `vrooli scenario port knowledge-observatory API_PORT`.

## REST Quick Calls

```bash
API_PORT=$(vrooli scenario port knowledge-observatory API_PORT)
curl -X POST "http://localhost:${API_PORT}/api/v1/knowledge/search" \
  -H "Content-Type: application/json" \
  -d '{"query":"agent workflows","limit":10}'
curl "http://localhost:${API_PORT}/api/v1/knowledge/health"
curl -X POST "http://localhost:${API_PORT}/api/v1/knowledge/graph" \
  -H "Content-Type: application/json" \
  -d '{"center_concept":"research","depth":3}'
```

## Knowledge Base Connect Service

`KnowledgeBaseService` exposes read-only, typed source search, inspection, review,
and index-status operations. Each unary request is sent as `POST` to
`/knowledge_observatory.v1.KnowledgeBaseService/<Method>`. The wire contract is the
repository [Knowledge Base proto](../../../../packages/proto/schemas/knowledge-observatory/v1/knowledge_base.proto).

| Method | Request fields | Result and boundary |
|---|---|---|
| `SearchDocuments` | `query` (non-empty, at most 2,000 characters), `scope` (`global`, `scenario`, or `path`), `target`, `mode` (`auto`, `hybrid`, `dense`, or `text`), `limit` (maximum 20; zero selects the default 5). | Ranked document paths, snippets, scores, metadata, search method, and reranker. |
| `InspectDocument` | Repository-relative `path`, character `offset`/`limit` (maximum 12,000; zero defaults to 4,000), optional `expectedSha256`, and `allowMissing`. | Exact source SHA-256, bounded UTF-8 content page, metadata, references, truncation flags, and a typed missing observation only when the path is absent and `allowMissing` is true. |
| `ReviewDocuments` | One to eight `paths`, a repository-relative `basePath`, and optional `maxFiles` (maximum 1,000; zero defaults to 300). | Selected source pages plus bounded duplicate/backlink observations, `truncated`, and explicit `gaps`; this is not proof of whole-repository reference closure or semantic agreement. |
| `KnowledgeStatus` | No fields. | Index availability, indexed count, last reconciliation time/outcome, and reranker. |

Paths use portable repository-relative `/` separators. Absolute paths, traversal,
Windows drive/UNC forms, symlink escapes, non-prose inspection targets, and
oversized/non-UTF-8 sources are rejected. A source revision mismatch returns a
precondition error; the caller must inspect the new revision before proceeding.
No method edits documents or authorizes retirement.

[CODE: api/knowledge_base.go]
[CODE: api/internal/knowledgebase/documents.go]

## Health
`GET /health`

Returns infrastructure health (API + dependency checks).

[CODE: api/server.go]

## Search
`POST /api/v1/knowledge/search`

**Request**
```json
{
  "query": "string",
  "collection": "string?",
  "namespaces": ["string"],
  "visibility": ["private|shared|global"],
  "tags": ["string"],
  "ingested_after": "RFC3339?",
  "ingested_before": "RFC3339?",
  "limit": 10,
  "threshold": 0.3
}
```

**Response**
```json
{
  "results": [
    {
      "id": "string",
      "score": 0.92,
      "content": "string",
      "metadata": {}
    }
  ],
  "query": "string",
  "took_ms": 12
}
```

[CODE: api/search.go]
[CODE: api/internal/aisearch/search.go]

## Documentation Search and Index Control

`POST /api/v1/search/query` searches the reconciled documentation corpus.

**Request fields**: `query` (required), `mode` (`auto`, `hybrid`, `dense`, or
`text`), `scope` (`global`, `scenario`, or `path`), `target`, `limit`, and
optional `facets` (`doc_type`, `audience`, `maturity`, `canonical_for`).

**Response fields**: `results`, `total`, `query`, `method`, and `reranker`.
`total` is the number of returned result objects (therefore no greater than
`limit`); it is not a count of every matching document in the corpus. The
vector engine searches a bounded candidate shortlist and does not provide an
exact corpus-wide match count for this endpoint.
The docs preset retrieves a bounded 240-chunk candidate pool, then keeps the
highest-ranked chunk per source document before limiting the page. Exact query
terms receive a bounded score boost so a short literal query can surface a
source that contains the requested term ahead of merely semantic neighbors.

`GET /api/v1/search/status` reports availability, Ollama/Qdrant readiness,
reranker, indexed document count, and last reconciliation outcome.

`POST /api/v1/reindex/run` plans or starts reconciliation. The body may include
`dry_run` and an optional `control_token`; an empty body starts an async run.
Dry runs return planned upsert/delete/refresh counts. Applied runs return
`running` (or `busy`) and planned counts. A presented control token must match
the token registered by search-hub; requests without a token remain available
for local operation.

`GET /api/v1/reindex/status` reports `idle`, `running`, `succeeded`, `failed`,
`cancelled`, or `unavailable`, with processed/total/deferred counts and timing.
`POST /api/v1/reindex/cancel` requests cancellation and returns whether a run
was active.

The documentation indexer is separate from knowledge collection ingest. There
is no `/api/v1/ingest/health` route; per-collection provenance and diagnostics
are available through collection inventory and diagnostics below.

[CODE: api/search_surface.go]
[CODE: api/internal/aisearch/indexer.go]

## Maintenance observation and routing

`POST /api/v1/knowledge/maintenance/inventory` returns bounded candidate records
with source hashes, owner hypotheses, artifact classifications, portability
signals, and inspect/propose handles. It is read-only and reports truncation.

`POST /api/v1/knowledge/maintenance/proposals` converts candidate records into
review-only dispositions. Every proposal remains `proposed`, cites its source
hash, names preservation obligations, and requires owner authorization.

`POST /api/v1/knowledge/maintenance/route` returns a dry-run or owner-admission
receipt. It refuses missing idempotency keys, stale source hashes, unresolved
actions, and unauthorized non-dry-run requests. It does not mutate Knowledge
Observatory or project artifacts.

## Health Metrics
`GET /api/v1/knowledge/health`

Returns collection health and aggregated quality metrics.

[CODE: api/metrics.go]

## Graph
`GET /api/v1/knowledge/graph` or `POST /api/v1/knowledge/graph`

**Request**
```json
{
  "center_concept": "string",
  "collection": "string?",
  "namespaces": ["string"],
  "visibility": ["private|shared|global"],
  "tags": ["string"],
  "depth": 1,
  "limit": 25,
  "threshold": 0.35
}
```

**Response**
```json
{
  "center": "string",
  "nodes": [{"id": "string", "label": "string", "score": 0.88}],
  "edges": [{"source": "string", "target": "string", "weight": 0.8, "relationship": "semantic_similarity"}],
  "took_ms": 25
}
```

[CODE: api/graph.go]
[CODE: api/internal/services/graph/service.go]

## Scenario List
`GET /api/v1/scenarios`

Returns documentation summary data for each scenario.

**Response**
```json
[
  {
    "name": "knowledge-observatory",
    "path": "scenarios/knowledge-observatory",
    "doc_count": 18,
    "health_score": 0.91,
    "has_manifest": true,
    "has_readme": true,
    "last_modified": "2026-01-26T03:12:00Z"
  }
]
```

[CODE: api/docs_search.go]
[CODE: api/internal/services/explorer/service.go]
[CODE: api/internal/services/dochealth/scenarios.go]

## Scenario Documentation Tree
`GET /api/v1/scenarios/{name}/docs`

Returns the documentation tree for a scenario, including doc type hints and warnings.

**Response**
```json
{
  "name": "knowledge-observatory",
  "path": "scenarios/knowledge-observatory",
  "type": "directory",
  "children": [
    {
      "name": "README.md",
      "path": "scenarios/knowledge-observatory/README.md",
      "type": "file",
      "doc_type": "readme",
      "size": 1024,
      "modified_at": "2026-01-26T03:12:00Z"
    }
  ]
}
```

[CODE: api/docs_explorer.go]
[CODE: api/internal/services/explorer/tree.go]

## Documentation Operations

The following registered REST endpoints are used by the `docs` CLI workflows.
The route registrations in `api/server.go` and their handlers are authoritative.
For terminal examples, use the [CLI reference](cli-commands.md).

### Audit and auto-fix

`GET /api/v1/scenarios/{name}/docs/audit` runs the full scenario audit and
returns `scenario_name`, `health_score`, `total_docs`, infrastructure findings,
broken or unknown marked references, orphaned docs, duplicate titles, and
undocumented targets. This is the REST audit result; the typed
`KnowledgeObservatoryService.DocHealth` contract is the canonical health check.

`POST /api/v1/scenarios/{name}/docs/autofix` accepts `{"dry_run":true}` and
returns `scenario_name`, `moved`, `skipped`, `health_before`, `health_after`,
and `dry_run`. Each moved item has `from_path`, `to_path`, and `doc_type`; each
skipped item also includes a reason. The handler's zero-value request defaults
to `dry_run:false`, so callers must send `true` to preview before applying moves.

### Typed document read and append

`GET /api/v1/scenarios/{name}/docs/{doc_type}/content` reads a manifest-owned
document by type. `format` accepts `raw` (default), `highlighted`, or `preview`.
The response uses the same content fields as `GET /api/v1/docs/content`, plus
`can_reset` and optional `reset_config` when the document type supports
append-log reset.

`POST /api/v1/scenarios/{name}/docs/{doc_type}/entries` accepts a required
`title` and optional `body`, `author`, and `status`. It appends only to a
manifest-supported append log and returns HTTP 201 with `scenario_name`,
`doc_type`, and `entry_added`. It does not accept arbitrary document paths.

### Access statistics and templates

`GET /api/v1/docs/stats` returns `{"stats":[...]}`. Optional `scenario` and
`doc_type` query parameters filter each row, which contains `scenario_name`,
`doc_type`, `read_count`, `write_count`, and `reset_count`.

`GET /api/v1/docs/templates` lists the documents in a resolved template as
`doc_type`, `expected_path`, `purpose`, and `title`. Select the source with
`template=<id>` or `scenario=<name>`. `GET /api/v1/docs/templates/{doc_type}`
returns the same metadata plus template `content`. Scenario names are resolved
against scenario manifests; these read endpoints do not apply templates.

[CODE: api/server.go]
[CODE: api/docs_audit.go]
[CODE: api/docs_heal.go]
[CODE: api/docs_entry.go]
[CODE: api/docs_template.go]

## Documentation Search
`POST /api/v1/docs/search/files`

**Request**
```json
{
  "pattern": "**/README.md",
  "scope": "global|scenario|path",
  "scenario": "knowledge-observatory",
  "base_path": "docs",
  "limit": 50,
  "include_content": true
}
```

**Response**
```json
[
  {
    "path": "scenarios/knowledge-observatory/README.md",
    "relative_path": "README.md",
    "scenario": "knowledge-observatory",
    "size": 1024,
    "modified_at": "2026-01-26T03:12:00Z",
    "doc_type": "readme",
    "content_preview": "# Knowledge Observatory"
  }
]
```

`POST /api/v1/docs/search/text`

**Request**
```json
{
  "query": "health score",
  "scope": "scenario",
  "scenario": "knowledge-observatory",
  "file_types": ["md", "json"],
  "case_sensitive": false,
  "limit": 50,
  "context_lines": 1
}
```

**Response**
```json
[
  {
    "path": "scenarios/knowledge-observatory/docs/reference/api-endpoints.md",
    "relative_path": "docs/reference/api-endpoints.md",
    "scenario": "knowledge-observatory",
    "line_number": 88,
    "content": "\"health_score\": 0.92,",
    "context_before": "{",
    "context_after": "\"total_docs\": 12,"
  }
]
```

`POST /api/v1/docs/search/unified`

**Request**
```json
{
  "query": "README.md",
  "scope": "global",
  "limit": 25,
  "use_semantic": true
}
```

**Response**
```json
{
  "results": [
    {
      "source": "file",
      "score": 0.9,
      "path": "scenarios/knowledge-observatory/README.md",
      "relative_path": "README.md",
      "scenario": "knowledge-observatory",
      "snippet": "# Knowledge Observatory",
      "doc_type": "readme"
    }
  ],
  "query": "README.md",
  "took_ms": 12
}
```

[CODE: api/docs_search.go]
[CODE: api/internal/services/docsearch/service.go]

## Documentation Deep Search
`POST /api/v1/docs/search/deep`

**Request**
```json
{
  "query": "How does deep search use agent-manager?",
  "scope": "global|scenario|path",
  "scenario": "knowledge-observatory",
  "base_path": "scenarios/knowledge-observatory/docs",
  "max_results": 10,
  "follow_refs": true,
  "timeout_seconds": 60
}
```

**Response**
```json
{
  "job_id": "uuid",
  "status": "running",
  "progress": "20% - scanning docs",
  "started_at": "2026-01-26T13:05:00Z"
}
```

`GET /api/v1/docs/search/deep/{job_id}`

**Response**
```json
{
  "job_id": "uuid",
  "status": "completed",
  "completed_at": "2026-01-26T13:05:45Z",
  "results": [
    {
      "path": "scenarios/knowledge-observatory/api/docs_deep_search.go",
      "relevance": 0.91,
      "summary": "Deep search handler wiring",
      "match_reason": "Handles /docs/search/deep endpoints",
      "references": ["scenarios/knowledge-observatory/api/internal/services/deepsearch/service.go"],
      "snippet": "handleDocsSearchDeepStatus"
    }
  ]
}
```

[CODE: api/docs_deep_search.go]
[CODE: api/internal/services/deepsearch/service.go]
[CODE: api/internal/services/deepsearch/parser.go]
[CODE: api/internal/services/deepsearch/ollama_parser.go]
[CODE: api/internal/services/docsearch/text.go]
[CODE: api/internal/services/docsearch/unified.go]

## Documentation Viewer
`GET /api/v1/docs/content`

Query params: `path` (repo-relative) and optional `format` (`raw`, `highlighted`, `preview`).

**Response**
```json
{
  "path": "scenarios/knowledge-observatory/docs/manifest.json",
  "content": "{...}",
  "format": "raw",
  "doc_type": "manifest",
  "size": 2048,
  "modified_at": "2026-01-26T03:12:00Z",
  "can_reset": false,
  "reset_config": {
    "max_age_days": 30,
    "keep_min_entries": 3
  }
}
```

`POST /api/v1/docs/reset`

**Request**
```json
{
  "path": "scenarios/knowledge-observatory/docs/internal/PROBLEMS.md",
  "max_age_days": 30,
  "keep_min_entries": 3,
  "preview_only": true
}
```

**Response**
```json
{
  "path": "scenarios/knowledge-observatory/docs/internal/PROBLEMS.md",
  "doc_type": "problems",
  "removed_count": 2,
  "kept_count": 5,
  "removed_entries": ["## 2025-10-01: Old entry"],
  "new_content": "# Problems...",
  "preview_only": true
}
```

[CODE: api/docs_viewer.go]
[CODE: api/internal/services/viewer/content.go]
[CODE: api/internal/services/viewer/reset.go]

## Documentation Health

`POST /knowledge_observatory.v1.KnowledgeObservatoryService/DocHealth` (Connect-RPC)

Single source of truth for scenario documentation health. Runs structural
placement (misplaced/missing/extra/temporary) + content checks
(markdown, mermaid, links, absolute paths) + bidirectional reference
audits (`[CODE:]`, `// DOC:`, marked refs) + manifest coverage in one
call. The legacy REST `GET /api/v1/scenarios/{name}/docs/health` was
removed; this is the only surface.

**Request (JSON unary)**

```json
{
  "scenarioName": "demo",
  "strictExternalLinks": false,
  "requireAllDocsRegistered": false,
  "skipExternalLinks": false
}
```

**Response (excerpt)**

```json
{
  "scenarioName": "demo",
  "healthScore": 0.92,
  "totalDocs": 12,
  "misplacedDocs": [
    {"actualPath": "docs/PROGRESS.md", "expectedPath": "docs/internal/PROGRESS.md", "docType": "progress", "severity": "DOC_HEALTH_SEVERITY_WARNING"}
  ],
  "missingDocs": [
    {"docType": "progress", "path": "docs/internal/PROGRESS.md", "severity": "DOC_HEALTH_SEVERITY_WARNING"}
  ],
  "extraDocs": ["docs/misc/NOTE.md"],
  "contentFindings": [],
  "referenceFindings": [],
  "manifestFindings": [],
  "counts": {"filesChecked": 12, "localLinks": 4, "externalLinks": 1}
}
```

See `packages/proto/schemas/knowledge-observatory/v1/api.proto` for the
full schema.

[CODE: api/handlers/dochealth/handler.go]
[CODE: api/internal/services/dochealth/service.go]

Proto schema (outside scenario tree): `packages/proto/schemas/knowledge-observatory/v1/api.proto`.

`POST /api/v1/scenarios/{name}/docs/reset`

**Request**
```json
{
  "doc_type": "problems",
  "max_age_days": 30,
  "keep_min_entries": 3,
  "preview": true
}
```

**Response**
```json
{
  "scenario_name": "string",
  "doc_type": "problems",
  "preview": true,
  "removed_count": 2,
  "kept_count": 5,
  "removed_entries": ["## 2025-10-01: Old entry"],
  "new_content": "# Problems..."
}
```

[CODE: api/docs_reset.go]
[CODE: api/internal/docvalidation/validation.go]
[CODE: api/internal/doclogs/logs.go]

## Documentation Healing
`POST /api/v1/scenarios/{name}/docs/heal`

Spawn a documentation healing agent for a scenario.

**Request**
```json
{
  "scenario_name": "knowledge-observatory",
  "issues": ["Missing: seams (docs/internal/SEAMS.md)"],
  "auto_approve": false,
  "dry_run": true
}
```

**Response**
```json
{
  "job_id": "uuid",
  "scenario_name": "knowledge-observatory",
  "status": "running",
  "progress": "10% - scanning docs",
  "health_before": 0.72,
  "started_at": "2026-01-26T18:45:02Z"
}
```

`GET /api/v1/docs/heal/{job_id}`

Check healing job status and diff preview.

**Response**
```json
{
  "job_id": "uuid",
  "scenario_name": "knowledge-observatory",
  "status": "needs_review",
  "health_before": 0.72,
  "health_after": 0.92,
  "diff": {
    "summary": "Moved SEAMS.md into docs/internal and updated manifest.",
    "files": [
      {
        "path": "scenarios/knowledge-observatory/docs/internal/SEAMS.md",
        "operation": "modify",
        "diff": "diff --git a/... b/..."
      }
    ]
  }
}
```

`POST /api/v1/docs/heal/{job_id}/approve`

Approve and apply healing changes.

`POST /api/v1/docs/heal/{job_id}/reject`

Reject and discard healing changes.

[CODE: api/docs_heal.go]
[CODE: api/internal/services/dochealing/service.go]
[CODE: api/internal/adapters/agentmanager/dochealing_client.go]
[CODE: api/internal/dochealing/sqlite.go]

## Collection Inventory

Knowledge Observatory currently exposes collection inspection and maintenance, not generic record/document upsert or ingest-job routes. Scenario documentation indexing is managed by the configured indexer.

`GET /api/v1/knowledge/collections`

Lists collections with ownership/provenance signals for triage.

**Response**
```json
{
  "collections": [
    {
      "name": "knowledge_chunks_v1",
      "total_points": 12600,
      "ownership": "knowledge_observatory",
      "ownership_label": "Likely KO-managed",
      "ingest_attempts": 850,
      "metadata_rows": 12440,
      "distinct_namespaces": 21,
      "last_ingest_at": "2026-02-08T18:31:15Z"
    }
  ],
  "timestamp": "2026-02-08T18:32:01Z"
}
```

[CODE: api/collections.go]

## Collection Records Preview
`GET /api/v1/knowledge/collections/{collection}/records?limit=25&offset=0&namespace=...&document_id=...&search=...`

Returns paginated record previews for investigation.

**Response**
```json
{
  "collection": "knowledge_chunks_v1",
  "total_count": 5820,
  "offset": 0,
  "limit": 25,
  "next_offset": 25,
  "records": [
    {
      "id": "point-id",
      "namespace": "ecosystem-manager",
      "document_id": "doc-123",
      "chunk_index": 2,
      "content_hash": "abc123",
      "ingested_at": "2026-02-08T18:31:15Z",
      "content_preview": "Chunk text preview..."
    }
  ]
}
```

[CODE: api/collections.go]

## Collection Diagnostics
`GET /api/v1/knowledge/collections/{collection}/diagnostics?mode=sample|full&limit=500`

Provides chunk/vector quality diagnostics and action recommendations.

**Response**
```json
{
  "collection": "knowledge_chunks_v1",
  "mode": "sample",
  "total_points": 12600,
  "analyzed_points": 1200,
  "vector_dimensions": [{"dimension": 768, "count": 1200}],
  "namespaces": [{"name": "ecosystem-manager", "count": 840}],
  "chunk_length": {"min_characters": 44, "max_characters": 1210, "avg_characters": 731.4},
  "missing_payload_fields": {"document_id": 2},
  "redundancy": {
    "duplicate_content_hashes": 14,
    "duplicate_point_count": 33,
    "duplicate_ratio": 0.0275
  },
  "stale_chunks": {
    "groups_detected": 8,
    "candidate_delete_rows": 11,
    "top_documents": [{"name": "ecosystem-manager/abc123", "count": 3}]
  },
  "ingest_history": {
    "total_attempts": 850,
    "success_count": 842,
    "failure_count": 8,
    "failure_count_last_24h": 1,
    "failure_rate": 0.0094
  },
  "recommendations": ["Run prune_stale_chunks ..."],
  "timestamp": "2026-02-08T18:32:01Z"
}
```

[CODE: api/knowledge_maintenance.go]

## Collection Maintenance: Prune Stale Chunks
`POST /api/v1/knowledge/collections/{collection}/maintenance/prune-stale-chunks`

Deletes superseded chunks per `(namespace, document_id, chunk_index)`, keeping newest.

**Request**
```json
{
  "dry_run": false,
  "max_deletes": 500
}
```

**Response**
```json
{
  "collection": "knowledge_chunks_v1",
  "action": "prune_stale_chunks",
  "dry_run": false,
  "analyzed_points": 1200,
  "candidate_delete_count": 11,
  "deleted_count": 11,
  "took_ms": 55
}
```

[CODE: api/knowledge_maintenance.go]

## Collection Maintenance: Dedupe Content
`POST /api/v1/knowledge/collections/{collection}/maintenance/dedupe-content`

Deletes duplicate chunks by `(namespace, content_hash)`, keeping newest.

[CODE: api/knowledge_maintenance.go]
