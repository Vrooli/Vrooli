# Whisper Resource

Managed Whisper speech-to-text runtime for local transcription and translation workflows.

## Intent

- Resource ID: `whisper`
- Category: `ai`
- Driver: `compose-service`
- Portability tier: `partial`

## Use Cases

- Transcribe speech-to-text for scenario ingestion and automation pipelines.
- Translate spoken audio into English for downstream LLM or search workflows.
- Preprocess recordings, meetings, or voice notes into text that other resources can use.

## Architecture

This resource uses the updated `compose-service` structure.

- `resource.json` is the declarative authority for lifecycle, compose orchestration, ports, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Whisper-specific Go logic when the manifest and shared control plane are not enough.

The intended escalation path is:

1. express behavior in `resource.json` and `docker/docker-compose.yml`
2. rely on the shared `vrooli resource ...` control plane
3. add Whisper-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/compose`: compose-specific runtime graph helpers
- `cli/internal/topology`: service dependency and readiness semantics
- `cli/internal/runtime`: runtime shaping helpers
- `cli/internal/health`: Whisper-specific readiness helpers
- `cli/internal/env`: environment export helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install whisper

# Check status through the shared control plane
resource-whisper status

# Default transcription endpoint
curl http://localhost:8090/
```

## Activity Edge

Whisper clients must use `127.0.0.1:8090`. That port is owned by the host-side
`activity-edge` companion, which forwards to the compose container on
`127.0.0.1:18090`.

The edge is intentionally part of the serving path: it is the only component that
can see every `POST /asr` request from host dictation and browser clients, so it
also reports whisper active/idle state to the capacity broker. If the container
is healthy but `8090` refuses connections, the container is not the failing
piece; the companion is down, STT is unavailable, and capacity reporting is
blind. `vrooli resource status whisper` now reports that state distinctly, and
`vrooli-autoheal` can recover it with a cheap `vrooli resource start whisper`
reconcile that respawns the companion without restarting the container.

The shared companion launcher also honors the resource's
`orchestration.recovery_attempts` as a crash-loop cap for repeated stale-pid
respawns. For whisper, two dead-edge respawns inside the rolling window are
allowed; the next reconcile writes `activity-edge.failed` beside the pidfile and
the status JSON reports `failed: true` with the terminal reason. A deliberate
`vrooli resource stop whisper` clears the companion crash state.

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for transcription workflows.
- Keep runtime state rooted in `${RESOURCE_*_DIR}` paths and compose-managed mounts rather than repo-local mutable directories.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/whisper/docs/OPERATIONS.md) as the architecture boundary for future migrations.
## Maturity

M4 (2026-08-05): lifecycle, health, platform gates, and Go CLI test evidence are covered by the fleet contract.
