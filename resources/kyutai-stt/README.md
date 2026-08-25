# Kyutai STT Resource

Managed Kyutai streaming speech-to-text runtime for low-latency local
transcription. Wraps the Kyutai delayed-streams-modeling model (via the `moshi`
package) behind a **stable** FastAPI + WebSocket contract consumed by the
audio-tools scenario.

## Intent

- Resource ID: `kyutai-stt`
- Category: `ai`
- Driver: `managed-service` (composed CPython runtime and wheels; no Docker)
- Portability tier: `platform-specific` (qualified Linux amd64)

## Use Cases

- Low-latency streaming transcription for voice and multimodal workflows.
- A reusable streaming STT service for scenarios and automation.
- Pair with the native `sherpa-onnx` resource for end-to-end voice pipelines.

## Model

- Default: `kyutai/stt-1b-en_fr` (~1B params, English + French, ~0.5 s delay).
- Native model sample rate 24 kHz; the contract accepts **16 kHz mono PCM
  s16le** and resamples internally.
- VRAM: ~3–4 GB resident (bf16) — fits the local RTX 4070 Ti SUPER (16 GB)
  budget. The heavier `kyutai/stt-2.6b-en` (English only) is selectable via
  `KYUTAI_STT_HF_REPO`.

## Architecture

- `resource.json` is the declarative authority for lifecycle, acquisition,
  model data, ports, exports, health, and freshness metadata.
- `docker/server.py` is the stable streaming contract source and
  `docker/requirements.lock` is the managed-service wheel lock. The historical
  Docker files remain development references, not lifecycle inputs.
- `cli/` is the thin binary entrypoint and delegated command wiring.
- `cli/internal/` is the home for Kyutai STT-specific Go logic when the manifest
  and shared control plane are not enough.

Internal package boundaries (`cli/internal/`): `compose`, `topology`,
`runtime`, `health`, `env`.

## API (stable contract)

| Endpoint | Method | Purpose |
|---|---|---|
| `/health` | GET | `{"status":"ok","model_loaded":bool,"device":"cuda|cpu"}` |
| `/v1/info` | GET | `{"backend":"kyutai","model":...,"device":...,"sample_rate":16000,"version":...}` |
| `/v1/stream` | WebSocket | streaming transcription; PCM s16le 16 kHz mono in, `partial`/`segment`/`done`/`error` JSON out |

See [docs/API.md](/home/matthalloran8/Vrooli/resources/kyutai-stt/docs/API.md)
for the authoritative contract and
[docs/USAGE_EXAMPLES.md](/home/matthalloran8/Vrooli/resources/kyutai-stt/docs/USAGE_EXAMPLES.md)
for client examples.

## Usage

```bash
# Install / start (first run composes the runtime and downloads model files)
vrooli resource install kyutai-stt

# Status through the shared control plane
resource-kyutai-stt status
```

Default endpoints:

- HTTP: `http://localhost:8094`
- WebSocket: `ws://localhost:8094/v1/stream`

Environment exports for scenarios: `KYUTAI_STT_URL`, `KYUTAI_STT_BASE_URL`,
`KYUTAI_STT_WS_URL`.

## Notes

- Requires the qualified Linux amd64 target. CUDA is preferred when an NVIDIA
  device is available; hosts without it receive the explicit CPU fallback
  state and should use Whisper or sherpa for real-time production streaming.
- First run downloads four checksum-pinned model files into the managed
  resource data directory. The server receives explicit local paths and does
  not contact Hugging Face during model load.
- The model commit and each file digest are declared in `resource.json`.
- See [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/kyutai-stt/docs/OPERATIONS.md)
  for the architecture boundary, VRAM details, and troubleshooting.
## Maturity

M4 (2026-08-05): lifecycle, health, platform gates, and Go CLI test evidence are covered by the fleet contract.
