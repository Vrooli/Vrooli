# Whisper Resource

Managed Whisper speech-to-text runtime for local transcription and translation workflows.

## Intent

- Resource ID: `whisper`
- Category: `ai`
- Driver: `managed-service`
- Portability tier: Linux native supported; Windows acquisition conditional; macOS unsupported until a signed native server build exists

The macOS candidate build recipe is documented in
[docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/whisper/docs/OPERATIONS.md).
It is a native-host build aid only; it does not promote macOS support or
replace the signed acquisition entry in `resource.json`.

## Use Cases

- Transcribe speech-to-text for scenario ingestion and automation pipelines.
- Translate spoken audio into English for downstream LLM or search workflows.
- Preprocess recordings, meetings, or voice notes into text that other resources can use.

## Architecture

This resource uses the `managed-service` structure.

- `resource.json` is the declarative authority for lifecycle, checksum-pinned native acquisition, ports, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` contains the activity edge and its native `/asr` compatibility adapter.

The intended escalation path is:

1. express behavior in `resource.json` and its acquisition targets
2. rely on the shared `vrooli resource ...` control plane
3. add Whisper-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/activityproxy`: canonical port, capacity activity, and native request adaptation
- `cli/internal/recommend`: resource sizing and capacity-degrade policy

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
`activity-edge` companion, which forwards to the supervised native
`whisper-server` on `127.0.0.1:18090`.

The edge is intentionally part of the serving path: it is the only component that
can see every `POST /asr` request from host dictation and browser clients, so it
also reports whisper active/idle state to the capacity broker. If the native
server is healthy but `8090` refuses connections, the companion is down, STT is
unavailable, and capacity reporting is blind. `vrooli resource status whisper`
reports that state distinctly, and `vrooli-autoheal` can recover it with a
cheap `vrooli resource start whisper` reconcile that respawns the companion
without restarting the server.

The shared companion launcher also honors the resource's
`orchestration.recovery_attempts` as a crash-loop cap for repeated stale-pid
respawns. For whisper, two dead-edge respawns inside the rolling window are
allowed; the next reconcile writes `activity-edge.failed` beside the pidfile and
the status JSON reports `failed: true` with the terminal reason. A deliberate
`vrooli resource stop whisper` clears the companion crash state.

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for transcription workflows.
- Keep runtime state rooted in `${RESOURCE_*_DIR}` paths; the native server and
  its GGML model are acquired and verified outside the repository.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/whisper/docs/OPERATIONS.md) as the architecture boundary for future migrations.
## Maturity

M4 (2026-08-05): lifecycle, health, platform gates, and Go CLI test evidence are covered by the fleet contract.
