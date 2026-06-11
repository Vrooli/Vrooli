# Reranker Resource

Cross-encoder relevance **reranking** for the second stage of hybrid retrieval.
Wraps the prebuilt HuggingFace **Text-Embeddings-Inference (TEI)** image serving
**`BAAI/bge-reranker-v2-m3`** — a multilingual cross-encoder that scores how well
each candidate passage answers a query. No custom inference code: TEI is the
engine; the model is auto-pulled from the HuggingFace Hub on first start.

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

First start downloads the model into a bind-mounted cache
(`${RESOURCE_DATA_DIR}/model-cache` → container `/data`); subsequent starts are
fully local and fast.

## GPU vs CPU

The base compose runs the **CPU** TEI image (`cpu-1.7`). When the `nvidia` probe
passes, the GPU overlay (`docker/docker-compose.gpu.yml`) swaps to the CUDA image
(`89-1.7`, matching an Ada/sm_89 GPU) and adds the device reservation.

> **Known CPU-fallback limitation (AMD hosts):** TEI's `cpu-1.7` image uses Intel
> MKL, which crashes (`Intel MKL ERROR: Parameter 13 ... SGEMM`) on AMD CPUs, and
> `bge-reranker-v2-m3` ships no ONNX weights (TEI CPU's preferred path). So on a
> GPU-less **AMD** host the default CPU image will not serve this model. Verified
> on an AMD Ryzen 9 7950X. Mitigations for a CPU-only AMD deployment: pin a
> reranker that ships ONNX weights (e.g. `BAAI/bge-reranker-base` has community
> ONNX exports) via `RERANKER_MODEL`, or run on Intel/GPU. The GPU path (this
> repo's primary target) is unaffected and fully validated.

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
bash resources/reranker/test/integration-test.sh    # live smoke (needs the running container)
```

## TEI API surface (served by the image)

- `GET /health` — 200 once the model is loaded.
- `GET /info` — model id, type, dtype, device.
- `POST /rerank` — body `{"query": str, "texts": [str], "raw_scores": bool, "return_text": bool}`; returns `[{"index": int, "score": float}, ...]` sorted by score descending.
