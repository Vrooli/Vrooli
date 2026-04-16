# {{RESOURCE_DISPLAY_NAME}}

{{RESOURCE_DESCRIPTION}}

This scaffold was generated from the `desktop-app` resource template on {{CURRENT_DATE}}.

## Intent

- Resource ID: `{{RESOURCE_NAME}}`
- Category: `{{RESOURCE_CATEGORY}}`
- Driver: `desktop-app`
- Portability tier: `{{RESOURCE_PORTABILITY_TIER}}`

## Architecture

This template keeps the generated CLI thin on purpose.

- `resource.json` is the declarative authority for install, invoke, freshness, platform support, and detection metadata.
- `cli/` is entrypoint and command wiring only.
- `internal/` is the default home for desktop-app-specific Go logic when the manifest and shared control plane are not enough.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add resource-specific Go code under `internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Generated placeholder packages:

- `internal/discovery`: host-path and application detection helpers
- `internal/install`: install/bootstrap helpers
- `internal/platform`: platform gating and support policy helpers
- `internal/health`: app-specific verification helpers

## Next Steps

1. Keep config/cache/log state outside the repo through the resource storage/runtime layer.
2. Keep the generated CLI contract manifest-driven: `resource.json` owns `cli.command`, `cli.install`, `cli.invoke`, and `cli.freshness`.
3. Keep `cli/main.go` focused on bootstrap and delegation; put platform/detection logic in `internal/...`.
4. Document supported platforms and unsupported behavior honestly.
