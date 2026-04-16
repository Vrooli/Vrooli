# {{RESOURCE_DISPLAY_NAME}}

{{RESOURCE_DESCRIPTION}}

This scaffold was generated from the `manual-resource` resource template on {{CURRENT_DATE}}.

## Intent

- Resource ID: `{{RESOURCE_NAME}}`
- Category: `{{RESOURCE_CATEGORY}}`
- Driver: `manual`
- Portability tier: `{{RESOURCE_PORTABILITY_TIER}}`

## Use Cases

Replace these bullets with the real scenario-facing uses for this resource.

- Serve as the operator-managed dependency for scenarios that need {{primary function}}.
- Support {{secondary workflow}} without each scenario owning its own setup instructions.
- Provide a foundation for {{integration pattern}} across the Vrooli stack.

## Architecture

This template keeps the generated CLI thin on purpose.

- `resource.json` is the declarative authority for install, invoke, freshness, validation, and limit metadata.
- `cli/` is the single binary entrypoint and command wiring surface.
- `cli/internal/` is optional and intentionally small; it is the default home for resource-specific Go validation or environment helpers when docs and manifest data are not enough.

The intended escalation path is:

1. express behavior in `resource.json` and the setup docs
2. rely on the shared `vrooli resource ...` control plane
3. add resource-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Generated placeholder packages:

- `cli/internal/validate`: validation helpers for documented manual setup
- `cli/internal/env`: environment/export helpers when required

## Next Steps

1. Keep any operator-managed state outside the repo through the resource storage/runtime layer.
2. Keep the generated CLI contract manifest-driven: `resource.json` owns `cli.command`, `cli.install`, `cli.invoke`, and `cli.freshness`.
3. Keep `cli/main.go` focused on bootstrap and delegation; keep any real validation logic under `cli/internal/...`.
4. Document manual prerequisites and validation probes explicitly.
