# Operations

`gemini` is organized as a `cloud-api` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative endpoint, credential, lifecycle, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Gemini-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.
- Shell scripts are no longer the implementation surface for this resource.

Do not turn `cli/main.go` into the implementation surface. If the resource needs specialized credential validation, endpoint shaping, provider-safe probes, or environment derivation, grow `cli/internal/config`, `cli/internal/auth`, `cli/internal/health`, or `cli/internal/env` first.

## Operator Checklist

- Keep Gemini endpoint and credential wiring declared in `resource.json`.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.
- Keep provider-specific state and request logic in `cli/internal/...` rather than reviving shell wrappers.
- Distinguish safe smoke checks from mutating or quota-consuming provider actions.
- Treat old shell-only actions like `generate`, `list-models`, or `content` as non-contractual until they are reintroduced explicitly through the Go CLI surface.
- `generate` and `list-models` have now been reintroduced explicitly as native Go commands; they should evolve from `cli/internal/app` plus the package-level provider logic, not from shell.
