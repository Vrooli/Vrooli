# @vrooli/iframe-bridge

Lightweight utilities for passing messages between host scenarios and embedded iframe children.

## Features

### Storage Shimming

When running in sandboxed iframe contexts (like Browserless for UI smoke tests), `localStorage` and `sessionStorage` may be blocked. The bridge automatically shims them with in-memory implementations.

This happens automatically when you call `initIframeBridgeChild()`. You can also call `shimStorage()` explicitly if you need the shim earlier:

```typescript
import { shimStorage, initIframeBridgeChild } from '@vrooli/iframe-bridge';

// Called automatically by initIframeBridgeChild, but can be called earlier if needed
shimStorage();

// Normal initialization
if (window.top !== window.self) {
  initIframeBridgeChild();
}
```

The shim results are available at `window.__VROOLI_UI_SMOKE_STORAGE_PATCH__` for inspection by smoke tests.

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
