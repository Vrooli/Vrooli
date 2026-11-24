# Kale TTS Nano (experimental)

This resource is a local-only experimental managed service for Kale TTS 1 Nano.
The model repository is pinned to `b6ce1887268c3667994fc3b0b2c82887c069915c`;
the int8 ONNX graph is 14,767,138 bytes (SHA-256
`4d5269974de4fba7a66f947608a9a5381b4e39c1a8a6073df6001ab7e7f76288`). The
Hugging Face metadata reports this pinned revision public, non-gated, and
Apache-2.0. This resource must remain experimental until product quality and
lifecycle qualification pass; license status alone does not qualify it as a
default provider.

The adapter uses the upstream Python ONNX Runtime implementation and its
QuickJS frontend. Runtime session construction and voice embedding loading are
lazy; voice listing reads only pinned JSON metadata. Synthesis is serialized by
the runtime lock, Uvicorn uses one worker with room for eight simultaneous HTTP
requests (including health probes), and ONNX/BLAS are configured for one CPU
thread. This bounds inference resources while keeping liveness probes from
rejecting synthesis at the former two-request server limit.
Requests accept up to 5,000 input characters and a 64 KiB JSON body. The server
binds to 127.0.0.1.

The acquisition contract fetches only the int8 ONNX graph, metadata, tokenizer,
the ten small voice embedding pairs, the upstream Python wrapper/frontend
bundle, the US-English frontend lexicon/overrides/terms/G2P asset, and the
upstream license notices. It does not fetch the fp32 graph, PyTorch weights,
teacher models, training data, diagrams, benchmarks, or the full research
repository. These source and model files use the pinned upstream revision.
The Linux/amd64 composed runtime now uses the pinned CPython
`install_only_stripped` archive. The managed bundle fell from 498 MiB to 257
MiB (about 241 MiB / 48%) and is installed and running with artifact SHA-256
`cf38c6dc79ff578c40454b79c12b35356f6c4049de43b013b525d3fb95facb63`. The
resource manifest retains a 512 MiB RAM and 768 MiB disk budget. After the
managed update, the product API selected `kale-tts-1-nano` and returned valid
24 kHz mono PCM16 WAVs: the first post-start request took 1,073.5 ms for 2.55
seconds, and a distinct warmed request took 113.8 ms for 3.05 seconds. A
documented 20-request sweep covered all ten voices (median 75.2 ms, max 614.13
ms, no empty, near-silent, or clipped output); the receipt is
`~/.local/share/vrooli/audio-tools/qualification/kale-nano-stripped-runtime-20261123.json`.
The serving process reached about 225 MiB high-water RSS during these requests.
These checks establish lifecycle, routing, and signal integrity, not
naturalness or held-out quality. The generic model resolver still lists
Kokoro as the static TTS default; Audio Tools conditionally routes synthesis
through Kale while it is healthy. Kale remains experimental pending human
quality review. See the dated evidence in
`scenarios/audio-tools/docs/internal/PROGRESS.md`.

## HTTP API

- `GET /healthz`: reports readiness from the local model graph and whether the
  model metadata and frontend bundle are present, plus whether the ONNX
  session has been loaded. This endpoint does not load the graph.
- `GET /v1/audio/voices`: lists the model's available voices.
- `POST /v1/audio/speech`: JSON `{ "model": "kale-tts-1-nano", "input":
  "Hello.", "voice": "kenzie", "response_format": "wav" }`. Returns mono,
  24 kHz WAV, or headerless signed 16-bit little-endian PCM for `pcm`; PCM
  responses include `X-Sample-Rate: 24000`.

The endpoint is intended for same-host callers only and has no authentication.
Do not expose it on a network interface. English is the only qualified
language. The upstream notice describes voices as synthetic; do not represent
generated audio as a human recording or use it to impersonate a person. Latency
and speech quality have not been independently qualified.

## Licensing and source scope

Upstream model and code declare Apache-2.0; the training dataset is CC BY 4.0.
The full upstream research repository is intentionally not included. The
runtime needs the int8 graph, voice embeddings, tokenizer, and English frontend
assets only. The acquisition retains upstream `LICENSE` and
`THIRD_PARTY_NOTICES.md`; frontend lexicon/G2P assets retain their upstream
notices. The notice says the model derives from Kokoro-82M and its voice data
from Maya-1-generated audio; keep those notices with installed copies.
