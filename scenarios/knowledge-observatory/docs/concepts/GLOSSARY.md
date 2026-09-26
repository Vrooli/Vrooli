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

## Materializer
Background loop (every 5 minutes) that samples vectors from each Qdrant collection and computes quality metrics (coherence, freshness, redundancy, coverage). Persists results and relationship edges into Postgres.
[CODE: api/metrics.go]

## Deep Search
Agent-powered documentation search. Spawns an agent-manager run with the `documentation-search` skill, polls for results, and parses structured JSON output. Async — returns a job ID for polling.
[CODE: api/internal/services/deepsearch/service.go]

## Doc Healing
Agent-powered documentation repair. Spawns a sandboxed agent-manager run with the `documentation-health` skill. Produces a diff that must be approved or rejected before changes are applied to the filesystem.
[CODE: api/internal/services/dochealing/service.go]

## Doc Health / Documentation Contract
Validation system that checks scenario documentation against the manifest contract declared by its source template or scenario-local `docs/manifest.json`. Produces a health score (0-1), lists missing and misplaced docs, and generates auto-fix hints.
[CODE: api/internal/doccontract/manifest.go]
[CODE: api/internal/docvalidation/validation.go]
[CODE: api/internal/services/dochealth/service.go]

## External ID Map
Idempotency mechanism for ingestion. Maps a `(namespace, external_id)` pair to a `record_id`, preventing duplicate records from being created when the same content is ingested multiple times.
[CODE: api/internal/adapters/metadatastore/postgres.go]

## Skill
A prompt template retrieved from prompt-manager that provides instructions for an agent-manager run. Knowledge Observatory uses two skills: `documentation-search` (deep search) and `documentation-health` (doc healing).
[CODE: api/internal/adapters/promptmanager/client.go]

## Agent Run
A sandboxed execution managed by agent-manager. Deep search and doc healing both create agent runs, poll their events for progress, and retrieve final output or diffs.
[CODE: api/internal/adapters/agentmanager/client.go]
