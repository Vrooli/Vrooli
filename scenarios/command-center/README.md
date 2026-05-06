# Command Center

Read-only kiosk-style aggregator that composes dashboard payloads from Swarm Manager, Vrooli Core, and LPBS, and renders six themed displays intended for an always-on TV or Xbox browser.

The canonical UI design contract is [DESIGN.md](DESIGN.md). Future UI work should follow `vrooli-command-display`, not the default operational-console app shell.

## Architecture

- **Go API (`api/`)** — HTTP aggregator with per-source TTL cache, gap registry loader, and four public endpoints:
  - `GET /api/v1/health`
  - `GET /api/v1/dashboards/{id}` — composed payload for one dashboard
  - `GET /api/v1/gaps` — all metrics flagged as `gap` or `partial`, grouped by dashboard
  - `GET|POST /api/v1/debug/r3f-stats` — R3F performance telemetry ring buffer
- **React + Vite UI (`ui/`)** — router with six themed dashboard pages (Mission Control, The Hive, The Forge, Ledger, Broadcast, Panorama). Mission Control is a full vertical slice; the other five are themed placeholders that sibling execute items fill in.
- **Gap registry (`config/gap-registry.json`)** — file-based JSON loaded at API startup. Source of truth for which metrics are `live`, `partial`, or `gap`.
- **BAS cases (`bas/`)** — page-load and structural assertions driven by test-genie playbooks.

Sibling execute items building on this scaffold:
- `execute/command-center-theming-engine` — replaces placeholder scenes + fills out the remaining five themes.
- `execute/command-center-kiosk-ux` — auto-cycle, D-pad spatial nav, fullscreen-on-load, hidden controls.
- `execute/lpbs-command-center-dashboard-endpoints` — lands LPBS `/api/v1/admin/dashboard/*` endpoints the scaffold's LPBS client already expects.

## Quick Start

```bash
make setup                              # build API + install UI deps + build UI bundle
vrooli scenario start command-center    # launch API + UI via lifecycle manager
make status                             # inspect running ports / PIDs
make logs                               # tail combined logs
make test                               # Go unit + vitest + BAS cases
make restart                            # clean stop + start
make stop                               # shut everything down
```

The lifecycle manager allocates ports from the ranges declared in `.vrooli/service.json` (API `15000-19999`, UI `35000-39999`) and exposes them via `API_PORT` / `UI_PORT` env vars. The Go server and Vite preview server both read their port from the env var — never a hardcoded value.

## Upstream Sources

| Source | Endpoint | Default TTL |
| --- | --- | --- |
| Swarm Manager | `GET /api/v1/stats`, `/api/v1/overview` | 30s |
| Vrooli Core | `GET http://localhost:8092/scenarios` | 60s |
| LPBS | `GET /api/v1/admin/dashboard/*` (`Authorization: Bearer ${LPBS_SERVICE_TOKEN}`) | 300s |

On upstream error the handler returns the last successful payload with a `staleness_ts` field. When there is no cached payload available, it falls through to gap-mode data from the registry.

## Further Reading

- `docs/ARCHITECTURE.md` — cache, registry, theming seams, Mission Control reference slice.
- `DESIGN.md` — fullscreen command-display design language for kiosk, TV, and war-room surfaces.
- `../swarm-manager/research/command-center-architecture/conclusion.md` — 17 research findings that shaped this scaffold.
- `../swarm-manager/initiatives/command-center-foundation/orchestration-summary.md` — brainstorming context and upstream data-source inventory.
