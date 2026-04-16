# Gemini Resource

Hosted Google Gemini API integration for multimodal and text-generation workloads.

## Intent

- Resource ID: `gemini`
- Category: `hosted-service`
- Driver: `cloud-api`
- Portability tier: `full`

## Use Cases

- Call hosted Gemini models for text generation and analysis.
- Add multimodal reasoning to scenarios without owning local model runtime.
- Provide a cloud-model fallback path alongside local AI resources.

## Architecture

This resource now follows the updated `cloud-api` structure.

- `resource.json` is the declarative authority for endpoint, credentials, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Gemini-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` is no longer the implementation surface for this resource.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add Gemini-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/config`: endpoint and model-configuration helpers
- `cli/internal/auth`: credential validation and translation helpers
- `cli/internal/health`: provider-safe connectivity helpers
- `cli/internal/env`: environment export helpers

Concrete Go-owned responsibilities now include:

- Vault, environment, and compatibility-file Gemini API key resolution
- derived runtime settings for endpoints, default models, cache flags, and storage paths
- repo-external prompt/template/function storage for future explicit command wiring
- safe Gemini probe and model-listing logic that does not belong in `cli/main.go`

## Usage

```bash
# Install or validate the resource contract
vrooli resource install gemini

# Check status through the shared control plane
resource-gemini status

# List available Gemini models
resource-gemini list-models

# Generate a response directly
resource-gemini generate "Explain vector embeddings simply"
```

Credentials can come from:

- `GEMINI_API_KEY`
- Vault secret ref: `secret/vrooli/gemini`

## Notes

- This is a cloud API resource, not a local runtime owner.
- Keep `cli/main.go` thin. Do not move provider logic into CLI wiring.
- Keep credential, endpoint, and health expectations declarative in `resource.json` whenever possible.
- The old shell-era `generate/content/cache` helpers were intentionally retired instead of being treated as an implicit CLI contract. If Gemini needs those operator actions again, add them back explicitly on top of the Go packages under `cli/internal/...`.
- `generate` and `list-models` are now explicit native commands. They are part of the Go CLI surface rather than leftover shell behavior.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/gemini/docs/OPERATIONS.md) as the architecture boundary for future migrations.
