# Operations

`opencode` is organized as an `external-cli` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative install, invoke, binary, version, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns OpenCode-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

Do not turn `cli/main.go` into the implementation surface. If the resource needs specialized binary discovery, install translation, version parsing, auth validation, or config shaping, grow `cli/internal/discovery`, `cli/internal/install`, `cli/internal/version`, `cli/internal/env`, or `cli/internal/auth` first.

## Operator Checklist

- Keep upstream binary/install/version expectations declared in `resource.json`.
- Route mutable config and auth files through canonical resource storage instead of repo-local paths.
- Separate auth/config validation from raw binary detection.
- Prefer shared lifecycle and invoke behavior before adding resource-local commands.
