# Kyutai STT Resource

Managed Kyutai streaming speech-to-text runtime for low-latency local
transcription. Wraps the Kyutai delayed-streams-modeling model (via the `moshi`
package) behind a **stable** FastAPI + WebSocket contract consumed by the
audio-tools scenario.

## Intent

- Resource ID: `kyutai-stt`
- Category: `ai`
- Driver: `compose-service` (builds a custom image from `docker/Dockerfile`)
- Portability tier: `platform-specific` (Linux + NVIDIA CUDA GPU)

## Use Cases

- Low-latency streaming transcription for voice and multimodal workflows.
- A reusable streaming STT service for scenarios and automation.
- Pair with the `kokoro` TTS resource for end-to-end voice pipelines.

## Model

- Default: `kyutai/stt-1b-en_fr` (~1B params, English + French, ~0.5 s delay).
- Native model sample rate 24 kHz; the contract accepts **16 kHz mono PCM
  s16le** and resamples internally.
- VRAM: ~3–4 GB resident (bf16) — fits the local RTX 4070 Ti SUPER (16 GB)
  budget. The heavier `kyutai/stt-2.6b-en` (English only) is selectable via
  `KYUTAI_STT_HF_REPO`.

## Architecture

- `resource.json` is the declarative authority for lifecycle, compose
  orchestration, ports, exports, health, and freshness metadata.
- `docker/` holds the server image: `Dockerfile` (pinned PyTorch CUDA base),
  pinned `requirements.txt`, and `server.py` (the stable streaming contract).
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
# Install / build / start (first run builds image + downloads weights)
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

- Requires an NVIDIA CUDA GPU for real-time streaming. The GPU compose overlay
  (`docker/docker-compose.gpu.yml`) is applied automatically when the nvidia
  probe succeeds.
- First run downloads multi-GB weights into the bind-mounted HF cache; they
  persist across container recreations.
- `KYUTAI_STT_HF_TOKEN` is optional (public models); only needed to avoid
  anonymous download rate limits.
- See [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/kyutai-stt/docs/OPERATIONS.md)
  for the architecture boundary, VRAM details, and troubleshooting.
