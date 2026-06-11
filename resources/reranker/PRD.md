# Reranker Resource — PRD

## Problem

Hybrid retrieval (BM25 sparse + dense vectors fused with RRF) produces a good
*shortlist*, but the top-1 ordering is often wrong: lexical and embedding signals
agree on the rough candidate set yet disagree on which is actually most relevant.
A **cross-encoder reranker** — which reads the (query, passage) pair jointly
rather than comparing independent embeddings — is the standard, highest-leverage
fix, pushing Recall@1/MRR materially higher on the fused shortlist.

Vrooli had no way to serve one. Ollama (the local model runtime) serves only
`embed` and `generate`; cross-encoders are a third model type it cannot host.

## Solution

A dedicated, compose-backed resource wrapping the prebuilt HuggingFace
**Text-Embeddings-Inference (TEI)** image serving **`BAAI/bge-reranker-v2-m3`**:

- **No inference code.** TEI is the engine; the model auto-pulls from the HF Hub
  on first start into a bind-mounted cache. One-time ~2.3GB download, fully local
  thereafter.
- **GPU-accelerated with CPU fallback.** CUDA image on hosts with an NVIDIA GPU
  (the `nvidia` probe + GPU overlay), CPU image otherwise; TEI degrades to CPU on
  its own if no device is visible.
- **A typed Go CLI gateway** (`resource-reranker gateway rerank|health|info`) so
  scenarios never hand-roll HTTP, resolving the service via the exported
  `RERANKER_URL`.

## Consumers

- **`packages/ai-go/search`** — `CrossEncoderReranker` (Phase 5 of the KO search
  cutover). Default-on when the resource is healthy.
- **`search-hub`** — same resource backs its reranker once it lands.
- Any scenario doing retrieval (cli-health, ui-health, security-health,
  knowledge-observatory).

## Non-goals

- Embeddings. TEI *can* also serve dense/sparse embeddings, but dense embedding
  stays on Ollama (nomic) for now; this resource is scoped to reranking. (A future
  consolidation could host the hybrid embedder here too — out of scope.)
- Being a hard dependency. Every consumer degrades gracefully (LLM reranker →
  fused order) when this resource is down.

## Success criteria

1. `vrooli resource validate|install|start|status reranker` all succeed; first
   start pulls the model and reaches healthy.
2. `POST /rerank` orders an obviously-relevant passage above noise
   (`test/integration-test.sh`).
3. GPU path runs on a GPU host; CPU fallback runs without one.
4. On the KO accuracy corpus, cross-encoder reranking improves MRR@3 over
   no-rerank and over the LLM reranker (measured in KO Phase 7 validation).
5. `make check` green; gateway unit tests pass with no live container.
