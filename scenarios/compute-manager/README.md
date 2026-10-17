# Compute Manager

Acquires, tracks and retires remote compute, and meters what it costs.

Compute Manager is the permanent capability for turning "I need a
machine" into a machine that is already a trusted Vrooli node, and it is
the one place that knows what that machine costs. It owns capacity and
cost, and nothing else. An operator uses it to buy capacity for their own
fleet without opening a provider console; other scenarios use it when
they need capacity the standing fleet does not have, such as validation
bursting to an operating system no current node runs.

Three properties shape everything in this tree:

- **Cost is on the same screen as inventory.** A provider console shows
  what exists and hides what it has cost you in a billing section nobody
  opens until the invoice arrives. Here, elapsed cost and remaining
  lifetime sit beside every instance, because both are irreversible once
  missed.
- **Destroy is the only stop.** A stopped instance still bills at the
  full rate on most providers, so there is no pause, suspend or shutdown
  verb anywhere in this scenario, and a structural test enforces that.
- **The reconciler reports and never resolves.** A bidirectional sweep
  compares provider inventory against local records in both directions
  and reports every divergence, because a reconciler bug that destroys is
  indistinguishable from one that deletes a paying customer's node.

Connecting a machine you already own stays free forever. Only capacity
this scenario provisions and pays for is metered.

> **Status: nothing is implemented.** This scenario was generated from
> the `react-vite` template on 2026-09-03 and carries the design contract
> only: the PRD, the requirements registry, the domain, data, flow and
> integration contracts, the experience contract, and the operations
> documentation. There is no provider adapter, no instance record, no
> reconciler, no expiry sweeper and no enrollment path. Every command
> named in the documentation below is an intended command surface unless
> it appears in Part 1 of
> [`docs/reference/cli-commands.md`](docs/reference/cli-commands.md).

> **Start here:** open [`docs/START-HERE.md`](docs/START-HERE.md). It
> owns the first-session initialization protocol — charter, requirements,
> domain map, design language, placeholder replacement, and first real
> vertical slice. Run `make orient` for a machine-readable gate status.

## What You Get

The standard full-stack Vrooli scenario shape:

- Go API (`api/`)
- React + TypeScript + Vite UI (`ui/`)
- CLI wrapper (`cli/`)
- Lifecycle + health wiring (`.vrooli/service.json`)
- Requirements registry, experience contract, and progress log
  (`requirements/`, `experience/`, `docs/internal/PROGRESS.md`)

And in more detail:

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
- The application shell, page headers, settings rows, empty states, cards,
  buttons, badges and async regions come from `react-component-library` as
  linked package imports. `ui/src/layout/` holds only this scenario's
  navigation data, mark, and three shell settings; nothing in the tree
  redraws chrome.
- Root-level `DESIGN.md` plus generated UI token assets from the
  selected design kit.
- `experience/` specs for the five real routes this scenario intends:
  inventory, instance, request capacity, findings and settings. Every
  page is status `draft` and every claim is tier `aspirational`, because
  no selector exists yet to check them against.
- A documentation contract in `docs/manifest.json` covering domains,
  flows, data, integrations, monetization, deployment, runbooks,
  observability, security, performance, and durable decisions.

## Customize Safely

The generated scaffold is intentionally not the product. When you build
the real UX, treat these as **placeholders** to replace:

- The home surface, marked `PLACEHOLDER:home-surface` in
  `ui/src/pages/DashboardPage.tsx`. It is replaced by the inventory
  surface specified in `docs/concepts/EXPERIENCE.md`.
- The two-destination navigation in `ui/src/layout/navItems.tsx`, which
  still reflects the template's routes rather than the five surfaces the
  experience contract declares.
- The bare-minimum settings surface, which needs the provider list and
  per-provider billing facts described in
  `experience/pages/settings.json`.

The template's worked example domain has already been removed by
`template-manager detemplate compute-manager`; the copyable vertical
slice it demonstrated lives in `templates/scenarios/react-vite/`.

Treat these as **durable seams** to preserve, even as you rewrite the
visual layout:

- i18n wiring (`SUPPORTED_LOCALES`, `useTranslation`, `setLocale`).
- Accessibility primitives (`role`, `aria-*`, `data-testid` selectors).
- Design tokens (`bg-app-background`, `rounded-panel`, etc.).
- Library components linked through `react-component-library adoptions link`;
  ask `adoptions suggest` before hand-rolling anything generic.
- The shell as a library import, configured in `ui/src/layout/AppShell.tsx`
  and never redrawn locally; Settings owns every preference.
- The feature-folder pattern under `ui/src/features/<name>/`.
- The proto → API → CLI → UI vertical-slice shape.

**Connect-RPC is the default transport.** Every domain endpoint goes
through a proto service and generated Connect handlers/clients. If
you find yourself writing `Path: "/api/v1/..."` as a literal string in
an `EndpointDescriptor`, stop — use a proto service method instead.
Codegen rejects literal Paths that lack an explicit `RESTException`
tag; the four allowed REST reasons (multipart upload, webhook
receiver, third-party shape, ops probe) are enumerated in
`api/internal/module/module.go`. This scenario expects to need none of
them: every planned domain carries typed wire contracts and no opaque
bytes.

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
`test-genie execute compute-manager --preset comprehensive` directly for
finer-grained presets.

## Documentation Map

| Need | Start Here |
|---|---|
| Initialize after generation | [`docs/START-HERE.md`](docs/START-HERE.md) |
| Establish UI design language | `DESIGN.md` at this scenario's root |
| Decide, configure and adopt the UI | [`docs/guides/choosing-ui.md`](docs/guides/choosing-ui.md), [`docs/concepts/EXPERIENCE.md`](docs/concepts/EXPERIENCE.md) |
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

Read `docs/guides/choosing-ui.md` before opening `ui/`. Then open
`scenarios/browser-automation-studio/` to see the same template shape taken to
completion on the API and CLI side; its UI predates the library shell.
