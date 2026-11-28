# Acquisition and qualification status

## Pinned inputs

- Upstream model repository: `KaleComputer/kale-tts-1-nano`, revision
  `b6ce1887268c3667994fc3b0b2c82887c069915c`.
- Hugging Face API metadata at the pinned revision reports the model as public,
  non-gated, and `apache-2.0` (last modified 2026-11-19). This updates earlier
  draft-status notes. Keep the resource experimental until product quality and
  lifecycle qualification are complete; license status alone does not qualify
  the model for a provider default.
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
`cf38c6dc79ff578c40454b79c12b35356f6c4049de43b013b525d3fb95facb63` for
Linux/amd64. It uses the upstream CPython `install_only_stripped` archive
(SHA-256 `5acfa3e9ba26b51ae161c83aff278da915b590d22373a424b2ba55b8afe91fcc`).
Offline candidate composition reproduces the artifact digest, and
`vrooli resource validate kale-tts` passes. Replacing the previous unstripped
runtime reduced the installed bundle from 498 MiB to 257 MiB (about 48%).
`vrooli resource status kale-tts` confirms the new digest is installed,
serving, and healthy; the manifest budgets 512 MiB RAM and 768 MiB disk.

The resource CLI exposes the standard shared lifecycle commands. After the
managed reacquire and start on 2026-11-23, the product route selected
`kale-tts-1-nano` and produced valid 24 kHz mono PCM16 WAVs. The first
post-start request took 1,073.5 ms for 2.55 seconds of audio; a distinct warm
request took 113.8 ms for 3.05 seconds. The documented integrity sweep then
completed 20 requests across all ten pinned voices (median 75.2 ms, max 614.13
ms; no empty, near-silent, or clipped responses). The saved receipt is
`~/.local/share/vrooli/audio-tools/qualification/kale-nano-stripped-runtime-20261123.json`,
SHA-256 `67b012475d456861740d13426f35ba648088d6348b749db40d93fe319f9dddf2`.
The service process reached about 225 MiB high-water RSS during the sweep.
The generic model resolver still reports Kokoro as its static TTS default;
Audio Tools conditionally routes through Kale while healthy. These checks
establish packaging, lifecycle, route, and signal integrity only, not speech
quality. An earlier synthesis attempt was blocked by pre-command policy; this
managed product-path verification supersedes that failed attempt.
The pinned upstream model is public and Apache-2.0 according to the [Hugging
Face model metadata](https://huggingface.co/api/models/KaleComputer/kale-tts-1-nano).
Keep it experimental until naturalness and held-out quality are reviewed.

### Bounded runtime integrity sweep

`python3 resources/kale-tts/tools/qualify.py` checks the ready managed service,
then synthesizes two fixed short probes for each voice returned by the pinned
voice catalog (20 requests for the current ten voices). It emits a JSON receipt
to stdout with per-request latency, duration, peak/RMS level, clipping count,
and WAV SHA-256. Use `--voice kenzie --voice luna` to bound a run further or
`--url` to target another local service. Each request is synchronous; the
resource's concurrency limit and one-thread CPU configuration remain in force.

The sweep rejects malformed WAV, empty or near-silent audio, and durations
outside 0.1–30 seconds. These checks establish service and signal integrity
only. The fixed probes are not human-reviewed references and do not measure
recognition accuracy, pronunciation correctness, intelligibility, or
naturalness. A passing receipt cannot promote Nano to an automatic default.

During first startup, the manifest's Python `entry_path` did not match the
composed archive layout. The installed artifact contains
`runtime/bin/python3.12`; `resource.json` now points to that pinned executable.
`vrooli resource validate kale-tts` passes with the corrected path.
