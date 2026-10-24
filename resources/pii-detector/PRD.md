# PII Detector Resource — PRD

## Problem

Vrooli has no PII detection anywhere. The ai-gateway carries `prompt_redacted` /
`response_redacted` route-evidence columns that nothing meaningfully populates,
agent-manager's W3 `redact` policy action is a recognized no-op, and there is no
reusable capability any scenario or program can call to find and mask PII in
text. A general small LLM (Ollama) cannot fill the gap: the strong PII models are
encoder-only token classifiers that llama.cpp cannot run, and an LLM is slower,
non-deterministic, and can hallucinate or emit malformed spans.

## Solution

A dedicated managed-service resource that detects PII in text with a **two-pass
hybrid**:

- **Deterministic regex + checksum pass** for structured IDs (email, phone, SSN,
  credit-card via Luhn, IBAN via mod-97). High precision, zero model cost,
  always available.
- **NER token-classification model** for names/orgs/locations and the broad
  category set, served over a **CPU ONNX Runtime** process (the operator-chosen
  runtime; the reranker's GPU-only TEI cannot do token classification).

Merged into one non-overlapping span set over `POST /detect`. Tiered by
`model-policy.json`: `pii.small` (INT8 BERT, default) and `pii.large`
(DistilBERT `ab-ai/pii_model_based_on_distilbert`, 65 labels) — both Apache-2.0
token-classification models served by the SAME engine; `pii.default` aliases the
small model for the gateway routing default. `fallbacks` + a capacity ladder
(`model` ↔ `regex-only`) and `upshift`, mirroring ollama/reranker. The planned
GLiNER zero-shot tiers were not shipped — see docs/ZERO_SHOT_DECISION.md.

## Architecture

- **CLI gateway** (`cli/`, root `resources` module) — `resource-pii-detector
  gateway detect|health|info`, `models list|status|activate`, lifecycle +
  capacity companions. Resolves the service via exported `PII_DETECTOR_URL`;
  scenarios never hand-roll HTTP. Mirrors `resources/reranker`.
- **Serving process** (`server/`, standalone module, `GOWORK=off`) — an
  engine-agnostic HTTP layer over a `NERModel` seam. The real ONNX inference is
  confined to a `//go:build pii_onnx` file; the default build ships a
  fail-closed stub so the service runs the regex pass only and remains buildable
  and testable without the native binding. Mirrors `resources/sherpa-onnx`.

## Consumers

- **ai-gateway** — `REQUEST_KIND_DETECT_PII`; `ProviderPiiDetector` adapter;
  `detect.pii.default` inference role. Populates the redacted route-evidence seam
  when a detect pass runs.
- **agent-manager** (follow-up) — the W3 `redact` action routes matched content
  through this detector at the import sink and/or search time.

## Non-goals

- Redaction/masking *policy* (which categories to mask, with what token). The
  detector returns spans; the caller decides masking. agent-manager's redact
  action owns that policy.
- A measured accuracy claim. The model-landscape numbers are self-reported; the
  `pii.detection` suite measurements stay zeroed (`TODO(measure)`) until our own
  corpus evaluation runs. `measurement_required` is deliberately `false` during
  the scaffold.

## Open checkpoints (honest completeness)

1. **onnxruntime Go binding** — a governed dependency approval is required before
   the `pii_onnx` build can link real inference. Not self-installable.
2. **Model acquisition** — the three HF ONNX models need staging with pinned
   sha256 checksums (mirror reranker's OCI pinning). `TODO(acquire)`.
3. **Measurement** — run the `pii.detection` corpus eval, populate measurements,
   then flip `measurement_required` to true.
