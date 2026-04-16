# Operations

`judge0` is organized as a `compose-service` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative lifecycle, compose, port, export, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Judge0-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.
- `lib/` contains retained shell behavior only until the resource is fully migrated.

Do not turn `cli/main.go` into the implementation surface. If the resource needs specialized compose graph handling, readiness semantics, runtime shaping, execution-safe probes, or environment derivation, grow `cli/internal/compose`, `cli/internal/topology`, `cli/internal/runtime`, `cli/internal/health`, or `cli/internal/env` first.

## Operator Checklist

- Keep compose topology, ports, and health checks declared in `resource.json` and `compose.yaml`.
- Keep mutable config and runtime state in canonical resource storage paths.
- Move shell workflows from `lib/` into `cli/internal/...` in focused slices instead of re-implementing them in CLI wiring.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.
