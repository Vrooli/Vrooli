# Operations

`postgres` is organized as a native `managed-service` resource. The Linux
server tree is acquired and tree-verified from a digest-pinned OCI source and
launched directly without Docker.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative runtime, lifecycle, port, export, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns PostgreSQL-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.
- `lib/` contains retained shell behavior only until the resource is fully migrated.

Do not turn `cli/main.go` into the implementation surface. Add a focused
`cli/internal/...` package only when PostgreSQL behavior cannot be expressed
in the manifest or shared control plane. Do not pre-create empty packages.

## Operator Checklist

- Keep runtime image, ports, volumes, and health checks declared in `resource.json`.
- Keep mutable runtime state in canonical resource storage paths rather than repo-local ad hoc paths.
- Move shell workflows from `lib/` into `cli/internal/...` in focused slices instead of re-implementing them in CLI wiring.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.
