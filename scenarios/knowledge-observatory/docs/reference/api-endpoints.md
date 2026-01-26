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
  "can_auto_fix": false
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
  "chunk_overlap": 150
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
  "took_ms": 120
}
```

[CODE: api/document_ingest.go]
[CODE: api/internal/services/ingest/chunking.go]

## Ingest Jobs (Async)
`POST /api/v1/ingest/jobs`

Enqueues an async document ingest job.

[CODE: api/jobs.go]
[CODE: api/internal/services/ingestjobs/runner.go]

`GET /api/v1/ingest/jobs/{job_id}`

Returns job status and timestamps.

[CODE: api/jobs.go]
