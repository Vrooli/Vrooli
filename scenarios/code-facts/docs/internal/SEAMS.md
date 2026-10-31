# Seams

| Seam | Production Wiring | Test Fake | Why It Exists |
|---|---|---|---|
| `targets.FileSystem` | Bounded OS filesystem adapter | Sorted map-backed filesystem | Resolve and inventory without global filesystem access in domain tests. |
| `targets.Resolver` | Repository-aware resolver | Table-driven resolver fake | Keep target policy independent from handlers. |
| `catalog.Repository` | SQLite catalog adapter | Transactional in-memory repository | Prove generation state without a process-global database. |
| `catalog.Clock` | Injected system clock | Fixed clock | Make generation timestamps deterministic. |
| `analysis.Analyzer` | Go/TypeScript graph clients | Recording analyzer | Avoid live providers and prove capability selection. |
| `analysis.ProjectionStore` | Normalized SQLite graph rows | In-memory projection store | Keep graph expansion out of cold analyzers. |
| `retrieval.Embedder` | Shared AI-search embedder | Copy-on-read vector fake | Make semantic tests deterministic. |
| `retrieval.VectorStore` | Qdrant adapter | Map-backed vector store | Test freshness and degradation without Qdrant. |
| `retrieval.Reranker` | Optional shared reranker | Ordered candidate fake | Test bypass and fallback behavior. |
| `retrieval.Admission` | Process-wide weighted controller | Deterministic capacity fake | Prove cancellation and capacity release. |
| `proof.ContractSource` | `descriptorimage.Source` adapter | Fixed descriptor contracts | Prove digest and last-known-good behavior. |
| `indexcontrol.JobStore` | SQLite durable job repository | In-memory state machine | Test restart and cancellation transitions. |
| `indexcontrol.ProcessRunner` | Governed process adapter where required | Recording runner | Keep commands injectable and cancellable. |
| `cache.Repository` | SQLite normalized cache adapter | Map-backed repository | Prove TTL, quotas, and no-write-on-hit behavior. |
| `logging.Logger` | Composition-owned standard logger | Buffer or recording logger | Prevent package-global logging and make output assertions deterministic. |

## Architecture Alignment Notes

Code Facts owns brokering and proof synthesis. Provider scenarios own parsing. Consumer health scenarios own policy.
