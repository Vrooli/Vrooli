# Home Assistant Resource

Managed Home Assistant runtime for local automation and device-integration workflows.

## Intent

- Resource ID: `home-assistant`
- Category: `automation`
- Driver: `compose-service`
- Portability tier: `partial`

## Use Cases

- Run local home and device automations without depending on cloud control.
- Integrate sensors, devices, and automation events into scenario workflows.
- Provide a reusable automation hub for internal experimentation and orchestration.

## Architecture

This resource is being aligned to the updated `compose-service` structure.

- `resource.json` is the declarative authority for lifecycle, compose orchestration, ports, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Home Assistant-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

The intended escalation path is:

1. express behavior in `resource.json` and `compose.yaml`
2. rely on the shared `vrooli resource ...` control plane
3. add Home Assistant-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/compose`: compose-specific runtime graph helpers
- `cli/internal/topology`: service dependency and readiness semantics
- `cli/internal/runtime`: runtime shaping helpers
- `cli/internal/health`: Home Assistant-specific readiness helpers
- `cli/internal/env`: environment export helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install home-assistant

# Check status through the shared control plane
resource-home-assistant status
```

Default endpoint:

- Web/API: `http://localhost:8123`

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for automation, backup, or voice workflows.
- Keep runtime state rooted in `${RESOURCE_*_DIR}` paths and compose-managed mounts rather than repo-local mutable directories.
- Existing shell-heavy workflows in `lib/` are transitional. New logic should land in Go under `cli/internal/...`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/home-assistant/docs/OPERATIONS.md) as the architecture boundary for future migrations.
