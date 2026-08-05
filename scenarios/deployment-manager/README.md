# Deployment Manager

**The governance plane for scenario deployments**

deployment-manager decides whether a build may ship, and records what shipped. It analyzes dependencies, scores platform fitness, holds deployment profiles, gates releases on recorded evidence, and keeps the release history. It is tier-agnostic by design: it never learns what a `.dmg` or an `.ipa` is.

## Purpose

**Without deployment-manager**: Scenarios are trapped in Tier 1 (local dev stack). Attempts to package scenarios fail because dependencies aren't portable (postgres doesn't run on mobile, ollama is too heavy for desktop bundles, cloud deployments lack secret strategies). Nothing records what was approved, so nothing can prove what shipped.

**With deployment-manager**: Any scenario can target any tier. The system analyzes dependencies, scores fitness for each tier, suggests swaps (postgres → sqlite for mobile), validates configurations, and holds the approval gate that a packaging ramp must pass before it publishes.

## Boundaries

deployment-manager is one of four planes. See [Deployment Hub](../../docs/deployment/README.md) for the full model.

| deployment-manager owns | A `scenario-to-*` ramp owns |
|---|---|
| Deployment profiles and versions | Build, package, sign, publish |
| Fitness scoring and dependency analysis | Target-specific runtime behavior |
| Approval gates and release records | Running the artifact on its targets |
| Release channels and promotion | Producing evidence from those runs |

**Direction of control**: a ramp calls deployment-manager. deployment-manager does not drive a ramp's pipeline. `scenario-to-desktop` calls three endpoints — create an approval, read the release gate, and generate a bundle manifest — and publishes only when the gate allows it.

**Not implemented yet**: the generic deploy endpoints (`POST /api/v1/deploy/{profile_id}` and `GET /api/v1/deployments/{deployment_id}`) do not orchestrate or track anything. Use the ramp directly. Cross-tier evidence review is design direction, not a current capability.

## Key Features (See PRD.md for full operational targets)

**Core Deployment Lifecycle (P0 - 39 operational targets)**:
- **Dependency Analysis**: Recursive dep fetching, circular dependency detection, resource/scenario aggregation
- **Fitness Scoring**: 0-100 scores for each of 5 deployment tiers with breakdown (portability, resources, licensing, platform support)
- **Dependency Swapping**: Suggest alternatives for blockers (postgres → sqlite), impact analysis, non-destructive profile-only swaps
- **Profile Management**: CRUD, versioning, export/import (JSON), auto-save
- **Secret Management**: Integration with secrets-manager for tier-specific secret strategies (.env templates for desktop, Vault refs for cloud)
- **Pre-Deployment Validation**: 6+ checks (fitness threshold, secrets, licensing, resource limits), cost estimation for SaaS/cloud
- **Deployment Orchestration**: One-click deploy, real-time log streaming (WebSocket), calls scenario-to-desktop/ios/android/saas packagers
- **Dependency Visualization**: Interactive React Flow graph with fitness color-coding, table view fallback for accessibility
- **Evidence-Backed Release Governance**: Validates the shared proto-first evidence contract and requires decision-grade desktop evidence before approval

**Enhanced Features (P1 - 36 operational targets)**:
- Post-deployment health monitoring, metrics collection, alerting
- Update/rollback automation with zero-downtime for SaaS
- Multi-tier deployment orchestration (desktop + iOS + SaaS simultaneously)
- AI agent integration for migration strategy suggestions (via claude-code/ollama)

**Advanced Features (P2 - 29 operational targets)**:
- Enterprise compliance (audit logs, approval workflows, license validation)
- CLI automation for CI/CD pipelines
- Visual dependency graph editor, deployment templates library

## Quick Start: Desktop Deployment

Deploy any scenario as a standalone Windows/macOS/Linux desktop app:

```bash
# 1. Create a deployment profile for your scenario
deployment-manager profile create my-profile my-scenario --tier 2

# 2. Build everything (manifest, binaries, Electron wrapper, installers)
deployment-manager deploy-desktop --profile my-profile

# Output: installers in scenarios/<scenario>/platforms/electron/dist-electron/
```

For complete walkthrough, see **[Desktop Deployment Guide](docs/DESKTOP-DEPLOYMENT-GUIDE.md)** or the **[Hello Desktop Tutorial](docs/tutorials/hello-desktop-walkthrough.md)**.

---

## Quick Start

```bash
# Setup (build API, install CLI, initialize SQLite schemas, build UI)
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

> The CLI is now implemented in Go using `packages/cli-core` for consistent cross-platform behavior. Install with `./packages/cli-core/install.sh scenarios/deployment-manager/cli --name deployment-manager` (or run the bundled `install.sh`). The legacy Bash CLI is preserved at `scenarios/deployment-manager/cli-old/` for reference only.

# Run tests
make test  # All phases: dependencies, structure, CLI, API, UI

## CLI cheat sheet (agent-friendly)
- Global output: prefix any command with `--json` or `--format table` (consumed once, applies to nested commands).
- Discovery: `deployment-manager status`, `deployment-manager analyze <scenario>`, `deployment-manager fitness <scenario> --tier 3`.
- Profiles: `profiles` (list), `profile create <name> <scenario> --tier <n>`, `profile export <id> --output /path`, `profile diff <id>`, `profile rollback <id> --version <n>`.
- Swaps: `swaps list <scenario>`, `swaps analyze <from> <to>`, `swaps apply <profile> <from> <to> --show-fitness`.
- Deployments: `deploy <profile> --dry-run`, `deploy-desktop --profile <id> --dry-run`, `validate <profile> --verbose`, `estimate-cost <profile> --verbose`, `package <profile> --packager <scenario-to-*>` (legacy stub), `logs <profile> --level error --format table`.
- Secrets: `secrets identify <profile>`, `secrets template <profile> --format env`, `secrets validate <profile>`.
```

## Deployment Workflows

For complete step-by-step deployment guides with exact commands and expected outputs:

| Workflow | Description | Guide |
|----------|-------------|-------|
| **Quick Start** | Deploy your first scenario in 5 minutes | [Quick Start](docs/guides/quick-start.md) |
| **Desktop (Tier 2)** | Deploy as Windows/macOS/Linux app | [Desktop Deployment](docs/workflows/desktop-deployment.md) |
| **Mobile (Tier 3)** | Deploy as iOS/Android app | [Mobile Deployment](docs/workflows/mobile-deployment.md) |
| **SaaS (Tier 4)** | Deploy to cloud infrastructure | [SaaS Deployment](docs/workflows/saas-deployment.md) |
| **Troubleshooting** | Common issues and solutions | [Troubleshooting](docs/workflows/troubleshooting.md) |

See the [documentation hub](docs/README.md) for the maintained index by task, component, and tier.

## Architecture

**Stack**:
- **API**: Go (orchestration, fitness scoring, profile management)
- **UI**: React + TypeScript + Vite + TailwindCSS + shadcn + React Flow (graphs) + Recharts (metrics)
- **CLI**: Go + `packages/cli-core` (cross-platform binary; auto-discovers API base via lifecycle)
- **Storage**: Per-domain SQLite through `modernc.org/sqlite` and the routed database boundary
- **Caching**: Redis (optional, for fitness score caching)
- **Real-time**: WebSocket (deployment log streaming)

**Dependencies**:
- **scenario-dependency-analyzer** (critical): Dependency tree data source
- **secrets-manager** (critical): Secret classification and template generation
- **app-issue-tracker** (critical): Migration task creation when swaps approved
- **scenario-to-desktop** (initial evidence producer): Decision-grade desktop journey and recording references
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

See [docs/README.md](docs/README.md) for full tier documentation.

## Documentation

- **[docs/README.md](docs/README.md)**: Deployment documentation hub (authoritative source)
- **[CLI reference](docs/cli/overview-commands.md)**: CLI command reference
- **[API reference](docs/api/bundles.md)**: representative API contract
- **[Workflows](docs/workflows/desktop-deployment.md)**: step-by-step deployment guide
- **[Guides](docs/guides/evidence-contract.md)**: technical deep-dives and release evidence
- **[Tier reference](docs/tiers/tier-2-desktop.md)**: desktop target requirements
- **PRD.md**: Full operational targets (104 OTs across P0/P1/P2)
- **requirements/**: Requirements registry (index.json + 15 modules mapping OTs to testable requirements)
- **docs/RESEARCH.md**: Uniqueness check, integration points, external references
- **docs/PROGRESS.md**: Implementation progress log
- **[docs/internal/PROBLEMS.md](docs/internal/PROBLEMS.md)**: Known issues, blockers, deferred decisions

## Status: Active governance plane

**Current State**: The proto-first governance plane is implemented and exercised through the shared evidence contract. The deployment-manager suite passes all 19 phases; the scenario-to-desktop suite passes all 21 phases; aggregate API, CLI, and UI coverage gates are met. Release evidence is stored as producer-owned references, not copied artifact bytes.

**Storage validation**: `storage-manager validate scenario deployment-manager --json` is the active storage-validation contract. It is runnable in the current environment; its status and any non-blocking findings are recorded as provider-owned validation output. No waiver or lowered threshold is used.

**Progress Tracking**: All progress logs go in `docs/PROGRESS.md` (not PRD.md). PRD checkboxes auto-flip via requirement sync.
- Artifacts live under `coverage/ui-smoke/` (screenshot, console.json, network.json, dom.html, raw.json) and the latest summary is stored at `coverage/ui-smoke/latest.json`.
- Structure tests invoke the harness automatically. Disable it temporarily by toggling `structure.ui_smoke.enabled` in `.vrooli/testing.json` or extend the default timeouts via `timeout_ms` / `handshake_timeout_ms`.
- Browser-automation-studio must be running (`browser-automation-studio status --json`). If it is offline the harness fails early so issues surface before release.

## Required Environment Variables
The lifecycle exports everything automatically when you run `vrooli scenario run`. If you start pieces manually, set these yourself (there are no fallbacks):

| Variable | Purpose |
|----------|---------|
| `API_PORT` | Port assigned to the Go API server |
| `UI_PORT` | Port assigned to the Vite dev server / production UI |
| `SQLITE_PATH` | Scenario-owned SQLite database path; lifecycle sets this under the scenario data directory |
| `VITE_API_BASE_URL` | UI → API bridge (set to `http://localhost:${API_PORT}/api/v1`) |

Optional service overrides are `SCENARIO_DEPENDENCY_ANALYZER_URL`,
`SECRETS_MANAGER_URL`, `SCENARIO_TO_DESKTOP_URL`, `SCENARIO_TO_CLOUD_URL`,
`LPBS_BASE_URL`, `LPBS_SERVICE_SECRET`, and
`DEPLOYMENT_MANAGER_TELEMETRY_DIR`. When omitted, inter-scenario URLs resolve
through Vrooli service discovery.

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
