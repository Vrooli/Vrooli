# Cloudflare AI Gateway Resource

Hosted Cloudflare AI Gateway control plane for routing AI traffic through a managed cache, policy, and analytics layer.

## Intent

- Resource ID: `cloudflare-ai-gateway`
- Category: `ai`
- Driver: `cloud-api`
- Portability tier: `full`

## Architecture

This resource now follows the `cloud-api` template structure.

- `resource.json` is the declarative authority for endpoint, credentials, health checks, freshness, and lifecycle metadata.
- `cli/` is the single binary entrypoint and command wiring surface.
- `cli/internal/` is the default home for Cloudflare-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` is no longer the implementation surface for this resource. Provider-specific logic now lives in Go under `cli/internal/...`.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add resource-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/config`: endpoint and provider configuration helpers
- `cli/internal/auth`: credential and auth validation helpers
- `cli/internal/health`: provider-safe connectivity and smoke probe helpers
- `cli/internal/env`: environment/export helpers

Concrete Go-owned responsibilities now include:

- credential resolution from Vault or environment variables
- repo-external config/state storage for gateway metadata and named config payloads
- safe connectivity probing against the Cloudflare API
- derived runtime paths and exported environment values

## Usage

```bash
# Install or validate the resource contract
vrooli resource install cloudflare-ai-gateway

# Check status against the shared control plane
resource-cloudflare-ai-gateway status
```

## Credentials

Use either:

- `CLOUDFLARE_ACCOUNT_ID`
- `CLOUDFLARE_API_TOKEN`

or the declared secret source in `resource.json`.

Keep auth wiring declarative in `resource.json`. Only add logic to `cli/internal/auth` when Cloudflare-specific validation or translation is genuinely required.

## Notes

- This resource is a cloud API integration, not a local runtime owner.
- Keep the CLI thin. Do not move provider logic into `cli/main.go`.
- Prefer shared control-plane behavior first; grow `cli/internal/config`, `cli/internal/auth`, `cli/internal/health`, and `cli/internal/env` only where Cloudflare-specific behavior is real.
- The old shell-era gateway helpers were removed. New behavior for this resource should land in Go under `cli/internal/...`.

## References

- [Cloudflare AI Gateway Docs](https://developers.cloudflare.com/ai-gateway/)
- [API Token Management](https://dash.cloudflare.com/profile/api-tokens)
