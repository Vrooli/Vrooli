# Decision: the large tier is a token-classification model swap, not GLiNER

Date: 2026-10-21 · Status: decided · Owner: pii-detector resource

## Context

The pii-detector resource ships a small default tier (`pii.small`,
`gravitee-io/bert-small-pii-detection`) that does real BERT token-classification
NER end-to-end: WordPiece tokenize → ONNX → per-token BIO logits → byte-offset
spans, in `server/engine_onnx.go` behind the `pii_onnx` build tag, merged with a
deterministic regex+checksum pass.

The original plan named the mid/large tiers as GLiNER models
(`knowledgator/gliner-pii-edge-v1.0`, `urchade/gliner_multi_pii-v1`). The
operator's hard constraint for finishing a larger tier was **clean over
complete**: do not force a fundamentally different architecture into the existing
engine if it would be messy.

## Why GLiNER is NOT a clean pure-Go fit here

GLiNER is a **zero-shot span architecture**, fundamentally different from the
small tier's BERT token classifier:

1. **Tokenizer.** GLiNER uses a DeBERTa-v2 / SentencePiece (unigram) tokenizer,
   not BERT WordPiece. The engine's offset-preserving WordPiece tokenizer could
   not be reused; a correct, offset-preserving SentencePiece unigram tokenizer
   (with the DeBERTa pre-tokenization and byte-fallback rules) is a substantial,
   separate implementation in pure Go, with no existing in-tree component to
   reuse.
2. **Protocol.** The entity label set is a **request-time input** encoded into
   the prompt (that is the "zero-shot" value — new categories need no
   retraining). The model takes extra label/prompt inputs the token classifier
   does not, so the `Detect(ctx, text)` seam would have to grow a label-set
   parameter and the HTTP contract would change.
3. **Decode.** The output is **span logits** (entity-start/entity-end or
   token-span scoring against the label embeddings), not per-token BIO. It needs
   GLiNER-specific span decoding (bi-encoder label/span matching + greedy
   span selection), not the existing `decodeBIO`.

None of these reuse the small tier's machinery; they replace it. Porting GLiNER
into the current engine would mean a second, parallel tokenizer + decode path of
real complexity — exactly the "messy port" the operator ruled out.

## Decision

Ship the large tier as a **larger token-classification PII model that reuses
`engine_onnx.go` verbatim**:

- `pii.large` = **`ab-ai/pii_model_based_on_distilbert`** — Apache-2.0, uncased
  `DistilBertForTokenClassification` (WordPiece `vocab.txt`, plain BIO, 65 labels
  over ~32 PII categories), ~265 MB fp32 ONNX (~9× the small INT8 model).

It is a pure model swap: the same WordPiece tokenizer, the same BIO decode, the
same ONNX graph-I/O + `id2label` resolution. The only engine change required was
making the tokenizer read `do_lower_case` from the model config instead of
hard-coding it — a clean, general improvement that lets the engine serve both
uncased and cased token classifiers correctly (a cased model lowercased at
tokenize time would drift the byte offsets). This keeps the engine uniform.

The large tier is validated live (`TestLargeModelDetectsPII`,
`TestLargePIIEval`: hybrid category recall 11/12 = 0.917) with byte-exact
offsets, and its artifacts are checksum-pinned in `model-policy.json`.

## If zero-shot PII is needed later

Zero-shot (arbitrary request-time categories without retraining) is a real,
distinct capability this resource does not provide. The clean ways to add it,
in preference order:

1. **GLiNER sidecar (recommended).** Run the official Python `gliner` /
   ONNX-gliner pipeline as a separate managed process (its own acquisition +
   model pin), exposed behind the same `POST /detect` contract with an optional
   `labels` field. This keeps the fast pure-Go token-classification path as the
   default and isolates the SentencePiece + span-decode complexity in the
   ecosystem that owns it. A new `pii.zeroshot` role would select it.
2. **A second, well-factored Go engine** (`engine_gliner.go`) behind the same
   `NERModel` seam, with its own SentencePiece tokenizer and span decoder,
   selected by the active model's family. Only worth it if the sidecar's process
   overhead is unacceptable; it is materially more code to get right than option 1.

Either path is a separate, scoped effort — not a prerequisite for the
token-classification large tier shipped here.
