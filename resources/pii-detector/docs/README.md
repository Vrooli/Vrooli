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
