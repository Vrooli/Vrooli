# Acquisition and qualification status

## Pinned inputs

- Upstream model repository: `KaleComputer/kale-tts-1-nano`, revision
  `b6ce1887268c3667994fc3b0b2c82887c069915c`.
- Model card status at that revision: **DRAFT, NOT PUBLISHED**. Keep this
  resource experimental and do not make it a provider default.
- Int8 ONNX graph: 14,767,138 bytes; SHA-256
  `4d5269974de4fba7a66f947608a9a5381b4e39c1a8a6073df6001ab7e7f76288`.
  The value is present in the upstream `SHA256SUMS`.
- CPython: 3.12.14 standalone, release `20260814`, Linux x86_64; SHA-256
  `3297691ae34f75fed81ac424e040145fccb0bafe8e581cd5cadbddfa1c0766c0`.
- Python dependencies: direct versions in `requirements.in`; complete
  Linux/Python 3.12 wheel closure and SHA-256 wheel hashes in `requirements.lock`.
  `onnxruntime` is the CPU package; no CUDA packages or indexes are used.
- Upstream wrapper, QuickJS bundle, licenses and the four US-English frontend
  files are acquired as individual data artifacts from the pinned revision.
  Their SHA-256 values were checked by streaming each remote file through
  `sha256sum`; model and voice SHA values come from the pinned upstream
  `SHA256SUMS`. `resource.json` records all individual values and URLs.

## Deliberate source scope

The installed runtime includes the Nano int8 graph, `voices.json`,
`tokenizer.json`, ten voice embedding/extra-embedding pairs, upstream
`python/kale_tts.py` and `python/kale_frontend.js`, US-English lexicon,
overrides, terms and G2P binary, plus upstream license and third-party notices.
The 30.5 MB fp32 graph, PyTorch weights, teacher/student research weights,
training data, diagrams, benchmark outputs, browser assets, British frontend,
other models and full research tree are excluded.

The official third-party notice describes Apache-2.0 model/code, CC BY 4.0
training data, Kokoro-82M-derived weights and synthetic Maya-1 voice data. The
license and third-party notices are acquired alongside code and frontend
assets. Preserve them when redistributing installed artifacts.

## Current qualification and lifecycle status

The managed-service artifact tree is pinned in `resource.json` with SHA-256
`f70d4e25a86a3bb4ef9c269d99cc34164d267e7ffc4ab86a7c8b5d8708fe3061` for
Linux/amd64. Offline candidate composition reproduced this digest, and
`vrooli resource validate kale-tts` passes. The composed runtime and CPU wheels
occupy about 498 MiB before the separately acquired model and frontend data.

The resource CLI now exposes the standard shared lifecycle commands. `vrooli
resource install kale-tts` installed the pinned artifact, and managed startup
passed its HTTP readiness checks on Linux/amd64. `/healthz` returned ready
without loading ONNX, and `/v1/audio/voices` returned all ten pinned voices.
The service was stopped after the check to avoid retaining an extra runtime
process. The synthesis POST was blocked by the current Vrooli pre-command
policy, so no managed synthesis or product-path quality result is claimed.
The pinned upstream model remains draft and must not become a default
provider.

During first startup, the manifest's Python `entry_path` did not match the
composed archive layout. The installed artifact contains
`runtime/bin/python3.12`; `resource.json` now points to that pinned executable.
`vrooli resource validate kale-tts` passes with the corrected path.
