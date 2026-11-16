# Seams

| Seam | Production Wiring | Test Fake | Why It Exists |
|---|---|---|---|
| `catalog.Repository` | SQLite catalog adapter | Transactional in-memory repository | Prove generation state without a process-global database. |
| `catalog.Clock` | Injected system clock | Fixed clock | Make generation timestamps deterministic. |
| `retrieval.Embedder` | Shared AI-search embedder | Copy-on-read vector fake | Make semantic tests deterministic. |
| `retrieval.VectorStore` | Qdrant adapter | Map-backed vector store | Test freshness and degradation without Qdrant. |
| `retrieval.Reranker` | Optional shared reranker | Ordered candidate fake | Test bypass and fallback behavior. |
| `retrieval.Admission` | Process-wide weighted controller | Deterministic capacity fake | Prove cancellation and capacity release. |
| `indexcontrol.JobStore` | SQLite durable job repository | In-memory state machine | Test restart and cancellation transitions. |
| `indexcontrol.ProcessRunner` | Governed process adapter where required | Recording runner | Keep commands injectable and cancellable. |
| `logging.Logger` | Composition-owned standard logger | Buffer or recording logger | Prevent package-global logging and make output assertions deterministic. |

## Architecture Alignment Notes

Code Facts owns brokering and proof synthesis. Provider scenarios own parsing. Consumer health scenarios own policy.
