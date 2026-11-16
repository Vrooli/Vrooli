# Operations

## Streaming transcription CPU tuning

The managed medium-model server uses 12 inference threads on the current
32-thread desktop host. This is the measured baseline setting for the
interactive audio-tools cell; the historical 12-thread benchmark in the
managed-service assessment remains unchanged. If the host has materially
fewer CPUs, qualify a separate setting rather than assuming this value.

`whisper` is organized as a checksum-verified `managed-service` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative lifecycle, native acquisition, port, export, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Whisper-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

Do not turn `cli/main.go` into the implementation surface. The native
compatibility and capacity edge belongs in `cli/internal/activityproxy`; model
and server delivery belongs in the managed-service acquisition contract.

## Operator Checklist

- Keep native server arguments, ports, and health checks declared in `resource.json`.
- Keep mutable uploads, models, and cache state in canonical resource storage paths.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.

## Native macOS Candidate Build

Upstream whisper.cpp v1.9.2 does not provide the native `whisper-server`
executable needed by this resource on macOS. A candidate can be built on a
matching Apple arm64 host from a source tree pinned to the v1.9.2 tag:

```bash
make -C resources/whisper build-native-macos-arm64 \
  WHISPER_CPP_ROOT=/path/to/whisper.cpp-v1.9.2
```

The recipe checks for Darwin/arm64 before invoking CMake, builds the upstream
`whisper-server` target in Release mode, and writes
`resources/whisper/dist/whisper-server_macos_arm64`. It deliberately does not
cross-compile, fetch dependencies, sign the executable, or update
`resource.json`. The candidate is not a supported acquisition artifact until
it has a reproducible release checksum, native-secure-store signing, and a
macOS arm64 managed-service smoke run.

## Capacity residency and wake ownership

The CPU rung starts the managed server with `--no-gpu`; it is CPU execution,
not an unloaded model. The public `activity-edge` companion is a required
resource-owned process and owns both the `/asr` demand boundary and the local
request endpoint. `/detect-language` is a compatibility probe, not
transcription demand.

The activity edge now owns a private Unix control socket beside its runtime-home
companion record. The managed-service restart path drains accepted work through
that socket, waits for the sequenced idle report, and keeps the edge alive while
the server is replaced. If a matched `/asr` request finds its private Unix
bridge absent, the edge requests a normal `vrooli resource start whisper` before
forwarding the body; the start still passes through capacity admission, health
checks, and observed placement. Health and language-detection probes do not
wake the model.

The `unloaded` capacity rung drains matched work, writes a durable model-bound
transition, stops the managed model server while retaining the edge, and
requires the managed process to be absent plus a fresh owner-attributed zero
VRAM sample before completing and releasing that exact claim. It can report
zero freed VRAM when the model was already on CPU; unload verification still
does not claim memory that was not measured as released. Failed or uncertain
work leaves the edge drained and the transition fenced. A later explicit start
replays or refreshes the transition before capacity admission, then resumes the
edge only after the restarted service passes health and placement checks. The
Unix control socket cleanup accepts only a stale socket belonging to the
recorded dead companion; live sockets, symlinks, regular files, and uncertain
listeners remain untouched.

These paths are source-tested only. No physical start, placement change, unload,
or wake journey was run in this delivery; actual serving placement, measured
freed bytes, timing, and recovery against host process/device inventory remain
unproven until an exact current operation admission exists.
