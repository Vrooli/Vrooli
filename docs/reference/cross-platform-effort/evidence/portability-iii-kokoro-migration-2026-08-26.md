# Kokoro native migration evidence — 2026-08-26

## Boundary

Kokoro is now a `managed-service` resource with a native composed acquisition.
The resource no longer owns Dockerfiles, Compose overlays, or a container
lifecycle. The qualified target is Linux amd64; arm64, macOS, and Windows are
explicitly unsupported for this release because their native wheels were not
qualified.

The acquisition is composed from:

- checksum-pinned CPython 3.12.14;
- target-specific, hash-pinned wheel closures for CPU and CUDA 12.6;
- the pinned Kokoro-FastAPI source archive; and
- the pinned model and configuration data artifacts.

The GPU target is selected by `gpu.cuda_compute >=8.9`; the CPU target is the
Linux amd64 fallback. Both targets use `--no-deps` installation from their
complete lock closure. Kokoro allows source distributions because the upstream
closure contains source-only packages; the exception is explicit as
`allow_sdists: true` rather than an implicit installer fallback.

Measured artifact digests:

| Target | Artifact SHA-256 |
|---|---|
| Linux amd64 CUDA | `db8375fc431b025f16217596d52973acca595e681c2864cbfeab253a0c075d57` |
| Linux amd64 CPU | `deb0268971cae48f88c089f90bf0937c9c321d17f56a1277ed54b1730a241ab8` |

Both lockfiles contain hashes for every resolved distribution. Dependency
approvals were recorded through Scenario Dependency Analyzer under the
`portability-truth-chain-iii` change authority; no raw package-manager install
was used for the resource dependency changes.

## Native runtime proof

The control plane installed the resource and started it with
`vrooli resource start kokoro --json`. The first attempt exposed a real
isolation defect: the standalone interpreter inherited an incompatible user
site `torchvision`, causing `torchvision::nms` registration to fail while
Transformers imported `AlbertModel`. The manifest now sets
`PYTHONNOUSERSITE=1`, so only the verified runtime and wheel tree participate
in imports.

The corrected start is healthy through the managed-service supervisor:

- artifact: `0.8.1-d5d1a695`;
- observed mode: CUDA, with `nvidia-smi` reporting the supervised Python PID;
- model warmup: completed on CUDA;
- voice packs loaded: 68;
- health endpoint: `GET http://127.0.0.1:8880/v1/audio/voices` returned HTTP
  200 and the voice list;
- resource status: `running=true`, `healthy=true`, `serving=true`,
  `mode_drift=false`.

Docker isolation could not be proven in this session. Docker was active with
the pre-existing `postgis-main` container, and both `sudo systemctl stop
docker` and unprivileged `systemctl stop docker` were rejected because this
session has no interactive sudo authentication. The successful start is
therefore native control-plane proof, not a Docker-stopped proof.

## Validation

- `go test ./internal/resources ./internal/resources/manifest`: pass.
- `make fleet-contract-check`: pass.
- `make acquisition-schema-check`: pass.
- `make lint-portability`: pass.
- `vrooli capability conformance --declarations-only --json`: zero findings and
  zero warnings for the declaration gate.
- `vrooli scenario test audio-tools`: server-owned run
  `20260826-175649-6c56343f` completed with 12/24 phases passing. The native
  service checks passed; the run also reports unrelated existing scenario
  portability, UI, provider-conformance, and baseline debt.
