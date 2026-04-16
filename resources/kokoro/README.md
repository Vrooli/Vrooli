# Kokoro Resource

Managed Kokoro text-to-speech runtime for local speech-synthesis workflows.

## Intent

- Resource ID: `kokoro`
- Category: `ai`
- Driver: `compose-service`
- Portability tier: `partial`

## Use Cases

- Generate spoken audio locally for voice and multimodal workflows.
- Provide a reusable text-to-speech service for scenarios and automation.
- Pair with speech-to-text resources for end-to-end voice pipelines.

## Architecture

This resource is being aligned to the updated `compose-service` structure.

- `resource.json` is the declarative authority for lifecycle, compose orchestration, ports, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Kokoro-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

The intended escalation path is:

1. express behavior in `resource.json` and `docker/docker-compose.yml`
2. rely on the shared `vrooli resource ...` control plane
3. add Kokoro-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/compose`: compose-specific runtime graph helpers
- `cli/internal/topology`: service dependency and readiness semantics
- `cli/internal/runtime`: runtime shaping helpers
- `cli/internal/health`: Kokoro-specific readiness helpers
- `cli/internal/env`: environment export helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install kokoro

# Check status through the shared control plane
resource-kokoro status
```

Default endpoint:

- API: `http://localhost:8880`

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for synthesis or voice workflows.
- Keep runtime state rooted in `${RESOURCE_*_DIR}` paths and compose-managed mounts rather than repo-local mutable directories.
- Existing shell-heavy workflows in `lib/` are transitional. New logic should land in Go under `cli/internal/...`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/kokoro/docs/OPERATIONS.md) as the architecture boundary for future migrations.
