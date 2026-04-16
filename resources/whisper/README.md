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

This resource is being aligned to the updated `compose-service` structure.

- `resource.json` is the declarative authority for lifecycle, compose orchestration, ports, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Whisper-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

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

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for transcription workflows.
- Keep runtime state rooted in `${RESOURCE_*_DIR}` paths and compose-managed mounts rather than repo-local mutable directories.
- Existing shell-heavy workflows in `lib/` are transitional. New logic should land in Go under `cli/internal/...`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/whisper/docs/OPERATIONS.md) as the architecture boundary for future migrations.
