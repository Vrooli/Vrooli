# Operations

`{{RESOURCE_NAME}}` is scaffolded as a `native-cli` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative install, invoke, freshness, portability, and exported environment metadata.
- `cli/` owns the binary entrypoint and install/build surface.
- `cli/internal/app` owns operator-facing command registration and CLI wiring.
- `cli/internal/domain` owns the real resource-specific Go logic.

Do not turn `cli/main.go` into the primary implementation surface. If the resource grows richer commands, wire them in `cli/internal/app` and implement the behavior in `cli/internal/domain` or sibling packages under `cli/internal/...`.

## Operator Checklist

- Keep mutable state outside the repo and resolve it through canonical resource storage paths.
- Keep manifest loading and source-root resolution in shared helpers under `cli/internal/discovery` and `cli/internal/version`.
- Route build/install behavior through `cli/internal/install`.
- Keep resource-specific business logic under `cli/internal/domain` instead of reintroducing shell glue.
- Keep `resource.json` as the declarative contract for command/install/invoke/freshness behavior.
