# Reranker — API & Operations

The reranker resource serves the prebuilt HuggingFace Text-Embeddings-Inference
(TEI) image with `BAAI/bge-reranker-v2-m3`. This page documents the HTTP surface
TEI exposes and how to operate the resource. For an overview see
[`../README.md`](../README.md); for product rationale see [`../PRD.md`](../PRD.md).

## HTTP API (served by TEI)

### `GET /health`
Returns `200 OK` once the model is loaded. Used by the Vrooli host-level health
check (`resource.json` → `health_checks`). The TEI image does not bundle `curl`,
so readiness is probed from the host, not inside the container.

### `GET /info`
Returns the served-model descriptor:

```json
{ "model_id": "BAAI/bge-reranker-v2-m3", "model_type": "reranker", "model_dtype": "float32", ... }
```

### `POST /rerank`
Request:

```json
{
  "query": "how do I restart a scenario",
  "texts": ["passage a", "passage b", "passage c"],
  "raw_scores": false,
  "return_text": false
}
```

Response — sorted by `score` **descending**; `index` is the position in the
request's `texts` array:

```json
[
  { "index": 1, "score": 0.987 },
  { "index": 0, "score": 0.421 },
  { "index": 2, "score": 0.011 }
]
```

Set `"return_text": true` to echo each passage back in the result.

## Configuration

Environment variables (set in the compose files; override per-deploy):

| Variable | Default | Meaning |
|---|---|---|
| `RERANKER_PORT` | `11453` | host port (container listens on 80) |
| `RERANKER_MODEL` / `MODEL_ID` | `BAAI/bge-reranker-v2-m3` | served model |
| `RERANKER_IMAGE` | `…text-embeddings-inference:cpu-1.7` (CPU) / `:89-1.7` (GPU) | TEI image tag |
| `RERANKER_MEMORY_LIMIT` | `6G` | container memory cap |

The HF model cache lives in a Docker-managed named volume (`reranker_models`,
mounted at container `/data`); it persists across restarts and auto-creates on
first start.

## Operations

- **First start** downloads ~2.3GB; `startup_timeout_seconds` is 180 to allow it.
  Watch progress: `vrooli resource logs reranker`.
- **Model cache** persists in the `reranker_models` named volume; removing it
  (`docker volume rm vrooli-reranker_reranker_models`) forces a re-download.
- **GPU**: the overlay (`docker/docker-compose.gpu.yml`) is applied only when the
  `nvidia` probe passes. To force CPU, start without the overlay or unset GPU
  visibility.
- **Swapping the model**: change `RERANKER_MODEL` (any TEI-compatible
  cross-encoder, e.g. `BAAI/bge-reranker-base`) and restart. `RERANKER_MODEL` is
  exported so consumers can report which model is active.

## Image tags

Pinned in the KO search cutover Phase 0 spike:
- GPU: `ghcr.io/huggingface/text-embeddings-inference:89-1.7` (Ada/sm_89, CUDA 12.x)
- CPU: `ghcr.io/huggingface/text-embeddings-inference:cpu-1.7`

Do not use `:*-latest` (unpinned).
