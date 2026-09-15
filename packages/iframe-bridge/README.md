# @vrooli/iframe-bridge

Lightweight utilities for passing messages between host scenarios and embedded iframe children.

## Features

### Storage Shimming

When running in sandboxed/headless browser containers (for UI smoke tests), `localStorage` and `sessionStorage` may be blocked. The bridge automatically shims them with in-memory implementations.

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

### Shortcut Intent Relay (Iframe -> Host)

When an embedded scenario wants to escalate a shortcut to the host (for example when a local shortcut is a no-op), emit a shortcut intent:

```typescript
import {
  emitShortcutIntent,
  HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER,
} from '@vrooli/iframe-bridge';

emitShortcutIntent({
  action: HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER,
  outcome: 'noop', // local handler was idempotent/no-op
  chord: 'mod+k',
  source: 'keyboard',
});
```

Guideline:
- Handle local shortcut first.
- If local action is `noop`/`unhandled`, relay intent to host.
- Still call `preventDefault()` on claimed browser-reserved chords like `Ctrl/Cmd+K`.

## Built-in gamepad navigation

Initialize navigation once in each application entry point. The shared
controller owns polling, default focus navigation, selection, Back, and host
relay. Repeated default initialization returns the same controller; competing
configuration is rejected. Dispose only when the application itself is torn down.

```tsx
import { initSpatialNav } from '@vrooli/iframe-bridge/spatial';
import { SpatialNavProvider } from '@vrooli/iframe-bridge/react';

const spatialNav = initSpatialNav();
if (import.meta.hot) import.meta.hot.dispose(() => spatialNav.dispose());
// Inside the application's React root:
<SpatialNavProvider controller={spatialNav}><App /></SpatialNavProvider>
```

The core and `./spatial` exports have no React runtime dependency. Only `./react`
requires the application's React peer (18 or later). The provider does not create
or dispose a controller, so StrictMode remounts and nested consumers share input.
`useSpatialNav()` reads that controller without starting a polling loop. Outside
a provider it can read the initialized application controller, including portals
and existing application integrations; it fails clearly if none exists.

### Custom controls and dialogs

```tsx
import { useRef } from 'react';
import { useGamepad, useSpatialScope } from '@vrooli/iframe-bridge/react';

function Dialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const ref = useRef<HTMLDivElement>(null);
  useSpatialScope(ref, open);
  useGamepad(ref, action => {
    if (action !== 'back') return false;
    onClose();
    return true;
  }, open);
  return open ? <div ref={ref} role="dialog"><button onClick={onClose}>Close</button></div> : null;
}
```

Handlers are eligible only while their element contains focus and lies within the
active modal scope. If focus has not yet entered an open modal, its own handler
can still receive input, including Back. Innermost handlers run first; the most recently registered
handler wins ties at the same element. Returning `true` consumes input; otherwise
it continues outward and then to default navigation/host relay. Use an explicit
handler for Back dismissal so closing a dialog changes application state.
A custom control should consume only actions it implements, leaving escape paths
available. `SpatialGroup` registers spatial, grid, passthrough, or modal groups
against the same controller. Passthrough does not create a raw input manager;
register custom actions with `useGamepad` on the focused control.

Modal cleanup removes its exact registration, even if a parent unmounts before a
child. Closing the top scope restores its previous focus when that target remains
connected and eligible. When conditionally mounting a referenced element, pass
its mounted/open condition to the hook so registration follows that lifecycle.

### Validation ownership

`pnpm test` in this package covers the engine and actual React adapters, including
StrictMode, callback updates, scoped consumption, cleanup, and fallback behavior.
Scenario tests cover their own controls, focus groups, and modal dismissal; they
do not copy the shared hook implementation or SDK mocks. The canonical template
keeps initialization in `ui/src/main.tsx` so new applications inherit support.

## Development

```bash
vrooli package build iframe-bridge
# Focused package tests (from packages/iframe-bridge):
pnpm test
```

## Propagating Package Updates

After editing this package, use the native refresh command:

```bash
vrooli package refresh iframe-bridge all
```

That rebuilds `@vrooli/iframe-bridge`, discovers governed dependents, runs `vrooli scenario setup`, without restarting consumers by default. Use the explicit `--restart` option
when a running consumer should restart.
