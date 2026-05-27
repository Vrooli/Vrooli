# Speaker Verification Resource

Managed SpeechBrain ECAPA-TDNN speaker-embedding runtime for local speaker
enrollment and verification workflows.

## Intent

- Resource ID: `speaker-verification`
- Category: `ai`
- Driver: `compose-service`
- Portability tier: `partial`

## Model

- Model: `speechbrain/spkrec-ecapa-voxceleb` (ECAPA-TDNN)
- Embedding dimension: **192**
- Input sample rate: **16000 Hz** mono
- Verification: cosine similarity vs a caller-supplied threshold
- Reported EER: 0.8% on VoxCeleb1-test (cleaned)
- CPU-friendly (small model); GPU is an optional overlay

## Use Cases

- Enroll a speaker once, then gate downstream actions on "is this the same
  voice?" (cosine similarity against the enrolled voiceprint).
- Pair with `whisper` to attribute or gate transcription by speaker identity.
- Provide a reusable speaker-recognition service for voice scenarios such as
  `audio-tools`.

## Architecture

This resource follows the `compose-service` structure.

- `resource.json` is the declarative authority for lifecycle, compose
  orchestration, ports, exports, health, and freshness metadata.
- `docker/` holds the custom image: `server.py` (FastAPI), pinned
  `requirements.txt`, and `Dockerfile` (CPU torch base + speechbrain + fastapi +
  uvicorn + python-multipart + ffmpeg).
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the home for Speaker Verification-specific Go logic when the
  manifest and shared control plane are not enough.
- `lib/` contains retained shell behavior during the migration; new logic should
  move into `cli/internal/...` over time.

Internal package boundaries:

- `cli/internal/compose`: compose-specific runtime graph helpers
- `cli/internal/topology`: service dependency and readiness semantics
- `cli/internal/runtime`: runtime shaping helpers
- `cli/internal/health`: readiness helpers
- `cli/internal/env`: environment export helpers

## Usage

```bash
# Install / build the resource image and start it
vrooli resource install speaker-verification

# Check status through the shared control plane
resource-speaker-verification status
```

Default endpoint:

- API: `http://localhost:11452`

Exported environment variable for consumers:

- `SPEAKER_VERIFICATION_URL` (e.g. `http://localhost:11452`) — audio-tools reads
  this to resolve the service.

## API

See [docs/API.md](docs/API.md) for the full endpoint contract (`/ready`,
`/v1/info`, `/v1/profiles`, `/v1/verify`, `/v1/extract` [reserved], and
`DELETE /v1/profiles/{id}`) and [docs/USAGE_EXAMPLES.md](docs/USAGE_EXAMPLES.md)
for worked examples.

> Target-speaker **extraction** (`/v1/extract`) is a **reserved** capability and
> currently returns HTTP `501`. ECAPA-TDNN provides embeddings, not source
> separation; a dedicated separation model would be required to implement it.

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface.
- The HTTP contract in `docker/server.py` is consumed byte-for-byte by
  audio-tools; do not let endpoint paths, multipart field names, or response
  JSON keys drift. See [docs/internal/SEAMS.md](docs/internal/SEAMS.md).
- Keep runtime state (enrolled profiles, model cache) in compose-managed mounts
  rather than repo-local mutable directories.
- Use [docs/OPERATIONS.md](docs/OPERATIONS.md) as the architecture boundary for
  future migrations.
