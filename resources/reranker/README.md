# Reranker Resource

Cross-encoder relevance **reranking** for the second stage of hybrid retrieval.
Runs the checksum-verified Hugging Face **Text-Embeddings-Inference (TEI)** server
binary serving
**`BAAI/bge-reranker-v2-m3`** — a multilingual cross-encoder that scores how well
each candidate passage answers a query. No custom inference code: TEI is the
engine; the model is auto-pulled from the HuggingFace Hub into the resource-owned
cache on first start.

This resource is the `CrossEncoderReranker` backend for
[`packages/ai-go/search`](../../packages/ai-go/search) and the shared reranker for
the `search-hub` router. It is a **soft** dependency everywhere — search degrades
to an LLM reranker (`qwen3:4b` via Ollama) and then to fused (RRF) order when the
reranker is unavailable.

## Why a dedicated resource (not an Ollama command)?

A cross-encoder is a *third* model type: it takes a (query, passage) **pair** and
emits a single relevance score. Ollama 0.11.7 exposes only `embed` and `generate`
— it has no `/rerank` endpoint — so it cannot serve a cross-encoder. TEI can, via
`POST /rerank`. If a future Ollama adds native rerank, consumers can retarget an
Ollama gateway command with no code change (the `Reranker` interface hides the
backend).

## Lifecycle

```bash
vrooli resource validate reranker     # manifest + contract export
vrooli resource install reranker      # build/install the CLI gateway
vrooli resource start reranker        # first start pulls ~2.3GB (watch logs)
vrooli resource status reranker       # healthy once the model is loaded
vrooli resource logs reranker         # follow model-pull / serving logs
vrooli resource stop reranker
```

First start downloads the model into `${RESOURCE_DATA_DIR}/model-cache`;
subsequent starts are fully local and fast. The managed-service supervisor owns
the TEI process and verifies the staged artifact checksum before every launch.

## GPU vs CPU

The currently staged Linux amd64 target requires a compatible CUDA host. The
CPU image is intentionally marked unavailable: single-file extraction loses
the image's `libiomp5.so` dependency, and no host-wide copy is assumed. The
runtime therefore rejects that target before installation instead of leaving
an artifact that cannot start.

The resource remains optional. Consumers fall back to the LLM reranker or fused
retrieval order when no validated reranker target is available. A future CPU
target must be shipped as a checksum-pinned executable tree (or a self-
contained executable) and pass runtime-closure and startup smoke tests before
the manifest can advertise CPU support.

## CLI gateway

```bash
# Rank passages by relevance to a query (cross-encoder scoring)
resource-reranker gateway rerank --query "restart a scenario" \
    --document "vrooli scenario restart <name>" \
    --document "bananas are rich in potassium" \
    --top-k 1 --json

# Or pipe passages from stdin (one per line)
printf '%s\n' "$passage_a" "$passage_b" | \
    resource-reranker gateway rerank --query "$q" --documents-stdin

resource-reranker gateway health      # probe /health
resource-reranker gateway info        # served model id/type/device
```

Scenarios resolve the service via the exported **`RERANKER_URL`** — never compute
the URL client-side.

## Exported environment

| Variable | Example | Purpose |
|---|---|---|
| `RERANKER_URL` | `http://localhost:11453` | base URL (consume this) |
| `RERANKER_RERANK_ENDPOINT` | `http://localhost:11453/rerank` | rerank endpoint |
| `RERANKER_MODEL` | `BAAI/bge-reranker-v2-m3` | served model id |
| `RERANKER_PORT` | `11453` | host port |

## Testing

```bash
make check                                          # Go gates (lint/type/test)
vrooli resource status reranker                     # readiness and model-serving state
resource-reranker gateway rerank --query "q" \
  --document "candidate passage" --json              # live gateway smoke
```

## TEI API surface (served by the managed service)

- `GET /health` — 200 once the model is loaded.
- `GET /info` — model id, type, dtype, device.
- `POST /rerank` — body `{"query": str, "texts": [str], "raw_scores": bool, "return_text": bool}`; returns `[{"index": int, "score": float}, ...]` sorted by score descending.
## Maturity

M5 (2026-08-15): the managed-service lifecycle, readiness, platform gates,
capacity profile, Go CLI tests, and Search Hub consumer evidence are covered by
the fleet contract. See [the full assessment](docs/maturity-assessment.md) for
the per-dimension score and the deliberately unsupported targets.
