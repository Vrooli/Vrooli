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

## GPU vs CPU

GPU is the default when an NVIDIA GPU is present. The image is built on the
PyTorch CUDA runtime base (torch ships with a CUDA build), and the manifest
`gpu` block probes for NVIDIA: when it passes, the resource manager applies the
GPU overlay (`docker/docker-compose.gpu.yml`, `runtime: nvidia` + device
reservation) and sets `SPEAKER_VERIFICATION_DEVICE=cuda`. ECAPA verification is
small, but SepFormer target-extraction is heavy enough that the GPU matters for
interactive latency.

The same image runs on CPU-only hosts: the server resolves the device at
startup and downgrades `cuda`→`cpu` when no GPU is visible (logged at boot and
reported by `/v1/info` as `device`, `torch_version`, `cuda_available`). Force a
mode with `VROOLI_GPU=on|off|auto` (auto is the default and runs the probe).

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
