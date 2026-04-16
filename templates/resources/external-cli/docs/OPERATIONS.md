# Operations

`{{RESOURCE_NAME}}` is scaffolded as an external CLI resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative install, invoke, binary, version, and health metadata.
- `cli/` owns entrypoint wiring and delegated command registration.
- `internal/` owns external-tool-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

Do not turn `cli/main.go` into the primary implementation surface. If the resource needs specialized discovery, version parsing, auth validation, or install translation, grow the matching package under `internal/` first.

## Operator Checklist

- Document install guidance per OS.
- Pin a minimum supported version.
- Route mutable files through canonical resource storage directories instead of repo-local `data/`.
- Separate auth/config probing from binary detection.
- Prefer shared control-plane behavior first; use `internal/discovery`, `internal/install`, `internal/version`, `internal/env`, and `internal/auth` only for real specialization.
- Describe any interactive steps that cannot yet be automated.
