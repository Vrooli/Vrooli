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
| `RERANKER_TOKENIZATION_WORKERS` | `4` | tokenizer worker count, independent of GPU batch-token limits |
| `RESOURCE_DATA_DIR` | platform-specific | resource-owned model cache root |

The HF model cache lives under the relocatable `model_cache` storage entry; it
persists across restarts and is measured by Storage Manager.

### Tokenizer memory

TEI clones the tokenizer for each worker. The resource explicitly defaults to
four workers instead of allowing TEI to select the host CPU count. Override
`RERANKER_TOKENIZATION_WORKERS` in the resource environment when a measured
workload needs another count, then use `vrooli resource restart reranker`.
Confirm the effective count with `resource-reranker gateway info --json`.

On 5 November 2026, a bounded 32/four/32-worker comparison on this CUDA host
measured about 7.21 GiB host RAM-plus-swap for each 32-worker arm versus 1.38 GiB
for four workers (81% lower). All six fixed relevance checks and distinguishable
rank-order checks passed; maximum normalized score difference was 0.000406.
The same baseline also varies between serial and batched requests. The earlier
exact-output gate failed and remains a rejected experiment. Serial sampled p95
was 94 ms; concurrency-four sampled p95 was 389 ms, within the prospectively
specified 20% regression bound against both control arms. These small shared-host
samples do not qualify production tails, high-concurrency throughput, or broader
retrieval quality. Model, dtype, client batch size and GPU batch-token settings
were unchanged. Retained protocol and raw results are in the local research home,
`topics/local-model-performance/runs/reranker-memory-confirmation-20261105`.

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
capability. The CPU target is currently an explicit unsupported terminal target
because the pinned image's `libiomp5.so` dependency is outside the single-file
extraction boundary; it must not be treated as a working fallback until a
relocatable, closure-verified CPU bundle is published. macOS and Windows remain
unsupported until signed native bundles and target smoke evidence exist.
