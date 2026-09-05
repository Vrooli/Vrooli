# Phase A6 — Docker migration audit — 2026-08-25

This phase is complete on 2026-08-25. x402, Kyutai, and Unstructured now use
managed-service lifecycle contracts on their qualified Linux targets, and the
fleet Docker blocker count is zero.

The current tree already satisfies the `audio-tools` optionality item:
`scenarios/audio-tools/.vrooli/service.json` declares `sherpa-onnx` with
`required: false`.

| Resource | Driver | Qualified route |
| --- | --- | --- |
| `x402-facilitator` | `managed-service` | Linux amd64/arm64 use the digest-pinned OCI filesystem extracted without Docker. macOS and Windows are explicitly unsupported because no signed native executable is published. |
| `kyutai-stt` | `managed-service` | Linux amd64 composes pinned CPython, governed wheels, and reviewed server source; four model files are acquired by commit-pinned URL and SHA-256. arm64, macOS, and Windows are explicitly unsupported. |
| `unstructured-io` | `managed-service` | Linux amd64 uses the pinned OCI-frozen Python runtime. macOS, Windows, and arm64 remain unsupported; `hi_res` is explicitly partial because `detectron2` is absent. |

The resource acquisition schema accepts URL, OCI image, composed and none
routes. The composed contract also permits a constrained local source step: it
copies only a resource-root-relative file into the composed tree, which is then
authenticated by the final tree digest. The x402 and Unstructured migrations
use OCI extraction without Docker, while Kyutai uses composed CPython/wheels
plus the local server source.

## x402 Linux lifecycle transcript

Commands were run through the control plane on 2026-08-25:

```text
vrooli resource install x402-facilitator --json       # pass
vrooli resource start x402-facilitator --json         # pass
vrooli resource status x402-facilitator --json       # running=true, healthy=true
curl -fsS http://127.0.0.1:14020/health             # HTTP 200
curl -fsS http://127.0.0.1:14020/supported          # HTTP 200, empty fail-closed catalog
vrooli resource stop x402-facilitator --json         # pass
```

The supervisor reported artifact version `1.3.0`, amd64 tree digest
`ea245e013e731945eef9e050b5670eb3d02e2534002af5010116c00b66c3e271`, and a
managed-service ownership token. The service logged a loopback bind at
`127.0.0.1:14020`.

## Unstructured Linux lifecycle transcript

The Linux amd64 artifact was acquired from the pinned upstream OCI image by the
shared binaryfetch path and launched through the managed-service supervisor. The
image is an acquisition source only; no Unstructured Docker container ran.

```text
go run ./cmd/vrooli --no-stale-check resource install unstructured-io --json # pass
go run ./cmd/vrooli --no-stale-check resource start unstructured-io --json   # pass
go run ./cmd/vrooli --no-stale-check resource status unstructured-io --json  # running=true, healthy=true
curl -fsS http://127.0.0.1:11450/healthcheck                           # HTTP 200
resource-unstructured-io health                                         # pass; readiness includes a text partition
ps -p 3229825 -o pid=,comm=,args=                                       # artifact loader + Python, no Docker command
go run ./cmd/vrooli --no-stale-check resource stop unstructured-io --json # pass
```

The supervised artifact version is `2025.09.11` with Linux amd64 tree digest
`38697ddb80a3ac739bd11d2255c70fd3d0d22b81786414b660b72b4476df4e35`. The
entrypoint is the image loader with the image library path and
`usr/bin/python3.12` as its explicit child; `PYTHONDONTWRITEBYTECODE=1` keeps
the authenticated tree immutable after launch. `hi_res` remains partial because
the frozen runtime does not include `detectron2`.

## Kyutai Linux lifecycle and hardware transcript

Commands were run through the control plane on 2026-08-25:

```text
go run ./cmd/vrooli --no-stale-check resource validate kyutai-stt --json # pass
go run ./cmd/vrooli --no-stale-check resource install kyutai-stt --json  # pass
go run ./cmd/vrooli --no-stale-check resource start kyutai-stt --json    # pass
go run ./cmd/vrooli --no-stale-check resource status kyutai-stt --json   # running=true, healthy=true, observed_mode=cuda
curl -fsS http://127.0.0.1:8094/health                           # model_loaded=true, device=cuda
curl -fsS http://127.0.0.1:8094/ready                            # HTTP 200
curl -fsS http://127.0.0.1:8094/v1/info                         # backend=kyutai, model=stt-1b-en_fr, device=cuda
python3 websocket smoke                                        # ready, processed_batches=1, done
go run ./cmd/vrooli --no-stale-check resource stop kyutai-stt --json # pass
```

The composed executable tree is version `2026.08.25` with Linux amd64 tree
digest `3046e77fb85334664053102c08b84ffc7f2678eea10c8088acf3ef369f73268f`.
The model source is Hugging Face commit
`1c34c6b4f7e9299bb61985f145052ff131005dde`; the manifest records separate
SHA-256 checksums for `config.json`, `model.safetensors`, Mimi weights, and the
English/French tokenizer. The service started from those local paths and the
WebSocket smoke completed without model download during startup.

Hardware evidence during the healthy run:

```text
NVIDIA GeForce RTX 4070 Ti SUPER, compute capability 8.9
nvidia-smi: Kyutai managed-service Python PID 3432551, 2878 MiB GPU memory
resource status: declared_mode=cuda, observed_mode=cuda, mode_drift=false
```

The old Docker-created root-owned `models/` directory was left untouched; the
managed data files use the user-owned resource data root. No Kyutai Docker
container or compose lifecycle was used by this migration.

## Fleet gate after all three migrations

`vrooli capability fleet --json` on 2026-08-25 reports zero Docker-blocked
cells. The baseline contained seven Docker-blocked cells, so the three
migrations remove all seven blockers. `audio-tools` still declares
`sherpa-onnx` optional, preserving its fallback behavior.
