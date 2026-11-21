# Operations

`twilio` is organized as a `cloud-api` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative endpoint, credential, lifecycle, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Twilio-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.
- There is no resource-local shell operator surface. The supported lifecycle is
  the Go CLI plus the shared control plane.

Do not turn `cli/main.go` into the implementation surface. If the resource needs specialized credential validation, request shaping, provider-safe probes, or environment derivation, grow `cli/internal/config`, `cli/internal/auth`, `cli/internal/health`, or `cli/internal/env` first.

## Operator Checklist

- Keep Twilio endpoint and credential wiring declared in `resource.json`.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.
- Distinguish safe smoke checks from quota-consuming or mutating provider actions.

## Provider Diagnostic

Use `resource-twilio provider-check` for the only supported provider-specific
operation. It sends an authenticated read-only request to the Accounts endpoint.
Missing credentials, credential rejection, and provider failures return errors.

## Shell Deletion Gate

The legacy `lib/`, `config/defaults.sh`, and `test/*.sh` tree was removed after
an inventory confirmed that every exported `twilio::` function was only
self-referenced inside that tree. No manifest command, Go CLI command,
scenario, or test invoked it. The supported behavior is the declarative
`resource.json` contract and the Go CLI's standard lifecycle commands.
