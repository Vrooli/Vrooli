# Token Economy

A real economy with money that isn't real: mint tokens, grant them with rules that outlive the grant, and let people who share one Vrooli instance earn and spend between themselves.

**The capability in one sentence:** a household declares its own currency and its own rewards, a child earns and redeems against them, and every balance is a query over an append-only journal that explains exactly why it is what it is.

**The load-bearing decision:** a grant *is* a mandate. A token grant carrying spend rules — what it buys, from whom, until when — is structurally the same object that authorizes real spending in [`treasury`](../treasury/). The two are authored congruent by design, which is why the eventual real-value rail is a new adapter rather than a rewrite. `treasury` is a **contract sibling, not a dependency**: neither calls the other, and this scenario works fully with `treasury` absent.

**Why local-only is a feature, not a limitation:** every rule worth having on real money — a cap, an expiry, an approval gate, a one-way narrowing to a delegate — can be modeled, shipped and broken here at zero cost. A policy engine that has only ever run against real money has never been safely wrong. No token carries a price, no redemption produces money, and no transfer leaves the instance; the constraint is enforced by the capability being **absent** from the service surface, not by a policy check (`OT-P0-014`).

**Its market position:** every allowance app on the market — Greenlight, BusyKid, FamZoo, Modak, Acorns Early — is a bank product requiring a linked account, a card issued to a child, KYC, and a monthly fee, and all of them can only pay out in dollars. This one has no bank, no card, no KYC, and rewards that can be anything the household decides: screen time, a trip, a chore traded between siblings. See [`docs/business/MONETIZATION.md`](docs/business/MONETIZATION.md) — the hypothesis is honest about being unvalidated.

**Read first:** [`docs/START-HERE.md`](docs/START-HERE.md) for the initialization gates, then [`PRD.md`](PRD.md) for the operational targets and [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) for the seven-domain map and build order.

> **Status: seven-domain skeleton in progress.** The removable template domain is gone, and the mints, journal, grants, holders, earning, catalog, and redemption package boundaries now exist. Product behavior and requirement evidence remain planned until their owning implementation phases land. See [`docs/internal/PROGRESS.md`](docs/internal/PROGRESS.md) for the measured frontier.

---

This scenario provides
the standard full-stack Vrooli scenario shape:

- Go API (`api/`)
- React + TypeScript + Vite UI (`ui/`)
- CLI wrapper (`cli/`)
- Lifecycle + health wiring (`.vrooli/service.json`)
- Requirements registry, generated L0 experience contract, and progress log
  (`requirements/`, `experience/`, `docs/internal/PROGRESS.md`)

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
- Generated `experience/` L0 specs for the starter routes. These are UX
  intent placeholders, not finished claims; grow them as routes become real.
- A documentation contract in `docs/manifest.json`, with stubs for
  domains, flows, data, integrations, monetization, deployment,
  runbooks, observability, security, performance, and durable
  decisions.

## Placeholders vs. Durable Scaffolding

The generated scaffold is intentionally not the product. When you build
the real UX, treat these as **placeholders** to replace:

- Starter page content such as the dashboard metric placeholders.
- The bare-minimum settings surface once your scenario needs more than
  theme and locale.

Treat these as **durable seams** to preserve, even as you rewrite the
visual layout:

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
`test-genie execute token-economy --preset comprehensive` directly for
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
