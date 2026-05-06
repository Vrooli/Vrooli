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
| Write tests | [`docs/internal/TESTING.md`](docs/internal/TESTING.md) |
| Add or update seams/fakes | [`docs/internal/SEAMS.md`](docs/internal/SEAMS.md) |
| Configure env vars, ports, CLI config | [`docs/reference/configuration.md`](docs/reference/configuration.md) |
| Add API endpoints | [`docs/reference/api-endpoints.md`](docs/reference/api-endpoints.md) |
| Add CLI commands | [`docs/reference/cli-commands.md`](docs/reference/cli-commands.md) |

## Customize Safely
1. **Read `docs/START-HERE.md` first.** It owns the first implementation workflow.
2. **Update PRD.md + requirements/** before feature work. Operational targets drive code + tests.
3. **Read root `DESIGN.md` before UI work.** Keep global styles, Tailwind theme, and primitives aligned with it.
4. **Append progress entries** to `docs/internal/PROGRESS.md` whenever you land work.
5. **Add resources** in `.vrooli/service.json` only when needed; the template ships with no resource dependencies (SQLite is in-process).
6. **Keep boundaries**: only edit within `scenarios/<your-scenario>/`.

## pnpm Everywhere
The template assumes pnpm. If you run another package manager, convert lockfiles yourself before committing. Scripts use `pnpm` directly (no `npm` fallbacks) to reduce drift.

## Need Inspiration?
Open `scenarios/browser-automation-studio/` to see this template taken to completion.
