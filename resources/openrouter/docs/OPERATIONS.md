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
- `resource-openrouter generate` (`--role` selects the model via policy; `--model` is an advanced override)
- `resource-openrouter configure`
- `resource-openrouter show-config`
- `resource-openrouter policy resolve|roles|models|constraints` — the model-role policy authority
- `resource-openrouter ensure --config-base64 <json>` — validate a scenario's declared `model_roles`

## Model Role Policy

`model-policy.json` is the single source of truth for concrete OpenRouter model
slugs. Roles (intent + capability + endpoint + request defaults) are the public
contract; consumers resolve them through `resource-openrouter policy resolve` and
never hard-code a slug or an `OPENROUTER_*_MODEL` env default. `ensure` validates
declared roles at scenario start and is non-destructive (cloud-hosted: it never
downloads). The default role exported to scenarios is `OPENROUTER_DEFAULT_ROLE`
(chat.default), not a model slug.

To retarget a role (e.g. a model was retired and `ensure` warns it is missing
from the live catalog), edit `model-policy.json` only — one file, with provenance.

## Operator Checklist

- Keep OpenRouter endpoint and credential wiring declared in `resource.json`.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.
- Keep provider-specific behavior implemented in `cli/internal/...`, not in ad hoc shell wrappers.
- Distinguish safe smoke checks from quota-consuming or mutating provider actions.
- Model selection changes happen in `model-policy.json`, never in consumer code.
