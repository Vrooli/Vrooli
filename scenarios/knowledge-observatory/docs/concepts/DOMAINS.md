# Domains

## Domain Inventory

| Domain | Purpose | Primary paths |
|---|---|---|
| knowledge-search | Query vector knowledge and return ranked semantic matches. | `api/search.go`, `api/internal/services/search/`, `ui/src/surfaces/search/` |
| documentation-contracts | Interpret scenario documentation manifests and validate docs health. | `api/internal/doccontract/`, `api/internal/doctemplates/`, `api/internal/docvalidation/`, `api/internal/services/dochealth/` |
| documentation-operations | Read, search, append, reset, audit, and heal scenario documentation. | `api/docs_*.go`, `api/internal/services/viewer/`, `api/internal/services/docsearch/`, `api/internal/services/dochealing/`, `cli/domains/docs/` |
| graph-and-metrics | Surface graph relationships and quality metrics. | `api/graph.go`, `api/metrics.go`, `api/internal/services/graph/` |
| ingest | Write records and documents into the knowledge stores. | `api/ingest.go`, `api/document_ingest.go`, `api/internal/services/ingest/` |

## Shared Concepts

Knowledge Observatory is a control plane, not a knowledge store. Qdrant and
PostgreSQL own durable data; this scenario owns interpretation, visibility,
and operational workflows.

## Cross-References

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [FLOWS.md](FLOWS.md)
- [DATA.md](DATA.md)
- [../internal/SEAMS.md](../internal/SEAMS.md)
