# Research Notes — from research/command-center-architecture

This file records guidance from `research/command-center-architecture/conclusion.md` that the scaffold execution must incorporate.

## Lift shared hooks, don't rewrite them (Action 4)

Two existing sibling-scenario hooks must be adapted into `scenarios/command-center/ui/src/hooks/`:

- **Wake Lock** — `scenarios/web-console/ui/src/hooks/useWakeLock.ts` uses `navigator.wakeLock.request("screen")` with exponential backoff (5 retries, 30s heartbeat) plus a hidden-video fallback for iOS reliability. Directly reusable.
- **Fullscreen** — `scenarios/landing-manager/ui/src/hooks/useFullscreenMode.ts` and `scenarios/tech-tree-designer/ui/src/hooks/useFullscreenManager.ts` handle `requestFullscreen()` + cross-browser vendor prefixes with a CSS-class layout-fullscreen fallback. Pattern is portable.

Command-center's posture differs from those sources: **auto-fullscreen-on-load, wake-lock-always-on**. Adapt the hooks to this posture rather than copying verbatim.

See conclusion Finding 12.

## Verification path matrix (incorporate into plan.md Testing section)

When the scaffold's `plan.md` is authored, its Testing section must incorporate the verification paths from conclusion Finding 14:

- **Scaffold conformance:** `vrooli scenario start command-center` succeeds; `vrooli scenario completeness command-center` reports clean; Makefile targets `start/test/logs/stop` behave like siblings.
- **Data aggregation:** Go API integration tests against test doubles (httptest servers) for each upstream source, plus contract tests for the cached response envelope (data + staleness timestamp).
- **Gap registry:** Unit tests loading the JSON registry; assert `/api/v1/gaps` returns every `gap`/`partial` entry grouped by dashboard; schema validation test on the JSON file itself.
- **Theme isolation:** Playwright visual regression per dashboard page verifying CSS custom properties don't leak across route boundaries.
- **Kiosk mode:** Playwright test launched with `--start-fullscreen`, assertions on `document.fullscreenElement`, Wake Lock acquisition events, auto-cycle timer advancement.
- **R3F performance on Xbox:** Manual device testing gated on a performance tier telemetry endpoint (auto-downgrade events logged, frame-drop counts exposed on `/api/v1/debug/r3f-stats`).

## Other settled decisions relevant to scaffold

- **Stack:** Go API + React+Vite UI + `service.json` v2.0 lifecycle (matches swarm-manager / LPBS conventions).
- **Visualization:** React Three Fiber + Recharts. Per-page `React.lazy()` code-splitting of scene modules (Finding 15).
- **Per-page R3F Canvas** with lazy mount/unmount (Finding 9).
- **Fade-through-black transitions** mask Canvas init flash and dynamic-import window (Finding 11).
- **Theming:** CSS custom properties scoped per route wrapper (Finding 5).
- **Gap registry:** File-based JSON loaded at API startup; drives `/api/v1/gaps` (Finding 6).
- **Caching TTLs:** Swarm 30s, Vrooli 60s, LPBS 5min; graceful degradation returns last cached data with staleness timestamp (Finding 7).
- **LPBS auth:** Bearer token on `/api/v1/admin/dashboard/*` via `requireAdminOrService` (Findings 4 & 10); LPBS-side endpoints tracked separately at `execute/lpbs-command-center-dashboard-endpoints`.
- **Controller input:** Consume `@vrooli/iframe-bridge` `useSpatialNav` + `<SpatialGroup>` — already bundled in the react-vite template (Finding 16). Verify the dependency is present in `scenarios/command-center/ui/package.json`.
