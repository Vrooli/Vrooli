# Implementation Plan: vrooli-emulator Standalone Management UI

## Greenfield Constraint

**This is greenfield work.** Do not add compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, renamed `_unused` variables, or re-exports of the old `Desktop*` types. The migration is a hard cutover (round-1 decision d3 option A, confirmed by user note "Make sure not to leave around old names for compatibility. Should be hard cutover").

The old `scenarios/scenario-to-desktop/ui/src/components/livedesktop/**` and `lib/api/livedesktop.ts` are **not** deleted by this item — that removal belongs to `chore/scenario-to-desktop-remove-livedesktop`. This item only writes into `scenarios/vrooli-emulator/**` and uses fresh names from the start.

## Purpose

Migrate the livedesktop session-management UI out of scenario-to-desktop and into `scenarios/vrooli-emulator/ui/` as the emulator scenario's first-class management surface, wired to `/api/v1/sessions/` and exposing session lifecycle events through `@vrooli/iframe-bridge`. This is the phase-2 UI counterpart to the phase-1 API extraction (completed in `execute/scaffold-vrooli-emulator-scenario`), closing the UI downtime window opened by the hard-cut migration (research Finding 15).

## Required Reading

Before executing this plan, run:

```bash
prompt-manager skill read vrooli-ui-interop react-coherence interoperability-steer cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Then review:

- `scenarios/swarm-manager/research/emulator-extraction-and-service-plan/conclusion.md` — Findings 4 (UI coupling), 8 (captures ownership), 9 (iframe embed pattern), 14 (iframe-bridge is a packaged library), 15 (hard-cut migration trade-off).
- `packages/iframe-bridge/README.md` and `packages/api-base/README.md` — consumer-facing APIs for the two shared packages this scenario uses.
- `scenarios/vrooli-emulator/api/livedesktop/handler.go` (and adjacent `*_handler.go`) — source of truth for the `/api/v1/sessions/` route shape.

## Problem Statement

The phase-1 extraction (`execute/scaffold-vrooli-emulator-scenario`) moved the **API** for session management from scenario-to-desktop into a dedicated `vrooli-emulator` scenario and renamed the route prefix from `/livedesktop/sessions` to `/api/v1/sessions/`. The **UI** was deliberately left behind at phase 1 — the session-management components (`DesktopControlsMenu`, `LiveDesktopDrawer`, `VncCanvas`, `DesktopToolbar`, `MetricsBar`, `PlatformSelector`), the API client (`lib/api/livedesktop.ts`), and the Zustand store (`store/liveDesktopStore.ts`) still live in `scenarios/scenario-to-desktop/ui/`, and scenario-to-desktop's desktop viewer is currently broken by design.

Without this migration, there is no supported way to create, view, or control emulator sessions. The phase-2 iframe-embed item (`execute/scenario-to-desktop-emulator-iframe-embed`) cannot proceed until this UI exists as an iframe-ready surface, and the `execute/vrooli-emulator-external-url-endpoint` item has no consumer to serve.

## Scope

### In Scope

- Migrate `components/livedesktop/*.tsx` (excluding `LiveDesktopDrawer.tsx`), `lib/api/livedesktop.ts`, `store/liveDesktopStore.ts`, and the required `components/ui/*` primitives from scenario-to-desktop into `scenarios/vrooli-emulator/ui/src/`.
- Rewrite all `/livedesktop/sessions…` path literals to `/sessions…` (which `@vrooli/api-base`'s `resolveApiBase({ appendSuffix: true })` resolves to `/api/v1/sessions…`).
- Build a router-based session management surface with `/sessions` (list) and `/sessions/:id` (detail) routes (round-1 decision d1 option B).
- Replace the source `LiveDesktopDrawer` with a plain full-page `SessionDetailPage.tsx` — no backdrop, no slide-in, no drawer chrome (round-2 decision d3 option B).
- Drop the captures-store coupling; capture responses are surfaced inline in the current session view (round-1 decision d2 option A).
- Rename all `Desktop*` prefixed types and components to `Session*` aligned with the `/sessions/` API path (round-1 decision d3 option A).
- Wire `@vrooli/iframe-bridge` child-side emission of session lifecycle events only: `session.created`, `session.state_changed`, `session.error`, `session.destroyed` (round-1 decision d4 option B), with the **identifier + summary fields** payload shape defined in Contract Decisions (round-2 decision d1 option B).
- Keep session state in a Zustand store (round-1 decision d5 option A); the executing agent may split server-state reads (session list, session detail) into React Query if doing so is materially cleaner (per user note "Can split up zustand store if desirable"), but the single-store approach is the default and acceptable.
- Add `@novnc/novnc`, `zustand`, and `react-router-dom` to `scenarios/vrooli-emulator/ui/package.json` (matching scenario-to-desktop's pinned versions).
- Tests: port `DesktopControlsMenu.test.tsx` to `SessionControlsMenu.test.tsx`; add new component tests for session list, session detail transitions, and iframe-bridge event emission. Mock the API client per-test using `vi.mock` (round-2 decision d2 option B), and cover the network-helper layer (`buildUrl`, `throwIfNotOk`) in dedicated tests at `lib/api/http.test.ts`.

### Out of Scope

- The `/embedded/emulator/external-url` endpoint (separate item: `execute/vrooli-emulator-external-url-endpoint`).
- scenario-to-desktop's iframe-embed restoration of the desktop viewer (separate item: `execute/scenario-to-desktop-emulator-iframe-embed`).
- Deletion of `scenarios/scenario-to-desktop/ui/src/components/livedesktop/**`, `lib/api/livedesktop.ts`, or `store/liveDesktopStore.ts` (separate item: `chore/scenario-to-desktop-remove-livedesktop`).
- Remote backend support (`remote-macos`, `remote-ios`) — lives behind the API's `PlatformBackend` abstraction.
- Captures *persistence* — the emulator UI displays capture results it gets back from the API but does not store them long-term. `useCapturesStore` is **not** migrated.
- Migration of `LiveDesktopDrawer.tsx` and the `components/ui/drawer.tsx` primitive — both are dropped from this item per round-2 d3 (the standalone surface no longer wraps content in a drawer). If the phase-2 iframe-embed item later wants a drawer wrapper, it builds it then.
- Responsive/mobile polish beyond "doesn't break on a narrow laptop viewport."
- Design-system primitive expansion beyond the set the migrated components already import (`input`, existing `button`).

## Current Technical Context

### Emulator UI (destination — `scenarios/vrooli-emulator/ui/`)

- `ui/src/` currently holds only: `App.tsx`, `main.tsx`, `lib/api.ts`, `lib/utils.ts`, `components/ui/button.tsx`, plus styles.
- `main.tsx` already calls `initIframeBridgeChild()` before React mount (vrooli-ui-interop slot [D]).
- `lib/api.ts` uses `resolveApiBase({ appendSuffix: true })` and has a placeholder `fetchHealth` — this is the emulator's slot [F] entry point. At most two files in the UI may call `resolveApiBase` directly; all other API fetches must go through `lib/api/*`.
- `vite.config.ts` sets `base: './'` (slot [B]). No changes needed.
- Dependencies already present: `@tanstack/react-query`, `@vrooli/api-base`, `@vrooli/iframe-bridge`, `lucide-react`, `class-variance-authority`, `clsx`, `tailwind-merge`, `tailwindcss`, `express`.
- Missing dependencies: `@novnc/novnc`, `zustand`, `react-router-dom`.

### Emulator API (contract — `scenarios/vrooli-emulator/api/`)

- Route registration lives in `api/livedesktop/handler.go`. The emulator mounts handlers under `/api/v1/sessions/` (route prefix `/api/v1`, sub-prefix `/sessions`).
- All existing route shapes (paths, methods, request/response bodies) match what `scenario-to-desktop/ui/src/lib/api/livedesktop.ts` already calls, minus the `/livedesktop` prefix.

### Source files to migrate (from `scenarios/scenario-to-desktop/ui/src/`)

| Source path | Lines | New path (target) |
|---|---|---|
| `components/livedesktop/VncCanvas.tsx` | 77 | `components/sessions/VncCanvas.tsx` |
| `components/livedesktop/DesktopToolbar.tsx` | 106 | `components/sessions/SessionToolbar.tsx` |
| `components/livedesktop/DesktopControlsMenu.tsx` | 393 | `components/sessions/SessionControlsMenu.tsx` |
| `components/livedesktop/MetricsBar.tsx` | 93 | `components/sessions/MetricsBar.tsx` |
| `components/livedesktop/PlatformSelector.tsx` | 64 | `components/sessions/PlatformSelector.tsx` |
| `components/livedesktop/index.ts` | 6 | `components/sessions/index.ts` |
| `lib/api/livedesktop.ts` | 139 | `lib/api/sessions.ts` |
| `store/liveDesktopStore.ts` | 192 | `store/sessionStore.ts` |
| `components/livedesktop/DesktopControlsMenu.test.tsx` | 143 | `components/sessions/SessionControlsMenu.test.tsx` |
| `components/ui/input.tsx` | 17 | `components/ui/input.tsx` (new in emulator UI) |

**Not migrated:** `components/livedesktop/LiveDesktopDrawer.tsx` and `components/ui/drawer.tsx` — replaced by a plain `SessionDetailPage.tsx` (round-2 d3 option B).

### New files (no source counterpart)

| New path | Purpose |
|---|---|
| `pages/SessionListPage.tsx` | `/sessions` route — session list + create-new control |
| `pages/SessionDetailPage.tsx` | `/sessions/:id` route — VNC canvas, toolbar, metrics, controls stacked on a plain page |
| `lib/api/http.ts` | Hosts `buildUrl()` + `throwIfNotOk()` helpers, delegates to `@vrooli/api-base` |
| `lib/bridge.ts` | Typed `postSessionEvent` helper + `SessionEvent` discriminated union |

### Coupling points that must break

- `store/liveDesktopStore.ts:13` — `import { useCapturesStore } from "@/store/capturesStore"`. This must be removed; capture responses become local state (see Contract Decisions).
- `lib/api/livedesktop.ts` — every path literal starts with `/livedesktop/sessions`. Every one rewrites to `/sessions`.
- `buildVncWsUrl()` (currently in `livedesktop.ts`) — composes a `ws://…/livedesktop/sessions/{id}/ws` URL. Must rewrite against the new path and route base resolution through `@vrooli/api-base` for tunnel/proxy correctness.
- `LiveDesktopDrawer.tsx` consumers (the source store opens/closes a drawer via `isOpen`) — replaced by router navigation. The `isOpen`/`open()`/`close()` slice is dropped from the migrated `sessionStore.ts`; navigation between list and detail uses `useNavigate()` from `react-router-dom`.

### Dependency items

- **Completed**: `execute/scaffold-vrooli-emulator-scenario` — emulator API with `/api/v1/sessions/` is live.
- **Downstream consumers** (cannot start until this item is done): `execute/vrooli-emulator-external-url-endpoint` → `execute/scenario-to-desktop-emulator-iframe-embed`.
- **Coordination**: `chore/scenario-to-desktop-remove-livedesktop` removes the source files on the scenario-to-desktop side. Ordering relative to this item is a coordination call, not a hard block (the source and destination live in different scenarios).

## Target End State

- `scenarios/vrooli-emulator/ui/src/` contains working session management at routes `/sessions` and `/sessions/:id`.
- The scenario's root `App.tsx` renders a `<BrowserRouter>` with a `basename` derived from the vrooli-ui-interop slot [E] proxy helper, mounting the two session routes.
- `/sessions/:id` renders `SessionDetailPage.tsx` — a plain page laying out the VNC canvas, session toolbar, metrics bar, and session controls menu directly. There is no drawer wrapper, no backdrop, and no slide-in animation anywhere in the standalone surface.
- `scenarios/vrooli-emulator/ui/src/lib/api/sessions.ts` is the single API client for session operations. It calls `buildUrl("/sessions…")` from `lib/api/http.ts`, which resolves to `/api/v1/sessions/…` via `@vrooli/api-base`.
- `scenarios/vrooli-emulator/ui/src/store/sessionStore.ts` is a Zustand store holding session state, with zero references to any captures store and no `isOpen`/drawer slice. The executing agent may factor the session-list and session-detail reads into React Query if it is materially cleaner; if they stay in Zustand, that is also acceptable.
- `scenarios/vrooli-emulator/ui/src/lib/bridge.ts` exposes a typed `postSessionEvent(event)` helper used at every session lifecycle transition. `main.tsx` initializes iframe-bridge with `appId: 'vrooli-emulator'`. Each posted event matches the payload schema in Contract Decisions below.
- No file under `scenarios/vrooli-emulator/ui/src/` references `/livedesktop`, `useCapturesStore`, `DesktopSession`, `LiveDesktopDrawer`, `Drawer` (from `components/ui/drawer`), or any other `Desktop*` identifier.
- `scenarios/vrooli-emulator/ui/package.json` has `@novnc/novnc`, `zustand`, and `react-router-dom` pinned to the same versions used in `scenarios/scenario-to-desktop/ui/package.json`.
- All component tests pass. Type check passes. Build produces a `dist/` with no warnings about missing deps.
- The emulator scenario starts cleanly via `vrooli scenario restart vrooli-emulator` (run by the user), and the UI loads a session list from the API.

## Implementation Strategy

### Phase 1 — API client and types (mechanical rewrite)

1. Copy `scenarios/scenario-to-desktop/ui/src/lib/api/livedesktop.ts` → `scenarios/vrooli-emulator/ui/src/lib/api/sessions.ts`.
2. Extract the minimal `buildUrl` + `throwIfNotOk` helpers into `scenarios/vrooli-emulator/ui/src/lib/api/http.ts` (they delegate to `@vrooli/api-base`'s `resolveApiBase({ appendSuffix: true })`). This is vrooli-ui-interop slot [F] — `resolveApiBase` must not be called in more than two files across the UI (`lib/api.ts` and `lib/api/http.ts`).
3. Rewrite every path literal: `/livedesktop/sessions…` → `/sessions…`. Rewrite `buildVncWsUrl()` to the new path and route through the slot [F] helpers.
4. Rename all `Desktop*` exported types → `Session*` (`DesktopSession` → `Session`, `DesktopSessionConfig` → `SessionConfig`, `DesktopSessionStatus` → `SessionStatus`, etc.). Update every type-level reference inside `sessions.ts` before leaving this phase.

### Phase 2 — Primitive components and router shell

5. Copy `scenario-to-desktop/ui/src/components/ui/input.tsx` → `scenarios/vrooli-emulator/ui/src/components/ui/`. Reconcile `button.tsx`: if scenario-to-desktop's copy has extra variants the migrated components use, port the missing variants into the emulator's copy; otherwise leave the emulator's `button.tsx` untouched. **Do not** copy `drawer.tsx` — it is no longer needed (round-2 d3).
6. Add `react-router-dom` to `ui/package.json`. Wrap `App.tsx` in `<BrowserRouter basename={…}>` using the vrooli-ui-interop slot [E] proxy-aware basename helper (derive from `import.meta.env.BASE_URL` or the `resolveApiBase` proxy hints).
7. Define two routes:
   - `/sessions` → `<SessionListPage>` (session list + create-new control).
   - `/sessions/:id` → `<SessionDetailPage>` (VNC canvas + toolbar + controls menu + metrics bar, stacked on a plain page).
8. `<SessionDetailPage>` is the landing target after a create or list-click. It renders its body directly — VNC canvas at the top, toolbar/metrics/controls beneath — with no drawer, backdrop, or slide-in. The phase-2 iframe-embed item builds its own drawer wrapper if needed.

### Phase 3 — Stateful components and store

9. Copy the five component files (`VncCanvas.tsx`, `DesktopToolbar.tsx` → `SessionToolbar.tsx`, `DesktopControlsMenu.tsx` → `SessionControlsMenu.tsx`, `MetricsBar.tsx`, `PlatformSelector.tsx`) into `scenarios/vrooli-emulator/ui/src/components/sessions/`. Compose them inside `SessionDetailPage.tsx`; do not introduce a `SessionViewer` wrapper component.
10. Migrate `store/liveDesktopStore.ts` → `store/sessionStore.ts`:
    - Strip the `useCapturesStore` import and replace every capture-related action. Capture responses (screenshot path, recording URL) become local state held either in the session store (`lastCapture?: { type, path, takenAt }`) or in component state held by `<SessionDetailPage>`. Either placement is acceptable; choose the one that keeps `<SessionControlsMenu>` props cleaner.
    - Remove the `isOpen`/`open()`/`close()` drawer slice. Navigation between list and detail uses `react-router-dom`'s `useNavigate()`.
    - Rename every `desktop*`/`LiveDesktop*` identifier.
    - Keep the side-effect orchestration (heartbeat interval, VNC disconnect handling) in the store — this is what Zustand is good at.
11. Optionally (per d5 user note): factor the session-list and session-detail **reads** into React Query with cache keys `['sessions']` and `['sessions', id]`. Timers, connection state, and error state stay in Zustand. Do this only if it removes meaningful duplication; do not add React Query scaffolding without payoff.

### Phase 4 — iframe-bridge lifecycle events

12. Add `appId: 'vrooli-emulator'` to the `initIframeBridgeChild(...)` call in `main.tsx`. Pass the parent origin from `import.meta.env.VITE_PARENT_ORIGIN` if set, otherwise leave unset.
13. Create `scenarios/vrooli-emulator/ui/src/lib/bridge.ts`:
    - Export `postSessionEvent(event: SessionEvent)` which wraps the iframe-bridge child-side `postMessage` helper.
    - Export a `SessionEvent` discriminated-union type covering `session.created`, `session.state_changed`, `session.error`, `session.destroyed` with the payload shape defined in Contract Decisions below.
14. Call `postSessionEvent(...)` from `sessionStore.ts` at every lifecycle transition: on successful create, on state changes (disconnected → connecting → connected, and any transition out of connected), on every caught error, and on destroy.

### Phase 5 — Tests and verification

15. Port `DesktopControlsMenu.test.tsx` → `SessionControlsMenu.test.tsx`. Update imports and rename identifiers. Remove any assertions that check `useCapturesStore` interactions.
16. Add new component/unit tests, all using `vi.mock('@/lib/api/sessions', () => ({ ... }))` (round-2 d2 option B):
    - `pages/SessionListPage.test.tsx` — empty state renders; populated state renders each session row; create-new flow calls the mocked `createSession()` and navigates to `/sessions/:id`.
    - `pages/SessionDetailPage.test.tsx` — state transitions disconnected → connecting → connected given a mocked `@novnc/novnc` RFB; error state surfaces a visible message; the page renders without any drawer chrome.
    - `store/sessionStore.test.ts` — lifecycle actions update state; capture actions set local `lastCapture` (or equivalent) and do **not** call any captures store; the store has no `isOpen`/`open`/`close` exports (grep-backed assertion in the test file).
    - `lib/bridge.test.ts` — every lifecycle transition posts the expected event payload (matching the Contract Decisions schema) to a stub bridge.
    - `lib/api/http.test.ts` — covers `buildUrl()` (correct prefixing through `@vrooli/api-base`) and `throwIfNotOk()` (throws with response body context on non-2xx). This is required because the page/store tests mock the API client layer above this seam.
17. Run the full gate: `pnpm --filter vrooli-emulator-ui run type-check && pnpm --filter vrooli-emulator-ui run test && pnpm --filter vrooli-emulator-ui run build`.

### Phase 6 — Cleanup & Scenario Health

18. Fix **all** lint, type, and test issues in the files this item touched — even pre-existing ones. Do not mark issues as "out of scope."
19. Surface the restart command and wait. The executing agent does not run `vrooli scenario restart vrooli-emulator` itself — the user runs it manually (project preference: agents do not restart scenarios on the user's behalf, even ones the agent isn't running inside).
20. After the user restarts, verify health:
    - `curl -s http://localhost:<emulator-port>/health` returns OK.
    - The UI loads at the emulator UI's dev URL; the session list renders; creating a 1280×720 Linux session and connecting via VNC works end-to-end.

## Contract Decisions

### API paths (settled)

All session operations go through `/api/v1/sessions/…`. The UI never uses the legacy `/livedesktop/` prefix. WS endpoints route through `@vrooli/api-base` (no hand-built `ws://localhost:…` strings).

### Routing contract (settled — round-1 d1 option B)

Two routes only:
- `GET /sessions` — session list page.
- `GET /sessions/:id` — session detail page.

The `:id` segment is the session UUID returned by `POST /sessions`. Direct deep-linking to `/sessions/:id` must work (supports the future `external-url` endpoint's `session_id` query param pattern).

### Captures contract (settled — round-1 d2 option A)

Capture *creation* is the emulator's job; capture *persistence* is not. The UI:
- Sends `POST /sessions/:id/captures` (screenshot or recording-start/stop) through `sessions.ts`.
- Displays the response (screenshot path, recording URL) inline in the current session view via `lastCapture` local state.
- Never calls a captures store. Never writes to any persistence layer. Refreshing the page is allowed to lose the `lastCapture` reference — that's by design.

### Naming convention (settled — round-1 d3 option A, hard cutover)

All `Desktop*` identifiers are renamed to `Session*` (or the corresponding domain-appropriate name) as part of the migration. No re-exports under the old names. No `// removed` comments. No temporary compatibility aliases.

### Layout for `/sessions/:id` (settled — round-2 d3 option B)

`SessionDetailPage.tsx` is a plain full-page layout. The VNC canvas, session toolbar, metrics bar, and controls menu are stacked directly on the page. There is no drawer wrapper, no backdrop, no slide-in animation, and no `Drawer`/`DrawerContent` component imported from `components/ui/`. Consequently `LiveDesktopDrawer.tsx` and `components/ui/drawer.tsx` are not migrated. If the phase-2 iframe-embed item later wants drawer semantics on the host side, it builds that wrapper at that time.

### iframe-bridge event surface (settled — round-1 d4 option B)

The emulator UI emits **outbound** lifecycle events only. It does **not** accept inbound control commands in this item (that is deferred to `execute/scenario-to-desktop-emulator-iframe-embed` when the host articulates what it needs).

Four events:
- `session.created` — fired once after a successful `POST /sessions` response.
- `session.state_changed` — fired on every connection-state transition (disconnected, connecting, connected, reconnecting, failed).
- `session.error` — fired on any caught error; includes error message/code.
- `session.destroyed` — fired once after a successful `DELETE /sessions/:id`.

### iframe-bridge event payload schema (settled — round-2 d1 option B)

All four events share a common envelope and payload shape. The shape is **identifier + summary fields**: enough for a host to render a session card without re-querying, but with no live metrics and no internal URLs.

```ts
type SessionEventType =
  | 'session.created'
  | 'session.state_changed'
  | 'session.error'
  | 'session.destroyed';

interface SessionEventPayload {
  sessionId: string;
  status: 'disconnected' | 'connecting' | 'connected' | 'reconnecting' | 'failed';
  createdAt: string;       // ISO-8601, the server-assigned create time
  backend: string;         // PlatformBackend identifier (e.g., 'linux')
  resolution: { width: number; height: number };
  error?: { code?: string; message: string };  // present for session.error; optional for state_changed
}

interface SessionEvent {
  type: SessionEventType;
  payload: SessionEventPayload;
}
```

Rules:
- Live metrics (CPU, memory, FPS) are **not** included in any payload — fetch them via the API on demand.
- Internal URLs (VNC WS URLs, capture file paths) are **not** included — they are not part of the cross-iframe contract.
- The same payload shape is used across all four event types; `error` is populated for `session.error` and may be populated for terminal `state_changed` transitions (e.g., `failed`).
- The payload is documented in `scenarios/vrooli-emulator/docs/embed-protocol.md` (the `chore/vrooli-emulator-documentation` item picks it up from there).

### State management (settled — round-1 d5 option A; split optional)

Zustand is the default store for session state. The executing agent has user-granted flexibility to factor session-list and session-detail **reads** into React Query if it meaningfully simplifies the code; the default remains single-Zustand-store.

### API mocking approach in tests (settled — round-2 d2 option B)

Component and store tests use `vi.mock('@/lib/api/sessions', () => ({ getSession: vi.fn(), listSessions: vi.fn(), createSession: vi.fn(), destroySession: vi.fn(), … }))`. No new devDependency (no MSW). The mocked surface is the API-client module, not `fetch`. Because this approach mocks the API-client layer, the `lib/api/http.ts` helpers (`buildUrl`, `throwIfNotOk`) get their own dedicated unit tests at `lib/api/http.test.ts` — those tests do not mock anything below their layer.

## Testing Plan

### Gates (must pass)

- `pnpm --filter vrooli-emulator-ui run type-check` — no TypeScript errors.
- `pnpm --filter vrooli-emulator-ui run test` — all tests pass, including the new component tests, the http helper tests, and the ported `SessionControlsMenu.test.tsx`.
- `pnpm --filter vrooli-emulator-ui run build` — produces `dist/` with no warnings about missing or unresolved dependencies.

### Tests added or ported in this item

| File | Mocking | Coverage |
|---|---|---|
| `pages/SessionListPage.test.tsx` | `vi.mock('@/lib/api/sessions', …)` | empty state; populated state; create-new flow calls `createSession` and navigates to `/sessions/:id` |
| `pages/SessionDetailPage.test.tsx` | `vi.mock('@/lib/api/sessions', …)` + mocked `@novnc/novnc` RFB | disconnected → connecting → connected transitions; error state visible; renders without drawer chrome (no `role="dialog"`/backdrop) |
| `store/sessionStore.test.ts` | `vi.mock('@/lib/api/sessions', …)` | lifecycle actions update state; capture actions set local `lastCapture`; no module references `useCapturesStore`; no `isOpen`/`open`/`close` exports (grep-backed assertions in the test file) |
| `lib/bridge.test.ts` | stub iframe-bridge child | each lifecycle transition posts the expected event payload (matching the Contract Decisions schema) |
| `lib/api/http.test.ts` | none (real `@vrooli/api-base`, no API server) | `buildUrl()` correctly composes `/sessions/...` → `/api/v1/sessions/...`; `throwIfNotOk()` throws with response body context on non-2xx |
| `components/sessions/SessionControlsMenu.test.tsx` (ported) | `vi.mock('@/lib/api/sessions', …)` | parity with the original `DesktopControlsMenu.test.tsx`, minus captures-store assertions |

### What is NOT tested in this item

- End-to-end against the live emulator API — that belongs to `execute/emulator-acceptance-tests-phase-1`.
- Iframe-embed host integration — that belongs to `execute/scenario-to-desktop-emulator-iframe-embed`.
- Cross-browser compatibility beyond Chromium (the scenario-to-desktop suite's baseline).

### Manual smoke (after user-run scenario restart)

1. Open the emulator UI dev URL.
2. Navigate to `/sessions`.
3. Create a session: 1280×720, Linux backend. Confirm POST to `/api/v1/sessions/` and a 201 response.
4. Page navigates to `/sessions/:id`. VNC canvas connects.
5. Take a screenshot. Confirm the result appears inline in the session view.
6. Destroy the session. Confirm the session is removed from `/sessions` list.
7. Check the emulator API logs: expect one create, one capture, and one destroy log entry.

## Rollout/Validation Checklist

- [ ] `pnpm --filter vrooli-emulator-ui run type-check` — passes
- [ ] `pnpm --filter vrooli-emulator-ui run test` — passes, includes the new component tests and `lib/api/http.test.ts`
- [ ] `pnpm --filter vrooli-emulator-ui run build` — produces `dist/` without missing-deps warnings
- [ ] `rg "/livedesktop" scenarios/vrooli-emulator/ui/src` — zero matches
- [ ] `rg "useCapturesStore" scenarios/vrooli-emulator/ui/src` — zero matches
- [ ] `rg "Desktop(Session|Controls|Toolbar|Drawer)" scenarios/vrooli-emulator/ui/src` — zero matches
- [ ] `rg "components/ui/drawer" scenarios/vrooli-emulator/ui/src` — zero matches (drawer primitive is not migrated)
- [ ] `rg "resolveApiBase" scenarios/vrooli-emulator/ui/src -l` — ≤2 files (slot [F] compliance)
- [ ] `scenarios/vrooli-emulator/docs/embed-protocol.md` — exists and describes the four session events with their payload shapes (matching the Contract Decisions schema verbatim)
- [ ] User runs `vrooli scenario restart vrooli-emulator` and reports it exited 0
- [ ] `curl -s http://localhost:<emulator-port>/health` — returns a healthy response
- [ ] Manual smoke (all 7 steps above) — succeeds end-to-end

## Risks + Mitigations

| Risk | Mitigation |
|---|---|
| `@novnc/novnc` RFB scaling regressions under `ResizeObserver` | Pin `@novnc/novnc` to the same `^1.6.0` that scenario-to-desktop uses; do not upgrade during migration. |
| Captures-store decoupling misses a branch | Before finalizing `sessionStore.ts`, grep the source store for every `useCapturesStore` reference and convert each explicitly. The `sessionStore.test.ts` grep-backed assertion catches regressions. |
| Tunnel/proxy context drift (WS URLs break under proxy) | Every URL (HTTP and WS) goes through `@vrooli/api-base` helpers. No hand-built `ws://localhost:…` strings in the migrated code. |
| iframe-bridge event payload drifts from documentation | The exact `SessionEventPayload` TypeScript type lives in `lib/bridge.ts` and is mirrored verbatim in `docs/embed-protocol.md`. `bridge.test.ts` asserts emitted payloads include exactly the documented fields (no more, no less) — adding a field requires updating both the doc and the test. |
| `vi.mock` stubs drift from the real `lib/api/sessions.ts` exports | The dedicated `lib/api/http.test.ts` tests cover the network seam directly, so a missing/renamed export in `sessions.ts` surfaces as a type error in the page tests rather than silently passing. |
| `@novnc/novnc` bundle size inflates the emulator UI | Acceptable trade-off — this is a management surface, not a landing page. Do not add a dynamic import for `novnc` unless a later perf issue materializes. |
| `react-router-dom` basename mismatch under the deployment proxy | Route basename resolution through vrooli-ui-interop slot [E] pattern; test under both root-hosted and proxy-hosted modes before marking done. |
| Reviewer confusion about what stays vs goes in scenario-to-desktop | Explicit non-goal above plus the out-of-scope cleanup note: deletion belongs to `chore/scenario-to-desktop-remove-livedesktop`. |
| Agent bails on "pre-existing" lint/type errors | Phase 6 step 18 is explicit: fix **all** issues in modified files, including pre-existing. The cleanup step is not optional. |
| Phase-2 iframe-embed wants drawer semantics later and finds nothing | Documented intentionally: building a drawer wrapper belongs to the iframe-embed item if needed. The standalone surface staying drawer-free is the better default per round-2 d3. |

## Non-goals / Prohibited Patterns

- **Do not** add re-exports of the old `Desktop*` names, temporary type aliases, or `// removed` comments. This is a hard cutover.
- **Do not** delete `scenarios/scenario-to-desktop/ui/src/components/livedesktop/**`, `lib/api/livedesktop.ts`, or `store/liveDesktopStore.ts` — that is the explicit charter of `chore/scenario-to-desktop-remove-livedesktop`.
- **Do not** copy `components/ui/drawer.tsx` from scenario-to-desktop. **Do not** create a `SessionViewer.tsx` wrapper that re-introduces drawer chrome. The standalone surface is drawer-free by decision.
- **Do not** add `react-router-dom` under a feature flag, lazy import, or optional dependency. It is a hard dep once d1 chose a router-based layout.
- **Do not** build design-system primitives beyond `input` + (possibly augmented) `button`. Copy-port, don't re-skin.
- **Do not** build a captures store, even "in-memory only."
- **Do not** compose API URLs by hand (`` `${window.location.origin}/api/v1/...` ``). Always route through the slot [F] helper.
- **Do not** call `resolveApiBase` in more than two files in the UI. Violations fail the checklist.
- **Do not** include live metrics, VNC WS URLs, or capture file paths in any `SessionEventPayload`. The cross-iframe contract is identifier + summary only.
- **Do not** add MSW or any other network-mocking dependency. The chosen approach (`vi.mock` of the API-client module + dedicated http helper tests) is a hard contract.
- **Do not** introduce a React Query query in any file other than the pages it backs (session-list and session-detail). No shared "query client fanout" modules.
- **Do not** leave `// TODO` or `// FIXME` comments in migrated code. Address or remove before commit.
- **Do not** run `vrooli scenario restart vrooli-emulator` from the executing agent. Surface the command and wait for the user to run it manually.

## Definition of Done

All of the following are true:

1. Every file listed in Current Technical Context → Source files has a corresponding migrated file in `scenarios/vrooli-emulator/ui/src/` at its new target path and name. The four "new files" entries also exist (`pages/SessionListPage.tsx`, `pages/SessionDetailPage.tsx`, `lib/api/http.ts`, `lib/bridge.ts`).
2. Every grep in the Rollout/Validation Checklist returns the stated zero-match or ≤2-match result.
3. Every test listed in Testing Plan passes; type check and build pass with no warnings about missing deps.
4. `scenarios/vrooli-emulator/docs/embed-protocol.md` exists and documents the four session events with payload shapes that match the `SessionEventPayload` type in `lib/bridge.ts` exactly.
5. The user has run `vrooli scenario restart vrooli-emulator` and reported success; the `/health` endpoint responds; and the manual smoke (7 steps) runs end-to-end without errors.
6. No compatibility shims, dead re-exports, `// removed` comments, `Desktop*`-prefixed identifiers, drawer primitives, or `SessionViewer` wrapper remain anywhere under `scenarios/vrooli-emulator/ui/src/`.
7. Downstream items (`execute/vrooli-emulator-external-url-endpoint`, `execute/scenario-to-desktop-emulator-iframe-embed`) can be queued without further UI-side prep work on this item's scope.
