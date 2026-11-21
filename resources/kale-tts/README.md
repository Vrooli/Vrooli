# Kale TTS Nano (experimental)

This resource is a local-only experimental managed service for Kale TTS 1 Nano.
The model repository is pinned to `b6ce1887268c3667994fc3b0b2c82887c069915c`;
the int8 ONNX graph is 14,767,138 bytes (SHA-256
`4d5269974de4fba7a66f947608a9a5381b4e39c1a8a6073df6001ab7e7f76288`). The
upstream model card labels this revision **DRAFT, NOT PUBLISHED**. This resource
must remain experimental and must not be selected as a default provider.

The adapter uses the upstream Python ONNX Runtime implementation and its
QuickJS frontend. Runtime session construction and voice embedding loading are
lazy; voice listing reads only pinned JSON metadata. Requests are serialized,
Uvicorn uses one worker, and ONNX/BLAS are configured for one CPU thread.
Requests accept up to 5,000 input characters and a 64 KiB JSON body. The server
binds to 127.0.0.1.

The acquisition contract fetches only the int8 ONNX graph, metadata, tokenizer,
the ten small voice embedding pairs, the upstream Python wrapper/frontend
bundle, the US-English frontend lexicon/overrides/terms/G2P asset, and the
upstream license notices. It does not fetch the fp32 graph, PyTorch weights,
teacher models, training data, diagrams, benchmarks, or the full research
repository. These source and model files use the pinned upstream revision.
The Linux/amd64 composed Python runtime and CPU wheels staged to 498 MiB; the
resource reserves 768 MiB for the runtime plus its model and frontend data.
The prior one-thread Python/ONNX CPU smoke used about 228 MiB RSS. Managed
startup was verified on Linux/amd64: `/healthz` returned ready without loading
ONNX, and `/v1/audio/voices` returned all ten pinned voices. The service was
stopped after this check. No managed synthesis or product-path quality result
has been recorded.

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
