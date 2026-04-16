# Operations

`twilio` is organized as a `cloud-api` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative endpoint, credential, lifecycle, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Twilio-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.
- `lib/` contains retained shell behavior only until the resource is fully migrated.

Do not turn `cli/main.go` into the implementation surface. If the resource needs specialized credential validation, request shaping, provider-safe probes, or environment derivation, grow `cli/internal/config`, `cli/internal/auth`, `cli/internal/health`, or `cli/internal/env` first.

## Operator Checklist

- Keep Twilio endpoint and credential wiring declared in `resource.json`.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.
- Move provider-specific shell workflows from `lib/` into `cli/internal/...` in focused slices.
- Distinguish safe smoke checks from quota-consuming or mutating provider actions.
