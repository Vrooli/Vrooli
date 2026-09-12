# React + Vite Scenario Template (Go API + CLI)

Use this template to bootstrap every new scenario. It packages the
standard full-stack Vrooli scenario shape:

- Go API (`api/`)
- React + TypeScript + Vite UI (`ui/`)
- CLI wrapper (`cli/`)
- Lifecycle + health wiring (`.vrooli/service.json`)
- Requirements registry + progress log (`requirements/`, `docs/internal/PROGRESS.md`)

## Copy the Template
```bash
# From the repo root
vrooli scenario generate react-vite \
  --id <your-scenario> \
  --display-name "Your Scenario" \
  --description "One sentence summary" \
  --design vrooli-default
cd scenarios/<your-scenario>/
```

After generation, work from `scenarios/<your-scenario>/` and let the
scenario lifecycle own setup, ports, start/stop, logs, and tests.
Use `make orient` or `vrooli scenario orient <your-scenario>` during
the first implementation session to see which initialization gates are
complete. When the required gates pass, run `vrooli scenario orient
<your-scenario> --finalize`; this removes only temporary orientation
metadata and leaves provenance, docs, requirements, and code intact.

> **The `notes` domain is a worked example, not a starting feature.**
> It demonstrates the canonical vertical slice: proto contract →
> API domain/service/repository → handler module → CLI domain → UI
> feature. Copy the structure for your first real domain, then delete
> the example once your real domain is green.

## What You Get
- Go API (`api/`), Go CLI (`cli/`), and React/Vite UI (`ui/`)
  coordinated through generated proto contracts.
- Lifecycle metadata, Makefile entrypoints, health checks, endpoint
  metadata, testing config, and CLI install wiring.
- Domain-first API shape with per-domain service, repository, schema,
  handler module, mocks, and tests.
- SQLite by default, with external resources added only when a scenario
  actually needs them.
- UI/CLI guardrails for i18n, accessibility, API base resolution,
  declarative command args, generated Connect clients, and report-shaped
  output.
- Root-level `DESIGN.md` plus generated UI token assets from the
  selected design kit.
- A scenario documentation contract in `docs/manifest.json`, including
  stubs for domains, flows, data, integrations, monetization,
  deployment, runbooks, observability, security, performance, and
  durable decisions.

## Setup Workflow
```bash
cd scenarios/<your-scenario>

# Build API + UI, install pnpm deps, install scenario CLI
make setup   # wraps `vrooli scenario setup`

# Start API + UI in the background
make start   # wraps `vrooli scenario start`
```

See [`docs/QUICKSTART.md`](docs/QUICKSTART.md) for the full clone-to-running flow.

Run tests with `make test` (which runs `vrooli scenario test`) or invoke `test-genie execute <your-scenario> --preset comprehensive` directly for finer-grained presets.

## Documentation Map

| Need | Start Here |
|---|---|
| Begin implementation after generation | [`docs/START-HERE.md`](docs/START-HERE.md) |
| Establish UI design language | `DESIGN.md` at the generated scenario root |
| Run the scenario | [`docs/QUICKSTART.md`](docs/QUICKSTART.md) |
| Understand the architecture | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) |
| Map product domains | [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) |
| Track workflows, data, and integrations | [`docs/concepts/FLOWS.md`](docs/concepts/FLOWS.md), [`docs/concepts/DATA.md`](docs/concepts/DATA.md), [`docs/concepts/INTEGRATIONS.md`](docs/concepts/INTEGRATIONS.md) |
| Capture monetization and launch strategy | [`docs/business/MONETIZATION.md`](docs/business/MONETIZATION.md), [`docs/business/GO-TO-MARKET.md`](docs/business/GO-TO-MARKET.md) |
| Prepare deployment and operations | [`docs/operations/DEPLOYMENT.md`](docs/operations/DEPLOYMENT.md), [`docs/operations/RUNBOOK.md`](docs/operations/RUNBOOK.md), [`docs/operations/OBSERVABILITY.md`](docs/operations/OBSERVABILITY.md) |
| Write tests | [`docs/internal/TESTING.md`](docs/internal/TESTING.md) |
| Add or update seams/fakes | [`docs/internal/SEAMS.md`](docs/internal/SEAMS.md) |
| Configure env vars, ports, CLI config | [`docs/reference/configuration.md`](docs/reference/configuration.md) |
| Add API endpoints | [`docs/reference/api-endpoints.md`](docs/reference/api-endpoints.md) |
| Add CLI commands | [`docs/reference/cli-commands.md`](docs/reference/cli-commands.md) |
| Map the UI surface | [`docs/concepts/ux-overview.md`](docs/concepts/ux-overview.md) |
| Understand the UI shell | [`docs/concepts/UI_ARCHITECTURE.md`](docs/concepts/UI_ARCHITECTURE.md) |

## UX At A Glance

The UI is a six-route SPA under `ui/` (React 18 + Vite, `react-router-dom` v7,
`@tanstack/react-query`, `@xyflow/react` + `dagre` for the state graph,
`recharts` for the run-outcome timeline). The application shell is full-width
with a resizable left sidebar on desktop and a drawer + bottom nav on mobile.
Every route is lazy-loaded and wrapped in a per-route `<ErrorBoundary>`.
Theme (light / dark / system) and other preferences are persisted server-side
via `/api/v1/settings`.

| Path | What you see |
|---|---|
| `/` | Dashboard — compact health, recent runs, and the run-outcome timeline. |
| `/flows` | Inventory — searchable / filterable / sortable table of discovered flows with per-row + bulk verify, URL-synced state. |
| `/flows/:flowId` | Flow Detail — tabs: **Graph**, **Traces**, **History**. |
| `/runs/:runId` | Run Detail — tabs: **Result** (metadata), **Counterexample** (JSON tree), **Raw output** (quint output + copy button). |
| `/settings` | Appearance + behavior preferences, persisted via `/api/v1/settings`. |
| `*` | NotFoundPage with a back-to-dashboard CTA. |

See [`docs/concepts/UI_ARCHITECTURE.md`](docs/concepts/UI_ARCHITECTURE.md) for
the shell, routing, and token system, and
[`docs/concepts/ux-overview.md`](docs/concepts/ux-overview.md) for per-route
data dependencies.

## CLI

```bash
flow-verifier flows list [--root <path>]
flow-verifier verify <flowId> [--root <path>]
flow-verifier runs list [--flow-id <flowId>] [--limit <n>]
flow-verifier settings get [--format text|json]
flow-verifier settings set theme=dark density=compact
```

All commands default to human-readable output; `--format json` is opt-in.

## Customize Safely
1. **Read `docs/START-HERE.md` first.** It owns the first implementation workflow.
2. **Run `make orient` as a progress check.** It reports template-owned initialization gates from `.vrooli/orientation.json`.
3. **Update PRD.md + requirements/** before feature work. Operational targets drive code + tests.
4. **Read root `DESIGN.md` before UI work.** Keep global styles, Tailwind theme, and primitives aligned with it.
5. **Update `docs/concepts/DOMAINS.md`** before adding product code.
6. **Keep `docs/manifest.json` accurate.** Durable docs should be registered there with a truthful maturity value.
7. **Append progress entries** to `docs/internal/PROGRESS.md` whenever you land work.
8. **Add resources** in `.vrooli/service.json` only when needed; the template ships with no resource dependencies (SQLite is in-process).
9. **Keep boundaries**: only edit within `scenarios/<your-scenario>/`.

## pnpm Everywhere
The template assumes pnpm. If you run another package manager, convert lockfiles yourself before committing. Scripts use `pnpm` directly (no `npm` fallbacks) to reduce drift.

## Need Inspiration?
Open `scenarios/browser-automation-studio/` to see this template taken to completion.
