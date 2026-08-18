# Notification Hub

Owner-facing notification spine that routes alerts to the right device and channel, relaying through the fleet when this host cannot deliver

Something happened. Decide who should hear about it and how, then make
sure they did. That sentence is the whole scenario.

Every scenario and agent in Vrooli produces things worth telling a human
about, and none of them should own retry logic, quiet hours, device
addresses, or channel credentials. Those live here once. A caller
supplies what happened and how urgent it is; this scenario decides the
channel, the timing, and — when this machine cannot reach the target
device — which other machine in the fleet can.

> **Status: regenerated 2026-08-17, documentation-first.** The contract
> in [`PRD.md`](PRD.md) and `docs/` is authored and validating; the
> implementation is the generated scaffold plus the template's `notes`
> reference domain. No product code exists yet, and the predecessor
> scenario's habit of reporting success while delivering nothing is
> exactly what the release checklist in
> [`docs/operations/DEPLOYMENT.md`](docs/operations/DEPLOYMENT.md) now
> guards against.

> **Start here:** open [`docs/START-HERE.md`](docs/START-HERE.md). It
> owns the first-session initialization protocol — charter, requirements,
> domain map, design language, placeholder replacement, and first real
> vertical slice. Run `make orient` for a machine-readable gate status.

## What You Get

The product shape, once built:

- **One send call.** Recipient, title, body, urgency, sensitivity.
  Everything else is this scenario's problem.
- **A device and channel registry** — which devices the owner has, which
  channels each accepts, and the address for each.
- **A routing core** that applies preferences, quiet hours, and
  duplicate suppression before anything leaves the machine, and that can
  be tested exhaustively without a network.
- **Delivery with retry and a stated terminal reason**, plus a timeline
  that answers "did it arrive" and "why that channel" for every
  notification.
- **Cross-node relay** so a channel this host cannot serve — macOS
  Notification Center, iMessage — is delivered by a fleet node that can.
- **A sensitivity model** so a notification body is safe on a locked
  screen, in a shared room, and in a third-party provider's logs.

The scaffold it is built on:

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

## Customize Safely

Three decisions are load-bearing and should not be undone without
superseding them in
[`docs/internal/DECISIONS.md`](docs/internal/DECISIONS.md):

1. **No resource dependencies.** `resource-postgres` and
   `resource-redis` are OCI-acquired and recorded `unsupported` on macOS
   and Windows. Adding either would make the scenario unable to run on
   the Mac node the relay lane exists to reach. Check
   `path:docs/reference/platform-support.md` before adding anything to
   `.vrooli/service.json`.
2. **No local accounts.** Identity comes from `scenario-authenticator`
   and recipients are keyed by the token subject. Do not add a profile
   table, an API key, or a password field. Multi-user support arrives by
   trust posture, not by schema.
3. **Sensitivity is NOT NULL with no default.** A caller decides once
   whether a body may leave the machine. A default would make the unsafe
   case silent.

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
`test-genie execute notification-hub --preset comprehensive` directly for
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
