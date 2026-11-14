# PII Detector — API & Operations

The pii-detector resource serves a Go CPU ONNX-Runtime process that detects PII
in text. For an overview see [`../README.md`](../README.md); for rationale see
[`../PRD.md`](../PRD.md).

## HTTP API (served by `server/`)

### `POST /detect`

Request:

```json
{ "text": "mail me at bob@example.com; card 4111 1111 1111 1111" }
```

Response:

```json
{
  "spans": [
    { "category": "EMAIL_ADDRESS", "start": 11, "end": 26, "text": "bob@example.com", "confidence": 0.99, "source": "regex" },
    { "category": "CREDIT_CARD",   "start": 33, "end": 52, "text": "4111 1111 1111 1111", "confidence": 0.95, "source": "regex" }
  ],
  "model_available": false,
  "model_id": ""
}
```

- `start`/`end` are **byte offsets** into the original UTF-8 text (half-open).
- `source` is `regex` (deterministic pass) or `model` (NER).
- `model_available:false` means the NER model is not loaded (stub build or no
  staged artifact); the regex/checksum pass still runs. This is a signalled
  degradation, not an error.
- Spans are **non-overlapping and sorted**. On overlap the higher-confidence span
  wins; on a tie the regex span wins (its checksums are exact).

### `GET /health`

`200 OK` with `{"status":"ok","model_loaded":<bool>,"model_id":"..."}`. Healthy
whenever the regex pass works; the model is a capability enhancement, not a
readiness precondition.

### `GET /info`

`{"model_id":"...","regex_pass":true,"categories":[...]}`.

## Deterministic categories

`EMAIL_ADDRESS`, `PHONE_NUMBER`, `US_SSN`, `CREDIT_CARD`, `IBAN_CODE`. Credit-card
requires a valid Luhn checksum; IBAN requires a valid ISO-13616 mod-97 check; SSN
rejects the structurally-invalid area/group/serial groups.

## Operations

```bash
resource-pii-detector gateway detect --text "<text>"          # human table
resource-pii-detector gateway detect --text-stdin --json       # JSON result
resource-pii-detector gateway health --json
resource-pii-detector models list                              # roles + catalog
resource-pii-detector models activate --model <id> --role pii.default
```

Capacity: the broker can `degrade --to regex-only` (release the NER model, serve
regex only) and `upshift` back to `model`.

## Building the server

```bash
make build-server          # regex-only stub engine (default; no ONNX)
make build-server-onnx ROOT=<onnxruntime-root>   # real NER (requires staged ONNX + approved Go binding)
```

See the `TODO(acquire)` markers in `../resource.json`, `../model-policy.json`,
and `server/engine_onnx.go` for the model + dependency checkpoints.

## Acquisition status

The serving binary is built from server/ (Go, reproducible -trimpath build). The pinned linux-amd64 sha256 is the DEFAULT (stub-engine, regex-only) build produced by `make build-server`; it serves the deterministic regex/checksum pass. REAL NER inference is implemented behind the `pii_onnx` build tag (server/engine_onnx.go) and uses the now-APPROVED governed dependency go/github.com/yalue/onnxruntime_go v1.25.0 (CGO-free dlopen; loads libonnxruntime.so at runtime). TWO token-classification tiers are served by the SAME engine (model-agnostic: reads graph I/O, id2label, and do_lower_case at load): the small tier (gravitee-io/bert-small-pii-detection, INT8, prebuilt ONNX from the Hub) and the large tier (ab-ai/pii_model_based_on_distilbert, DistilBERT, 65 labels; its ONNX is produced deterministically from the Hub safetensors by scripts/export_large_model.py). Both are checksum-pinned in model-policy.json models.*.artifact.files. Validated end-to-end: small TestRealModelDetectsPII + TestPIIEval (recall 13/13); large TestLargeModelDetectsPII + TestLargePIIEval (recall 11/12) — all green against ONNX Runtime 1.23.0 (ORT_API_VERSION 23, matching the v1.25.0 binding). Re-pin this binary sha when a `pii_onnx` release artifact (server + bundled libonnxruntime.so, sherpa-style) is published. The GLiNER zero-shot tiers from the original plan were deliberately not shipped (see docs/ZERO_SHOT_DECISION.md).

## `pii_onnx` environment

The pii_onnx build additionally reads: PII_DETECTOR_MODEL_DIR (explicit model dir; else resolved under PII_DETECTOR_MODEL_CACHE/<model-id>), PII_DETECTOR_ONNX_FILE (default model.quant.onnx), PII_DETECTOR_ONNXRUNTIME_LIB (path to libonnxruntime.so; else <bin>/../lib/libonnxruntime.so, the sherpa-style bundle layout), and the capacity ladder's PII_DETECTOR_LOAD_MODEL=on|off (off pins the server to the regex-only rung and releases the NER model's RAM). The default stub build ignores all of these.
