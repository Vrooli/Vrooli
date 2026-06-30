# OpenRouter Resource

Hosted OpenRouter API gateway for access to multiple upstream model providers through one endpoint.

## Intent

- Resource ID: `openrouter`
- Category: `hosted-service`
- Driver: `cloud-api`
- Portability tier: `full`

## Use Cases

- Route scenario traffic across multiple hosted model providers behind one API.
- Compare or switch models without rewriting every scenario client.
- Add fallback and provider diversity for AI-powered workflows.

## Architecture

This resource now follows the updated `cloud-api` structure.

- `resource.json` is the declarative authority for endpoint, credentials, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for OpenRouter-specific Go logic when the manifest and shared control plane are not enough.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add OpenRouter-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/config`: endpoint and provider configuration helpers
- `cli/internal/auth`: credential resolution and redaction helpers
- `cli/internal/health`: provider-safe connectivity and generation helpers
- `cli/internal/env`: environment, path, and defaults helpers
- `cli/internal/app`: native operator commands over the internal Go packages

## Usage

```bash
# Install or validate the resource contract
vrooli resource install openrouter

# Check status through the shared control plane
resource-openrouter status

# List available models
resource-openrouter list-models
resource-openrouter content models --json --limit 20

# Resolve a model ROLE to a concrete model (the only model-selection authority)
resource-openrouter policy resolve --role chat.default --json
resource-openrouter policy resolve --role image.generate.logo --field model
resource-openrouter policy roles
resource-openrouter policy models
resource-openrouter policy constraints --json

# Generate a response (model is selected by ROLE via policy, not a slug)
resource-openrouter generate "Summarize OpenRouter routing in one paragraph"
echo "Explain tool routing simply" | resource-openrouter generate --role chat.small
resource-openrouter generate --role chat.quality --max-tokens 640 --prompt-file ./prompt.txt
# --model is an explicit advanced override; prefer --role.

# Store credentials for future commands
resource-openrouter configure --api-key sk-or-v1-example

# Show resolved runtime configuration (default role + resolved model)
resource-openrouter show-config --json
```

## Model Role Policy

OpenRouter is the **policy authority** for model selection. Concrete OpenRouter
model slugs live in exactly one place — [`model-policy.json`](/home/matthalloran8/Vrooli/resources/openrouter/model-policy.json).
Scenarios and resources declare the **roles** they need and resolve them through
`resource-openrouter policy resolve`; they never hard-code a provider slug or an
`OPENROUTER_*_MODEL` env default (enforced by the `openrouter_policy_facts`
repo-contract check).

- A **role** describes intent + capability (e.g. `chat.default`, `agent.tools`,
  `image.generate.logo`, `image.edit.identity`), an **endpoint** family (`chat`
  or `images`), and bounded **request defaults**.
- The policy carries the concrete default model plus an ordered **fallbacks**
  list and per-model capabilities/modalities/pricing metadata.
- Scenarios declare needed roles in `service.json` under
  `dependencies.resources.openrouter.model_roles`. At scenario start,
  `resource-openrouter ensure --config-base64 <json>` validates every declared
  role against the policy (it never downloads anything — OpenRouter is cloud
  hosted) and best-effort checks the live catalog.
- To change a model, edit `model-policy.json` (the single update point) — never a
  consumer.

Credentials can come from:

- `OPENROUTER_API_KEY`
- Vault secret ref: `secret/openrouter`

## Notes

- This is a cloud API resource, not a local runtime owner.
- Keep `cli/main.go` thin. Do not move provider logic into CLI wiring.
- Keep endpoint, credential, and health expectations declarative in `resource.json` whenever possible.
- `generate`, `list-models`, and `content models` are now native Go commands, not retained shell wrappers.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/openrouter/docs/OPERATIONS.md) as the architecture boundary for future migrations.
