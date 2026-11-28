# Glossary

## Collection
A named Qdrant vector set used for a corpus, such as `vrooli-docs` or
`knowledge_chunks_v1`. Knowledge Observatory can inspect and maintain
collections; source systems own their indexed inputs.
[CODE: api/collections.go]

## Ingest Provenance
Operational records of ingest attempts associated with a collection. The
collection diagnostics surface reports these records when present; Knowledge
Observatory does not expose generic record or document upsert routes.
[CODE: api/internal/ingest/repository.go]

## Document
A scenario-owned documentation file indexed for search. The index is derived;
the source repository remains authoritative.
[CODE: api/internal/aisearch/indexer.go]

## Visibility
Access scope values used by knowledge search: `private`, `shared`, and
`global`.
[CODE: api/knowledge_schema.go]

## Tags
Optional labels that can filter knowledge search and graph queries.
[CODE: api/knowledge_schema.go]
[CODE: api/graph.go]

## Quality Metrics
Scores derived from sampled vectors: coherence, freshness, redundancy, and
coverage. Samples and collection rollups are stored in the scenario database.
[CODE: api/metrics.go]
[CODE: api/internal/quality/sqlite.go]

## Graph Node / Edge
Nodes represent indexed records; edges represent derived semantic similarity
relationships.
[CODE: api/internal/services/graph/service.go]
[CODE: api/internal/graph/sqlite.go]

## Materializer
Background work that samples Qdrant collections, calculates quality metrics,
and stores the results and relationship edges.
[CODE: api/metrics.go]

## Documentation Index / Reindex
The derived searchable view of scenario documentation and the operation that
reconciles it with its source files. Reindex can plan changes, run in the
background, report status, or cancel the current reconciliation.
[CODE: api/search_surface.go]
[CODE: api/internal/aisearch/indexer.go]

## Deep Search
Agent-powered documentation search. It creates a job, tracks progress, and
returns results for polling.
[CODE: api/internal/services/deepsearch/service.go]
[CODE: api/internal/deepsearch/sqlite.go]

## Doc Healing
Agent-powered documentation repair. It produces a reviewable diff and requires
approval or rejection before applying changes.
[CODE: api/internal/services/dochealing/service.go]
[CODE: api/internal/dochealing/sqlite.go]

## Doc Health / Documentation Contract
Validation system that checks scenario documentation against the source
manifest and reports missing, misplaced, or invalid documents.
[CODE: api/internal/doccontract/manifest.go]
[CODE: api/internal/docvalidation/validation.go]
[CODE: api/internal/services/dochealth/service.go]

## Search Skill / Agent Run
Prompt Manager supplies task guidance; Agent Manager creates and tracks runs
used by deep search and documentation healing.
[CODE: api/internal/adapters/promptmanager/client.go]
[CODE: api/internal/adapters/agentmanager/client.go]
