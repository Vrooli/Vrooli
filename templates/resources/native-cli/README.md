# {{RESOURCE_DISPLAY_NAME}}

{{RESOURCE_DESCRIPTION}}

This scaffold was generated from the `native-cli` resource template on {{CURRENT_DATE}}.

## Intent

- Resource ID: `{{RESOURCE_NAME}}`
- Category: `{{RESOURCE_CATEGORY}}`
- Driver: `native-cli`
- Portability tier: `{{RESOURCE_PORTABILITY_TIER}}`

Use this template for repo-owned Go resource binaries like `sqlite`, where the installed CLI is the real operator surface rather than a thin wrapper over Docker or a third-party executable.

## Use Cases

Replace these bullets with the real scenario-facing uses for this resource.

- Provide a shared Go-native capability for scenarios that need a repo-owned binary interface.
- Keep resource-specific operator workflows in one repo-owned binary instead of scattering them across scripts.
- Establish a foundation for cross-scenario native resource reuse across the Vrooli stack.

## Architecture

This template keeps `cli/main.go` thin while making the repo-owned binary a first-class implementation surface.

- `resource.json` is the declarative authority for install, invoke, freshness, portability, and exported environment contracts.
- `cli/` is the single binary entrypoint and build/install surface.
- `cli/internal/app` is the default home for command registration and CLI wiring.
- `cli/internal/domain` is the default home for resource-specific Go logic.
- `cli/internal/discovery`, `cli/internal/install`, `cli/internal/version`, and `cli/internal/env` carry the shared native-resource concerns around runtime resolution and build/install behavior.

The intended escalation path is:

1. express behavior in `resource.json`
2. keep `cli/main.go` as bootstrap only
3. add operator-facing command wiring in `cli/internal/app`
4. implement real resource behavior in `cli/internal/<domain>` packages

Generated placeholder packages:

- `cli/internal/app`: command registration and CLI wiring helpers
- `cli/internal/domain`: resource-specific implementation surface
- `cli/internal/discovery`: runtime and source-root resolution helpers
- `cli/internal/install`: binary rebuild/install helpers
- `cli/internal/version`: manifest/build metadata helpers
- `cli/internal/env`: environment/config helpers

## Next Steps

1. Keep mutable runtime state outside the repo and expose it through canonical resource storage paths.
2. Keep the generated CLI contract manifest-driven: `resource.json` owns `cli.command`, `cli.install`, `cli.invoke`, and `cli.freshness`.
3. Keep `cli/main.go` focused on bootstrap; put command wiring in `cli/internal/app` and resource logic in `cli/internal/domain`.
4. Replace placeholder domain commands with the real operator surface for the resource.
