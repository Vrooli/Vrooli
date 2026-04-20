# Implementation Plan — `execute/command-center-scenario-scaffold`

> Scaffold the `command-center` scenario at `scenarios/command-center/` with a React+Vite UI, Go API, and `service.json` v2.0 lifecycle. This is the foundation that `execute/command-center-theming-engine` and `execute/command-center-kiosk-ux` build on.

## 1. Purpose

Create a clean, runnable `scenarios/command-center/` directory that conforms to Vrooli scenario conventions (sibling parity with `swarm-manager` and `landing-page-business-suite`) and provides the structural seams required by the sibling execute items in the `command-center-foundation` initiative. After this item lands:

- `vrooli scenario start command-center` starts API + UI cleanly.
- `vrooli scenario completeness command-center` reports clean.
- The 6 dashboard routes (Mission Control, The Hive, The Forge, Ledger, Broadcast, Panorama) exist as navigable pages — **Mission Control is a full vertical slice** (real Ground Control theme values, real R3F scene, real data fetch through to gap badges); the other 5 are themed placeholders (one `data-theme` wrapper + a trivial lazy R3F placeholder scene).
- The Go API serves `/api/v1/health`, `/api/v1/dashboards/{id}`, `/api/v1/gaps`, and `/api/v1/debug/r3f-stats` with a functioning per-source TTL cache layer and a file-based gap registry populated for all 6 dashboards.
- The theming-engine and kiosk-ux items have clean landing seams to fill (theming-engine replaces the 5 remaining placeholder scenes + fills in 5 themes; kiosk-ux wires auto-cycle + hidden controls + D-pad + fullscreen-on-load).

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read storage-steer react-coherence test
prompt-manager skill read browser-automation-studio e2e-testing
```

Background reading (domain/context):
- `scenarios/swarm-manager/execute/command-center-scenario-scaffold/research-notes.md` — distilled research notes for this specific scaffold.
- `scenarios/swarm-manager/research/command-center-architecture/conclusion.md` — full research conclusion (17 findings, all settled architectural decisions).
- `scenarios/swarm-manager/initiatives/command-center-foundation/orchestration-summary.md` — brainstorming context, theme assignments per dashboard, upstream data source detail.

Reference implementations:
- `scenarios/swarm-manager/` — canonical scenario layout (`api/`, `ui/`, `.vrooli/service.json`, `Makefile`).
- `scenarios/landing-page-business-suite/` — second reference for sibling parity.
- `templates/scenarios/react-vite/` — starting point for the entire scenario scaffold. `template.json`, `.vrooli/service.json`, `cli/`, `api/`, and `ui/` must be consumed as-is where possible; only diverge when command-center's needs demand it (documented below).
- `scenarios/web-console/ui/src/hooks/useWakeLock.ts` — Wake Lock hook source (to adapt).
- `scenarios/landing-manager/ui/src/hooks/useFullscreenMode.ts` — Fullscreen hook source (to adapt).
- `scenarios/prompt-manager/ui/src/components/world/WorldCanvas.tsx` — R3F Canvas/tier-aware rendering reference.
- `scenarios/browser-automation-studio/bas/cases/01-foundation/` — BAS case JSON examples.
- `scenarios/scenario-completeness-scoring/bas/` — reference BAS layout (actions, flows, cases, seeds, registry.json).
- `scenarios/prd-control-tower/.vrooli/testing.json` — reference `testing.json` shape (adapt `playbooks.enabled: true` for our case).

## 3. Problem Statement

`scenarios/command-center/` does not yet exist. The initiative has three dependent execute items (scaffold → theming-engine → kiosk-ux) and one cross-scenario item (`execute/lpbs-command-center-dashboard-endpoints`) that cannot proceed until the scaffold is in place. The scaffold needs to:

1. Establish the directory layout and lifecycle configuration so `vrooli scenario` commands work.
2. Wire up enough of the Go API and React UI that the sibling items land on clear seams, not a blank slate.
3. Ship Mission Control as a complete vertical slice so the end-to-end pipeline (upstream fetch → cache → API payload → UI query → themed scene + gap badges) is exercised and validated before sibling items fan out.
4. Avoid doing the sibling items' work for the OTHER 5 dashboards (per-theme CSS, auto-cycle logic, D-pad input, full R3F scenes) — those items own that scope.

## 4. Scope

### In scope
- Scenario directory skeleton at `scenarios/command-center/` with `api/`, `ui/`, `.vrooli/service.json`, `.vrooli/testing.json`, `Makefile`, `README.md`, `config/` (for the gap registry JSON), `bas/` (for e2e cases), and `docs/` folder.
- Go API with `/api/v1/health`, `/api/v1/dashboards/{id}`, `/api/v1/gaps`, `/api/v1/debug/r3f-stats` endpoints.
- Per-source TTL cache layer (Swarm 30s, Vrooli 60s, LPBS 5min) with graceful degradation (last-cached data + `staleness_ts`).
- File-based JSON gap registry loaded at API startup, **fully populated for all 6 dashboards** from the orchestration summary's data-availability tables.
- Upstream client interfaces + initial implementations for Swarm Manager, Vrooli Core, and LPBS. LPBS implementation calls the (future) `/api/v1/admin/dashboard/*` with Bearer token wired; falls through to gap-mode cleanly on 404 until the sibling LPBS endpoints item ships.
- React+Vite UI with router, 6 route pages (5 placeholders + 1 full vertical slice), shared layout, `ThemeProvider` populated with the full Ground Control theme values (CSS custom properties) + stub entries for the other 5 themes.
- **Mission Control vertical slice:** real Ground Control theme CSS values, real R3F scene (instanced-points starfield + 3–5 rotating satellite node meshes wired to the Ground Control palette — a single reusable scene module that exercises both static and animated rendering), real API data fetch via react-query, real gap badges on metrics pulled from `/api/v1/dashboards/mission-control`. No live-metric visualization binding in the slice — theming-engine owns metric visualization design.
- **Other 5 routes:** themed placeholder — `data-theme` wrapper + trivial rotating-cube R3F scene via `React.lazy()` (so the code-split seam is exercised on all 6 routes), H1 title, metric list from registry with gap badges.
- `SceneCanvas.tsx` wrapper + per-page lazy scene modules (5 trivial placeholder scenes + 1 Mission Control real scene).
- Adapted `useWakeLock` and `useFullscreen` hooks in `ui/src/hooks/` (not auto-invoked — kiosk-ux wires the policies).
- **Tests:**
  - Go: unit + integration tests (httptest doubles for upstream sources), registry load tests, cache envelope contract, handler tests, LPBS gap-mode fallback.
  - UI: vitest component tests (`DashboardLayout`, `GapBadge`, `StaleBadge`, `SceneCanvas` fallback, Mission Control page renders with real theme + data).
  - E2E: **BAS/test-genie** via `.vrooli/testing.json` with `playbooks.enabled: true`. Scaffold ships a minimal `bas/` tree: 1 action (navigate-to-route), 1 flow (navigate-all-routes), 6 route-loads cases under `bas/cases/01-foundation/` (one per route: page loads, no console errors, `data-theme` attribute is set), plus 1 Mission Control case asserting gap badges render.
- Makefile targets: `start`, `stop`, `restart`, `status`, `logs`, `test`, `setup`, `open` (delegating to `vrooli scenario ...`).
- UI proxy configuration so `/api/*` routes hit the Go API.
- Port allocation: use the **react-vite template standard ranges** (API `15000-19999`, UI `35000-39999`, WS `25000-29999` — WS omitted since not needed). The lifecycle manager allocates within the range at start time; no hardcoded ports are committed in `.vrooli/service.json`.

### Out of scope
- Full per-theme CSS, typography, and background treatments for the 5 non-Mission-Control themes (owned by `execute/command-center-theming-engine`).
- Replacing the trivial placeholder R3F scenes on the 5 non-Mission-Control routes with real scene content (theming-engine scope).
- Auto-cycle timer, fade-through-black transitions, settings panel, hidden controls, D-pad input wiring, auto-fullscreen-on-load (owned by `execute/command-center-kiosk-ux`).
- New LPBS `/api/v1/admin/dashboard/*` endpoints (owned by `execute/lpbs-command-center-dashboard-endpoints`).
- Xbox Edge device testing (owned by kiosk-ux verification).
- Visual-regression baselines per theme (owned by theming-engine).
- Live-metric visualization binding inside the Mission Control scene (owned by theming-engine — scaffold's slice is starfield + satellite nodes only, not metric-bound visualizations).
- Perf/load testing, security scanning, audit review (out of scaffold scope entirely).

## 5. Current Technical Context

### Directory targets
- Final location: `scenarios/command-center/` (path confirmed by `acceptance_allow: scenarios/command-center/**`).
- Sibling parity targets: `scenarios/swarm-manager/`, `scenarios/landing-page-business-suite/`, `scenarios/scenario-completeness-scoring/` (for BAS layout).

### Port allocation
- Use the **react-vite template defaults verbatim** from `templates/scenarios/react-vite/.vrooli/service.json`:
  - API port → `env_var: "API_PORT"`, `range: "15000-19999"`.
  - UI port → `env_var: "UI_PORT"`, `range: "35000-39999"`.
  - Websocket entry is omitted (no WS usage in scaffold).
- The lifecycle manager auto-allocates a free port inside each range at `vrooli scenario start` time and exposes them to the process via `API_PORT` / `UI_PORT` env vars. **Do not hardcode a `port` field** — stay range-only. This avoids collisions with siblings (swarm-manager 36234, prd-control-tower 36300/18600, landing-page-business-suite and others) without hand-picking numbers.
- If a sibling item later requires a deterministic port for cross-scenario plumbing, that item can propose pinning one in a follow-up.

### Key files to author
- `scenarios/command-center/.vrooli/service.json` — lifecycle v2.0, port ranges above, health checks, dependencies (swarm-manager, landing-page-business-suite, vrooli core).
- `scenarios/command-center/.vrooli/testing.json` — test-genie config, `playbooks.enabled: true`, `structure.additional_dirs: ["ui", "bas"]`, Go coverage thresholds consistent with siblings.
- `scenarios/command-center/Makefile` — fork swarm-manager's Makefile pattern; `SCENARIO_NAME := $(notdir $(CURDIR))`.
- `scenarios/command-center/api/main.go`, `routes.go`, `server.go`, `cache.go`, `registry.go`, plus `upstream/` package with per-source clients.
- `scenarios/command-center/config/gap-registry.json` — loaded at startup. Loader reads a `version` field + unmarshals into Go structs (struct tags enforce shape). **No explicit validation pass, no external JSON-schema library.** If the file is syntactically malformed or a required field is missing via `json:"...,required"` semantics via a decoder with `DisallowUnknownFields`, the startup returns an error and exits non-zero. Otherwise the registry is trusted as authored.
- `scenarios/command-center/ui/` — fork of `templates/scenarios/react-vite/ui/` with additions.
- `scenarios/command-center/ui/src/pages/{MissionControl,Hive,Forge,Ledger,Broadcast,Panorama}.tsx` — 6 route pages (1 full slice + 5 placeholders using a shared `PlaceholderPage.tsx`).
- `scenarios/command-center/ui/src/components/DashboardLayout.tsx`, `ThemeProvider.tsx`, `GapBadge.tsx`, `StaleBadge.tsx`, `SceneCanvas.tsx`, `PlaceholderPage.tsx`.
- `scenarios/command-center/ui/src/scenes/` — lazy-loaded R3F scene modules. `missionControl.tsx` is the real scene (starfield + satellite nodes); `trivialCube.tsx` is the shared placeholder for the other 5 routes.
- `scenarios/command-center/ui/src/themes/` — CSS custom-property tokens. `ground-control.css` fully populated; `bioluminescent.css`, `foundry.css`, `vault.css`, `signal-tower.css`, `cosmos.css` stub files with palette-level defaults only (theming-engine fills these).
- `scenarios/command-center/ui/src/hooks/useWakeLock.ts`, `useFullscreen.ts` — adapted from sibling scenarios.
- `scenarios/command-center/ui/src/lib/api.ts` — API client calling `/api/v1/dashboards/*`, `/api/v1/gaps`.
- `scenarios/command-center/bas/actions/navigate-to-route.json`, `bas/flows/navigate-all-routes.json`, `bas/cases/01-foundation/{mission-control,hive,forge,ledger,broadcast,panorama}-loads.json`, `bas/cases/01-foundation/mission-control-gap-badges-render.json`, `bas/registry.json` (generated via `test-genie registry build`).

### Dependencies to add to `ui/package.json`
Starting from the react-vite template, the UI package needs these additions:
- `react-router-dom` (routing)
- `recharts` (charts)
- `three`, `@react-three/fiber`, `@react-three/drei` (R3F — Finding 3)
- `framer-motion` (page transitions; kiosk-ux item may replace with simpler primitives)

The template already has: `@tanstack/react-query`, `@vrooli/iframe-bridge`, `tailwind-merge`, `clsx`, `lucide-react`, `react`, `react-dom`. No need to re-add.

### Upstream endpoints the scaffold must call
- Swarm Manager: `GET http://localhost:${SWARM_MANAGER_API_PORT}/api/v1/stats`, `/api/v1/overview`.
- Vrooli Core: `GET http://localhost:8092/scenarios` (port 8092 per research).
- LPBS: `GET /api/v1/admin/dashboard/*` with `Authorization: Bearer ${LPBS_SERVICE_TOKEN}` — endpoints don't exist yet; the LPBS client ships with Bearer-token plumbing already wired and falls through to gap-mode on 404/connection-refused so the scaffold is self-contained.

### r3f-stats telemetry storage
- In-memory ring buffer, capacity 1024 events. Events older than 1 hour are evicted on insert. No persistence (restart clears).
- GET returns the current buffer contents (newest first); POST appends one event.
- **No auth on this endpoint in the scaffold** — it's a local-only debug surface, consistent with command-center being a local-network aggregator with no user auth of its own. If a future security review objects, a sibling item can add `X-Debug-Token` gating without breaking contracts.

## 6. Target End State

After this item lands, a human running the following commands from a clean checkout should get a green path:

```bash
cd scenarios/command-center
make setup                                                                   # succeeds: builds Go API, installs pnpm deps, builds UI
vrooli scenario start command-center                                         # succeeds: API and UI up (ports auto-allocated within template ranges)
curl -sf http://localhost:$API_PORT/api/v1/health | jq '.status'             # "ok"
curl -sf http://localhost:$API_PORT/api/v1/gaps | jq '.dashboards | keys'    # ["broadcast","forge","hive","ledger","mission-control","panorama"]
curl -sf http://localhost:$API_PORT/api/v1/dashboards/mission-control | jq '.metrics' # array of metric entries with live/gap/partial status
open http://localhost:$UI_PORT/                                              # UI loads, router renders, 6 routes navigable
make test                                                                    # Go unit+integration + vitest + BAS cases (via test-genie) all pass
vrooli scenario completeness command-center                                  # clean report
vrooli scenario restart command-center                                       # canonical cycle: clean stop + restart produces identical green state
vrooli scenario stop command-center                                          # clean shutdown; no orphan processes
```

All 6 dashboard routes render with the correct theme-wrapper `data-theme` attribute applied. Mission Control renders with the real Ground Control palette + real scene (starfield + satellite nodes) + real API data + gap badges on gap/partial metrics. The other 5 render with stub palette values (variant accent color per route so they're visually distinguishable), the shared trivial-cube lazy scene, and gap badges driven by the registry.

## 7. Implementation Strategy (phased)

### Phase 1 — Directory skeleton and lifecycle wiring
1. Create `scenarios/command-center/` with empty `api/`, `ui/`, `config/`, `docs/`, `bas/`, `.vrooli/` subdirs.
2. Author `.vrooli/service.json`: copy the react-vite template verbatim, then override `service.name`, `displayName`, `description`, `category: "business-application"`, `tags: ["react-ui", "go-api", "dashboard", "monitoring", "kiosk"]`. **Leave the `ports` block as-is** (ranges only: API `15000-19999`, UI `35000-39999`) — do not add concrete `port` values. Remove the `websocket` entry.
3. Author `.vrooli/testing.json`: fork prd-control-tower's file, set `playbooks.enabled: true`, add `ui` + `bas` to `structure.additional_dirs`, include Go coverage thresholds consistent with siblings.
4. Author `Makefile` by forking swarm-manager's Makefile verbatim (no changes needed — `SCENARIO_NAME := $(notdir $(CURDIR))` makes it portable).
5. Author `README.md` following sibling conventions (purpose, quick-start commands, architecture overview, link to plan file).
6. Smoke check: `vrooli scenario status command-center` reports "not running" cleanly (no lifecycle errors). Port allocation is deferred to start time; the service.json is range-based, so no static-port grep is necessary.

**Convergence checkpoint:** `vrooli scenario setup command-center` runs without error (even on empty `api/` + `ui/` — steps are guarded by `condition.file_exists`).

### Phase 2 — Go API foundation
1. `cd api && go mod init github.com/Vrooli/Vrooli/scenarios/command-center/api`.
2. Author `main.go` (env-driven port via `API_PORT`, gorilla/mux router, graceful shutdown, structured logging — follow swarm-manager patterns).
3. Author `server.go` with a `Server` struct carrying config, cache, registry, upstream clients, and the r3f-stats ring buffer.
4. Author `routes.go` with `setupRoutes(s *Server)` and `register*Routes(s)` per domain (health, dashboards, gaps, debug).
5. Author minimal handlers that return static `{"status":"ok"}` for health, empty payloads for dashboards/gaps. Full behavior added in later phases.
6. `go build ./... && go vet ./...` clean.

**Convergence checkpoint:** API binary builds; the binary starts under the lifecycle manager with the allocated `API_PORT`; `curl /api/v1/health` returns 200.

### Phase 3 — Gap registry + cache layer + upstream clients
1. Author `config/gap-registry.json` with **all 6 dashboards' metric entries fully populated** from `orchestration-summary.md`:
   - Mission Control (~70% live): scenario health, swarm stats, revenue (gap).
   - The Hive (~80% live): scenario metadata/health/completeness, usage frequency (gap).
   - The Forge (~90% live): throughput, timing, scope, blocking, agent, dashboard, review stats.
   - Ledger (~60% live): subscriber counts by tier, revenue, churn, credit balances/consumption.
   - Broadcast (~40% live): LPBS analytics (live), social/SEO/campaign metrics (gap).
   - Panorama (composite — marked partial where its constituent dashboards are partial/gap).
   Schema: `{"version": "1.0.0", "dashboards": {"<id>": [{"id", "label", "dataSource", "upstreamSource", "description", "whatIsNeeded"}, ...], ...}}`.
2. Author `registry.go`: loads the JSON at startup and unmarshals into typed Go structs. Use a `json.Decoder` with `DisallowUnknownFields()` so typos in the JSON are caught by the decoder itself. **No hand-written validation pass and no external JSON-schema library** — the Go struct shape is the contract. If decoding fails, log + exit non-zero. If decoding succeeds, the registry is trusted as authored.
3. Author `cache.go` with a TTL cache keyed by `(source, request)`. Cache envelope: `{data, cached_at, staleness_ts, from_cache, source}`. TTLs: Swarm 30s, Vrooli 60s, LPBS 5min.
4. Author `upstream/swarm.go`, `upstream/vrooli.go`, `upstream/lpbs.go` — client interfaces with `Fetch(ctx, path)` methods. Swarm and Vrooli clients call real endpoints. LPBS client is **fully wired with `Authorization: Bearer ${LPBS_SERVICE_TOKEN}`** from env; gracefully falls through to gap-mode on 404, 5xx, or connection-refused (since endpoints don't exist yet).
5. Wire cache in front of upstream clients. On upstream error: return last cached data with `staleness_ts` if any, else return gap-mode data from the registry.
6. Author `/api/v1/gaps` handler that reads registry and returns all entries with `dataSource ∈ {gap, partial}` grouped by dashboard.
7. Author `/api/v1/dashboards/{id}` handler that returns a pre-composed payload (registry entries + cached upstream data) for the requested dashboard.
8. Author `/api/v1/debug/r3f-stats`: in-memory ring buffer (cap 1024, 1h TTL on insert). POST appends; GET returns newest-first. No auth.

**Convergence checkpoint:** `go test ./...` passes all handler, cache, registry, and upstream-client tests; `curl /api/v1/gaps` returns entries for all 6 dashboards; `curl /api/v1/dashboards/mission-control` returns a non-empty payload with the expected shape.

### Phase 4 — UI scaffold (clone template, add deps, wire router)
1. `cp -r templates/scenarios/react-vite/ui/* scenarios/command-center/ui/` (or equivalent — no symlinks).
2. Update `ui/package.json`: set `name` to `command-center-ui`, add deps (`react-router-dom`, `recharts`, `three`, `@react-three/fiber`, `@react-three/drei`, `framer-motion`).
3. Update `ui/vite.config.ts`: add proxy config so `/api/*` forwards to `http://localhost:${API_PORT}` (read from env at startup).
4. Replace `ui/src/App.tsx` with a `BrowserRouter` containing 6 routes (`/mission-control`, `/hive`, `/forge`, `/ledger`, `/broadcast`, `/panorama`), default `/` redirects to `/mission-control`.
5. Author `ui/src/themes/ground-control.css` with the full Ground Control palette + typography + accent values (electric blue, monospace, space aesthetic). Stub files for the other 5 themes with palette-only defaults (distinct accent colors so routes are visually distinguishable at placeholder fidelity).
6. Author `ui/src/components/DashboardLayout.tsx` — wraps children in `<div data-theme={themeKey}>`, loads the theme's CSS module via dynamic import.
7. Author `ui/src/components/SceneCanvas.tsx` — R3F `<Canvas>` wrapper with performance-tier defaults; accepts a `scene` React node via a lazy-loaded prop.
8. Author `ui/src/scenes/trivialCube.tsx` — single rotating cube using the theme's accent color. Shared by 5 routes.
9. Author `ui/src/scenes/missionControl.tsx` — real scene: **instanced-points starfield + 3–5 rotating satellite node meshes** using simple geometry (spheres or boxes), wired to the Ground Control palette. Fits on a single screen of code. No metric-bound visualizations — theming-engine owns those.
10. Author `ui/src/components/PlaceholderPage.tsx` — shared placeholder route that calls the API + renders H1 + gap badges; imports `trivialCube` as its scene.
11. Author 6 route components — `ui/src/pages/MissionControl.tsx` uses the full slice (fetches `/api/v1/dashboards/mission-control` via react-query, renders `missionControl` scene, renders gap badges on gap/partial metrics). The other 5 pages render `<PlaceholderPage themeKey="..." dashboardId="..." />`.
12. Author `ui/src/lib/api.ts` — `fetchDashboard(id)`, `fetchGaps()` functions using `@tanstack/react-query`.
13. Author `ui/src/hooks/useWakeLock.ts` (adapted from web-console) and `ui/src/hooks/useFullscreen.ts` (adapted from landing-manager). **Do NOT auto-invoke** — kiosk-ux wires the policy.
14. Author `ui/src/components/GapBadge.tsx` and `ui/src/components/StaleBadge.tsx` — minimal renderings; theming-engine item refines per-theme visuals. GapBadge carries `data-testid="gap-badge"` for BAS assertions.
15. `pnpm run type-check && pnpm run lint && pnpm run build` all clean.

**Convergence checkpoint:** `vite preview` (or `node server.js` after build) serves the UI; all 6 routes navigable; no console errors; Mission Control shows the real scene (starfield + satellites) + real data + gap badges; the other 5 show the rotating cube + list of metrics with badges.

### Phase 5 — Testing
1. **Go API tests** (`api/*_test.go`): handler tests with `httptest`, cache tests, registry load tests (valid-file round-trip + decode-error exit), upstream client tests against test doubles (httptest servers), LPBS gap-mode fallback test, r3f-stats ring buffer tests.
2. **UI tests** (`ui/src/**/*.test.ts(x)`): vitest component tests for `DashboardLayout`, `GapBadge`, `StaleBadge`, `SceneCanvas` fallback, Mission Control page (render with mocked react-query + mocked scene).
3. **BAS/test-genie e2e** (`bas/`):
   - `bas/actions/navigate-to-route.json` — reusable navigate action parameterized by route.
   - `bas/flows/navigate-all-routes.json` — composes `navigate-to-route` for all 6 routes.
   - `bas/cases/01-foundation/{route}-loads.json` — 6 cases, one per route: navigate, assert `data-theme` attribute present, assert no console errors on load, screenshot.
   - `bas/cases/01-foundation/mission-control-gap-badges-render.json` — 1 case asserting `[data-testid="gap-badge"]` exists on at least one metric after page load, plus the Mission Control scene canvas is mounted.
   - `bas/registry.json` — generated via `test-genie registry build`.
4. `make test` runs the combined suite (Go + UI + BAS/test-genie playbooks).

**Convergence checkpoint:** `make test` passes; `vrooli scenario completeness command-center` reports clean.

### Phase 6 — Lifecycle + docs polish + final restart
1. Verify `.vrooli/service.json` still uses range-based ports only (no concrete `port` values crept in).
2. Verify `make start`/`stop`/`restart`/`status`/`logs`/`test` all work end-to-end.
3. Author `docs/ARCHITECTURE.md` summarising the per-source cache, gap registry, theming seams (including the Mission Control reference slice), and where the sibling items land.
4. Update scenario catalog/index if applicable.
5. **Final cleanup:** from the scenario directory (or repo root), run `vrooli scenario restart command-center` to exercise the canonical stop+start cycle against the finished scaffold. Verify: the restart exits 0, both health endpoints return 200 within the grace period, no orphan processes remain from the previous invocation, and the allocated ports are the same or different-but-free (range-allocated is fine either way). This is the canonical final validation step per project convention.

**Final convergence:** `vrooli scenario restart command-center` is green; full `vrooli scenario` lifecycle is green; `scenarios/swarm-manager` recognizes `command-center` as a peer in its dependency graph if relevant.

## 8. Contract Decisions

### API contracts
- `GET /api/v1/health` → `{"status": "ok", "version": "0.0.1", "uptime_seconds": <int>}`.
- `GET /api/v1/dashboards/{id}` → `{"dashboard": "<id>", "generated_at": "<iso>", "metrics": [<metric_entry>...], "sources": {"<source>": {"from_cache": <bool>, "staleness_ts": "<iso|null>"}}}`.
- `GET /api/v1/gaps` → `{"generated_at": "<iso>", "dashboards": {"<id>": [<metric_entry_with_status_gap_or_partial>...]}}`.
- `GET /api/v1/debug/r3f-stats` → `{"events": [{"tier": <int>, "fps": <float>, "ts": "<iso>"}, ...]}`; `POST` accepts a single event.
- All errors: `{"error": "<code>", "message": "<human>", "details": <object|null>}` with proper 4xx/5xx.

### Gap registry JSON schema (loaded at startup)
```json
{
  "version": "1.0.0",
  "dashboards": {
    "mission-control": [
      {
        "id": "active_scenarios",
        "label": "Active scenarios",
        "dataSource": "live",
        "upstreamSource": "vrooli",
        "description": "Count of scenarios currently running.",
        "whatIsNeeded": null
      }
    ]
  }
}
```
`dataSource ∈ {"live", "partial", "gap"}`. `upstreamSource ∈ {"vrooli", "swarm", "lpbs", "none"}`. `whatIsNeeded` is prose describing what blocks the metric (used by `/api/v1/gaps` consumers).

**Validation strategy:** trust Go struct unmarshalling. Use `json.Decoder` with `DisallowUnknownFields()` at startup; this catches malformed JSON, unknown keys, and type mismatches. Beyond that, the registry is trusted as authored — no hand-written enum-range checks, no duplicate-ID scans, no external schema library. Rationale: scaffold simplicity, single-producer/single-consumer file, and the registry author (us, via this item) controls all content. If future sibling items demand stricter enforcement, they can add it as a follow-up.

### Port allocation (contract with the lifecycle manager)
`.vrooli/service.json` declares port **ranges only** (API `15000-19999`, UI `35000-39999`), inherited directly from `templates/scenarios/react-vite/.vrooli/service.json`. The lifecycle manager is responsible for:
1. Picking a free port inside each range at `vrooli scenario start` time.
2. Exposing the chosen ports via the `API_PORT` / `UI_PORT` env vars.
3. Keeping the selection stable across the scenario's running session.
The scaffold's Go API and Vite dev server both read their port from the env var — never from a hardcoded value.

### Cache envelope (internal)
```go
type CachedResponse[T any] struct {
    Data        T
    CachedAt    time.Time
    StalenessTS *time.Time
    FromCache   bool
    Source      string
}
```
When upstream returns an error, the envelope is returned with `StalenessTS` set to `CachedAt` of the previous successful fetch. When there's never been a successful fetch, the handler falls back to gap-mode data from the registry.

### UI theme key convention
Theme keys are the slugified page name: `ground-control`, `bioluminescent`, `foundry`, `vault`, `signal-tower`, `cosmos`. The `<div data-theme="{key}">` wrapper is set by `DashboardLayout`. CSS custom properties are consumed by descendants only. **Ground Control is fully populated in this item; the other 5 have palette-only stubs for theming-engine to expand.**

### Mission Control scene contract
- Single scene module at `ui/src/scenes/missionControl.tsx`, lazy-loaded by `MissionControl.tsx`.
- Content: instanced-points starfield + 3–5 rotating satellite node meshes (simple geometry, simple materials) wired to Ground Control palette via CSS custom properties or theme context.
- Must not import metric data — it's a pure visual scene. Metric display happens via the surrounding `DashboardLayout` + `GapBadge` components, not inside the R3F canvas.
- Must not import from any other scenario's code (hooks are adapted into `ui/src/hooks/`, not imported).

### BAS test case taxonomy (this scaffold)
All scaffold BAS cases live under `bas/cases/01-foundation/` and target page-load and structural assertions only. Visual regression, kiosk-mode behavior, auto-cycle, spatial-nav, and theme-contrast cases are deferred to sibling items (theming-engine → `02-builder/` or `03-execution/`; kiosk-ux → `04-ai/` or `06-nonfunctional/` depending on focus).

## 9. Testing Plan

Verification paths from research/command-center-architecture Finding 14 and the user's round-1/round-2 direction:

| Area | Scope | Verification |
|---|---|---|
| Scaffold conformance | `vrooli scenario start/stop/restart/status/logs/test` + `vrooli scenario completeness` | Manual smoke in a clean workspace; Makefile targets behave like siblings. Final Phase 6 restart must be green. |
| Data aggregation | Go API handler + cache + upstream clients | Go integration tests against `httptest` servers per upstream; contract tests on cache envelope (data + `staleness_ts`). |
| Gap registry | JSON registry + `/api/v1/gaps` | Unit tests loading the registry with `DisallowUnknownFields`; assert `/api/v1/gaps` returns every `gap`/`partial` entry grouped by dashboard; decode-error test (malformed JSON → startup exit non-zero). No semantic enum/uniqueness tests — those are deferred with the validation strategy. |
| LPBS gap-mode fallback | `upstream/lpbs.go` | Go test: point client at a test double that returns 404; assert handler returns gap-mode payload from registry. |
| Mission Control vertical slice | Full pipeline (API → UI → scene + badges) | BAS case `mission-control-gap-badges-render.json`: navigate, assert `[data-testid="gap-badge"]` exists, assert scene canvas is present (no WebGL error), screenshot. |
| Theme isolation (scaffold-level) | `DashboardLayout` wrapper per page | Vitest component test: mount two themed wrappers, assert computed CSS custom-property values differ. (Visual regression per theme is theming-engine scope.) |
| Fullscreen / Wake Lock hooks | `useWakeLock`, `useFullscreen` | Vitest: JSDOM tests for state transitions; mock `navigator.wakeLock.request` and `document.fullscreenElement`. Xbox real-device testing is kiosk-ux scope. |
| R3F performance telemetry | `/api/v1/debug/r3f-stats` | Go test POSTing events and GETting them back; ring-buffer eviction test. |
| Route-level smoke (all 6) | BAS cases `{route}-loads.json` | Per-route BAS case: navigate, assert `data-theme` attribute set, assert no console errors, screenshot. Runs via `make test` → test-genie playbooks phase. |

**Not tested in this item (owned elsewhere):** auto-cycle timing, fade-through-black transitions, D-pad spatial navigation, real R3F scene visuals for the 5 non-Mission-Control routes, per-theme visual regression, Xbox Edge device smoke test, semantic registry validation (enum-range, duplicate-ID, required-field beyond struct-tag shape).

## 10. Rollout / Validation Checklist

- [ ] `ls scenarios/command-center/` shows expected layout (including `bas/`, `config/`, `.vrooli/testing.json`).
- [ ] `.vrooli/service.json` declares API + UI as ranges only (no hardcoded `port` values).
- [ ] `vrooli scenario completeness command-center` → clean.
- [ ] `make setup` → green.
- [ ] `vrooli scenario start command-center` → API + UI up, health endpoint returns 200; `API_PORT`/`UI_PORT` env vars are set by the lifecycle manager inside the declared ranges.
- [ ] `curl /api/v1/gaps` → shape matches contract, all 6 dashboards present as keys.
- [ ] `curl /api/v1/dashboards/{id}` → shape matches contract for each of the 6 dashboards.
- [ ] All 6 UI routes render without console errors.
- [ ] Mission Control renders real starfield + satellite-node scene + real data + gap badges on gap/partial metrics.
- [ ] Gap registry decoder rejects malformed JSON and unknown keys (decode-error test); valid registry loads cleanly.
- [ ] `make test` → full suite passes (Go + vitest + BAS cases).
- [ ] `vrooli scenario restart command-center` → green end-to-end (Phase 6 final cleanup step); no orphan processes, health endpoints recover within the grace period.
- [ ] `vrooli scenario stop command-center` → clean shutdown; no orphan processes.
- [ ] Initiative dependency graph updates: `research/command-center-architecture` satisfied; `execute/command-center-theming-engine` and `execute/command-center-kiosk-ux` unblocked.

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| R3F + Three.js bundle size blows up cold-start on Xbox before code-splitting kicks in. | Xbox kiosk boot latency regression. | Per-page `React.lazy()` on scene modules. Measure shared bundle size in Phase 5. Real scene only lives on Mission Control route; the other 5 share a trivial scene. |
| Mission Control vertical slice pulls in data that isn't available yet (e.g., revenue). | Mission Control page fails to render. | Revenue is tagged as `gap` in the registry — it renders a gap badge, not a broken metric. All Mission Control metrics route through the same `metric_entry` contract whether live or gap. |
| LPBS client can't be integration-tested against real endpoints (don't exist yet). | LPBS code paths untested at merge. | Unit tests against `httptest` server returning 404/5xx/connection-refused; contract test confirming graceful fall-through to gap-mode. When LPBS endpoints ship, no code change needed — just point config at real URL. |
| Range-based port allocation collides with a sibling that's concurrently starting. | Intermittent `make start` failure. | Lifecycle manager already handles retry-on-collision within the range; if flakes emerge, the fallback is to pin a concrete port later (follow-up item), not in this scaffold. |
| Hooks lifted from sibling scenarios introduce imports or types that don't exist in `command-center/ui`. | UI build breaks. | Adapt hooks to use only template-available deps; no cross-scenario imports. Verify with `pnpm run type-check` per phase. |
| Scope creep into theming-engine or kiosk-ux territory. | Schedule slip; duplicated work when sibling items run. | Re-read Section 4 (Scope → Out of scope) at the start of every phase. If a task feels like polish rather than structural seam or the Mission Control vertical slice, push it to the sibling item. |
| Gap registry JSON content drifts under the trust-the-decoder model and silent miscounting creeps in. | Wrong gap reporting; downstream dashboards show stale or misclassified metrics. | The trade-off was accepted: only the file's author (us) writes this file in the scaffold. The decode-error test still catches shape bugs. If a future item needs semantic validation (enum ranges, duplicate-ID detection), it can introduce it via a follow-up without breaking the contract. |
| BAS/test-genie setup is unfamiliar; Phase 5 blows up. | Schedule slip. | Copy the `bas/` structure directly from `scenarios/scenario-completeness-scoring/`; only 1 action + 1 flow + 7 cases to author, all page-load-level. test-genie's playbooks phase handles orchestration. |
| Test-genie playbooks phase requires BAS runtime available at test time. | Flaky `make test`. | Verify BAS is up (or degrades gracefully with a skipped-phase warning) in the same way sibling scenarios handle it. If BAS is down, document the fallback (unit tests pass, playbooks skipped with warning). |
| `@vrooli/iframe-bridge` spatial nav dependency not actually shipped in the template version pinned. | kiosk-ux item blocked. | Phase 4 verifies `@vrooli/iframe-bridge` presence in `ui/package.json`; add it if missing. |
| Test flakiness from httptest upstream doubles running concurrently. | Flaky CI. | Use `t.Parallel()` only where clients are stateless; pin cache TTLs to short values in tests. |

## 12. Non-goals / Prohibited Patterns

- **No** full per-theme CSS, typography, or background treatments for the 5 non-Mission-Control themes — only palette stubs + the `data-theme` wrapper + ThemeProvider structure. Full theming is theming-engine scope.
- **No** auto-cycle, fade-through-black transitions, settings panel, hidden controls, D-pad input wiring, or auto-fullscreen-on-load invocation — kiosk-ux scope.
- **No** Gamepad API plumbing, `navigator.getGamepads()` polling, or keycode mapping. Use `@vrooli/iframe-bridge`'s `useSpatialNav` + `<SpatialGroup>` consumers when kiosk-ux runs.
- **No** new LPBS handlers in this item — separate execute item (`execute/lpbs-command-center-dashboard-endpoints`) owns that.
- **No** database (Postgres, SQLite, etc.) for command-center itself — purely a read-only aggregator. Any storage need signals a scope violation.
- **No** breaking changes to sibling scenarios' APIs — this item only consumes `/api/v1/*` endpoints, never modifies their contracts.
- **No** post-processing effects (Bloom, Vignette) baked into scaffold R3F wrapper — they're disabled in the reference prompt-manager setup for compatibility reasons.
- **No** live-metric visualization binding inside the Mission Control scene (no Recharts embedded in the R3F canvas, no data-driven mesh positions). Scene is starfield + satellites only. Metric visualizations are theming-engine scope.
- **No** cross-scenario imports (`import ... from "../../swarm-manager/..."`). Hooks are adapted (copied + modified), not imported.
- **No** Playwright or playwright-based fixtures — e2e is BAS/test-genie only, per round-1 direction. If a test case would be easier in Playwright, rewrite it as a BAS case or fold it into vitest.
- **No** hand-written registry validation passes (enum-range checks, duplicate-ID scans, required-field loops) and **no** external JSON-schema libraries. Decoder-level shape checks only.
- **No** hardcoded `port` values in `.vrooli/service.json` — ranges only, allocated by the lifecycle manager.
- **No** compatibility shims or legacy fallbacks — this is greenfield. If a sibling pattern is wrong, either adapt it on the way in or raise a separate item to fix the sibling.

## 13. Definition of Done

All of the following are satisfied:

- [ ] `scenarios/command-center/` exists with the layout in Section 5.
- [ ] `.vrooli/service.json` passes schema validation (`../../../../.vrooli/schemas/service.schema.json`); ports are **range-only** (API `15000-19999`, UI `35000-39999`); no hardcoded `port` values.
- [ ] `.vrooli/testing.json` sets `playbooks.enabled: true` and lists `ui` + `bas` in `structure.additional_dirs`.
- [ ] `make setup && vrooli scenario start command-center` produces a running API + UI with green health checks; `API_PORT` / `UI_PORT` are populated by the lifecycle manager.
- [ ] `curl /api/v1/health`, `/api/v1/dashboards/{id}` (all 6), `/api/v1/gaps`, `/api/v1/debug/r3f-stats` all return contract-correct payloads.
- [ ] Per-source cache enforces TTLs (Swarm 30s, Vrooli 60s, LPBS 5min) and falls back to last-cached data + `staleness_ts` on upstream error.
- [ ] Gap registry JSON loads at startup via `DisallowUnknownFields` decoder; malformed JSON exits non-zero; valid registry is trusted as authored; `/api/v1/gaps` output matches entries with `dataSource ∈ {gap, partial}`; all 6 dashboards populated with metrics from the orchestration summary.
- [ ] LPBS client is wired with Bearer token + gap-mode fallback on 404/5xx/connection-refused; unit-tested against an httptest double.
- [ ] Mission Control page renders the real Ground Control palette, the starfield + satellite-node R3F scene, real API data via react-query, and gap badges on gap/partial metrics. BAS case `mission-control-gap-badges-render.json` passes.
- [ ] All 6 UI routes render with the correct `data-theme` attribute on the layout wrapper; the 5 non-Mission-Control routes render a shared trivial-cube scene via `React.lazy()`.
- [ ] `useWakeLock` and `useFullscreen` hooks exist in `ui/src/hooks/` and pass vitest coverage. They are **not** auto-invoked.
- [ ] `bas/` contains 1 action, 1 flow, and 7 cases (6 route-load + 1 Mission Control gap badge); `bas/registry.json` is generated.
- [ ] `make test` passes with the full suite (Go + UI vitest + BAS cases via test-genie).
- [ ] `vrooli scenario completeness command-center` reports clean.
- [ ] `vrooli scenario restart command-center` completes green as the Phase 6 final cleanup step — no orphan processes from prior runs, health recovers within the grace period.
- [ ] Scope violations are zero: no code that belongs to theming-engine or kiosk-ux was shipped in this item (verify against Section 4 / 12).
- [ ] `execute/command-center-theming-engine` and `execute/command-center-kiosk-ux` can start their workshops against the scaffolded structure without requiring scaffold rework.
