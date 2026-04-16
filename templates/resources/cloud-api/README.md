# {{RESOURCE_DISPLAY_NAME}}

{{RESOURCE_DESCRIPTION}}

This scaffold was generated from the `cloud-api` resource template on {{CURRENT_DATE}}.

## Intent

- Resource ID: `{{RESOURCE_NAME}}`
- Category: `{{RESOURCE_CATEGORY}}`
- Driver: `cloud-api`
- Portability tier: `{{RESOURCE_PORTABILITY_TIER}}`

## Architecture

This template keeps the generated CLI thin on purpose.

- `resource.json` is the declarative authority for install, invoke, freshness, endpoint, credential, and health metadata.
- `cli/` is entrypoint and command wiring only.
- `internal/` is the default home for provider-specific Go logic when the manifest and shared control plane are not enough.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add resource-specific Go code under `internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Generated placeholder packages:

- `internal/config`: endpoint and provider configuration helpers
- `internal/auth`: credential and auth validation helpers
- `internal/health`: provider-specific connectivity and safe probe helpers
- `internal/env`: environment/export helpers

## Next Steps

1. Keep any cached/config state outside the repo through the resource storage/runtime layer.
2. Keep the generated CLI contract manifest-driven: `resource.json` owns `cli.command`, `cli.install`, `cli.invoke`, and `cli.freshness`.
3. Keep `cli/main.go` focused on bootstrap and delegation; put provider-specific config/auth/health logic in `internal/...`.
4. Replace the placeholder endpoint, health probe, and credential expectations.
