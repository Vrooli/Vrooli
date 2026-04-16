# Operations

`{{RESOURCE_NAME}}` is scaffolded as a cloud API resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative endpoint, credential, lifecycle, and health metadata.
- `cli/` owns entrypoint wiring and delegated command registration.
- `internal/` owns provider-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

Do not turn `cli/main.go` into the primary implementation surface. If the resource needs specialized credential validation, safe probe logic, or provider-specific configuration shaping, grow the matching package under `internal/` first.

## Operator Checklist

- Replace the placeholder endpoint and health URL.
- Wire credentials to the real secret source.
- Keep any local cache/config state in canonical resource storage directories, not repo-local `data/`.
- Prefer shared control-plane behavior first; use `internal/config`, `internal/auth`, `internal/health`, and `internal/env` only for real specialization.
- Document auth rotation and failure modes.
- Clarify which API actions are safe for smoke checks.
