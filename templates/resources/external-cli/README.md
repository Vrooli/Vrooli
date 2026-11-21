# {{RESOURCE_DISPLAY_NAME}}

{{RESOURCE_DESCRIPTION}}

This scaffold was generated from the `external-cli` resource template on {{CURRENT_DATE}}.

## Intent

- Resource ID: `{{RESOURCE_NAME}}`
- Category: `{{RESOURCE_CATEGORY}}`
- Driver: `external-cli`
- Portability tier: `{{RESOURCE_PORTABILITY_TIER}}`

Use this template for CLIs like `codex`, `claude-code`, `terraform`, or `ffmpeg`.

## Use Cases

Replace these bullets with the real scenario-facing uses for this resource.

- Serve as the shared host CLI integration for scenarios that need a third-party executable.
- Support repeatable operator workflows without each scenario owning its own binary setup.
- Provide a foundation for cross-scenario CLI reuse across the Vrooli stack.

## Architecture

This template keeps the generated CLI thin on purpose.

- `resource.json` is the declarative authority for install, invoke, freshness, binary probing, and health metadata.
- `cli/` is the single binary entrypoint and command wiring surface.
- `cli/internal/` is the default home for external-tool-specific Go logic when the manifest and shared control plane are not enough.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add resource-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Generated placeholder packages:

- `cli/internal/discovery`: host binary detection and probing helpers
- `cli/internal/install`: install/bootstrap helpers
- `cli/internal/version`: version parsing and compatibility helpers
- `cli/internal/env`: environment/config helpers
- `cli/internal/auth`: auth/config validation helpers when the external CLI needs them

## Next Steps

1. Keep mutable runtime state outside the repo; if the CLI needs files, resolve them through the resource storage/runtime layer.
2. Keep the generated CLI contract manifest-driven: `resource.json` owns `cli.command`, `cli.source_build`, `cli.distribution`, `cli.invoke`, and `cli.freshness`; do not add resource-local installer scripts.
3. Keep `cli/main.go` focused on bootstrap and delegation; put binary/version/auth logic in `cli/internal/...`.
4. Replace placeholder install/version checks with the real binary contract.
