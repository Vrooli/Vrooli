# Seams

| Seam | Production Wiring | Test Fake | Why It Exists |
|---|---|---|---|
| `catalog.Repository` | SQLite catalog adapter | Transactional in-memory repository | Prove generation state without a process-global database. |
| `catalog.Clock` | Injected system clock | Fixed clock | Make generation timestamps deterministic. |
| `retrieval.Embedder` | Shared AI-search embedder | Copy-on-read vector fake | Make semantic tests deterministic. |
| `gateway.EmbeddingClient` | Generated AI Gateway Connect client | Fake generated-client interface | Keep RPC transport outside retrieval policy and verify embedding request/response rules. |
| `retrieval.VectorStore` | Qdrant adapter | Map-backed vector store | Test freshness and degradation without Qdrant. |
| `retrieval.Reranker` | Optional shared reranker | Ordered candidate fake | Test bypass and fallback behavior. |
| `retrieval.Admission` | Process-wide weighted controller | Deterministic capacity fake | Prove cancellation and capacity release. |
| `indexcontrol.JobStore` | SQLite durable job repository | In-memory state machine | Test restart and cancellation transitions. |
| `indexcontrol.ProcessRunner` | Governed process adapter where required | Recording runner | Keep commands injectable and cancellable. |
| `logging.Logger` | Composition-owned standard logger | Buffer or recording logger | Prevent package-global logging and make output assertions deterministic. |

## Architecture Alignment Notes

Code Facts owns brokering and proof synthesis. Provider scenarios own parsing. Consumer health scenarios own policy.

## Production transport inventory

- API ingress mounts generated `CodeFactsService` and `SearchControlService`
  Connect handlers from `api/handlers/facts/module.go`.
- CLI commands bind generated Code Facts and Search Control RPC clients.
- UI fact operations use generated `CodeFactsService` through the shared
  scenario Connect transport. The only UI REST call is `/health`, an operational
  probe parsed with the generated health response and error-envelope schemas.
- Outbound scenario calls use generated clients for TypeScript Code Graph,
  Architecture Cartographer audit/signals, Search Hub registry/eval, and
  AI Gateway inference. The AI Gateway transport adapter lives in
  `api/internal/gateway/`, outside transport-free retrieval policy. Optional provider failures degrade through the owning
  provider behavior; registration is bounded best-effort; embedding is required
  only when semantic search is selected.

There are no multipart or other production REST exceptions in this scenario.
The unused generic multipart helper and its fixture test were removed from the
UI client; they described a notes attachment route this scenario does not own.
