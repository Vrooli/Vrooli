# Glossary

## Collection
A Qdrant collection that stores knowledge vectors. Default is `knowledge_chunks_v1`.
[CODE: api/knowledge_schema.go]

## Namespace
Logical owner or domain for a record (e.g., a scenario name). Required for ingestion.
[CODE: api/ingest.go]
[CODE: api/internal/services/ingest/service.go]

## Record
The atomic knowledge unit stored in Qdrant (vector + payload). Created via record upsert.
[CODE: api/ingest.go]

## Document
A larger payload that is chunked into multiple records.
[CODE: api/document_ingest.go]
[CODE: api/internal/services/ingest/chunking.go]

## Visibility
Access scope for records: `private`, `shared`, or `global`.
[CODE: api/knowledge_schema.go]

## Tags
Optional labels for filtering in search and graph queries.
[CODE: api/search.go]
[CODE: api/graph.go]

## Quality Metrics
Scores derived from sampled vectors: coherence, freshness, redundancy, coverage.
[CODE: api/metrics.go]

## Ingest Job
An async document ingest request stored in Postgres and processed by a runner.
[CODE: api/jobs.go]
[CODE: api/internal/services/ingestjobs/runner.go]

## Graph Node / Edge
Graph view nodes represent records; edges represent semantic similarity.
[CODE: api/internal/services/graph/service.go]
