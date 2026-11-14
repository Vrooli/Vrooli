# PII Detector Resource

Tiered **PII detection** for text. A managed CPU **ONNX-Runtime** serving process
combines two passes:

1. A **deterministic regex + checksum pass** ("Presidio-style") for structured
   identifiers — email, phone, US SSN, credit-card (**Luhn**), IBAN (**mod-97**).
   Cheap, deterministic, high-precision; always on, even with no model staged.
2. An **NER token-classification model** (BERT / GLiNER) for names, orgs,
   locations, and the broader category set. Tiered via `model-policy.json`.

The two passes are **merged** into one non-overlapping, sorted span set
(`POST /detect` → `{spans:[{category,start,end,text,confidence}]}`).

## Why a dedicated resource (not Ollama)?

Strong PII models are **encoder-only token classifiers** (BERT/DeBERTa/GLiNER).
Ollama / llama.cpp serve only `embed` and `generate` and cannot produce token
spans, so they cannot host this capability. The precedent is `resources/reranker`
(a third model type behind a managed service) and `resources/sherpa-onnx`
(the Go-server-wraps-native-ONNX build pattern). This resource follows both:
a typed Go CLI gateway + a Go serving process whose native inference is confined
behind a build tag.

## Tiers (model-policy roles)

| Role | Model | Size | License | Notes |
|---|---|---|---|---|
| `pii.small` (default) | `gravitee-io/bert-small-pii-detection` | ~29MB INT8 ONNX | Apache-2.0 | Uncased BERT token classifier, 25 Presidio-aligned categories; floor of every fallback chain. |
| `pii.default` | `gravitee-io/bert-small-pii-detection` | ~29MB INT8 ONNX | Apache-2.0 | Default routing alias (ai-gateway `detect.pii.default` resolves here). |
| `pii.large` | `ab-ai/pii_model_based_on_distilbert` | ~265MB fp32 ONNX | Apache-2.0 | Uncased DistilBERT token classifier, 65 labels / ~32 PII categories (names, SSN, IBAN, card, password, username, URL, age…). **Reuses the same engine** as small — a clean model swap. Degrades to small. |

Both served tiers are **token-classification** models that run through the same
`server/engine_onnx.go` (WordPiece → ONNX → BIO → byte-offset spans). The large
tier is a pure model swap, not a new code path. `fallbacks` + a capacity ladder
(`model` ↔ `regex-only`) with `upshift`, mirroring ollama/reranker.

The originally-planned GLiNER zero-shot tiers were **deliberately not shipped** —
GLiNER is a different architecture (SentencePiece + request-time label prompts +
span-score decode) that would not reuse the engine cleanly. See
[docs/ZERO_SHOT_DECISION.md](docs/ZERO_SHOT_DECISION.md).

## Status

What runs today:

- The **regex/checksum pass**, span-merge, HTTP `/detect` + `/health` + `/info`,
  the CLI gateway, and model-policy load/validate are **complete and tested**
  (`make test`). The server builds and serves **regex-only** out of the box
  (`model_available:false`).
- **Real NER inference** (both the small and large token-classification tiers)
  runs behind the `pii_onnx` build tag via the governed, approved dependency
  `go/github.com/yalue/onnxruntime_go` (CGO-free dlopen of `libonnxruntime.so`).
  Both tiers are **validated live** with byte-exact offsets + a labeled-fixture
  eval (`make test-server-onnx MODEL_DIR=… ONNXRUNTIME_LIB=…`).
- **Model artifacts** are checksum-pinned in `model-policy.json`. The small tier
  downloads a prebuilt INT8 ONNX from the Hub; the large tier's ONNX is produced
  deterministically from the Hub safetensors by
  `scripts/export_large_model.py` (the HF source files are the pinned,
  reproducibly-downloadable artifacts; the exported `model.onnx` sha256 is a
  recorded build-output fingerprint).

## Lifecycle

```bash
vrooli resource validate pii-detector     # manifest + contract export
vrooli resource install pii-detector       # build/install the CLI gateway
vrooli resource start pii-detector          # start the managed serving process
resource-pii-detector gateway detect --text "mail me at bob@example.com"
resource-pii-detector gateway health
resource-pii-detector models list
```

## Consumers

- **ai-gateway** — `REQUEST_KIND_DETECT_PII` / the `detect.pii.default` role; the
  first non-ollama/openrouter detector provider.
- **agent-manager** (follow-up, after W2+W5 land) — the W3 `redact` action
  scrubs matched content on ingest and/or at search time.

See [`docs/README.md`](docs/README.md) for the HTTP surface and operations,
[`PRD.md`](PRD.md) for product rationale.
