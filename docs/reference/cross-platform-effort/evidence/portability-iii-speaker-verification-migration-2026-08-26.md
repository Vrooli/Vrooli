# Speaker Verification native migration evidence — 2026-08-26

## Boundary

Speaker Verification is now a `managed-service` resource with one native
composed acquisition. Dockerfiles, Compose overlays, and the container
lifecycle were removed. The qualified target is Linux amd64; arm64, macOS,
and Windows are explicitly unsupported for this release because the native
PyTorch/SpeechBrain wheel set was not qualified there.

The single Linux amd64 artifact uses a CUDA-enabled PyTorch 2.8.0 wheel lock.
The server selects CUDA automatically when available and retains a CPU fallback
with the same artifact, so there is one artifact for two device modes rather
than a second unverified CPU artifact. The lock is hash-pinned and the
acquisition installs the complete closure with `--no-deps`.

Measured artifact digest:

`46cae27aef0907cc04b0bcb3e07446775764a8d2a2521e9d96c032606c1a4bef`

The artifact includes `_soundfile_data/libsndfile_x86_64.so`, so libsndfile is
not a host dependency. The server can use `ffmpeg` for container decoding and
optional spectral denoise; `ffmpeg` is declared explicitly in `hostTools` as
an optional setup/development requirement. Dependency approvals were recorded
through Scenario Dependency Analyzer under the
`portability-truth-chain-iii` change authority.

## Native runtime proof

The control plane installed the resource and started it with
`vrooli resource start speaker-verification --json`. The first start exposed
stale root-owned cache state left by the former container path. The cache is
regenerable, so the manifest now declares a fresh `model-cache-native`
subdirectory and sets `PYTHONNOUSERSITE=1` for an isolated native Python
runtime; the old cache was left untouched.

After model warmup, the managed-service status was:

- artifact: `2026.08.26-torch-2.8.0`;
- health endpoint: `GET http://127.0.0.1:11452/ready` returned HTTP 200 with
  `model_loaded=true`, `profile_store_ok=true`, and `temp_dir_ok=true`;
- resource status: `running=true`, `healthy=true`, `serving=true`;
- observed mode: CUDA, with `nvidia-smi` reporting the supervised Python PID;
- `mode_drift=false`.

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
