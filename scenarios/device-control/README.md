# Device Control

Device authentication is a governed capability for owner-trusted phones. Use
`device-control auth create` to save reference-only policy, use
`device-control auth update` for metadata changes, pipe a credential to
`device-control auth provision <profile-id>`, acquire a device lease, and run
`device-control auth unlock` to obtain a typed, live-verified result. Delete
the authority-held value with `device-control auth delete-credential` before
revoking a temporary profile. The credential authority owns the secret;
device-control owns binding, method policy, bounded attempts, and postcondition
verification.

Promoted wireless Android devices retain their stable device id across service
restarts. Recover a stale saved endpoint with
`device-control device reconnect <device-id> --json`; the reconnect verifies
the original hardware serial before persisting a discovered TLS endpoint.

Drive owner-trusted physical and virtual devices through pluggable control strategies, with shared vision-based understanding, reusable automation flows, and agent-driven goal completion.

This scenario packages the standard full-stack Vrooli scenario shape:

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

## Durable Scaffolding

The product surfaces are implemented around device inventory, strategy
conformance, leased sessions, flow validation, and evidence. Treat these as
durable seams to preserve as the visual layout evolves:

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
`api/internal/module/module.go`. REST exceptions remain reserved for the
explicitly documented third-party and operational shapes.

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

## Customize Safely

Keep device-control capability probing, leases, evidence retention, and
fail-closed dispositions intact when customizing this scenario. Add behavior
through a declared domain and its API, CLI, UI, requirements, and documentation
surfaces; prefer the shared component canon before introducing a new primitive.

Run tests with `make test` (which runs `vrooli scenario test`) or invoke
`test-genie execute device-control --preset comprehensive` directly for
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
