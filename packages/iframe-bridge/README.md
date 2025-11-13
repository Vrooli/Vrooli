# @vrooli/iframe-bridge

Lightweight utilities for passing messages between host scenarios and embedded iframe children.

## Development

```bash
pnpm --filter @vrooli/iframe-bridge install
pnpm --filter @vrooli/iframe-bridge build
```

## Propagating Package Updates

After editing this package, refresh the scenarios that depend on it so they reinstall the new build:

```bash
./scripts/scenarios/tools/refresh-shared-package.sh iframe-bridge <scenario|all> [--no-restart]
```

The helper rebuilds `@vrooli/iframe-bridge`, filters to scenarios that declare this dependency, runs `vrooli scenario setup`, and restarts only the scenarios that were already running (use `--no-restart` to opt out and restart manually later).
