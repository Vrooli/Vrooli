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

This resource is being aligned to the updated `cloud-api` structure.

- `resource.json` is the declarative authority for endpoint, credentials, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Gemini-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

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

## Usage

```bash
# Install or validate the resource contract
vrooli resource install gemini

# Check status through the shared control plane
resource-gemini status
```

Credentials can come from:

- `GEMINI_API_KEY`
- Vault secret ref: `secret/vrooli/gemini`

## Notes

- This is a cloud API resource, not a local runtime owner.
- Keep `cli/main.go` thin. Do not move provider logic into CLI wiring.
- Keep credential, endpoint, and health expectations declarative in `resource.json` whenever possible.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/gemini/docs/OPERATIONS.md) as the architecture boundary for future migrations.
