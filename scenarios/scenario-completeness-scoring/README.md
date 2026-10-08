# Scenario Completeness Scoring

Fast cached scenario status, maturity, and next-focus reader.

```bash
scenario-completeness-scoring score get <scenario> [--json]
scenario-completeness-scoring score trend <scenario> [--limit 20] [--json]
scenario-completeness-scoring score list [--sort score|rung|last-scored|scenario|priority] [--limit 25] [--json]
```

One command answers "what is the current state of scenario X and what
should I focus on next?" in under a second, from cached on-disk artifacts
on the core path — it never runs tests, and optional service enrichment is
hard-budgeted and omitted on miss:

- **Maturity rung** R0–R4 via the shared `packages/maturity-go` ladder
  gates, labeled **"as of digest td:…"** — never presented as
  swarm-manager's live state.
- **0–100 composite** with classification and a per-group breakdown
  (quality / coverage / quantity / UI), each line showing observed
  counts, thresholds, and awarded points.
- **Prioritized recommendations** with estimated point impact plus a
  phased action plan.
- **Per-phase freshness verdicts** (fresh/stale/unknown via the shared
  `packages/freshness-go` digest + run-index contract) with a
  copy-pastable refresh command. Never-tested scenarios read "unknown",
  not fake-fresh.
- **Optional importance enrichment** from scenario-dependency-analyzer
  centrality and swarm-manager recent activity, shown only when at least
  one source responds inside the budget.
- **Digest-deduplicated score history** persisted by the API sweeper so
  reports can show previous-score deltas and `score trend` can return a
  per-scenario series. The sweeper is controlled by
  `SCS_SCORE_SWEEP_INTERVAL`, `SCS_SCORE_SWEEP_CONCURRENCY`,
  `SCS_SCORE_SWEEP_START_JITTER`, and `SCS_SCORE_SWEEP_DISABLED`.
- **Fleet-scale bulk view** through `ScoreService.ListScores` / `score list`,
  served from latest persisted snapshots with server-side sort, filter, and
  pagination. The normal path is O(query) over stored rows and never computes
  the full fleet on demand.
- **Federated fleet measures** for measures-health/search-hub consumers:
  count below rung, average composite, and score series, all served from the
  snapshot store.
- **Degradation honesty:** every signal collector runs behind a circuit
  breaker; malformed artifacts disable that collector and surface in the
  output instead of crashing the read.

It is the fast **cached** layer of the scenario-status stack: test-genie
is the slow fresh deep layer that *produces* the artifacts this scenario
reads; swarm-manager computes the same rung predicates in-process on
live findings. All three share maturity-go/freshness-go, so their answers
agree by construction.

This scenario provides
the standard full-stack Vrooli scenario shape:

- Go API (`api/`) — domains `signals` (cached-artifact collectors),
  `freshness` (digest + verdicts), `importance` (optional enrichment),
  `scoring` (assembly + ScoreService + snapshot store/sweeper)
- React + TypeScript + Vite UI (`ui/`)
- CLI wrapper (`cli/`)
- Lifecycle + health wiring (`.vrooli/service.json`)
- Requirements registry + progress log (`requirements/`, `docs/internal/PROGRESS.md`)

> **Start here:** open [`docs/START-HERE.md`](docs/START-HERE.md). It
> owns the first-session initialization protocol — charter, requirements,
> domain map, design language, placeholder replacement, and first real
> vertical slice. Run `make orient` for a machine-readable gate status.

## What's In This Scenario

- Go API (`api/`), Go CLI (`cli/`), and React/Vite UI (`ui/`)
  coordinated through generated proto contracts.
- Lifecycle metadata, Makefile entrypoints, health checks, endpoint
  metadata, testing config, and CLI install wiring.
- Domain-first API shape with per-domain service, repository, schema,
  handler module, mocks, and tests.
- SQLite by default. Add external resources to `.vrooli/service.json`
  only when this scenario actually needs them.
- UI/CLI guardrails for i18n, accessibility, API base resolution,
  declarative command args, generated Connect clients, and report-shaped
  output.
- Baseline PWA branding metadata: web app manifest, standalone-mode
  mobile tags, and generic placeholder icons ready for scenario-specific
  replacement.
- Root-level `DESIGN.md` plus generated UI token assets from the
  selected design kit.
- A documentation contract in `docs/manifest.json`, with stubs for
  domains, flows, data, integrations, monetization, deployment,
  runbooks, observability, security, performance, and durable
  decisions.

## Placeholders vs. Durable Scaffolding

The template's `notes` worked example has been removed (orientation
Gate 7); the scoring domain is the product. Remaining template
scaffolding that may evolve with the product:

- The `AppShell` layout in `ui/src/`.
- The bare-minimum settings surface (currently just theme + locale
  switching).

Treat these as **durable seams** to preserve, even as you rewrite the
visual layout:

- i18n wiring (`SUPPORTED_LOCALES`, `useTranslation`, `setLocale`).
- Accessibility primitives (`role`, `aria-*`, `data-testid` selectors).
- Design tokens (`bg-app-background`, `rounded-panel`, etc.).
- The feature-folder pattern under `ui/src/features/<name>/`.
- The proto → API → CLI → UI vertical-slice shape.

**Connect-RPC is the default transport.** Every domain endpoint goes
through a proto service and generated Connect handlers/clients. If
you find yourself writing `Path: "/api/v1/..."` as a literal string in
an `EndpointDescriptor`, stop — use a proto service method instead.
Codegen rejects literal Paths that lack an explicit `RESTException`
tag; the four allowed REST reasons (multipart upload, webhook
receiver, third-party shape, ops probe) are enumerated in
`api/internal/module/module.go`.

[`docs/START-HERE.md`](docs/START-HERE.md) describes the replacement
workflow in full.

## Running The Scenario

```bash
# Build API + UI, install pnpm deps, install scenario CLI
make setup   # wraps `vrooli scenario setup`

# Start API + UI in the background
make start   # wraps `vrooli scenario start`
```

See [`docs/QUICKSTART.md`](docs/QUICKSTART.md) for the full clone-to-running flow.

Run tests with `make test` (which runs `vrooli scenario test`) or invoke
`test-genie execute scenario-completeness-scoring --preset comprehensive` directly for
finer-grained presets.

## Documentation Map

| Need | Start Here |
|---|---|
| Initialize after generation | [`docs/START-HERE.md`](docs/START-HERE.md) |
| Establish UI design language | `DESIGN.md` at this scenario's root |
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

## Working Rules

1. **Read [`docs/START-HERE.md`](docs/START-HERE.md) first.** It owns the first implementation workflow.
2. **Run `make orient`** as a progress check — it reports initialization gates from `.vrooli/orientation.json`.
3. **Update `PRD.md` and `requirements/`** before feature work. Operational targets drive code + tests.
4. **Read root `DESIGN.md` before UI work.** Tokens, motion, and status semantics are binding; specific component lists in the design are illustrative — implement everything your scenario actually needs.
5. **Update `docs/concepts/DOMAINS.md`** before adding product code.
6. **Keep `docs/manifest.json` accurate.** Durable docs should be registered there with a truthful maturity value.
7. **Append progress entries** to `docs/internal/PROGRESS.md` whenever you land work.
8. **Add resources** to `.vrooli/service.json` only when needed; this scenario ships with no resource dependencies (SQLite is in-process).
9. **Keep boundaries**: only edit within this scenario's directory.

## pnpm Everywhere

This scenario assumes pnpm. If you run another package manager, convert
lockfiles yourself before committing. Scripts use `pnpm` directly (no
`npm` fallbacks) to reduce drift.

## Need Inspiration?

Open `scenarios/browser-automation-studio/` to see the same template
shape taken to completion.
