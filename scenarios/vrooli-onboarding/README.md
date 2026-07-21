# Vrooli Onboarding

Central configuration hub guiding users through Vrooli resource setup with guided flows, secret validation, and health visualization.

> **Configuration substrate is documented at [`/docs/configuration/`](../../docs/configuration/).** That folder is the canonical contract this scenario implements; new configurability lands as a doc page first, then becomes a wizard step. The v2 rework flow and wireframes are in [`docs/WIZARD_FLOW.md`](docs/WIZARD_FLOW.md).

## Architecture

- **Go API** (`api/`) - REST endpoints for resource discovery, config generation/validation, and progress persistence
- **React + TypeScript + Vite UI** (`ui/`) - Onboarding wizard. Current shipped flow: Welcome → Select Resources → Review → Complete (4 steps). V2 rework planned: Scenarios → Resources → Secrets → Integrations → Host → Operating-mode → Validation (see [`docs/WIZARD_FLOW.md`](docs/WIZARD_FLOW.md))
- **CLI** (`cli/`) - Command-line interface for health checks and configuration
- **Lifecycle + health wiring** (`.vrooli/service.json`)
- **Requirements registry + progress log** (`requirements/`, `docs/PROGRESS.md`)

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/resources` | List all available resources with status and category |
| GET | `/api/v1/resources/{name}` | Get details for a single resource |
| GET | `/api/v1/progress` | Get onboarding progress |
| PUT | `/api/v1/progress` | Save/update onboarding progress |
| POST | `/api/v1/config/generate` | Generate service.json from selected resources |
| POST | `/api/v1/config/validate` | Validate config against resource dependencies |

## What You Get
- **Clean UI scaffold**: Vite + Tailwind + shadcn-style primitives, pnpm-based scripts, Vitest + Testing Library pre-configured, `.env.example` for API URL.
- **Go API skeleton**: `go.mod` + `cmd/server` entrypoint ready for feature modules.
- **CLI installer**: Installs into `~/.vrooli/bin` by default (prints PATH guidance if needed).
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

# Install dependencies (Go 1.22+ + pnpm must be available)
corepack pnpm install --dir ui --ignore-workspace

# Build API + UI + CLI via lifecycle
vrooli scenario run <your-scenario> --setup

# Start dev servers (API + Vite)
make dev   # wraps `vrooli scenario run --dev`
```

Run tests with `make test` or `test-genie execute --scenario <your-scenario>`.

### UI Smoke Harness

- `vrooli scenario ui-smoke <your-scenario>` launches a browser smoke-test session (BAS/Playwright) against the production UI bundle, waits for `@vrooli/iframe-bridge` to signal readiness, captures a screenshot, HTML snapshot, console logs, and network trace.
- Artifacts live under `coverage/ui-smoke/` (screenshot, console.json, network.json, dom.html, raw.json) and the latest summary is stored at `coverage/ui-smoke/latest.json`.
- Structure tests invoke the harness automatically. Disable it temporarily by toggling `structure.ui_smoke.enabled` in `.vrooli/testing.json` or extend the default timeouts via `timeout_ms` / `handshake_timeout_ms`.
- browser-automation-studio must be running (`browser-automation-studio status --format json`). If it is offline the harness fails early so issues surface before release.

## Required Environment Variables
The lifecycle exports everything automatically when you run `vrooli scenario run`. If you start pieces manually, set these yourself (there are no fallbacks):

| Variable | Purpose |
|----------|---------|
| `API_PORT` | Port assigned to the Go API server |
| `UI_PORT` | Port assigned to the Vite dev server / production UI |
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
- The control plane installs the declared Go CLI into `~/.vrooli/bin` when the scenario starts.
- The shared onboarding lifecycle state is stored in your user config directory at `~/.config/vrooli/config.json`.
- Run `vrooli-onboarding configure api_base http://localhost:<API_PORT>/api/v1` (and optionally `vrooli-onboarding configure token <token>`) to point at a remote or non-standard API.

## Customize Safely
1. **Update PRD.md + requirements/** first. Operational targets drive code + tests.
2. **Append progress entries** to `docs/PROGRESS.md` whenever you land work.
3. **Add resources** in `.vrooli/service.json` only when needed; Postgres is the sole default.
4. **Keep boundaries**: only edit within `scenarios/<your-scenario>/`.

## pnpm Everywhere
The template assumes pnpm. If you run another package manager, convert lockfiles yourself before committing. Scripts use `pnpm` directly (no `npm` fallbacks) to reduce drift.

## Need Inspiration?
Open `scenarios/browser-automation-studio/` to see this template taken to completion.
