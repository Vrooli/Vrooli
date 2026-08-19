# Domains — Infrastructure Manager

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

This scenario is a **thin, read-mostly aggregator**. Every domain below
measures and surfaces; none re-implements another scenario's measurement,
performs the improvement, or makes a judgment call. That constraint is the
scenario's defining property and the reason the domain set is small.

Four product domains are planned — `targets`, `readings`, `supervision`,
`focus` — alongside the scaffold's `health`. **Build order is
`targets` → `readings` → `supervision` → `focus`**, because each reads the
one before it: readings need typed targets to band against, supervision
needs the derived set, and focus ranks what the other three produce.
`targets` + `readings` is the first real vertical slice (Gate 6).

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
| targets | Read and validate the setpoint. | Turn the team's authored sensor map into typed targets so the rest of the scenario has something to measure against. | No product data; parses an upstream document. | reporting | query, validation | Target, Deadband, Actuator, SensorRef | `api/internal/targets/`, `api/handlers/targets/`, `cli/domains/targets/` |
| readings | Take live readings and qualify their trust. | Answer "what does each sensor say right now, and can that reading be believed?" | Reading history (never verdicts). | reporting | service, timeseries | Reading, TrustVerdict, Band | `api/internal/readings/`, `api/handlers/readings/`, `cli/domains/readings/` |
| supervision | Reconcile the check registry against the derived should-be-supervised set. | Answer "is everything that should be watched actually watched, and is anything watched that no longer exists?" | No product data; derives its set at read time. | reporting | query, reconciliation | SupervisedSet, GhostCheck, UnsupervisedElement | `api/internal/supervision/`, `api/handlers/supervision/`, `cli/domains/supervision/` |
| focus | Rank the error surface and grade actuation. | Give a member one ordered answer to "what should I do next?", and record whether past fixes moved their sensor. | Findings and their sensor-efficacy join. | workflow | service, ranking | Finding, GapSource, Efficacy | `api/internal/focus/`, `api/handlers/focus/`, `cli/domains/focus/` |

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

### targets

- Purpose: read the setpoint and prove it is well-formed. The setpoint is the sensor map in `docs/infra-health/strategy/RELIABILITY_TARGETS.md`, owned by the infra-health team and the operator.
- Primary archetype: reporting / query.
- Secondary traits: upstream-document parsing, integrity validation.
- Owns: the typed target model (id, sensor reference, deadband, actuator, honesty flag, gap dates) and the integrity findings that come from parsing it.
- **Does not own the setpoint itself.** Authoring a target, choosing a deadband, or changing an actuator is an operator decision made through a `reliability-target-update`. A target authored here would be the instrument confirming itself — deviation `D6` in `docs/agent-system/TARGET_MODEL.md`.
- Also owns setpoint-drift detection: diffing each target's named sensor against the fleet's live command surface, so a target whose sensor no longer resolves — or an empty cell a shipped verb could already fill — is reported rather than silently believed.
- API: `api/internal/targets/`, `api/handlers/targets/`.
- CLI: `targets status`, `targets show <id>`, `targets validate`, `targets drift`.
- Storage: none. The setpoint is read at query time, never cached as truth.
- Requirements: `IFM-P0-001`, `IFM-P1-004`.
- Related docs: [`SETPOINT-MODEL.md`](SETPOINT-MODEL.md).

### readings

- Purpose: take a live reading from each named sensor, band it against its deadband, and attach a trust verdict.
- Primary archetype: reporting / timeseries.
- Secondary traits: bounded fan-out, independent per-source degradation.
- Owns: the reading history and the trust-verdict computation.
- **Owns readings, never verdicts.** In-band status is recomputed at query time against the *current* deadband, so tightening a target re-grades history instead of stranding stale judgments. This is the one deliberate divergence from `meta-optimization-manager`, which never stores a numerator at all — uptime over thirty days *is* history, and Gap 11 names the failure of not keeping it: an outage becomes indistinguishable from missing data after the fact.
- Does not own: what a sensor means. Each source owns its own semantics; this domain reads the derived output and never re-runs or re-implements a measurement.
- API: `api/internal/readings/`, `api/handlers/readings/`.
- CLI: `readings status`, `readings trust`, `readings explain <target>`, `readings history <target>`.
- Storage: domain-owned SQLite reading history.
- Requirements: `IFM-P0-002`, `IFM-P0-003`, `IFM-P1-002`.
- Related docs: [`TRUST-MODEL.md`](TRUST-MODEL.md), [`INTEGRATIONS.md`](INTEGRATIONS.md).

### supervision

- Purpose: report the two-direction diff between the autoheal check registry and the set that should be supervised — ghost checks on one side, unsupervised plant on the other.
- Primary archetype: reporting / reconciliation.
- Secondary traits: derived-set computation.
- Owns: the reconcile diff and its findings.
- **Never owns a roster.** The should-be-supervised set is computed fresh at read time — the core-set closure from `scenario-dependency-analyzer` union the load-bearing declared capability members. Operating-model rule 6 forbids enumeration anywhere in this team's surfaces, and a cached member list would be exactly the central capability-health roster that `INSTRUMENTATION_ROADMAP.md` Gap 11 rejects.
- Does not own: the check registry (that is `vrooli-autoheal`) or the closure computation (that is `scenario-dependency-analyzer`).
- API: `api/internal/supervision/`, `api/handlers/supervision/`.
- CLI: `supervision status`, `supervision reconcile`, `supervision explain <element>`.
- Storage: none; the diff is computed live.
- Requirements: `IFM-P1-001`.

### focus

- Purpose: merge every error term into one ranked surface, and grade whether completed work actually moved its sensor.
- Primary archetype: workflow / ranking.
- Secondary traits: multiplexed sources, efficacy join.
- Owns: findings, their ranking, and the finding-to-sensor efficacy record.
- Sources are named and independently degradable — out-of-band readings, untrusted readings, uninstrumented targets, supervision gaps, and open-loop self-report. An unreadable source becomes a visible availability entry rather than silently removing findings.
- **Surfaces, does not decide.** Ranking is a recommendation. Whether to repair, defer, or retire stays with the team and the operator, and no path in this scenario actuates anything.
- Actuation efficacy implements update-protocol rule 5 of `RELIABILITY_TARGETS.md`: a finding names its sensor and expected in-band return, the next read after the work lands re-grades it, and a fix that does not move the sensor re-opens the finding. The sensor grades the fix, not its author.
- API: `api/internal/focus/`, `api/handlers/focus/`.
- CLI: `focus next`, `focus show <id>`, `focus efficacy`.
- Storage: domain-owned SQLite findings and efficacy records.
- Requirements: `IFM-P0-004`, `IFM-P0-005`, `IFM-P1-003`.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `watchdog-supervision` | The plant does not exist yet. A watchdog tier — liveness of autoheal and the core set, on a seconds clock, with an enumerated action set — is a separate operator decision precisely because it is the only piece that would touch a running system. Modelling its supervision now would inflate the open-loop count with a hole nobody can close. | The operator authorizes a watchdog tier. Then this becomes a domain here (`OT-P2-003`), supervising its liveness, enumerated-action counts, and claim-suppression rate. |
| `capability-availability` | Owner-side work, not ours. `INSTRUMENTATION_ROADMAP.md` Gap 11 places availability persistence with each capability owner, and building it centrally would recreate the roster rule 6 forbids. The `readings` domain carries an interim read-time reachability proxy, flagged `estimate` with an expiry. | Capability owners persist queryable availability aggregates. The proxy is then retired in favour of owner-derived history (`OT-P2-002`). |
| `platform-code-audit` | Judgment-shaped, not measurement-shaped. Cross-platform debt and internal platform-code quality belong to the `platform-code-auditor` member's lane; forcing them into a coverage ratio would repeat the mistake the production-ledger archetype exists to prevent. | Only if a defensible denominator for platform-code conformance is ever authored. Absent that, this stays prose. |
| `remediation` | Out of scope by contract, permanently. Operating-model rule 3 is "supervise, don't operate", and live incident response belongs to autoheal, system-monitor, agent-manager and the operator. | Never. A remediation domain here would make the instrument a controller — the boundary violation `TARGET_MODEL.md` names as the whole reason sensors carry no authority. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`SETPOINT-MODEL.md`](SETPOINT-MODEL.md) — what a target is, where the setpoint lives, and why this scenario may not author it
- [`TRUST-MODEL.md`](TRUST-MODEL.md) — the trust axis: when a reading is evidence and when it is instrument fault
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy

Upstream canon this scenario implements (cited, never restated):

- `docs/agent-system/TARGET_MODEL.md` — the instrument contract and the deviation catalogue
- `docs/infra-health/strategy/RELIABILITY_TARGETS.md` — the setpoint itself
- `docs/infra-health/operating/OPERATING_MODEL.md` — the plant's layer map and routing rules
- `docs/infra-health/evidence/INSTRUMENTATION_ROADMAP.md` — the gaps behind every empty sensor cell
