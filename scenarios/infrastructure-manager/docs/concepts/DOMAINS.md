# Domains — Infrastructure Manager

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

This scenario is a **thin, read-mostly aggregator**. Every domain below
measures and surfaces; none re-implements another scenario's measurement,
performs the improvement, or makes a judgment call. That constraint is the
scenario's defining property and the reason the domain set is small.

Three product domains are planned — `coverage`, `condition`, `focus` —
alongside the scaffold's `health`. They are the three axes of
[`COVERAGE-MODEL.md`](COVERAGE-MODEL.md) made into code, and the mapping is
one-to-one on purpose: a reader who understands the models already knows
where any behaviour lives.

**Build order is `coverage` → `condition` → `focus`**, because each reads the
one before it: condition needs the cell grid to band against, and focus ranks
what the other two produce. Each domain ships as a **vertical** — API, CLI and
its board page together. No page ships without its domain; no domain ships
without its page.

The scaffold also ships one clearly fenced worked example domain (never
product scope) as a copyable reference; `template-manager detemplate
infrastructure-manager` removes every fenced example once the real domains
are green.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/infrastructure-manager/v1/shared/health.proto` |
| coverage | Join owner-authored spaces with the checked-in setpoint into a live cell grid. | Answer "how much of the platform's reliability is instrumented at all, and against what bar?" | No product data; reads spaces live and parses a checked-in file. | reporting | query, validation | Projection, Cell, Denominator, Confidence, OpenLoop | `api/internal/coverage/`, `api/handlers/coverage/`, `cli/domains/coverage/`, `ui/src/features/coverage/` |
| condition | Take live readings on every `NOW` cell, qualify their trust, and band them. | Answer "for what we can see, is the platform in band — and can the reading be believed?" | Reading history and trust verdicts (never band verdicts). | reporting | service, timeseries | Leg, Reading, TrustVerdict, BandVerdict | `api/internal/condition/`, `api/handlers/condition/`, `cli/domains/condition/`, `ui/src/features/condition/` |
| focus | Rank the error surface and grade actuation. | Give a member one ordered answer to "what should I do next?", and record whether past fixes moved their sensor. | Findings and their sensor-efficacy join. | workflow | service, ranking | Finding, GapSource, Efficacy | `api/internal/focus/`, `api/handlers/focus/`, `cli/domains/focus/`, `ui/src/features/focus/` |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships `notes` as a worked CRUD vertical slice with a binary
upload exception. Copy its shape for your own domains, then remove it.

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| notes | Provide the worked CRUD reference with attachment upload exception. | Demonstrate the expected vertical slice for a real domain. | Notes and attachment metadata. | crud | service | Note, Attachment | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/infrastructure-manager/v1/notes/` |

- Purpose: demonstrate the expected vertical slice for a real domain.
- Primary archetype: CRUD / entity.
- Secondary traits: binary/blob attachment upload, upload workflow.
- Owns: note records, attachment metadata, note validation, note
  service/repository seams, UI note interactions, CLI notes commands.
- Does not own: product scope for a generated scenario.
- API: `api/internal/notes/`, `api/handlers/notes/`.
- CLI: `cli/domains/notes/`.
- UI: `ui/src/features/notes/`, `ui/src/api/notes.ts`.
- Storage: domain-owned SQLite schema in `api/internal/notes/schema.sql`.
- Requirements: template starter only; replace with PRD-specific
  requirements.
- Tests: repository, service, handler, CLI, UI, accessibility, and
  workflow tests.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).
<!-- EXAMPLE-DOMAIN:notes END -->

## Domain Details

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Requirements: starter scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

### coverage

- Purpose: produce the live cell grid — which reliability dimensions are instrumented, at what confidence, against which bar.
- Primary archetype: reporting / query.
- Secondary traits: multi-owner fan-out, declarative-file parsing, integrity validation, drift detection.
- Owns: the typed projection and cell model, the denominator join, denominator-confidence, the open-loop set with its dates, and the integrity findings that come from parsing the setpoint.
- **Does not own either half of the denominator.** The *space* — which cells exist — is authored by each control layer in its own `docs/spaces/<projection>-space.md` and read through that owner's `space --projection <p> --json` verb. The *bar* is the checked-in setpoint file, changed only through an approved `reliability-target-update` in a reviewed commit. Authoring either here would be the instrument confirming itself — deviation `D6` in `docs/agent-system/TARGET_MODEL.md`.
- Also owns setpoint drift: a cell naming a sensor whose typed operation no longer resolves, and a `MISSING` cell whose gap a shipped verb could already close. Both are computed against the fleet's live surface so neither can go stale the way the thing it checks can.
- API: `api/internal/coverage/`, `api/handlers/coverage/`.
- CLI: `coverage status`, `coverage show <projection>`, `coverage cells`, `coverage validate`, `coverage drift`, `coverage open-loop`.
- UI: `ui/src/features/coverage/` — the cell grid, per-projection confidence, and the dated open-loop set.
- Storage: **none.** Spaces are read live; the setpoint is parsed per query. A `coverage` table is almost certainly a cached denominator, which is `D6`.
- Requirements: `IFM-P0-001`, `IFM-P0-002`, `IFM-P0-005`, `IFM-P1-004`; board page `IFM-P1-005`.
- Related docs: [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md), [`SETPOINT-MODEL.md`](SETPOINT-MODEL.md).

### condition

- Purpose: for every cell that resolved `NOW`, take a live reading, attach a trust verdict, and band it against the current deadband.
- Primary archetype: reporting / timeseries.
- Secondary traits: bounded concurrent fan-out, independent per-source degradation, derived leg population.
- Owns: the reading history, the trust-verdict computation, and band evaluation.
- **Owns readings, never band verdicts.** In-band status is recomputed at query time against the *current* deadband, so tightening a target re-grades history instead of stranding stale judgments. Trust verdicts are the one stored exception, because nothing can reconstruct after the fact whether a check was saturated at the moment of the read.
- **Never caches its leg population.** The legs are derived per read from the cells that resolved `NOW`. A stored leg list is the roster operating-model rule 6 forbids and `INSTRUMENTATION_ROADMAP.md` Gap 11 rejects — an architectural defect, not an optimization.
- Does not own: what a sensor means. Each control layer owns its own semantics; this domain reads the derived output and never re-runs or re-implements a measurement.
- API: `api/internal/condition/`, `api/handlers/condition/`.
- CLI: `condition status`, `condition trust`, `condition explain <cell>`, `condition history <cell>`.
- UI: `ui/src/features/condition/` — per-cell band state, the trust distribution triple, and drilldown to the evidence behind each verdict.
- Storage: domain-owned SQLite reading history, with a retention floor derived from the longest declared window.
- Requirements: `IFM-P0-003`, `IFM-P1-001`, `IFM-P1-002`; board page `IFM-P1-005`.
- Related docs: [`CONDITION-MODEL.md`](CONDITION-MODEL.md), [`TRUST-MODEL.md`](TRUST-MODEL.md), [`INTEGRATIONS.md`](INTEGRATIONS.md).

### focus

- Purpose: merge every error term into one ranked surface, and grade whether completed work actually moved its sensor.
- Primary archetype: workflow / ranking.
- Secondary traits: multiplexed sources, efficacy join, cascade-ordered ranking.
- Owns: findings, their ranking, and the finding-to-sensor efficacy record.
- Sources are named and independently degradable — out-of-band readings, untrusted readings, open-loop cells, coverage drift, and source unavailability. An unreadable source becomes a visible availability entry rather than silently removing findings.
- **Ranks by cascade order and says so.** Sensor-channel integrity is the innermost layer, so trust findings outrank condition findings whenever both are present. A reordering the reader cannot see is indistinguishable from a ranking bug.
- **Surfaces, does not decide.** Ranking is a recommendation. Whether to repair, defer, or retire stays with the team and the operator, and no path in this scenario actuates anything.
- Actuation efficacy implements update-protocol rule 5: a finding names its sensor and expected in-band return, the next read after the work lands re-grades it, and a fix that does not move the sensor re-opens the finding. The sensor grades the fix, not its author.
- API: `api/internal/focus/`, `api/handlers/focus/`.
- CLI: `focus next`, `focus show <id>`, `focus efficacy`, `focus sources`.
- UI: `ui/src/features/focus/` — the ranked surface and the `triage-next-finding` journey.
- Storage: domain-owned SQLite findings and efficacy records.
- Requirements: `IFM-P0-004`, `IFM-P1-003`; board page and `triage-next-finding` journey `IFM-P1-005`.
- Related docs: [`CONDITION-MODEL.md`](CONDITION-MODEL.md) § Actuation efficacy.

## Projections Are Cells, Not Domains

The ten reliability projections — `supervision`, `availability`, `recovery`,
`capacity`, `headroom`, `durability`, `attribution`, `validation-cost`,
`agent-throughput`, `commissioning` — are **data in the cell grid, not code
structures.** Adding an eleventh projection must require zero new domains,
zero new tables, and zero new handlers: an owner authors a space doc, the
setpoint gains bars, and the grid grows a column.

This is the topological payoff the whole design exists for. If adding a
projection ever requires a code change here, that is the signal the grid has
been special-cased and the change should be reverted rather than extended.

> **`supervision` was originally modelled as a domain and is not one.** It is
> one projection among ten, distinguished only by having a two-direction
> reconcile as its numerator instead of a scalar read. Modelling it as a
> domain gave one reliability dimension a structural privilege the other nine
> did not have, and made the tenth projection harder to add than the second.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Projection | One reliability question, owned by the control layer holding its ground truth. | [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md). |
| Cell | One answerable fact inside a projection; the unit of coverage. | [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md). |
| Leg | The smallest unit an owner can measure independently; the unit of condition. | [`CONDITION-MODEL.md`](CONDITION-MODEL.md). |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `watchdog-supervision` | The plant does not exist yet. A watchdog tier — liveness of autoheal and the core set, on a seconds clock, with an enumerated action set — is a separate operator decision precisely because it is the only piece that would touch a running system. Modelling its supervision now would inflate the open-loop count with a hole nobody can close. | The operator authorizes a watchdog tier. It then becomes an eleventh **projection**, not a domain, supervising liveness, enumerated-action counts, and claim-suppression rate. |
| `capability-availability` | Owner-side work, not ours. `INSTRUMENTATION_ROADMAP.md` Gap 11 places availability persistence with each capability owner, and building it centrally would recreate the roster rule 6 forbids. `condition` carries an interim read-time reachability proxy, flagged `estimate` with an expiry. | Capability owners persist queryable availability aggregates. The proxy is then retired in favour of owner-derived history. |
| `platform-code-audit` | Judgment-shaped, not measurement-shaped. Cross-platform debt and internal platform-code quality belong to the `platform-code-auditor` member's lane; forcing them into a coverage ratio would repeat the mistake the production-ledger archetype exists to prevent. | Only if a defensible denominator for platform-code conformance is ever authored. Absent that, this stays prose, and `docs/infra-health/evidence/CROSS_PLATFORM_LEDGER.md` stays with the team. |
| `remediation` | Out of scope by contract, permanently. Operating-model rule 3 is "supervise, don't operate", and live incident response belongs to autoheal, system-monitor, agent-manager and the operator. | Never. A remediation domain here would make the instrument a controller — the boundary violation `TARGET_MODEL.md` names as the whole reason sensors carry no authority. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/sources/` — typed read clients per control layer, resolved through `api-core/discovery`. Transport, not product capability; it holds no reliability vocabulary and no verdict logic.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md) — projections, cells, denominators, confidence
- [`CONDITION-MODEL.md`](CONDITION-MODEL.md) — legs, banding, reading history, efficacy
- [`TRUST-MODEL.md`](TRUST-MODEL.md) — the closed trust vocabulary
- [`SETPOINT-MODEL.md`](SETPOINT-MODEL.md) — the bar, and why this scenario cannot move it
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy

Upstream canon this scenario implements (cited, never restated):

- `docs/agent-system/TARGET_MODEL.md` — the instrument contract and the deviation catalogue
- `docs/infra-health/operating/OPERATING_MODEL.md` — the plant's layer map and routing rules
- `scenarios/meta-optimization-manager/docs/concepts/{COVERAGE,CONDITION}-MODEL.md` — the sibling instrument's models
