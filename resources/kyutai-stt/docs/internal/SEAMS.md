# Kyutai STT Resource — Internal Seams

This file enumerates the intentional testability / contract boundaries in the
kyutai-stt resource. New seams should land here when added; renames should
update the row.

## The stable WebSocket streaming contract (`server.py`)

| | |
|---|---|
| **Seam** | The `/v1/stream` wire protocol between the server and any client (primarily the audio-tools scenario). |
| **Surface** | `docker/server.py` — `start`/binary-PCM/`end` client frames; `partial`/`segment`/`done`/`error` server frames. Input is canonical PCM s16le 16 kHz mono. |
| **Why it exists** | audio-tools' canonical PCM substrate feeds this directly. Keeping the contract at 16 kHz (with internal resampling to the model's 24 kHz) decouples the scenario from the model's native rate. Changing the frame shapes or sample-rate contract is a breaking change and must bump the server major version. |
| **Validation** | `test/integration-test.sh` asserts `/v1/info` reports `backend=kyutai` + `sample_rate=16000`, `/health` reports `status=ok`, and the `/v1/stream` route is mounted. |

## Audio-rate normalization boundary (`Model.resample_to_model`)

| | |
|---|---|
| **Seam** | The single place 16 kHz contract audio is resampled to the model's native rate. |
| **Surface** | `docker/server.py::Model.resample_to_model` (uses `julius.resample_frac`, 16000 → `mimi.sample_rate`). |
| **Why it exists** | The model is format-blind internally; all rate handling is confined to one function so the public contract (16 kHz) never leaks the model's 24 kHz native rate. If a future model changes its native rate, only this function and `model_sample_rate` change. |

## Manifest-driven lifecycle (`resource.json`)

| | |
|---|---|
| **Seam** | Declarative contract for ports, env exports, health, GPU overlay, and freshness — consumed by the shared `vrooli resource ...` control plane and the `cli-core` resource app. |
| **Surface** | `resource.json` (`ports`, `environment_exports`, `health_checks`, `gpu`, `cli.freshness`). |
| **Why it exists** | Scenarios resolve `KYUTAI_STT_URL` / `KYUTAI_STT_WS_URL` from the manifest's derived exports; never hard-code the port. The GPU overlay (`docker/docker-compose.gpu.yml`) is applied only when the nvidia probe succeeds. |

## CLI app construction (`cli/main.go::newApp`)

| | |
|---|---|
| **Seam** | The resource CLI binary's wiring into `cli-core`'s `ResourceApp`. |
| **Surface** | `cli/main.go::newApp` → `cliapp.NewResourceApp(...)` + `StandardLifecycleCommands()`. |
| **Test fake** | `cli/main_test.go` asserts the stale-checker contract (`SourceContextPath=".."`, `ManifestSourcePath="resource.json"`, freshness inputs `cli/**` + `resource.json`). |
| **Why it exists** | Keeps `main.go` thin and delegates lifecycle to the shared control plane. Resource-local Go logic, if needed, grows under `cli/internal/...`. |
