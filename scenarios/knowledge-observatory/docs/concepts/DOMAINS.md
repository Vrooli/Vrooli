# Domains

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| knowledge-search | Query vector knowledge and return ranked semantic matches. | Help agents find relevant docs, records, and semantic context. | Qdrant vector collections and search indexes. | service | query | SearchResult, KnowledgeSource, Ranking | `api/search.go`, `api/internal/services/search/`, `ui/src/surfaces/search/` |
| documentation-contracts | Interpret scenario documentation manifests and validate docs health. | Keep scenario docs registered, typed, and conforming to template contracts. | None; reads docs and manifests. | validation | service | Manifest, Contract, DocValidation, DocHealth | `api/internal/doccontract/`, `api/internal/doctemplates/`, `api/internal/docvalidation/`, `api/internal/services/dochealth/` |
| documentation-operations | Read, search, append, reset, audit, and heal scenario documentation. | Provide the canonical docs CLI/API workflow for agents and operators. | Scenario documentation files. | service | mutation | DocViewer, DocSearch, DocHealing, AppendLog | `api/docs_*.go`, `api/internal/services/viewer/`, `api/internal/services/docsearch/`, `api/internal/services/dochealing/`, `cli/domains/docs/` |
| graph-and-metrics | Surface graph relationships and quality metrics. | Show documentation graph links and health metrics. | None; reports derived graph and metrics. | reporting | query | Graph, Metrics, Relationship | `api/graph.go`, `api/metrics.go`, `api/internal/services/graph/` |
| ingest | Write records and documents into the knowledge stores. | Load docs and records into searchable stores. | PostgreSQL records and Qdrant vectors. | mutation | provider | Ingest, DocumentIngest, Record | `api/ingest.go`, `api/document_ingest.go`, `api/internal/services/ingest/` |

## Shared Concepts

Knowledge Observatory is a control plane, not a knowledge store. Qdrant and
PostgreSQL own durable data; this scenario owns interpretation, visibility,
and operational workflows.

## Cross-References

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [FLOWS.md](FLOWS.md)
- [DATA.md](DATA.md)
- [../internal/SEAMS.md](../internal/SEAMS.md)
