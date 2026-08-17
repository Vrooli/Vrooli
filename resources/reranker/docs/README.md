# Reranker — API & Operations

The reranker resource serves the checksum-verified Hugging Face
Text-Embeddings-Inference (TEI) binary with `BAAI/bge-reranker-v2-m3`. This page documents the HTTP surface
TEI exposes and how to operate the resource. For an overview see
[`../README.md`](../README.md); for product rationale see [`../PRD.md`](../PRD.md).

## HTTP API (served by TEI)

### `GET /health`
Returns `200 OK` once the model is loaded. Used by the Vrooli host-level health
check (`resource.json` → `health_checks`).

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

Environment values exported by the resource and supplied to the supervised process:

| Variable | Default | Meaning |
|---|---|---|
| `RERANKER_PORT` | `11453` | supervised service port |
| `RERANKER_MODEL` | `BAAI/bge-reranker-v2-m3` | served model |
| `RESOURCE_DATA_DIR` | platform-specific | resource-owned model cache root |

The HF model cache lives under the relocatable `model_cache` storage entry; it
persists across restarts and is measured by Storage Manager.

## Operations

- **First start** downloads ~2.3GB; `startup_timeout_seconds` is 180 to allow it.
  Watch progress: `vrooli resource logs reranker`.
- **Model cache** persists in the resource-owned model-cache directory. Model
  cleanup must go through the resource/provider contract rather than deleting
  individual blobs.
- **GPU**: the native process uses the host driver when available. The status
  surface reports the active model and TEI device; do not infer GPU use from a
  host capability flag alone.
- **Swapping the model**: use the resource model-policy command once the target
  catalog entry has a measured `router.routing` result. Restarting with an
  unmeasured model is not a supported operator path.

## Artifact provenance

The Linux amd64 artifact is TEI 1.7.4's `text-embeddings-router` executable.
The acquisition ladder selects the CUDA 8.9+ image when the host reports that
capability and otherwise selects the digest-pinned CPU image; both selected
executables are verified by SHA-256 before launch. macOS and Windows remain
unsupported until signed native bundles and target smoke evidence exist.
