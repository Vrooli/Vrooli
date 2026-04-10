# API Endpoints

Base URL: `http://localhost:${API_PORT}` (API port is assigned by the scenario lifecycle).

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
[CODE: api/internal/services/search/service.go]

## Health Metrics
`GET /api/v1/knowledge/health`

Returns collection health and aggregated quality metrics.

[CODE: api/metrics.go]

## Ingest Health
`GET /api/v1/ingest/health`

Returns ingest runner/backlog telemetry.

**Response**
```json
{
  "runner_interval_ms": 500,
  "pending_jobs": 0,
  "running_jobs": 0,
  "failed_jobs": 2,
  "successful_jobs": 42,
  "failures_last_24h": 1,
  "oldest_pending_age_ms": 0,
  "status": "healthy|warning|degraded|unknown",
  "timestamp": "2026-02-08T18:32:01Z"
}
```

[CODE: api/knowledge_maintenance.go]

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
`GET /api/v1/scenarios/{name}/docs/health`

Returns documentation health for a scenario (misplaced, missing, extra docs).

**Response**
```json
{
  "scenario_name": "string",
  "health_score": 0.92,
  "total_docs": 12,
  "misplaced_docs": [
    {"actual_path": "docs/PROGRESS.md", "expected_path": "docs/internal/PROGRESS.md", "doc_type": "progress", "severity": "warning"}
  ],
  "missing_docs": ["progress"],
  "extra_docs": ["docs/misc/NOTE.md"],
  "warnings": [
    {"type": "misplaced", "message": "Documentation file is in the wrong location", "expected_path": "docs/internal/PROGRESS.md", "severity": "warning"}
  ],
  "can_auto_fix": true
}
```

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

[CODE: api/docs_health.go]
[CODE: api/internal/docschema/validation.go]
[CODE: api/internal/docschema/reset.go]

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
[CODE: api/internal/adapters/dochealingstore/postgres.go]

## Upsert Record
`POST /api/v1/knowledge/records/upsert`

**Request**
```json
{
  "namespace": "string",
  "collection": "string?",
  "record_id": "string?",
  "external_id": "string?",
  "content": "string",
  "tags": ["string"],
  "metadata": {},
  "visibility": "private|shared|global",
  "source": "string?",
  "source_type": "string?"
}
```

**Response**
```json
{
  "record_id": "string",
  "collection": "string",
  "namespace": "string",
  "content_hash": "string",
  "upserted": true,
  "took_ms": 18
}
```

[CODE: api/ingest.go]
[CODE: api/internal/services/ingest/service.go]

## Delete Record
`DELETE /api/v1/knowledge/records/{record_id}`

**Response**
```json
{
  "record_id": "string",
  "collection": "string",
  "deleted": true,
  "took_ms": 6
}
```

[CODE: api/delete.go]

## Document Ingest
`POST /api/v1/knowledge/documents/ingest`

Chunked, synchronous ingest for long documents.

**Request**
```json
{
  "namespace": "string",
  "collection": "string?",
  "document_id": "string?",
  "external_id": "string?",
  "content": "string",
  "tags": ["string"],
  "metadata": {},
  "visibility": "private|shared|global",
  "source": "string?",
  "source_type": "string?",
  "chunk_size": 1200,
  "chunk_overlap": 150,
  "prune_stale": true
}
```

**Response**
```json
{
  "document_id": "string",
  "collection": "string",
  "namespace": "string",
  "chunk_count": 4,
  "record_ids": ["string"],
  "content_hash": "string",
  "pruned_stale_count": 2,
  "took_ms": 120
}
```

[CODE: api/document_ingest.go]
[CODE: api/internal/services/ingest/chunking.go]

## Ingest Jobs (Async)
`POST /api/v1/ingest/jobs`

Enqueues an async document ingest job.

[CODE: api/jobs.go]

## Collection Inventory
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

## Delete Document
`POST /api/v1/knowledge/documents/delete`

Delete all chunks for a namespace+document.

**Request**
```json
{
  "namespace": "ecosystem-manager",
  "collection": "knowledge_chunks_v1",
  "document_id": "string?",
  "external_id": "string?",
  "dry_run": false
}
```

**Response**
```json
{
  "collection": "knowledge_chunks_v1",
  "namespace": "ecosystem-manager",
  "document_id": "string",
  "candidate_delete_count": 18,
  "deleted_count": 18,
  "took_ms": 25
}
```

[CODE: api/knowledge_maintenance.go]
[CODE: api/internal/services/ingestjobs/runner.go]

`GET /api/v1/ingest/jobs/{job_id}`

Returns job status and timestamps.

[CODE: api/jobs.go]
