# Treasury

Holds what an agent is permitted to spend and permitted to charge, evaluates every attempt against a grant a human made, and settles it through whichever rail fits.

**The capability in one sentence:** an agent can be given spending power without being given a credential — authority becomes a signed object with a cap, a counterparty scope and an expiry, and the agent that read the merchant page never holds the decision.

**The load-bearing decision:** the core object is a **mandate**, not a balance. Rails — manual settlement, x402, a scoped card — are adapters satisfying one execution contract, which is why no P0 target names a payment vendor. This is [Money Ledger](../money-ledger/)'s one-contract-many-adapters decision applied to the other direction.

**Its siblings:** [Offer Desk](../offer-desk/) holds what *should* earn. [Money Ledger](../money-ledger/) holds what *did* happen. This scenario holds what *may happen right now*. It can never remember financial position, and the ledger can never cause anything — that separation is what keeps the journal a neutral record.

**The security shape worth knowing before reading anything else:** the agent-facing Connect service declares **no method** that mutates policy, budgets or the approval gate. The guarantee is the absence of the RPC, asserted against the generated proto descriptor — not a permission check an injected prompt might argue past.

**Read first:** [`docs/START-HERE.md`](docs/START-HERE.md) for the initialization gates, then [`PRD.md`](PRD.md) for the operational targets, [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) for the domain map, and [`docs/internal/DECISIONS.md`](docs/internal/DECISIONS.md) for the choices that were made deliberately and should not be relitigated.

> **Status: documented, not implemented.** Orientation gates 1-5b are complete — PRD, requirements registry, domain map, data and flow models, dependency contracts, design adaptation, experience contract, and the business, operations and internal docs. **No implementation code exists yet**, and every security mitigation described in [`docs/internal/SECURITY.md`](docs/internal/SECURITY.md) is `designed` rather than verified. Gate 0 (scaffold health) is blocked by an inherited template defect recorded in [`docs/internal/PROBLEMS.md`](docs/internal/PROBLEMS.md).

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

- The `notes` domain (proto, API, CLI, UI feature) — a worked vertical
  slice meant to be copied once and then deleted.
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
`api/internal/module/module.go`. The notes attachments endpoint is
the worked REST example.

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
`test-genie execute treasury --preset comprehensive` directly for
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
