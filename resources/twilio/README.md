# Twilio Resource

Hosted Twilio API integration for messaging and communications workflows.

## Intent

- Resource ID: `twilio`
- Category: `communications`
- Driver: `cloud-api`
- Portability tier: `full`

## Use Cases

- Send SMS or voice notifications from scenarios and operator workflows.
- Integrate communications into automation pipelines without owning telephony infrastructure.
- Standardize Twilio credential and connectivity checks across local development.

## Architecture

This resource is being aligned to the updated `cloud-api` structure.

- `resource.json` is the declarative authority for endpoint, credentials, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Twilio-specific Go logic when the manifest and shared control plane are not enough.
- The supported operator surface is the Go CLI and shared control plane; no
  resource-local shell runtime is required.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add Twilio-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/config`: endpoint and account configuration helpers
- `cli/internal/auth`: credential validation and translation helpers
- `cli/internal/health`: provider-safe connectivity helpers
- `cli/internal/env`: environment export helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install twilio

# Check status through the shared control plane
resource-twilio status
```

Credentials can come from:

- `TWILIO_ACCOUNT_SID`
- `TWILIO_AUTH_TOKEN`
- Vault secret ref: `secret/resources/twilio`

## Notes

- This is a cloud API resource, not a local runtime owner.
- Keep `cli/main.go` thin. Do not move provider logic into CLI wiring.
- Keep endpoint, credential, and health expectations declarative in `resource.json` whenever possible.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/twilio/docs/OPERATIONS.md) as the architecture boundary for future migrations.
