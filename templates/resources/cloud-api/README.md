# {{RESOURCE_DISPLAY_NAME}}

{{RESOURCE_DESCRIPTION}}

This scaffold was generated from the `cloud-api` resource template on {{CURRENT_DATE}}.

## Intent

- Resource ID: `{{RESOURCE_NAME}}`
- Category: `{{RESOURCE_CATEGORY}}`
- Driver: `cloud-api`
- Portability tier: `{{RESOURCE_PORTABILITY_TIER}}`

## Use Cases

Replace these bullets with the real scenario-facing uses for this resource.

- Serve as the hosted provider integration for scenarios that need {{primary function}}.
- Support {{secondary workflow}} without each scenario owning provider-specific auth and endpoint handling.
- Provide a foundation for {{integration pattern}} across the Vrooli stack.

## Architecture

This template keeps the generated CLI thin on purpose.

- `resource.json` is the declarative authority for install, invoke, freshness, endpoint, credential, and health metadata.
- `cli/` is the single binary entrypoint and command wiring surface.
- `cli/internal/` is the default home for provider-specific Go logic when the manifest and shared control plane are not enough.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add resource-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Generated placeholder packages:

- `cli/internal/config`: endpoint and provider configuration helpers
- `cli/internal/auth`: credential and auth validation helpers
- `cli/internal/health`: provider-specific connectivity and safe probe helpers
- `cli/internal/env`: environment/export helpers

## Next Steps

1. Keep any cached/config state outside the repo through the resource storage/runtime layer.
2. Keep the generated CLI contract manifest-driven: `resource.json` owns `cli.command`, `cli.install`, `cli.invoke`, and `cli.freshness`.
3. Keep `cli/main.go` focused on bootstrap and delegation; put provider-specific config/auth/health logic in `cli/internal/...`.
4. Replace the placeholder endpoint, health probe, and credential expectations.
