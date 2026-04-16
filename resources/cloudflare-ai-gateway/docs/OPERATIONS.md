# Operations

`cloudflare-ai-gateway` is organized as a `cloud-api` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative endpoint, credential, lifecycle, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Cloudflare-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

Do not turn `cli/main.go` into the implementation surface. If the resource needs specialized credential validation, provider shaping, or safe probe logic, grow `cli/internal/auth`, `cli/internal/config`, `cli/internal/health`, or `cli/internal/env` first.

## Operator Checklist

- Keep Cloudflare account and token wiring declared in `resource.json`.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.
- Keep provider-specific validation and request shaping in `cli/internal/...`, not in shell wrappers.
- Document which API actions are safe for smoke checks versus mutating administrative actions.
