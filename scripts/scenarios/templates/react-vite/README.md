# React + Vite Scenario Template (Go API + CLI)

Use this template to bootstrap every new scenario. It mirrors the patterns from `browser-automation-studio`:

- Go API (`api/`)
- React + TypeScript + Vite UI (`ui/`)
- CLI wrapper (`cli/`)
- Lifecycle + health wiring (`.vrooli/service.json`)
- Requirements registry + progress log (`requirements/`, `docs/PROGRESS.md`)

## Copy the Template
```bash
# From the repo root
vrooli scenario generate react-vite \
  --id <your-scenario> \
  --display-name "Your Scenario" \
  --description "One sentence summary"
cd scenarios/<your-scenario>/
```

Immediately replace placeholder tokens (scenario name, description, maintainer info, etc.).

## What You Get
- **Clean UI scaffold**: Vite + Tailwind + shadcn-style primitives, pnpm-based scripts, Vitest + Testing Library pre-configured, `.env.example` for API URL.
- **Go API skeleton**: `go.mod` + `cmd/server` entrypoint ready for feature modules.
- **CLI installer**: Installs into `~/.vrooli/bin` and updates shell rc files.
- **Lifecycle-ready service.json**: ports aligned with the platform, only Postgres required by default, lifecycle steps that build API/UI and start dev servers.
- **Iframe-ready UI**: Automatically initializes `@vrooli/iframe-bridge` so App Monitor and other hosts can embed the scenario without extra work.
- **Smart API resolution**: UI uses `@vrooli/api-base` to resolve the correct API + WebSocket URLs across localhost/dev/proxy contexts.
- **Requirements seed**: `requirements/index.json` + `requirements/modules/foundation.json` show how operational targets trace to technical requirements.
- **Lifecycle metadata seed**: `.vrooli/service.json`, `endpoints.json`, `testing.json`, and `lighthouse.json` so status/health/testing commands work immediately after copy.
- **Progress log**: `docs/PROGRESS.md` so improvers track deltas outside PRD.md.
- **Database placeholder**: `initialization/storage/postgres/seed.sql` to remind agents where to place migrations/seeds without shipping fake data.

## Setup Workflow
```bash
cd scenarios/<your-scenario>

# Install dependencies (Go + pnpm must be available)
pnpm install --dir ui

# Build API + UI + CLI via lifecycle
vrooli scenario run <your-scenario> --setup

# Start dev servers (API + Vite)
make dev   # wraps `vrooli scenario run --dev`
```

Run tests with `make test` or `test-genie execute --scenario <your-scenario>`.

### UI Smoke Harness

- `vrooli scenario ui-smoke <your-scenario>` launches a Browserless session against the production UI bundle, waits for `@vrooli/iframe-bridge` to signal readiness, captures a screenshot, HTML snapshot, console logs, and network trace.
- Artifacts live under `coverage/ui-smoke/` (screenshot, console.json, network.json, dom.html, raw.json) and the latest summary is stored at `coverage/ui-smoke/latest.json`.
- Structure tests invoke the harness automatically. Disable it temporarily by toggling `structure.ui_smoke.enabled` in `.vrooli/testing.json` or extend the default timeouts via `timeout_ms` / `handshake_timeout_ms`.
- Browserless must be running (`resource-browserless status --format json`). If it is offline the harness fails early so issues surface before release.

## Required Environment Variables
The lifecycle exports everything automatically when you run `vrooli scenario run`. If you start pieces manually, set these yourself (there are no fallbacks):

| Variable | Purpose |
|----------|---------|
| `API_PORT` | Port assigned to the Go API server |
| `UI_PORT` | Port assigned to the Vite dev server / production UI |
| `WS_PORT` | WebSocket channel for live updates |
| `DATABASE_URL` *or* `POSTGRES_HOST/PORT/USER/PASSWORD/DB` | PostgreSQL connection details |
| `N8N_BASE_URL` | Base URL for workflow automation calls |
| `UI_BASE_URL` | Base URL for the Vrooli UI shell / iframe bridge |
| `API_TOKEN` | Shared secret the CLI/API uses for authentication |
| `VITE_API_BASE_URL` | UI → API bridge (set to `http://localhost:${API_PORT}/api/v1`) |

> Tip: when running outside the lifecycle, fetch ports with `vrooli scenario port <name> API_PORT` (or `UI_PORT`) and then export `VITE_API_BASE_URL` accordingly:

```bash
API_PORT=$(vrooli scenario port <name> API_PORT)
UI_PORT=$(vrooli scenario port <name> UI_PORT)
cd ui && VITE_API_BASE_URL="http://localhost:${API_PORT}/api/v1" pnpm run dev -- --host --port "$UI_PORT"
```

## Iframe Bridge + API Base
- `src/main.tsx` initializes `@vrooli/iframe-bridge` automatically whenever the UI is rendered inside App Monitor or another host.
- All API calls go through `@vrooli/api-base`, which means the UI works no matter where it’s served (localhost dev server, Cloudflare tunnel, proxied iframe, production ingress). Just keep `VITE_API_BASE_URL` pointed at `http://localhost:${API_PORT}/api/v1` during local work.

## CLI Auto-Detection
- The CLI writes `~/.{{SCENARIO_ID}}/config.json` on first run. Leave `api_base` blank to let it auto-detect the correct URL from `vrooli scenario port` output.
- If you point the API at a remote host, run `{{SCENARIO_ID}} configure api_base https://api.example.com/v1` to override detection.
- The CLI requires the scenario to be running through the lifecycle; otherwise it will warn that it cannot discover the API.

## Customize Safely
1. **Update PRD.md + requirements/** first. Operational targets drive code + tests.
2. **Append progress entries** to `docs/PROGRESS.md` whenever you land work.
3. **Add resources** in `.vrooli/service.json` only when needed; Postgres is the sole default.
4. **Keep boundaries**: only edit within `scenarios/<your-scenario>/`.

## pnpm Everywhere
The template assumes pnpm. If you run another package manager, convert lockfiles yourself before committing. Scripts use `pnpm` directly (no `npm` fallbacks) to reduce drift.

## Need Inspiration?
Open `scenarios/browser-automation-studio/` to see this template taken to completion.
