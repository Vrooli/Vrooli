# Operations

`sqlite` is a Go-native resource that follows the `native-cli` template philosophy: the installed CLI is a repo-owned binary and a first-class operator surface.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative install metadata, binary metadata, freshness inputs, portability, and exported environment contracts.
- `cli/` is the single binary entrypoint and stays as small as possible.
- `cli/internal/app` owns the richer operator-facing command surface for SQLite-specific actions.
- `cli/internal/env`, `cli/internal/discovery`, `cli/internal/install`, and `cli/internal/version` carry the template-aligned native-cli concerns.
- `cli/internal/sqlite` is the SQLite-specific engine for content, replication, migrations, query helpers, and stats.

Do not grow new behavior in `cli/main.go`. If the resource needs additional specialization, first decide whether it belongs in the manifest contract, the operator-facing app, or one of the `cli/internal/...` packages above.

## Operator Checklist

- Keep mutable databases, backups, replicas, and migrations under canonical resource storage paths.
- Route install and rebuild behavior through `cli/internal/install`.
- Keep manifest loading and source-root resolution in `cli/internal/version` and `cli/internal/discovery`.
- Extend `cli/internal/sqlite` for SQLite-specific features instead of reintroducing shell helpers.
- Preserve the serverless contract: `start`, `stop`, and `restart` remain acknowledgements, not daemon management.
