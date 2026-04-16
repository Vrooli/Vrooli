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

# Generate a response
resource-openrouter generate "Summarize OpenRouter routing in one paragraph"
echo "Explain tool routing simply" | resource-openrouter generate --model openai/gpt-4o-mini
resource-openrouter generate --model openai/gpt-4o-mini --max-tokens 640 --prompt-file ./prompt.txt

# Store credentials for future commands
resource-openrouter configure --api-key sk-or-v1-example

# Show resolved runtime configuration
resource-openrouter show-config --json
```

Credentials can come from:

- `OPENROUTER_API_KEY`
- Vault secret ref: `secret/openrouter`

## Notes

- This is a cloud API resource, not a local runtime owner.
- Keep `cli/main.go` thin. Do not move provider logic into CLI wiring.
- Keep endpoint, credential, and health expectations declarative in `resource.json` whenever possible.
- `generate`, `list-models`, and `content models` are now native Go commands, not retained shell wrappers.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/openrouter/docs/OPERATIONS.md) as the architecture boundary for future migrations.
