# Operations

`openrouter` is organized as a `cloud-api` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative endpoint, credential, export, lifecycle, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns OpenRouter-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

Do not turn `cli/main.go` into the implementation surface. If the resource needs specialized credential validation, model/provider shaping, provider-safe probes, prompt handling, or environment derivation, grow `cli/internal/config`, `cli/internal/auth`, `cli/internal/health`, `cli/internal/env`, or `cli/internal/app` first.

The explicit native command surface is:

- `resource-openrouter list-models`
- `resource-openrouter content models`
- `resource-openrouter generate`
- `resource-openrouter configure`
- `resource-openrouter show-config`

## Operator Checklist

- Keep OpenRouter endpoint and credential wiring declared in `resource.json`.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.
- Keep provider-specific behavior implemented in `cli/internal/...`, not in ad hoc shell wrappers.
- Distinguish safe smoke checks from quota-consuming or mutating provider actions.
