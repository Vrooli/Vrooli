# Operations

`claude-code` is organized as an `external-cli` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative install, invoke, binary, version, export, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Claude Code-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.
- `lib/` contains retained shell behavior only until the resource is fully migrated.

Do not turn `cli/main.go` into the implementation surface. If the resource needs specialized binary discovery, install translation, version parsing, config-path handling, or auth validation, grow `cli/internal/discovery`, `cli/internal/install`, `cli/internal/version`, `cli/internal/env`, or `cli/internal/auth` first.

## Operator Checklist

- Keep install guidance and minimum supported version declared in `resource.json`.
- Route mutable config and runtime files through canonical resource storage instead of repo-local ad hoc paths.
- Separate binary discovery from login/config validation.
- Prefer shared lifecycle and invoke behavior before adding resource-local commands.
