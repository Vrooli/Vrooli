# Operations

`k6` is organized as an `external-cli` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative install, invoke, binary, version, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns k6-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

Do not turn `cli/main.go` into the implementation surface. If the resource needs specialized binary discovery, install translation, version parsing, or cloud auth validation, grow `cli/internal/discovery`, `cli/internal/install`, `cli/internal/version`, `cli/internal/env`, or `cli/internal/auth` first.

## Operator Checklist

- Keep install guidance and minimum supported version declared in `resource.json`.
- Route mutable files through canonical resource storage instead of repo-local ad hoc paths.
- Separate binary discovery from optional Grafana Cloud auth or config checks.
- Prefer shared lifecycle and invoke behavior before adding resource-local commands.
