# {{RESOURCE_DISPLAY_NAME}}

{{RESOURCE_DESCRIPTION}}

This scaffold was generated from the `compose-service` resource template on {{CURRENT_DATE}}.

## Intent

- Resource ID: `{{RESOURCE_NAME}}`
- Category: `{{RESOURCE_CATEGORY}}`
- Driver: `compose-service`
- Portability tier: `{{RESOURCE_PORTABILITY_TIER}}`

Use this template when the resource needs a coordinated runtime graph instead of a single container.

## Architecture

This template keeps the generated CLI thin on purpose.

- `resource.json` is the declarative authority for lifecycle, install, invoke, freshness, and orchestration metadata.
- `cli/` is the single binary entrypoint and command wiring surface.
- `cli/internal/` is the default home for compose-specific Go logic when the manifest and shared control plane are not enough.

The intended escalation path is:

1. express behavior in `resource.json` and `compose.yaml`
2. rely on the shared `vrooli resource ...` control plane
3. add resource-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Generated placeholder packages:

- `cli/internal/compose`: compose-specific graph and command helpers
- `cli/internal/topology`: service dependency and readiness semantics
- `cli/internal/runtime`: runtime shaping helpers
- `cli/internal/health`: resource-specific readiness helpers
- `cli/internal/env`: environment/export helpers

## Next Steps

1. Keep runtime state outside the repo; use `${RESOURCE_*_DIR}` or platform-native equivalents when bind mounts are needed.
2. Keep the generated CLI contract manifest-driven: `resource.json` owns `cli.command`, `cli.install`, `cli.invoke`, and `cli.freshness`.
3. Keep `cli/main.go` focused on bootstrap and delegation; put compose-specific logic in `cli/internal/...`.
4. Replace placeholder images and ports in `compose.yaml`.
5. Document real dependency and readiness semantics in `docs/OPERATIONS.md`.
