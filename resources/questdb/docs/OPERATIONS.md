# Operations

`questdb` is organized as a `docker-service` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative runtime, lifecycle, port, export, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns QuestDB-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.
- `lib/` contains retained shell behavior only until the resource is fully migrated.

Do not turn `cli/main.go` into the implementation surface. If the resource needs specialized runtime shaping, richer status interpretation, QuestDB-specific probes, or environment derivation, grow `cli/internal/install`, `cli/internal/runtime`, `cli/internal/status`, `cli/internal/health`, or `cli/internal/env` first.

## Operator Checklist

- Keep runtime image, ports, volumes, and health checks declared in `resource.json`.
- Keep mutable runtime state in canonical resource storage paths rather than repo-local ad hoc paths.
- Move shell workflows from `lib/` into `cli/internal/...` in focused slices instead of re-implementing them in CLI wiring.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.
