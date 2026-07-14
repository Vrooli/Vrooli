# Command Center — Architecture

Command Center is a **read-only aggregator** that composes kiosk-style dashboard payloads from three upstream Vrooli data sources: **Swarm Manager**, **Vrooli Core**, and **LPBS** (Landing Page Business Suite). It ships a Go API and a React + Vite UI.

This scaffold lands the structure. Two sibling execute items extend it:
- `command-center-theming-engine` — fills in the 5 non-Mission-Control themes and scenes.
- `command-center-kiosk-ux` — adds auto-cycle, fade transitions, hidden settings, D-pad input, and fullscreen-on-load.

## Directory layout

```
scenarios/command-center/
├── .vrooli/
│   ├── service.json       # lifecycle v2.0; port ranges only (API 15000-19999, UI 35000-39999)
│   └── testing.json       # playbooks enabled; ui + bas in additional_dirs
├── api/                   # Go API server (flat `package main`, per-domain handler files)
├── bas/                   # test-genie / browser-automation-studio cases
├── config/
│   └── gap-registry.json  # loaded at API startup; 6 dashboards × N metrics
├── docs/
│   └── ARCHITECTURE.md    # this file
├── ui/                    # React + Vite + R3F client
├── Makefile               # delegates to `vrooli scenario ...`
└── README.md
```

## API

The API is a plain Go HTTP server built on gorilla/mux. It serves:

| Path | Purpose |
|---|---|
| `GET /health`, `GET /api/v1/health` | Lifecycle health check. |
| `GET /api/v1/dashboards/{id}` | Per-dashboard payload: registry metrics + per-source cache metadata. |
| `GET /api/v1/gaps` | All `gap`/`partial` entries grouped by dashboard. |
| `GET /api/v1/debug/r3f-stats` | Read R3F performance events. |
| `POST /api/v1/debug/r3f-stats` | Append a single R3F event. |

### Gap registry (`config/gap-registry.json`)

Loaded once at startup via a `json.Decoder` with `DisallowUnknownFields()`. Malformed JSON or unknown keys cause the process to exit non-zero. Valid files are trusted as authored — there is no semantic enum-range or duplicate-ID validation pass (see `plan.md` §8 for rationale).

Each entry has:
- `id`, `label`, `dataSource` (`live`/`partial`/`gap`), `upstreamSource` (`swarm`/`vrooli`/`lpbs`/`none`), optional `description`, optional `whatIsNeeded`.

### Per-source cache

`cache.go` implements a map keyed by `"<source>:<path>"`. TTLs:

| Source | TTL |
|---|---|
| Swarm | 30 s |
| Vrooli | 60 s |
| LPBS | 5 min |

On upstream error, the last-good entry is returned with its `StalenessTS` set so the UI can surface a `<StaleBadge>`. If there has never been a successful fetch, the handler falls back to gap-mode data from the registry.

### Upstream clients (`api/upstream/`)

Uniform `Client` interface with `Name()` and `Fetch(ctx, path)`. Three concrete clients:

- **`NewSwarm(baseURL)`** — plain JSON GET against the Swarm Manager API.
- **`NewVrooli(baseURL)`** — plain JSON GET against `http://localhost:8092/scenarios` by default.
- **`NewLPBS(baseURL, bearerToken)`** — adds `Authorization: Bearer <token>` when token is non-empty. **On 404 it returns `ErrNotAvailable`** so the scaffold is self-contained while the LPBS `/api/v1/admin/dashboard/*` endpoints are being shipped by `execute/lpbs-command-center-dashboard-endpoints`.

Environment variables consumed by the server:
- `API_PORT` (lifecycle-assigned; range 15000-19999)
- `SWARM_MANAGER_API_PORT`, `SWARM_MANAGER_BASE_URL` (optional)
- `VROOLI_CORE_BASE_URL` (default `http://localhost:8092`)
- `LPBS_API_PORT`, `LPBS_BASE_URL` (optional — when empty, LPBS client returns `ErrNotAvailable`)
- `LPBS_SERVICE_TOKEN` (bearer token for LPBS admin endpoints)
- `COMMAND_CENTER_REGISTRY_PATH` (test override)

### R3F stats buffer (`api/stats.go`)

In-memory ring buffer, capacity 1024 events, TTL 1 h on insert-time pruning. `GET` returns newest-first. **No auth** — this is a local-only debug surface.

## UI

React + Vite with React Router. Six routes, one per dashboard:

| Route | Dashboard ID | Theme key |
|---|---|---|
| `/mission-control` | `mission-control` | `ground-control` |
| `/hive` | `hive` | `bioluminescent` |
| `/forge` | `forge` | `foundry` |
| `/ledger` | `ledger` | `vault` |
| `/broadcast` | `broadcast` | `signal-tower` |
| `/panorama` | `panorama` | `cosmos` |

Each route wraps its content in a `<DashboardLayout themeKey={...}>`. `ThemeProvider` loads the theme's CSS module and sets `data-theme` on `<html>`, so descendant components read palette/typography via CSS custom properties (`--cc-bg`, `--cc-accent`, …).

### Themes (`ui/src/themes/`)

- `ground-control.css` — **fully populated** (electric blue, monospace, space aesthetic). This is the Mission Control theme.
- `bioluminescent.css`, `foundry.css`, `vault.css`, `signal-tower.css`, `cosmos.css` — palette-only stubs, each with a distinct `--cc-accent`. **`command-center-theming-engine` fills these in.**

### Scenes (`ui/src/scenes/`)

All scenes are lazy-loaded so the R3F/Three.js bundle is code-split per route.

- `missionControl.tsx` — **real** scene: instanced-points starfield + 3-5 rotating satellite node meshes using the Ground Control palette. No metric-bound visuals.
- `trivialCube.tsx` — shared placeholder used by the other 5 routes; a single rotating cube with the theme's accent color. **`command-center-theming-engine` replaces this per route.**

### Mission Control vertical slice

`pages/MissionControl.tsx` is the full end-to-end path:

1. `fetchDashboard('mission-control')` via `@tanstack/react-query`.
2. `<DashboardLayout themeKey="ground-control">` applies Ground Control palette.
3. `<SceneCanvas>` lazy-mounts `missionControl.tsx` (starfield + satellites).
4. `<MetricList>` renders metric rows with `<GapBadge>` on `gap`/`partial` entries.
5. `<StaleBadge>` surfaces per-source `staleness_ts` from the API response.

### Hooks (`ui/src/hooks/`)

- `useWakeLock.ts` (adapted from `scenarios/web-console`) — **not auto-invoked**.
- `useFullscreen.ts` — **not auto-invoked**.

Both are wired into the policy layer by `command-center-kiosk-ux`.

## Lifecycle

`.vrooli/service.json` declares **port ranges only**:

- `API_PORT` → `15000-19999`
- `UI_PORT` → `35000-39999`

The lifecycle manager picks free ports at `vrooli scenario start` and exposes them via env vars. The Go API and Vite dev server (`server.js`) both read from env — never hardcode.

Health endpoints:
- API: `http://localhost:${API_PORT}/health`
- UI: `http://localhost:${UI_PORT}/health`

## Testing

| Surface | Location | Runner |
|---|---|---|
| Go unit + integration | `api/**/*_test.go` | `go test ./...` |
| UI component + hook | `ui/src/**/*.test.tsx` | `vitest run` |
| Route smoke (all 6) | `bas/cases/01-foundation/*-loads.json` | `test-genie` playbooks |
| Mission Control end-to-end | `bas/cases/01-foundation/mission-control-gap-badges-render.json` | `test-genie` playbooks |

`.vrooli/testing.json` enables `playbooks: true` and lists `ui` + `bas` in `structure.additional_dirs` so the test harness discovers all three surfaces.

## What sibling items land on

- **`execute/command-center-theming-engine`** fills in the 5 stub themes (full palette, typography, backgrounds) and swaps the trivial-cube scene for per-theme scenes on 5 routes. The `ThemeProvider` and `SceneCanvas` seams are already in place.
- **`execute/command-center-kiosk-ux`** wires `useWakeLock` + `useFullscreen`, implements auto-cycle transitions, the hidden settings panel, and D-pad input. The hooks are already adapted into `ui/src/hooks/` — kiosk-ux owns the policy, not the primitives.
- **`execute/lpbs-command-center-dashboard-endpoints`** ships LPBS `/api/v1/admin/dashboard/*`. The scaffold's LPBS client is already wired with Bearer-token plumbing and returns `ErrNotAvailable` on 404 so no code change is needed on the command-center side when those endpoints land.
