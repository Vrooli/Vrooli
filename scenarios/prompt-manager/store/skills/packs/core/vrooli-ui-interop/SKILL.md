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
- `docs/agent-system/SKILL_AUTHORING.md`
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
| **[F]** | `ui/src/api/client.ts` | API base resolution + URL building | Import of `resolveApiBase` + `buildApiUrl` |
| **[G]** | `ui/src/hooks/useKeyboardShortcuts.ts` | Central shortcut manager with iframe relay | Import of `emitShortcutIntent` |
| **[H]** | `ui/src/hooks/useSpatialNav.ts` | Spatial navigation hook with focus group registration | Import of `initSpatialNav` from `@vrooli/iframe-bridge/spatial` |
| **[I]** | `ui/src/hooks/useGamepad.ts` | Raw gamepad input hook for custom handling | Import of `GamepadInputManager` from `@vrooli/iframe-bridge/spatial` |

**Why fixed slots?** A future CLI tool can verify interop compliance by checking 9 known file paths with simple grep/AST patterns. Scattering these across arbitrary files makes automated verification impractical.

**Flexibility note:** Slots [F] and [G] allow one naming alternative each (noted below). All other slots are fixed. Slots [H] and [I] are provided by the template and rarely need customisation.

---

## Section 2: Slot Details — `@vrooli/api-base` Adoption

### [A] `ui/package.json` — Dependencies

Both packages must be declared. Vrooli scenarios are independent applications
(not pnpm/yarn workspace members), so dependencies use relative `file:` paths:

```json
{
  "dependencies": {
    "@vrooli/api-base": "file:../../../packages/api-base",
    "@vrooli/iframe-bridge": "file:../../../packages/iframe-bridge"
  }
}
```

> **Important:** Do NOT use `"workspace:*"` — that syntax requires a unified
> pnpm/yarn workspace, which Vrooli's architecture does not use. Each scenario
> is a standalone application that references shared packages via relative
> file paths.

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

Must use `startScenarioServer()` (or `createScenarioServer()` when custom route setup is needed), never a custom Express/http setup:

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

For scenarios that need custom routes before listening (e.g., embedded scenario proxying), use `createScenarioServer()` with the `setupRoutes` callback:

```javascript
import { createScenarioServer } from '@vrooli/api-base/server';

const app = createScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: '<scenario-name>',
  corsOrigins: '*',
  setupRoutes: (expressApp) => {
    // Custom routes here
  },
});

app.listen(process.env.UI_PORT);
```

Audit: `rg "startScenarioServer|createScenarioServer" ui/server.js` — must match. `rg "express\(\)|http\.createServer|http\.listen" ui/server.js` — must NOT match.

### [E] `ui/src/App.tsx` — Proxy-Aware Router Basename

If the scenario uses React Router with **BrowserRouter**, the router must receive a proxy-aware basename. HashRouter uses URL hashes (`/#/path`) which are never sent to the server, making it inherently proxy-compatible without a basename — this slot is N/A for HashRouter scenarios. The exact code shape for BrowserRouter:

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

### [F] `ui/src/api/client.ts` — API Base Resolution

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
- `resolveApiBase()` is called in at most 2 production files (ideally 1; a second is acceptable when SSE/streaming connections need a separate base, e.g. `resolveApiBase({ appendSuffix: false })`)
- All other files import `buildUrl` (or the equivalent helper) from the primary API client file
- No file anywhere in `ui/src/` should contain hardcoded `localhost:PORT` URLs for API calls
- Test files (`*.test.ts`, `*.spec.ts`, etc.) are excluded from these audits — mock data in tests commonly references localhost URLs

Audit: `rg "resolveApiBase" ui/src/ --files-with-matches` — must return at most 2 files (excluding test files). `rg "localhost:\d+" ui/src/` — must return 0 matches in production files (excluding comments/docs/tests).

---

## Section 3: Slot Details — `@vrooli/iframe-bridge` Adoption

### [D] `ui/src/main.tsx` — Bridge Initialization

The iframe bridge must be initialized **before** React mounts, in `main.tsx`.

The `@vrooli/iframe-bridge` package exports `initIframeBridgeChild` from both the root entrypoint and the `./child` subpath. Both imports are equivalent:

```tsx
// Either of these is valid (index.ts re-exports from iframeBridgeChild.ts):
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge/child";
```

The recommended code shape:

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

## Section 4.5: Iframe-Safe Scroll & Viewport APIs

### The Problem

Several DOM APIs implicitly traverse the iframe boundary, affecting the **host document** (app-monitor) instead of — or in addition to — the scenario's own document. This causes content to shift irreversibly in the proxy/iframe context while appearing to work fine on localhost.

The most common offender is `scrollIntoView()`. When called on an element inside an iframe, it scrolls **every** scrollable ancestor in the chain — including ancestors in the host document outside the iframe. If the host uses `overflow: hidden` on the iframe shell (as app-monitor does), the content shifts up with no scrollbar to recover.

### Banned APIs

These APIs must **never** be used directly in scenario UI code:

| API | Why it breaks | Safe alternative |
|-----|---------------|-----------------|
| `element.scrollIntoView()` | Scrolls all ancestors including host document | `container.scrollTo()` targeting your own scroll container |
| `window.scrollTo()` / `window.scroll()` | In an iframe, `window` is the iframe's window — usually harmless, but can interact unpredictably with host scroll state | Use a ref to your scroll container and call `container.scrollTo()` |
| `window.scrollBy()` | Same issue as `window.scrollTo()` | `container.scrollBy()` on your own scroll container |
| `document.documentElement.scrollTop = N` | Moves the iframe's root scroll position, which can desync with host layout | `container.scrollTop = N` on your own scroll container |
| `element.focus()` without `preventScroll` | `focus()` implicitly calls `scrollIntoView` on the focused element | `element.focus({ preventScroll: true })` then manually scroll your container if needed |

### The Safe Pattern

Always scroll by targeting the specific scrollable container you own:

```tsx
// ╔══════════════════════════════════════════════════════════════╗
// ║  INTEROP-CRITICAL: Iframe-safe scrolling                     ║
// ║                                                              ║
// ║  Never use scrollIntoView() — it crosses iframe boundaries   ║
// ║  and shifts the host document's scroll position.             ║
// ║                                                              ║
// ║  Always scroll by targeting your own scroll container.       ║
// ╚══════════════════════════════════════════════════════════════╝

const scrollContainerRef = useRef<HTMLDivElement>(null);
const targetRef = useRef<HTMLDivElement>(null);

function scrollToTarget() {
  const container = scrollContainerRef.current;
  const target = targetRef.current;
  if (container && target) {
    container.scrollTo({
      top: target.offsetTop - container.offsetTop,
      behavior: "smooth",
    });
  }
}

// Layout: root is overflow-hidden, inner container is overflow-auto
return (
  <div className="h-full flex flex-col overflow-hidden">
    <div ref={scrollContainerRef} className="flex-1 overflow-auto">
      {/* ... content ... */}
      <div ref={targetRef} />
      {/* ... more content ... */}
    </div>
  </div>
);
```

### Layout Rule: `h-full` not `h-screen`

`h-screen` compiles to `height: 100vh`. Inside an iframe, `100vh` can refer to the **outer window's viewport height**, not the iframe's actual dimensions. This causes the content to be sized incorrectly.

Use `h-full` (`height: 100%`) with an explicit height chain instead:

```css
/* In styles.css */
html, body, #root {
  height: 100%;
  margin: 0;
}
```

```tsx
/* In App.tsx — uses h-full, NOT h-screen */
<div className="h-full flex flex-col overflow-hidden">
  <div className="flex-1 overflow-auto">
    {/* content */}
  </div>
</div>
```

This works in all three contexts because `height: 100%` correctly inherits from the parent — whether that parent is the browser viewport (localhost/tunnel) or an iframe element (proxy).

### Self-Detection

These patterns degrade gracefully:

| Pattern | Localhost | Proxy/iframe |
|---|---|---|
| `container.scrollTo()` | Scrolls the container (same as scrollIntoView would) | Scrolls only the container, never the host |
| `h-full` + height chain | 100% of viewport (same as h-screen) | 100% of iframe element |
| `focus({ preventScroll: true })` | Focuses without scrolling (then you scroll manually) | Same — no host document scroll side effect |

Audit: `rg "scrollIntoView" ui/src/` — must return 0 matches in production files. `rg "h-screen" ui/src/` — should return 0 matches (use `h-full` with height chain instead).

---

## Section 4.6: Spatial Navigation & Gamepad Support

Scenario UIs must be navigable with game controllers (Xbox, PlayStation, Switch) via spatial navigation — 2D directional focus movement driven by D-pad and analog stick input.

### Initialisation — Slot [H]

`initSpatialNav()` is called once in `main.tsx`, after `initIframeBridgeChild()`:

```typescript
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";

// After bridge init...
initSpatialNav();
```

Unlike the bridge init, spatial nav works in all contexts (not just iframes), so no `window.top !== window.self` guard is needed. The call is already included in both UI templates.

### Focus Groups — `<SpatialGroup>` Component

Complex UIs should register focus groups to control navigation behavior per-container. The `<SpatialGroup>` wrapper renders with `display: contents` so it generates no layout box — it won't break flex/grid parent layouts.

```tsx
import { useSpatialNav } from "./hooks/useSpatialNav";
import { SpatialGroup } from "./hooks/SpatialGroup";

function App() {
  const spatialNav = useSpatialNav();
  return (
    <>
      <SpatialGroup controllerRef={spatialNav} mode="spatial">
        <Sidebar />
      </SpatialGroup>
      <SpatialGroup controllerRef={spatialNav} mode="passthrough">
        <GraphCanvas />
      </SpatialGroup>
      <SpatialGroup controllerRef={spatialNav} mode="spatial">
        <DetailsPanel />
      </SpatialGroup>
    </>
  );
}
```

### Mode Selection Heuristic

| Mode | When to use | Examples |
|---|---|---|
| `spatial` (default) | Standard UI — D-pad moves between focusable children | Lists, forms, buttons, cards, nav menus, toolbars |
| `passthrough` | Component handles arrow keys for its own internal purpose | Canvas/graph views, map views, drawing tools, rich text editors, video players |
| `grid` | Uniform grid layout | Image galleries, dashboard cards, calendar views |
| `modal` | Overlay that must trap D-pad focus inside it | Dialogs, drawers, floating panels, command overlays, help panels |

**Rule of thumb:** If the component already handles arrow keys for panning, cursor movement, or cell navigation, use `passthrough`. If it's an overlay/dialog that appears on top of other content, use `modal`. Otherwise use `spatial`.

### Modal Focus Trapping (Critical)

**Every overlay, dialog, drawer, floating panel, and full-screen overlay MUST trap spatial navigation inside it while open.** Without this, D-pad navigation leaks through to elements behind the overlay, which is confusing and unusable on consoles.

There are two approaches:

**Approach 1 — `<SpatialGroup mode="modal">` (preferred for new components):**

```tsx
<SpatialGroup controllerRef={spatialNav} mode="modal">
  <MyDialog>{children}</MyDialog>
</SpatialGroup>
```

When mode is `modal`, SpatialGroup calls `pushScope(element)` on mount and `popScope()` on unmount, constraining all D-pad navigation to within that element. Scopes nest correctly (dialog within dialog works).

**Approach 2 — `SpatialNavContext` + manual scope (for existing/portaled components):**

Components rendered via `createPortal` or deeply nested components that can't receive `controllerRef` as a prop should use context:

```tsx
// In your root layout (e.g., GraphWorkspace):
import { SpatialNavProvider } from "./hooks/SpatialNavContext";

function GraphWorkspace() {
  const spatialNav = useSpatialNav();
  return (
    <SpatialNavProvider controllerRef={spatialNav}>
      {/* All descendants can now access the controller */}
    </SpatialNavProvider>
  );
}

// In any overlay component (Dialog, Drawer, FloatingPanel, etc.):
import { useSpatialNavContext } from "./hooks/SpatialNavContext";

function MyOverlay({ isOpen, children }) {
  const spatialNavRef = useSpatialNavContext();
  const overlayRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const ctrl = spatialNavRef?.current;
    const el = overlayRef.current;
    if (!isOpen || !ctrl || !el) return;
    ctrl.pushScope(el);
    return () => { ctrl.popScope(); };
  }, [isOpen, spatialNavRef]);

  if (!isOpen) return null;
  return <div ref={overlayRef}>{children}</div>;
}
```

**Checklist for overlay components — ALL of these need scope management:**
- `Dialog` / `AlertDialog` components
- `Drawer` / side-panel components
- `FloatingPanel` (draggable non-modal panels)
- Full-screen overlays (command post, settings, help panels)
- Any custom overlay div that appears on top of other content

Missing scope management on even one overlay type will cause D-pad to navigate behind it on consoles.

### Mode Switching

Mode switching between cursor and spatial navigation is automatic:
- **D-pad press → enter spatial mode.** Browser cursor is hidden via CSS, a visible focus ring appears on the nearest focusable element.
- **Mouse move / touch → exit spatial mode.** Cursor is restored, focus ring disappears.
- **Bumper buttons (LB/RB)** cycle between top-level focus groups regardless of mode.
- **B button** calls `history.back()` — this is the only reliable escape mechanism on console browsers where the virtual cursor may not be available. On Xbox Edge, this navigates back one page; inside a single-page app, it pops the last history entry.

### Iframe Safety

All spatial navigation follows the same iframe-safe rules from Section 4.5:
- Never use `scrollIntoView()` — the spatial nav engine uses `container.scrollTo()` internally.
- Always `element.focus({ preventScroll: true })` — the engine handles this automatically.

### Focus Ring Styling

The engine injects a default blue focus ring via `[data-spatial-focus="true"]` attribute. Scenarios can:
1. **Override** by writing CSS for `[data-spatial-focus="true"]` in their own stylesheets.
2. **Opt out** by passing `{ injectDefaultFocusStyle: false }` to `initSpatialNav()` and styling `:focus-visible` themselves.

### Opt-Out

Scenarios that do not need gamepad support can add `// spatial-nav: disabled` to `main.tsx` instead of calling `initSpatialNav()`. The auditor rule will accept this as a valid opt-out.

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
| `container.scrollTo()` | N/A (always contained) | Scrolls container (same effect) | Scrolls only container, never host |
| `h-full` + height chain | N/A (always %) | 100% of viewport | 100% of iframe element |

**Rule: No conditional branches for deployment context in component code.** The interop layer (slots [B]-[G]) absorbs all context differences. Components just call `buildUrl("/endpoint")` and `navigate("/page")` — they never check whether they're proxied.

---

## Section 6: Compliance Verification

### Discovery & Planning (Before Implementation)

List rules by priority to understand what needs to be implemented before writing code:

```bash
app-monitor rules                                      # All rules by priority
app-monitor rules --scenario browser-automation-studio  # Filter for scenario
app-monitor rules --severity critical,high              # Critical + high only
app-monitor rules --json                                # Raw JSON for tooling
```

### Automated Checking (Preferred)

Use the app-monitor CLI for automated compliance verification:

```bash
# Check a single scenario
app-monitor interop <scenario-name>
```

These checks are also included in `app-monitor diagnostics <scenario-name>` (the aggregated diagnostics bundle) and registered with scenario-auditor as external rules under the `interop` category.

### Check Reference

The automated scanner verifies 21 compliance checks mapped to slots [A]-[I] plus cross-cutting rules:

| # | Check | Slot | Severity | What It Verifies |
|---|-------|------|----------|------------------|
| 1 | api-base dep | [A] | critical | `@vrooli/api-base` in ui/package.json |
| 2 | iframe-bridge dep | [A] | critical | `@vrooli/iframe-bridge` in ui/package.json |
| 3 | No hardcoded localhost | [F] | high | No `localhost:PORT` in ui/src/ (excludes test files) |
| 4 | Relative Vite base | [B] | critical | `base: './'` in ui/vite.config.ts |
| 5 | Router basename | [E] | high | BrowserRouter has proxy-aware basename (or no router) |
| 6 | Standard server | [C] | medium | Uses `startScenarioServer()`/`createScenarioServer()`, not custom Express |
| 7 | Bridge init | [D] | critical | `initIframeBridgeChild` in ui/src/main.tsx |
| 8 | Single API base | [F] | high | `resolveApiBase` in at most 2 production files |
| 9 | Shortcut relay | [G] | medium | `emitShortcutIntent` in shortcut chain (if shortcuts exist) |
| 10 | No scattered keydown | [G] | medium | App-level keydown only in hooks/ and dismissible UI components |
| 11 | Bridge appId | [D] | medium | Bridge init includes `appId` param |
| 12 | Protective comments | [B],[D] | low | `INTEROP-CRITICAL` markers present |
| 13 | Iframe guard | [D] | high | Bridge init guarded with `window.parent !== window` or `window.top !== window.self` |
| 14 | Capture settings enabled | [D] | medium | captureLogs/captureNetwork not disabled in bridge init |
| 15 | Proxy base preservation | [F] | high | resolveApiBase output not rebuilt with window.location.origin |
| 16 | Secure UI tunnel | [C] | high | Custom server routes API calls through proxyToApi |
| 17 | Standard server functions | [C] | medium | Server file uses `startScenarioServer` or `createScenarioServer` from `@vrooli/api-base/server` |
| 18 | No scrollIntoView | §4.5 | high | No `scrollIntoView` calls in ui/src/ production files |
| 19 | No h-screen | §4.5 | medium | No `h-screen` class usage in ui/src/ (use `h-full` with height chain) |
| 20 | Spatial nav init | [H] | medium | `initSpatialNav` in main.tsx or `// spatial-nav: disabled` opt-out comment |
| 21 | Focus visible styles | — | low | `focus-visible` classes or `data-spatial-focus` styling in UI component files |
| 22 | Modal scope trapping | §4.6 | high | All overlay components (Dialog, Drawer, FloatingPanel, custom overlays) call `pushScope`/`popScope` or use `<SpatialGroup mode="modal">` |

Checks 5, 9, 10, 13, 14, 16, and 17 are conditional — they skip automatically when the feature isn't used (no router, no keyboard shortcuts, no bridge call, no custom server, no server file). Checks 20-22 support an explicit opt-out comment (`// spatial-nav: disabled`).

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
