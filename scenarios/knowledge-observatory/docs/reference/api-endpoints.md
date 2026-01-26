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
