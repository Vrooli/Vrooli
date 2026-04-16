# Operations

`{{RESOURCE_NAME}}` is scaffolded as a single-container Docker-backed resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative lifecycle, runtime, port, health, and export metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns resource-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

Do not turn `cli/main.go` into the primary implementation surface. If the resource needs specialized install/runtime/status/health/env logic, grow the matching package under `cli/internal/` first.

## Operator Checklist

- Pin a production-safe image tag.
- Keep runtime storage rooted in `${RESOURCE_CONFIG_DIR}`, `${RESOURCE_DATA_DIR}`, `${RESOURCE_CACHE_DIR}`, `${RESOURCE_LOGS_DIR}`, and `${RESOURCE_STATE_DIR}` rather than repo-local `data/`.
- Document volume backup/restore expectations.
- Replace placeholder health checks with service-specific readiness probes.
- Prefer shared control-plane behavior first; use `cli/internal/install`, `cli/internal/runtime`, `cli/internal/status`, `cli/internal/health`, and `cli/internal/env` only for real specialization.
- Keep `environment_exports` as the canonical scenario-facing contract; do not reintroduce shell-based export wrappers.
- Treat custom CLI commands as an exception path, not the default architecture.
- Validate both the resource manifest and at least one consuming scenario before treating the scaffold as complete.
- Define upgrade and rollback procedures.
