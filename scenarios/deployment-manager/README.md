# Deployment Manager

**Control tower for scenario deployments across all platform tiers**

deployment-manager analyzes dependencies, scores platform fitness, guides users through portable deployments, orchestrates platform-specific packagers (scenario-to-desktop, scenario-to-ios, etc.), and monitors deployed scenarios across Tier 1 (local dev) through Tier 5 (enterprise hardware).

## Purpose

**Without deployment-manager**: Scenarios are trapped in Tier 1 (local dev stack). Attempts to package scenarios fail because dependencies aren't portable (postgres doesn't run on mobile, ollama is too heavy for desktop bundles, cloud deployments lack secret strategies).

**With deployment-manager**: Any scenario can target any tier. The system analyzes dependencies, scores fitness for each tier, suggests swaps (postgres → sqlite for mobile), validates configurations, orchestrates packaging via scenario-to-* tools, and monitors health across all deployments — all through an intuitive UI and CLI.

## Key Features (See PRD.md for full operational targets)

**Core Deployment Lifecycle (P0 - 37 operational targets)**:
- **Dependency Analysis**: Recursive dep fetching, circular dependency detection, resource/scenario aggregation
- **Fitness Scoring**: 0-100 scores for each of 5 deployment tiers with breakdown (portability, resources, licensing, platform support)
- **Dependency Swapping**: Suggest alternatives for blockers (postgres → sqlite), impact analysis, non-destructive profile-only swaps
- **Profile Management**: CRUD, versioning, export/import (JSON), auto-save
- **Secret Management**: Integration with secrets-manager for tier-specific secret strategies (.env templates for desktop, Vault refs for cloud)
- **Pre-Deployment Validation**: 6+ checks (fitness threshold, secrets, licensing, resource limits), cost estimation for SaaS/cloud
- **Deployment Orchestration**: One-click deploy, real-time log streaming (WebSocket), calls scenario-to-desktop/ios/android/saas packagers
- **Dependency Visualization**: Interactive React Flow graph with fitness color-coding, table view fallback for accessibility

**Enhanced Features (P1 - 36 operational targets)**:
- Post-deployment health monitoring, metrics collection, alerting
- Update/rollback automation with zero-downtime for SaaS
- Multi-tier deployment orchestration (desktop + iOS + SaaS simultaneously)
- AI agent integration for migration strategy suggestions (via claude-code/ollama)

**Advanced Features (P2 - 26 operational targets)**:
- Enterprise compliance (audit logs, approval workflows, license validation)
- CLI automation for CI/CD pipelines
- Visual dependency graph editor, deployment templates library

## Quick Start

```bash
# Setup (build API, install CLI, initialize postgres, build UI)
vrooli scenario setup deployment-manager

# Start services (API + UI)
vrooli scenario start deployment-manager
# OR: cd scenarios/deployment-manager && make start

# Access UI
# Opens at http://localhost:{UI_PORT} (port assigned by lifecycle system)

# CLI usage
deployment-manager --help
deployment-manager analyze picker-wheel
deployment-manager fitness picker-wheel --tier 2  # Desktop tier
deployment-manager profiles list

# Run tests
make test  # All phases: dependencies, structure, CLI, API, UI
```

## Architecture

**Stack**:
- **API**: Go (orchestration, fitness scoring, profile management)
- **UI**: React + TypeScript + Vite + TailwindCSS + shadcn + React Flow (graphs) + Recharts (metrics)
- **CLI**: Bash wrapper calling API endpoints
- **Storage**: PostgreSQL (profiles, audit logs, fitness rules, swap database)
- **Caching**: Redis (optional, for fitness score caching)
- **Real-time**: WebSocket (deployment log streaming)

**Dependencies**:
- **scenario-dependency-analyzer** (critical): Dependency tree data source
- **secrets-manager** (critical): Secret classification and template generation
- **app-issue-tracker** (critical): Migration task creation when swaps approved
- **claude-code or ollama** (optional): AI-powered migration strategy suggestions
- **At least one scenario-to-* packager** (required for orchestration validation): e.g., scenario-to-extension, scenario-to-desktop

## Deployment Tiers

| Tier | Name | Description | deployment-manager's Role |
|------|------|-------------|---------------------------|
| 1 | Local/Dev Stack | Full Vrooli + app-monitor tunnel | Baseline reference (fitness = 100) |
| 2 | Desktop | Windows/macOS/Linux standalone apps | Score fitness, suggest swaps (postgres → sqlite), orchestrate scenario-to-desktop |
| 3 | Mobile | iOS/Android native apps | Score fitness, suggest swaps (heavy deps → cloud APIs), orchestrate scenario-to-ios/android |
| 4 | SaaS/Cloud | DigitalOcean/AWS/bare metal | Score fitness, estimate costs, orchestrate scenario-to-saas |
| 5 | Enterprise | Hardware appliances with compliance | Validate licensing, enforce approval workflows, orchestrate scenario-to-enterprise |

See `/docs/deployment/README.md` for full tier documentation.

## Documentation

- **PRD.md**: Full operational targets (99 OTs across P0/P1/P2)
- **requirements/**: Requirements registry (index.json + 14 modules mapping OTs to testable requirements)
- **docs/RESEARCH.md**: Uniqueness check, integration points, external references
- **docs/PROGRESS.md**: Implementation progress log
- **docs/PROBLEMS.md**: Known issues, blockers, deferred decisions
- **/docs/deployment/**: Deployment hub (tier system, fitness scoring, dependency swapping guides)

## Status: Initialization Phase

**Current State**: Scaffolded from react-vite template, PRD seeded, requirements registry initialized (99 requirements), .vrooli configuration complete.

**Next Steps for Improvers**:
1. Implement P0 Module 01 (Dependency Analysis) - integrate with scenario-dependency-analyzer API
2. Implement P0 Module 02 (Dependency Swapping) - create swap database and suggestion engine
3. Implement P0 Module 03 (Profile Management) - postgres schema + CRUD API
4. See `requirements/README.md` for full implementation priority and critical path

**Progress Tracking**: All progress logs go in `docs/PROGRESS.md` (not PRD.md). PRD checkboxes auto-flip via requirement sync.
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
- The CLI writes `~/.deployment-manager/config.json` on first run. Leave `api_base` blank to let it auto-detect the correct URL from `vrooli scenario port` output.
- If you point the API at a remote host, run `deployment-manager configure api_base https://api.example.com/v1` to override detection.
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
