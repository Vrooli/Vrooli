# Money Ledger

Holds where money is and where it went as an auditable journal, admits every source through one contract, and computes financial position rather than asserting it.

**The capability in one sentence:** every money event — a card charge, a bank transaction, a hand-typed cash sale — lands in one journal in one shape, carrying how much it can be trusted; position is a query over that journal and can never be stale.

**The load-bearing decision:** the scenario does not integrate with named systems. It defines one inbound shape — a dated, signed, attributed money event with provenance and a basis — and lets any source satisfy it. No P0 target names a specific upstream, and that is deliberate.

**Its sibling:** [Offer Desk](../offer-desk/) holds what should be sold. This scenario holds what happened.

**How the two are adopted:** Offer Desk is the `monetization` team's *instrument* — the one address its members read — and it reads this scenario for actuals and financial posture. This scenario is *covered by* that instrument rather than being it, which is a work-queue role and not a ranking: this is the marketed surface and the operator's daily tool. The direction is one-way and stays that way; this scenario has no knowledge of offers. See [`docs/internal/DECISIONS.md`](docs/internal/DECISIONS.md), 2026-08-13.

**Read first:** [`docs/START-HERE.md`](docs/START-HERE.md) for the initialization gates, then [`PRD.md`](PRD.md) for the operational targets and [`docs/concepts/FLOWS.md`](docs/concepts/FLOWS.md) for the diagrams showing how the pieces fit together.

> **Status:** the core journal, ingestion, position, goal, statement, CLI, and console vertical slices are implemented. Run `make test` for the scenario-owned validation suite; see [`docs/internal/PROBLEMS.md`](docs/internal/PROBLEMS.md) for remaining integration and experience-contract findings.

---

This scenario follows Vrooli's standard full-stack scenario shape:

- Go API (`api/`)
- React + TypeScript + Vite UI (`ui/`)
- CLI wrapper (`cli/`)
- Lifecycle + health wiring (`.vrooli/service.json`)
- Requirements registry, experience contract, and progress log
  (`requirements/`, `experience/`, `docs/internal/PROGRESS.md`)

> **Start here:** open [`docs/START-HERE.md`](docs/START-HERE.md), then use
> `make orient` for the current lifecycle and contract gate status.

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
- Baseline PWA/native-readiness metadata: web app manifest,
  standalone-mode mobile tags, proxy-safe relative install asset URLs,
  a minimal app-shell service worker, safe-area CSS tokens, and generic
  placeholder icons ready for scenario-specific replacement.
- Canonical responsive shell plus adopted-provenance UI primitives from
  `react-component-library` for common shared surfaces such as buttons,
  cards, data tables, empty states, inputs, selects, status badges, sidebar
  shell, and bottom navigation.
- Root-level `DESIGN.md` plus generated UI token assets from the
  selected design kit.
- Experience contracts for the six operator pages, including live-data,
  degraded, empty, and mobile journey states.
- A documentation contract in `docs/manifest.json`, with stubs for
  domains, flows, data, integrations, monetization, deployment,
  runbooks, observability, security, performance, and durable
  decisions.

## Durable seams

The product-specific implementation preserves these cross-cutting seams:

- i18n wiring (`SUPPORTED_LOCALES`, `useTranslation`, `setLocale`).
- Accessibility primitives (`role`, `aria-*`, `data-testid` selectors).
- Design tokens (`bg-app-background`, `rounded-panel`, etc.).
- Adopted shared UI primitives under `ui/src/components/ui/`; prefer
  `react-component-library adoptions apply` over hand-rolling a new primitive.
- The responsive shell floors: full viewport height, overflow-contained main
  content, desktop sidebar, fixed safe-area mobile bottom nav, and Settings
  ownership of locale switching.
- The feature-folder pattern under `ui/src/features/<name>/`.
- The proto → API → CLI → UI vertical-slice shape.

**Connect-RPC is the default transport.** Every domain endpoint goes
through a proto service and generated Connect handlers/clients. If
you find yourself writing `Path: "/api/v1/..."` as a literal string in
an `EndpointDescriptor`, stop — use a proto service method instead.
Codegen rejects literal Paths that lack an explicit `RESTException`
tag; the four allowed REST reasons (multipart upload, webhook
receiver, third-party shape, ops probe) are enumerated in
`api/internal/module/module.go`. REST endpoints are limited to the
explicitly approved exception cases.

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
`test-genie execute money-ledger --preset comprehensive` directly for
finer-grained presets.

## Documentation Map

| Need | Start Here |
|---|---|
| Initialize after generation | [`docs/START-HERE.md`](docs/START-HERE.md) |
| Establish UI design language | `DESIGN.md` at this scenario's root |
| Author UX intent | [`experience/README.md`](experience/README.md) |
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
5. **Keep `experience/` aligned with routes.** Start at L0, then add priorities, claims, bindings, states, and journeys before flipping pages active.
6. **Update `docs/concepts/DOMAINS.md`** before adding product code.
7. **Keep `docs/manifest.json` accurate.** Durable docs should be registered there with a truthful maturity value.
8. **Append progress entries** to `docs/internal/PROGRESS.md` whenever you land work.
9. **Add resources** to `.vrooli/service.json` only when needed; this scenario ships with no resource dependencies (SQLite is in-process).
10. **Keep boundaries**: only edit within this scenario's directory.

## pnpm Everywhere

This scenario assumes pnpm. If you run another package manager, convert
lockfiles yourself before committing. Scripts use `pnpm` directly (no
`npm` fallbacks) to reduce drift.

## Need Inspiration?

Open `scenarios/browser-automation-studio/` to see the same template
shape taken to completion.
