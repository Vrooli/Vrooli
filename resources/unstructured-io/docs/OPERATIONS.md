# Operations

`unstructured-io` is organized as a `docker-service` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative runtime, lifecycle, port, export, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Unstructured-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.
- `resource-unstructured-io` and the docker-service driver are the supported operator surface.

Do not turn `cli/main.go` into the implementation surface. If the resource needs specialized runtime shaping, richer status interpretation, document-safe probes, or environment derivation, grow `cli/internal/install`, `cli/internal/runtime`, `cli/internal/status`, `cli/internal/health`, or `cli/internal/env` first.

## Operator Checklist

- Keep runtime image, ports, and health checks declared in `resource.json`.
- Keep mutable runtime state in canonical resource storage paths rather than repo-local ad hoc paths.
- Use `health`, `formats`, and `process` for retained document operations.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.
