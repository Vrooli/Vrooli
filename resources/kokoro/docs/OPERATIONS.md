# Operations

`kokoro` is organized as a native `managed-service` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative lifecycle, acquisition, port, export, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Kokoro-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

Do not turn `cli/main.go` into the implementation surface. Keep native lifecycle decisions in the manifest and shared managed-service control plane.

## Operator Checklist

- Keep native acquisition, ports, and health checks declared in `resource.json`.
- Keep mutable config and runtime state in canonical resource storage paths.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.
