# Domains

## Domain Inventory

This page defines domain responsibilities, purpose, data ownership, archetypes,
traits, and vocabulary. The [Architecture Domain Map](ARCHITECTURE.md#domain-map)
is the single maintained source for implementation paths and runtime surfaces.

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| collection-maintenance | Inspect and maintain indexed knowledge collections and their provenance. | Diagnose collection state and apply bounded cleanup actions. | Ingest provenance in scenario SQLite; Qdrant owns vectors and source systems own indexed inputs. | service | mutation, reporting | CollectionInventory, Diagnostics, Provenance | Architecture Domain Map |
| knowledge-search | Query vector knowledge and return ranked semantic matches. | Help agents find relevant docs, records, and semantic context. | None; reads Qdrant vector collections and scenario documentation. | service | query | SearchResult, KnowledgeSource, Ranking | Architecture Domain Map |
| knowledge-base | Search and inspect source-owned documents and review bounded references/duplicates. | Ground answers and audits in revision-pinned source evidence. | None; source artifacts remain owned by their repositories. | service | query | KnowledgeBase, Revision, Review, ScanGap | Architecture Domain Map |
| deep-search | Run agent-backed searches across scenario documentation. | Find relevant material through a bounded asynchronous workflow. | Search-job and result state in scenario SQLite. | orchestration | query | DeepSearch, SearchJob | Architecture Domain Map |
| documentation-indexing | Reconcile scenario documentation into the searchable corpus. | Keep documentation search current with source repositories. | Derived index in Qdrant; source scenarios own the indexed documents. | orchestration | service | Indexer, Reconcile, Reindex | Architecture Domain Map |
| graph | Derive and display relationships between knowledge concepts. | Reveal semantic connections among stored records. | Derived relationship edges in scenario SQLite; source vectors remain in Qdrant. | reporting | query | Graph, Relationship | Architecture Domain Map |
| metrics | Calculate and report collection quality and activity metrics. | Show collection quality and operational trends. | Quality samples and collection rollups in scenario SQLite. | reporting | query | Metrics, Coherence, Freshness, Redundancy, Coverage | Architecture Domain Map |
| documentation-health | Interpret scenario documentation manifests and validate docs health. | Keep scenario docs registered, typed, and conforming to template contracts. | None; reads source-owned docs and manifests. | validation | service | Manifest, Contract, DocValidation, DocHealth | Architecture Domain Map |
| documentation-operations | Explore, read, append, reset, audit, and retrieve scenario documentation and templates. | Provide the canonical docs API/CLI workflow for agents and operators. | No independent document corpus; source scenarios own documentation files. | service | mutation | DocViewer, DocSearch, AppendLog | Architecture Domain Map |
| documentation-healing | Coordinate agent-backed documentation fixes and approval. | Turn docs health findings into reviewed, bounded repairs. | Healing job, diff, and approval state in scenario SQLite; source scenarios own documentation files. | orchestration | mutation | DocHealing, DiffPreview, Approval | Architecture Domain Map |

Knowledge Observatory is a control plane, not the owner of the source knowledge
corpus. Qdrant holds indexed vectors; source repositories own their
documentation. Scenario SQLite holds domain metadata and operational records,
including job state, quality samples, collection rollups, and derived
relationship edges.

## Cross-References

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [FLOWS.md](FLOWS.md)
- [DATA.md](DATA.md)
- [../internal/SEAMS.md](../internal/SEAMS.md)
