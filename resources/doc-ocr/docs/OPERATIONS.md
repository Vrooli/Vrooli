# Doc OCR operations

`doc-ocr` is the tier-2 local OCR handler. It is a native Go CLI with a
checksum-verified model data artifact, no runtime network access, and a stable
result contract: every run has text, a bounding box, and confidence.

## Engine comparison

| Candidate | License | Portable artifact | Model size | Accuracy/latency evidence | Decision |
| --- | --- | --- | ---: | --- | --- |
| Embedded text baseline (selected) | MIT | Go binary plus JSON data on Linux/macOS/Windows amd64+arm64 (cross-build smoke passed for all six pairs) | 0.3 KiB | deterministic fixtures: 0% CER for text-bearing pages; sub-10 ms | selected for the portable free path |
| ocrs + RTen | MIT | Rust/WASM described upstream, but early-preview runtime/model packaging was not produced in this phase | not measured | no comparable local fixture run | revisit when release artifacts stabilize |
| RapidOCR/PaddleOCR + ONNX Runtime | Apache-2.0/model-specific | multi-platform possible, but ONNX runtime and model bundle were not produced in this phase | not measured | no comparable local fixture run | rejected for this phase's no-toolchain target |
| Bundled Tesseract | Apache-2.0 | native libraries and language packs per platform | 20–80 MiB | the existing unstructured-io path measured 9.143 s on a low-resolution synthetic page and misread `SCANNED`/`scanned` | retained as an optional alternative |

The selected baseline is intentionally conservative: it guarantees a local,
checksum-verifiable positioned-run contract on every claimed pair. A model
upgrade must preserve the contract and add fixture CER/latency evidence before
changing the selection.

## Fixture CER comparison

The fixture set covers clean 300 DPI, low-resolution, rotated, multi-column,
and table pages. The portable baseline is exact on its text-bearing fixture
inputs; raster fixtures currently return a positioned low-confidence page run,
which is recorded as `N/A` for character error rate rather than being presented
as a successful transcription.

| Fixture class | doc-ocr CER | unstructured-io/Tesseract CER | Result |
| --- | ---: | ---: | --- |
| clean 300 DPI text | 0.00 | measured per host image | doc-ocr deterministic |
| low-resolution | N/A (confidence 0.35) | known synthetic substitutions (`SCANNED`→`SCANMED`, `scanned`→`scaimed`) | Tesseract remains the fallback |
| rotated page | N/A (confidence 0.35) | not measured in this phase | escalate by confidence |
| multi-column page | N/A (confidence 0.35) | not measured in this phase | preserve geometry for later model |
| table page | N/A (confidence 0.35) | not measured in this phase | preserve page anchor |

## Installation and readiness

The model is installed below `RESOURCE_DATA_DIR` as `ocr-model.json`. The CLI
checks its SHA-256 before `health` or `ocr`; missing or corrupt data fails
closed. The active operating mode is `cpu`, and the status output exposes that
mode. There is no GPU mode to silently downgrade from.

The release stager must publish the model at the manifest URL with SHA-256
`98d33ace6e54a09f43eb2671950324f50b49861b53617742269f8e54723482a5`.
