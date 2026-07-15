# Operations

`kyutai-stt` is organized as a `compose-service` resource that builds a local
Docker image from `docker/Dockerfile` (a custom FastAPI + websockets server
wrapping the Kyutai `moshi` streaming STT stack).

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative lifecycle, compose, port, export, and health
  metadata.
- `docker/` owns the server image: `Dockerfile`, pinned `requirements.txt`, and
  `server.py` (the stable streaming contract implementation).
- `cli/` owns the binary entrypoint, wiring, and delegated command
  registration. Keep `cli/main.go` thin.
- `cli/internal/` owns Kyutai STT-specific Go logic that cannot be expressed
  through the manifest or shared control-plane packages.
- `lib/` contains retained shell behavior (Docker lifecycle, status, API
  helpers, install) used by the shared control plane.

Do not turn `cli/main.go` into the implementation surface. Grow
`cli/internal/{compose,topology,runtime,health,env}` first if specialization is
needed.

## Hardware & VRAM

| Property | Value |
|---|---|
| GPU | NVIDIA CUDA required (RTX 40-series / Ada validated target) |
| Default model | `kyutai/stt-1b-en_fr` |
| Weights VRAM (bf16) | ~2–3 GB |
| Resident VRAM (with streaming buffers + CUDA context) | ~3–4 GB |
| Host GPU | RTX 4070 Ti SUPER, 16 GB (≈8 GB already used by other resources) |

The 1B model was chosen specifically because ~3–4 GB resident fits comfortably
in the remaining VRAM budget. The `kyutai/stt-2.6b-en` model (~6–8 GB, English
only, ~2.5 s delay) can be selected by setting `KYUTAI_STT_HF_REPO` but watch
the VRAM headroom against other resident resources.

CPU execution is possible but **not** real-time and is unsupported for
production streaming; the resource warns and continues if no GPU is present.

## First-run model download

On first start the container downloads model weights from Hugging Face into the
bind-mounted HF cache (`${RESOURCE_DATA_DIR}/models`, container `/models`).
This is multi-GB and can take several minutes. The startup timeout in
`resource.json` (`startup_timeout_seconds: 180`) and the lib wait window
(`KYUTAI_STT_STARTUP_MAX_WAIT=600`) account for this. Weights persist across
container recreations because the cache is a host bind mount.

The models are public; `KYUTAI_STT_HF_TOKEN` is optional and only needed to
avoid anonymous download rate limits.

## Ports & Environment Exports

| Name | Value | Notes |
|---|---|---|
| Host port | `8094` | container `8000`; chosen to avoid the registry (whisper 8090, llamaindex 8091, vrooli-api 8092, blender 8093) |
| `KYUTAI_STT_URL` | `http://localhost:8094` | HTTP base for `/health`, `/v1/info` |
| `KYUTAI_STT_BASE_URL` | `http://localhost:8094` | alias |
| `KYUTAI_STT_WS_URL` | `ws://localhost:8094/v1/stream` | streaming endpoint URL pattern for audio-tools |

### Segmentation / commit-cadence tuning

Kyutai is a *delayed-streams* model at 12.5 Hz. A durable `segment` is committed
on any of three triggers; the two frame-count knobs below are env-configurable
(set on the container, e.g. via `docker-compose*.yml`).

| Name | Default | Notes |
|---|---|---|
| `KYUTAI_STT_SILENCE_COMMIT_FRAMES` | `16` (~1.3 s) | A run of this many padding (no-text) frames after words is treated as a speaking pause and commits the pending segment. |
| `KYUTAI_STT_MAX_SEGMENT_FRAMES` | `48` (~3.8 s) | During *continuous* speech (no pause), force-commit the pending segment once it has spanned this many frames, at the next word boundary. Prevents a long unbroken utterance from stalling as a volatile partial until the end flush (silent tail-loss). `0` disables force-commit. |

The end flush (on `{"type":"end"}`) always drains the ~0.5 s delayed-streams
tail and commits the final segment regardless of these knobs; a cold
disconnect with no `end` does **not** flush, so clients must send `end`.

### Backpressure-safety / decoupling tuning

The decode loop is decoupled from socket sends: it enqueues events and keeps
stepping regardless of consumer speed, while a background send worker drains
them. Per the event-durability contract, `partial` events are coalesced to the
latest value and may be dropped under pressure (they never back-pressure
decode); `segment`/`done`/`error` events are ordered and lossless. This is what
makes a slow/stalled browser consumer unable to freeze the backend decode loop.
The knobs below bound teardown and the single-session lock.

| Name | Default | Notes |
|---|---|---|
| `KYUTAI_STT_SEND_DRAIN_TIMEOUT_S` | `5` | Bounded wait for the send worker to flush queued durable events to a slow consumer during teardown before the socket is force-closed. Committed text is flushed within this window; a dead consumer cannot hang teardown past it. |
| `KYUTAI_STT_ADMISSION_MAX_DEPTH` | `8` | Maximum FIFO admission queue depth, excluding the active decoder holder. Excess sessions receive `rejected/admission_full` before sending audio. |
| `KYUTAI_STT_ADMISSION_MAX_WAIT_S` | `30` | Bounded FIFO admission wait. Expired sessions receive `timed_out/admission_timeout`; clients retain audio for explicit recovery. |
| `KYUTAI_STT_START_FRAME_TIMEOUT_S` | `5` | Maximum wait for the required `start` control frame before joining admission. A bare WebSocket is rejected and cannot reserve the decoder. |
| `KYUTAI_STT_LOCK_TIMEOUT_S` | `10` | Bounded wait to acquire the single-session model lock before a new connection inspects the holder. |
| `KYUTAI_STT_ACTIVITY_WEDGE_S` | `5` | If the current holder decoded a frame within this many seconds, it is active and the newcomer receives `{"type":"error","code":"stt_busy",...}` instead of cancelling it. Only holders idle beyond this threshold are reaped as wedged. |
| `KYUTAI_STT_TORCH_COMPILE` | `0` | Enables moshi's experimental lazy `torch.compile` path when set to `1`. It is disabled by default because its first inference can spend many minutes compiling on a local GPU after health is already ready; CUDA graphing remains enabled. Enable only with measured cold and warm throughput evidence. |

Every stream logs a close summary with reason, frames consumed, segments
emitted, and duration. Reap warnings include holder age and idle time so active
dictation kills are distinguishable from real wedge recovery in logs.

### Throughput qualification

Run the opt-in live regression probe after changing the image, CUDA stack, or
decode configuration:

```bash
KYUTAI_STT_LIVE_PROBE=1 python3 -m pytest docker/test_live_throughput.py
```

It sends 30 seconds of canonical audio as 100 ms WebSocket frames and fails at
90 seconds. It also requires all 300 accepted batches to be acknowledged before
`done`; a terminal frame alone is not enough evidence of complete processing.

## Operator Checklist

- Keep compose topology, ports, and health checks declared in `resource.json`
  and `docker/docker-compose*.yml`.
- Keep mutable state (HF cache) in canonical resource storage paths via the
  compose bind mount; never repo-local.
- Pin Python deps in `docker/requirements.txt` and the base image tag in
  `docker/Dockerfile`. `torch` ships with the CUDA base image and is
  intentionally absent from `requirements.txt`.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding
  resource-local commands.

The Docker and Vrooli lifecycle checks use `/ready`, not `/health`: the latter
only proves that the process is alive, while `/ready` proves model admission is
safe.

## Common Operations

```bash
# Install / build / start (first run builds the image + downloads weights)
vrooli resource install kyutai-stt

# Status (text or JSON)
resource-kyutai-stt status
resource-kyutai-stt status --json

# Logs
resource-kyutai-stt logs

# Restart / stop
vrooli scenario restart kyutai-stt   # if consumed as a dependency
resource-kyutai-stt manage stop
```

## Troubleshooting

| Symptom | Likely cause | Action |
|---|---|---|
| `/health` 200 but `model_loaded:false` for minutes | first-run weight download | check `resource-kyutai-stt logs`; wait |
| Container unhealthy, CUDA errors in logs | no nvidia runtime / driver mismatch | confirm `docker info | grep nvidia` and `nvidia-smi` |
| `error: unsupported sample_rate` on stream | client sent non-16k `start` | send `sample_rate:16000`; resample client-side |
| OOM on model load | VRAM budget exhausted by other resources | free VRAM or switch to 1B model / lower concurrency |
