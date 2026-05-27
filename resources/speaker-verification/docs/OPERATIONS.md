# Operations

`speaker-verification` is organized as a `compose-service` resource that builds
a custom image (`docker/Dockerfile`) wrapping a FastAPI server around
SpeechBrain's ECAPA-TDNN speaker-embedding model.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative lifecycle, compose, port, export, and health metadata.
- `docker/` owns the server (`server.py`), its pinned dependencies (`requirements.txt`), and the image (`Dockerfile`) plus compose topology.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Speaker Verification-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.
- `lib/` contains retained shell behavior only until the resource is fully migrated.

Do not turn `cli/main.go` into the implementation surface. If the resource needs
specialized compose graph handling, readiness semantics, runtime shaping, probe
logic, or environment derivation, grow `cli/internal/compose`,
`cli/internal/topology`, `cli/internal/runtime`, `cli/internal/health`, or
`cli/internal/env` first.

## Server Contract (load-bearing)

The HTTP contract in `docker/server.py` is consumed byte-for-byte by the
audio-tools Go client
(`scenarios/audio-tools/api/internal/stt/pipeline/speaker_client.go`). Endpoint
paths, multipart field names (`profile_id`, `display_name`, `notes`,
`threshold`, `verify`, and the `audio` file field), and response JSON keys MUST
NOT drift. See [docs/API.md](API.md) and [docs/internal/SEAMS.md](internal/SEAMS.md).

## CPU vs GPU

CPU is the default and recommended path — ECAPA-TDNN is a small model. The GPU
overlay (`docker/docker-compose.gpu.yml`) is opt-in: the manifest `gpu` block
probes for NVIDIA and, when present, flips `SPEAKER_VERIFICATION_DEVICE=cuda`.
The server falls back to CPU automatically if CUDA is unavailable at runtime.

## Operator Checklist

- Keep compose topology, ports, and health checks declared in `resource.json` and `docker/docker-compose.yml`.
- Keep enrolled profiles and the model cache in the bind-mounted volumes (`/data/profiles`, `/data/model-cache`), not repo-local mutable directories.
- The first start builds the image and (on first embedding request) downloads the ECAPA-TDNN weights into the model-cache volume. Budget time for both.
- Pin every docker base image and pip dependency version. Never introduce unpinned installs.
- Move shell workflows from `lib/` into `cli/internal/...` in focused slices instead of re-implementing them in CLI wiring.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.

## Live Validation

The image build and model download require an operator with Docker and network
access:

```bash
vrooli resource install speaker-verification
# or, directly:
cd resources/speaker-verification && docker compose -f docker/docker-compose.yml up --build -d
resource-speaker-verification status
bash test/integration-test.sh
```
