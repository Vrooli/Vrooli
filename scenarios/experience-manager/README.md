# Experience Manager

Experience-axis authority: machine-checkable UX spec contract (experience/), form-based authoring and workshop studio, and validation of built UIs against declared experience intent.

## What This Scenario Is

Vrooli's business-logic track has design-first rigor end to end: `PRD.md` (intent) → `requirements/` (testable contract) → test cases, enforced by business-health. The experience track has had nothing equivalent — which is why scenario UX gets trapped at whatever generation produced. Experience Manager is the missing mirror:

| | Business track (exists) | Experience track (this scenario) |
|---|---|---|
| Intent | PRD operational targets | PRD operational targets |
| Testable contract | `requirements/` | `experience/` — UX spec per page/state |
| Evidence | test cases (test-genie `business` phase) | reconciliation + perceptual checks (test-genie `experience` phase) |

The spec is **claim-based and open-world**: typed assertions about perceivable outcomes ("exactly one action visually dominates", "identity is in the first reading-order region"), never an exhaustive layout description — anything unclaimed is free, so the schema can never pigeonhole novel UX. Claims carry enforcement tiers (`machine` gates CI; `manual` is attested with expiry; `aspirational` is tracked but advisory), use WAI-ARIA roles as the element vocabulary, and separate stable intent from volatile selector bindings. Machine-tier claims are checked deterministically against the accessibility tree BAS captures at runtime — v1 is deliberately zero-ML.

Around that contract: a form-based **Studio** (author specs with live validation, render deterministic wireframes, compare variants side-by-side), **bas/ scaffolding** (derive test-case stubs from spec entries for workflow-health governance), an **attestation ledger**, and a fleet-wide **spec-debt sweep**. Boundaries: ui-health owns "built correctly", this scenario owns "the intended experience"; workflow-health owns the `bas/` substrate this scenario scaffolds into. The full rationale — every fork resolved during design — lives in [`docs/internal/DECISIONS.md`](docs/internal/DECISIONS.md); open prerequisites (BAS a11y-tree capture, the schema spike gate) live in [`docs/internal/PROBLEMS.md`](docs/internal/PROBLEMS.md).

**Status:** v1 implementation is live in API, CLI, and UI. The self-spec dogfood currently has 4 active pages, 1 draft page, 2 draft journeys, and 37 claims (17 machine / 17 manual / 3 aspirational), with all page specs schema-valid and BAS case stubs registered for every active page. The cockpit surfaces are wired to real backend data: Fleet, Scenario Explorer, Evidence, Studio, and Findings no longer rely on sample rows. Full live `vrooli scenario test experience-manager` closure is still the final verification gate for this follow-up.

This scenario was generated from the `react-vite` template and now packages
the full-stack Vrooli scenario shape:

- Go API (`api/`)
- React + TypeScript + Vite UI (`ui/`)
- CLI wrapper (`cli/`)
- Lifecycle + health wiring (`.vrooli/service.json`)
- Requirements registry + progress log (`requirements/`, `docs/internal/PROGRESS.md`)

> **Start here:** open [`docs/START-HERE.md`](docs/START-HERE.md). It
> owns the first-session initialization protocol — charter, requirements,
> domain map, design language, placeholder replacement, and first real
> vertical slice. Run `make orient` for a machine-readable gate status.

## What You Get

- Go API (`api/`), Go CLI (`cli/`), and React/Vite UI (`ui/`)
  coordinated through generated proto contracts.
- Lifecycle metadata, Makefile entrypoints, health checks, endpoint metadata,
  testing config, and CLI install wiring.
- Domain-first API shape for spec parsing, state coverage, reconciliation
  evidence, studio authoring, rendering, manual attestations, fleet sweep, and
  deterministic autofix.
- A live cockpit UI: Fleet reads `ListFleet`; Scenario Explorer reads parsed
  specs; Evidence reads persisted reconciliation evidence; Studio calls
  session/render/compare/promote APIs; Findings calls validate/preview/apply.
- SQLite by default. Add external resources to `.vrooli/service.json`
  only when this scenario actually needs them.
- UI/CLI guardrails for i18n, accessibility, API base resolution,
  declarative command args, generated Connect clients, and report-shaped
  output.
- Baseline PWA/native-readiness metadata: web app manifest,
  standalone-mode mobile tags, proxy-safe relative install asset URLs,
  a minimal app-shell service worker, safe-area CSS tokens, and generic
  placeholder icons ready for scenario-specific replacement.
- Root-level `DESIGN.md` plus generated UI token assets from the
  selected design kit.
- A documentation contract in `docs/manifest.json`, with stubs for
  domains, flows, data, integrations, monetization, deployment,
  runbooks, observability, security, performance, and durable
  decisions.

## Customize Safely

The template placeholder app has been replaced. Preserve these durable seams
when extending the scenario:

- i18n wiring (`SUPPORTED_LOCALES`, `useTranslation`, locale catalogs, generated strings).
- Accessibility primitives (`role`, `aria-*`, `data-testid` selectors) because
  BAS reconciliation depends on them.
- Design tokens (`bg-app-background`, `rounded-panel`, status colors) from `DESIGN.md`.
- The feature-folder pattern under `ui/src/features/<name>/`.
- The proto → API → CLI → UI vertical-slice shape.
- The `experience/` contract as intent. UI selectors may move, but binding
  updates must keep claims honest.

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
`test-genie execute experience-manager --preset comprehensive` directly for
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
