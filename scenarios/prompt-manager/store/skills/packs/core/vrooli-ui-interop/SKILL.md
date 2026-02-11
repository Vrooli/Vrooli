## Steer focus: Vrooli UI Interop

Ensure scenario UIs work reliably across all three Vrooli deployment contexts —
localhost development, Cloudflare tunnel, and app-monitor proxy/iframe — by
adopting `@vrooli/api-base` and `@vrooli/iframe-bridge` with consistent
file-level conventions.

This skill is about **deployment-context correctness**, not UX design, component
stability, or code organization. Those concerns belong to react-coherence and
react-stability.

Do **not** break functionality, regress tests, or introduce unrelated features.

Required reading:
- `prompt-manager skill read skill-principles`
- `prompt-manager skill read react-coherence`
- `packages/api-base/README.md`
- `packages/iframe-bridge/README.md`

---

## Section 0: The Three Deployment Contexts

| Context | URL shape | ProxyInfo? | iframe? | What breaks without this skill |
|---|---|---|---|---|
| Localhost | `http://localhost:PORT` | No | No | Nothing (defaults work) |
| Tunnel | `https://subdomain.trycloudflare.com` | No | No | API calls if hardcoded to localhost |
| Proxy/iframe | `https://host/apps/NAME/proxy/` | Yes | Yes | Routing, API calls, assets, keyboard shortcuts, storage |

Core principle: **Zero-config in localhost, auto-detected everywhere else.** Every pattern in this skill must degrade gracefully to a no-op when its context isn't present.

---

## Section 1: Canonical File Layout (The Convergence Pattern)

This is the central organizing structure. Each interop concern maps to exactly one file slot. Agents and CLI tools can scan these slots to verify compliance.

| Slot | Canonical Path | Responsibility | Scannable Marker |
|---|---|---|---|
| **[A]** | `ui/package.json` | Declares `@vrooli/api-base` + `@vrooli/iframe-bridge` deps | `dependencies["@vrooli/api-base"]` exists |
| **[B]** | `ui/vite.config.ts` | `base: './'` for relative asset URLs | Line matching `base: './'` |
| **[C]** | `ui/server.js` | `startScenarioServer()` from `@vrooli/api-base/server` | Import of `startScenarioServer` |
| **[D]** | `ui/src/main.tsx` | iframe-bridge init (before React mount) | Call to `initIframeBridgeChild` |
| **[E]** | `ui/src/App.tsx` | Proxy-aware router basename via `getProxyInfo()` | Import of `getProxyInfo` + `basename` prop on router |
| **[F]** | `ui/src/lib/api-client.ts` | API base resolution + URL building | Import of `resolveApiBase` + `buildApiUrl` |
| **[G]** | `ui/src/hooks/useKeyboardShortcuts.ts` | Central shortcut manager with iframe relay | Import of `emitShortcutIntent` |

**Why fixed slots?** A future CLI tool can verify interop compliance by checking 7 known file paths with simple grep/AST patterns. Scattering these across arbitrary files makes automated verification impractical.

**Flexibility note:** Slots [F] and [G] allow one naming alternative each (noted below). All other slots are fixed.

---

## Section 2: Slot Details — `@vrooli/api-base` Adoption

### [A] `ui/package.json` — Dependencies

Both packages must be declared:

```json
{
  "dependencies": {
    "@vrooli/api-base": "workspace:*",
    "@vrooli/iframe-bridge": "workspace:*"
  }
}
```

Audit: `jq '.dependencies["@vrooli/api-base"] // empty' ui/package.json` — must produce output.

### [B] `ui/vite.config.ts` — Relative Asset Base

Must include `base: './'` with a protective comment explaining why:

```typescript
export default defineConfig({
  // ╔══════════════════════════════════════════════════════════════╗
  // ║  INTEROP-CRITICAL: Relative base for proxy/tunnel contexts  ║
  // ║                                                              ║
  // ║  When served through app-monitor's proxy at                  ║
  // ║  /apps/<name>/proxy/, absolute asset URLs (base: '/')        ║
  // ║  resolve to the domain root, breaking all JS/CSS loading.    ║
  // ║  Relative base ('./') makes assets resolve from the          ║
  // ║  current directory, which works in all three contexts.       ║
  // ║                                                              ║
  // ║  DO NOT change to '/' or remove this setting.                ║
  // ╚══════════════════════════════════════════════════════════════╝
  base: './',
  // ...
});
```

Audit: `rg "base:\s*['\"]\./" ui/vite.config.ts` — must match.

### [C] `ui/server.js` — Scenario Server

Must use `startScenarioServer()`, never a custom Express/http setup:

```javascript
import { startScenarioServer } from '@vrooli/api-base/server';

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: '<scenario-name>',
  corsOrigins: '*',
});
```

Audit: `rg "startScenarioServer" ui/server.js` — must match. `rg "express|createServer|http\.listen" ui/server.js` — must NOT match.

### [E] `ui/src/App.tsx` — Proxy-Aware Router Basename

If the scenario uses React Router (BrowserRouter, HashRouter, etc.), the router must receive a proxy-aware basename. The exact code shape:

```tsx
import { getProxyInfo } from "@vrooli/api-base";

/**
 * Compute BrowserRouter basename from proxy context.
 *
 * When served through app-monitor at /apps/<name>/proxy/,
 * React Router needs the proxy path as basename so that
 * navigate("/page") resolves to /apps/<name>/proxy/page
 * instead of /page.
 *
 * Returns "" outside proxy context (localhost, tunnel).
 */
function getRouterBasename(): string {
  const proxyInfo = getProxyInfo();
  const proxyPath = proxyInfo?.primary?.path ?? proxyInfo?.basePath;
  if (proxyPath) {
    return proxyPath.replace(/\/+$/, "");
  }
  return "";
}

export default function App() {
  const basename = getRouterBasename();

  return (
    <BrowserRouter basename={basename}>
      {/* routes */}
    </BrowserRouter>
  );
}
```

Key rules:
- `getRouterBasename()` is a top-level function in App.tsx (not inline, not in a separate file)
- Strips trailing slashes to prevent double-slash URLs
- Returns `""` (not `"/"`) when not proxied — this is the React Router default
- All internal navigation uses paths without the proxy prefix (React Router handles it)

Audit: `ast-grep --lang tsx --pattern '<BrowserRouter basename={$_}>' ui/src/App.tsx` — must match. `ast-grep --lang tsx --pattern '<BrowserRouter>' ui/src/App.tsx` (no basename) — must NOT match if React Router is used.

**Scenarios without React Router** (e.g., agent-inbox uses manual `pushState`, app-issue-tracker uses modals): This slot is N/A. The skill does not require React Router — only that IF a router is used, it must be proxy-aware.

### [F] `ui/src/lib/api-client.ts` — API Base Resolution

**Allowed alternative path:** `ui/src/services/api.ts`

This is the single file where `resolveApiBase()` and `buildApiUrl()` are called. No other file in the UI should import these directly.

Recommended code shape:

```typescript
import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

// ╔══════════════════════════════════════════════════════════════╗
// ║  INTEROP-CRITICAL: Centralized API URL resolution            ║
// ║                                                              ║
// ║  resolveApiBase() auto-detects the correct API endpoint      ║
// ║  for localhost, tunnel, and proxy contexts.                  ║
// ║  buildApiUrl() normalizes paths against that base.           ║
// ║                                                              ║
// ║  DO NOT hardcode localhost URLs or construct API URLs         ║
// ║  manually anywhere else in the codebase.                     ║
// ╚══════════════════════════════════════════════════════════════╝

const API_BASE = resolveApiBase({ appendSuffix: true });

/**
 * Build a full API URL from a path segment.
 * All API calls in the UI must use this function.
 */
export function buildUrl(path: string): string {
  return buildApiUrl(path, { baseUrl: API_BASE });
}
```

For scenarios where proxy metadata may not be injected before module load (rare — only if lazy-loaded modules call API before mount), use the lazy pattern:

```typescript
let _apiBase: string | null = null;

function getApiBase(): string {
  if (_apiBase === null) {
    _apiBase = resolveApiBase({ appendSuffix: true });
  }
  return _apiBase;
}

export function buildUrl(path: string): string {
  return buildApiUrl(path, { baseUrl: getApiBase() });
}
```

Key rules:
- `resolveApiBase()` is called in exactly one file
- All other files import `buildUrl` (or the equivalent helper) from this file
- No file anywhere in `ui/src/` should contain hardcoded `localhost:PORT` URLs for API calls

Audit: `rg "resolveApiBase" ui/src/ --files-with-matches` — must return exactly 1 file. `rg "localhost:\d+" ui/src/` — must return 0 matches (excluding comments/docs).

---

## Section 3: Slot Details — `@vrooli/iframe-bridge` Adoption

### [D] `ui/src/main.tsx` — Bridge Initialization

The iframe bridge must be initialized **before** React mounts, in `main.tsx`. The recommended code shape:

```tsx
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";

// ╔══════════════════════════════════════════════════════════════╗
// ║  INTEROP-CRITICAL: Iframe bridge initialization              ║
// ║                                                              ║
// ║  Must run BEFORE React mount so that:                        ║
// ║  1. Storage shimming is in place before any component        ║
// ║     accesses localStorage/sessionStorage                     ║
// ║  2. The bridge message channel is ready for host commands    ║
// ║                                                              ║
// ║  The window.parent check ensures this is a no-op when        ║
// ║  running outside an iframe (localhost, tunnel).              ║
// ╚══════════════════════════════════════════════════════════════╝

declare global {
  interface Window {
    __<scenarioName>BridgeInitialized?: boolean;
  }
}

if (
  typeof window !== "undefined" &&
  window.parent !== window &&
  !window.__<scenarioName>BridgeInitialized
) {
  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch {
    // Fall back to default origin when parsing fails.
  }

  initIframeBridgeChild({ parentOrigin, appId: "<scenario-name>" });
  window.__<scenarioName>BridgeInitialized = true;
}

// React mount follows...
const root = ReactDOM.createRoot(document.getElementById("root")!);
root.render(<App />);
```

Key rules:
- **Location**: Always in `main.tsx`, never in `App.tsx` or a component
- **Guard**: Use `window.parent !== window` (equivalent to `window.top !== window.self` but more explicit)
- **Idempotency flag**: `window.__<scenarioName>BridgeInitialized` prevents double-init from HMR/StrictMode
- **parentOrigin**: Extract from `document.referrer` for correct cross-origin postMessage targeting
- **appId**: Always pass the scenario name — this lets the host identify which child sent a message
- **Before mount**: The `initIframeBridgeChild()` block must appear before `ReactDOM.createRoot()`

The simplest valid form (for trivial scenarios) is:

```tsx
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";

if (window.top !== window.self) {
  initIframeBridgeChild();
}
```

This is acceptable but the enhanced form above is recommended for production scenarios.

Audit: `rg "initIframeBridgeChild" ui/src/main.tsx` — must match.

---

## Section 4: Slot Details — Keyboard Shortcut Architecture

### [G] `ui/src/hooks/useKeyboardShortcuts.ts`

**Allowed alternative name:** `useHotkeys.ts` (if the scenario already uses that name)

This hook is the single root keyboard listener for the app. It must satisfy four requirements for interop:

1. **Single root listener** — one `addEventListener("keydown", ...)` at the app shell level
2. **Input suppression** — skip shortcuts when focus is in `<input>`, `<textarea>`, `contentEditable`, or known editor elements (e.g., Monaco)
3. **Local-first, relay-on-noop** — handle locally first; if the shortcut was a noop/unhandled, relay to host via `emitShortcutIntent()`
4. **Shared host action constants** — use `HOST_SHORTCUT_ACTION_*` from `@vrooli/iframe-bridge`, never scenario-specific strings

Recommended minimal code shape:

```typescript
import { useEffect, useCallback } from "react";
import {
  emitShortcutIntent,
  HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER,
  type BridgeShortcutOutcome,
} from "@vrooli/iframe-bridge";

/**
 * Central keyboard shortcut handler for this scenario.
 *
 * Architecture rules (see vrooli-ui-interop skill):
 * - This is the ONE place that owns window keydown listeners
 * - Components do NOT add their own keydown listeners for app shortcuts
 *   (dialog-local Escape handlers are fine)
 * - Unhandled shortcuts are relayed to the host via iframe-bridge
 */

interface ShortcutHandlers {
  /** Return true if the shortcut was handled locally */
  onSearch?: () => boolean;
  // ... other scenario-specific handlers
}

function isInputElement(el: HTMLElement): boolean {
  return (
    el.tagName === "INPUT" ||
    el.tagName === "TEXTAREA" ||
    el.isContentEditable ||
    el.closest(".monaco-editor") !== null
  );
}

export function useKeyboardShortcuts(handlers: ShortcutHandlers): void {
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      const target = event.target as HTMLElement;
      if (isInputElement(target)) return;

      const mod = event.metaKey || event.ctrlKey;

      // Ctrl/Cmd+K — Global search / switcher
      if (mod && event.key === "k") {
        event.preventDefault();
        const handled = handlers.onSearch?.() ?? false;
        const outcome: BridgeShortcutOutcome = handled ? "handled" : "noop";
        if (outcome !== "handled") {
          emitShortcutIntent({
            action: HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER,
            outcome,
            chord: "mod+k",
            source: "keyboard",
          });
        }
        return;
      }

      // ... other shortcuts follow the same pattern:
      // 1. preventDefault if claiming the chord
      // 2. Try local handler
      // 3. Relay to host if noop/unhandled
    },
    [handlers],
  );

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);
}
```

This hook is then called once at the app shell level (typically in `MainLayout` or `App.tsx`):

```tsx
// In MainLayout.tsx or App.tsx
useKeyboardShortcuts({
  onSearch: () => { setSearchOpen(true); return true; },
});
```

**Decision tree for shortcut handling:**

```
Keyboard event arrives
        │
        ▼
Focus in input/editor? ──YES──▶ Return (let browser handle)
        │ NO
        ▼
Modifier+Key matches local shortcut?
        │
   ┌────┴────┐
  YES        NO
   │          │
   ▼          ▼
preventDefault()    Relay to host via
Execute handler     emitShortcutIntent()
   │                with outcome: "unhandled"
   ▼
Handler returned true (handled)?
   │
┌──┴──┐
YES   NO
│      │
▼      ▼
Done  Relay to host via
      emitShortcutIntent()
      with outcome: "noop"
```

**What about complex scenarios?** Scenarios with many shortcuts or dynamic shortcut registration (like browser-automation-studio's store-based approach, or app-monitor's scope/priority provider) can use more sophisticated internal architectures. The interop requirement is only that:
- The iframe relay integration exists somewhere in the shortcut chain
- `emitShortcutIntent` is imported from `@vrooli/iframe-bridge`
- The hook/provider lives at the canonical path [G]

**Exceptions for dialog-local listeners:** Component-scoped Escape handlers for dialogs/modals are fine — they're UI-local and don't need iframe relay. The skill's audit for "scattered keydown listeners" specifically flags listeners that handle *app-level* shortcuts (Ctrl+K, Ctrl+S, etc.) outside the central hook.

Audit: `rg "emitShortcutIntent" ui/src/hooks/useKeyboardShortcuts.ts` — must match if the scenario has keyboard shortcuts.

---

## Section 5: Self-Detection & Graceful Degradation

Every pattern in this skill auto-detects its context and degrades to a no-op when the context isn't present:

| Pattern | Detection | Localhost behavior | Proxy behavior |
|---|---|---|---|
| `getRouterBasename()` | `getProxyInfo()` returns null | Returns `""` (default basename) | Returns `/apps/<name>/proxy` |
| `initIframeBridgeChild()` | `window.parent !== window` | Skipped entirely | Initializes bridge |
| `resolveApiBase()` | Checks proxy globals, hostname, env | Returns `http://localhost:PORT/api/v1` | Returns proxy-relative API path |
| `emitShortcutIntent()` | Bridge initialized? | postMessage to self (harmless no-op) | Relays to host |
| `base: './'` in Vite | N/A (always relative) | Assets load from `/` | Assets load from proxy path |

**Rule: No conditional branches for deployment context in component code.** The interop layer (slots [B]-[G]) absorbs all context differences. Components just call `buildUrl("/endpoint")` and `navigate("/page")` — they never check whether they're proxied.

---

## Section 6: Red Flags & Audit Checklist

Commands are written to run from the scenario root (e.g., `scenarios/<name>/`).

| # | Red Flag | Audit Command | Expected |
|---|---|---|---|
| 1 | Missing api-base dep | `jq '.dependencies["@vrooli/api-base"]' ui/package.json` | Non-null |
| 2 | Missing iframe-bridge dep | `jq '.dependencies["@vrooli/iframe-bridge"]' ui/package.json` | Non-null |
| 3 | Hardcoded localhost in source | `rg 'localhost:\d+' ui/src/ -l` | 0 files |
| 4 | Missing relative base | `rg "base:\\s*['\"]\\.\\/['\"]" ui/vite.config.ts` | 1 match |
| 5 | Router without basename | `ast-grep --lang tsx --pattern '<BrowserRouter>' ui/src/` then check for `basename` prop | Has basename or no router |
| 6 | Custom server | `rg "express\|createServer\|http\\.listen" ui/server.js` | 0 matches |
| 7 | No bridge init | `rg "initIframeBridgeChild" ui/src/main.tsx` | >= 1 match |
| 8 | resolveApiBase in multiple files | `rg "resolveApiBase" ui/src/ -l \| wc -l` | Exactly 1 |
| 9 | Shortcut relay missing | `rg "emitShortcutIntent" ui/src/hooks/` | >= 1 match (if shortcuts exist) |
| 10 | Scattered app-level keydown | `rg "addEventListener.*keydown" ui/src/ -l` | Only hooks/ and dialog components |
| 11 | Missing appId in bridge init | `rg "initIframeBridgeChild\\(" ui/src/main.tsx` then check for `appId` param | Has appId |
| 12 | Missing protective comments | `rg "INTEROP-CRITICAL" ui/vite.config.ts ui/src/main.tsx` | >= 1 match per file |

---

## Section 7: Relationship to Other Skills

| Skill | Boundary |
|---|---|
| **react-stability** | Stability = crash prevention (error boundaries, null guards). Interop = deployment-context correctness. No overlap. |
| **react-coherence** | Coherence = code organization, component structure, state management. Interop = where interop-specific code lives (slots [A]-[G]). Coherence §0.5 references this skill for keyboard shortcut implementation; this skill owns the full pattern including iframe relay. |
| **interoperability-steer** | Interop-steer = proto-first API contracts between services. UI-interop = client-side deployment context adaptation. Different layers. |
| **platform-scope** | Platform-scope = session boundaries when working on shared packages. UI-interop = how scenarios consume those packages. |
| **platform-package-consumption-audit** | Consumption-audit = general audit framework for any package. UI-interop = specific patterns for api-base + iframe-bridge. |
| **navigation-integrity-audit** | Nav-integrity = UX correctness (dead links, orphan routes). UI-interop = routing infrastructure works through proxy. |
