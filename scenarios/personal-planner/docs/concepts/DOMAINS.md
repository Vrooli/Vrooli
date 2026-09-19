# Domains — Personal Planner

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The domain set below is derived from the approved implementation plan
(the *Adaptive Personal Planner* specification): its domain concepts
(plan §4), logical entity catalog (plan §19.2), and module boundaries
(plan §22.1). The single most load-bearing rule for this product is
**one authoritative owner per mutable fact** (INV-02): the planner owns
native planning records, accepted allocations, availability, private
activity, forecast snapshots, proposals, and sharing projections; source
apps own their content and authoritative domain status; external calendar
providers own their imported events. Imported and referenced objects are
*projections* that retain origin identity and revision.

`health` is the one real infrastructure domain the scaffold ships. The
scaffold also ships one clearly fenced worked example domain (`notes`,
never product scope) as a copyable reference; `template-manager
detemplate personal-planner` removes every fenced example once the first
real domain is green. The product domains below are the target map to
build against — their source paths describe intended structure, not
files that already exist.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md). Dependency contracts belong in
[`INTEGRATIONS.md`](INTEGRATIONS.md).

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/personal-planner/v1/shared/health.proto` |
| workspace | Bind the authenticated subject to one private workspace and own the planning profile, availability rules/exceptions, and appearance preferences. | Establish tenant isolation, timezone/week-start, capacity policy, and the day/night appearance model from the first session. | Workspace, planning profile, availability rule, availability exception, preference records. | service | crud, validation | Workspace, PlanningProfile, AvailabilityRule, AvailabilityException | `api/internal/workspace/`, `ui/src/features/workspace/`, `packages/proto/schemas/personal-planner/v1/workspace/` |
| work | Own native work items, effort estimate/remaining history, definition of done, quick capture, and the unscheduled backlog. | Let users capture and edit the units of human effort that planning schedules, keeping estimate/remaining/actual distinct. | Work item, effort revision, dependency edge, capture inbox entries. | service | crud, mutation | WorkItem, EffortRevision, DependencyEdge, Backlog | `api/internal/work/`, `cli/domains/work/`, `ui/src/features/work/`, `packages/proto/schemas/personal-planner/v1/work/` |
| calendar | Own native fixed events, routines (fixed recurrence + flexible frequency), routine occurrences, schedule allocations, and the temporal model. | Hold the authoritative accepted schedule and the hybrid fixed/timed/date-assigned/backlog placement model without double-counting demand. | Native event, routine definition, routine occurrence, allocation (timed + date). | service | mutation, validation | Event, Routine, RoutineOccurrence, Allocation, TemporalExtent | `api/internal/calendar/`, `cli/domains/calendar/`, `ui/src/features/calendar/`, `packages/proto/schemas/personal-planner/v1/calendar/` |
| goals | Own goals, lightweight initiatives/references, and evidence-based milestones. | Connect time to intended outcomes and observable checkpoints without treating elapsed hours as achievement. | Goal, initiative reference, milestone. | service | crud | Goal, Initiative, Milestone | `api/internal/goals/`, `cli/domains/goals/`, `ui/src/features/goals/`, `packages/proto/schemas/personal-planner/v1/goals/` |
| commitments | Own explicit promises, their revision/acknowledgment history, and definition-of-done/beneficiary metadata. | Preserve original and renegotiated promises so a forecast can never silently rewrite what was promised. | Commitment, commitment revision. | service | mutation, validation | Commitment, CommitmentRevision, Acknowledgment | `api/internal/commitments/`, `cli/domains/commitments/`, `ui/src/features/commitments/`, `packages/proto/schemas/personal-planner/v1/commitments/` |
| capacity | Compute interval-aware capacity: usable intervals, union of exclusions, reserves counted once, demand, fragmentation, and unknown-effort honesty. | Answer how much usable time exists and whether work fits, using the same values the UI, text, and API display. | No product tables (derived from workspace/calendar/work). | aggregation | scoring, query | Capacity, Reserve, AllocatableBudget, Fragmentation | `api/internal/capacity/`, `ui/src/features/capacity/`, `packages/proto/schemas/personal-planner/v1/capacity/` |
| planning | Own the deterministic scheduling kernel, immutable proposals, proposal applications, and side-effect-free what-if scenarios. | Produce explainable candidate placements and apply them atomically after revalidation, never mutating accepted records during generation. | Proposal, proposal application. | orchestration | service, validation | Proposal, ProposalApplication, WhatIf, ScheduleRevision | `api/internal/planning/`, `cli/domains/planning/`, `ui/src/features/planning/`, `packages/proto/schemas/personal-planner/v1/planning/` |
| forecasts | Own forecast snapshots over the shared personal resource model, risk classification, and compact change explanations. | Show explainable central/cautious completion outlooks and why a forecast changed, without reserving time or altering the schedule. | Forecast snapshot, change record. | scoring | reporting | Forecast, Scenario, RiskState, ChangeRecord | `api/internal/forecasts/`, `ui/src/features/forecasts/`, `packages/proto/schemas/personal-planner/v1/forecasts/` |
| focus | Own durable focus sessions, session segments, actual-activity records, and corrections. | Track working sessions and real elapsed effort as facts distinct from scheduled time and task completion. | Focus session, session segment, manual actual, correction history. | service | mutation | FocusSession, SessionSegment, Actual, Correction | `api/internal/focus/`, `cli/domains/focus/`, `ui/src/features/focus/`, `packages/proto/schemas/personal-planner/v1/focus/` |
| review | Own daily/weekly review read models and conservative, evidence-linked learning insights. | Turn recorded history into descriptive review and inspectable setting suggestions the user explicitly accepts. | Review record, learning insight. | aggregation | reporting, classification | Review, LearningInsight, Coverage | `api/internal/review/`, `cli/domains/review/`, `ui/src/features/review/`, `packages/proto/schemas/personal-planner/v1/review/` |
| integrations | Own source registration/projection contracts, read-only external calendar providers, imported events, and ICS import/export. | Bring authoritative external obligations into capacity as revisioned read-only projections with honest freshness. | Source registration, source object projection, provider connection, provider calendar, imported event. | provider | service, mutation | SourceRegistration, SchedulingIntent, ProviderConnection, ImportedEvent | `api/internal/integrations/`, `cli/domains/integrations/`, `ui/src/features/integrations/`, `packages/proto/schemas/personal-planner/v1/integrations/` |
| sharing | Own share grants and the server-side viewer projection layer. | Let owners expose selected commitment fields to a recipient-bound viewer without leaking private data. | Share grant, viewer projection. | provider | validation, query | ShareGrant, FieldMask, ViewerProjection | `api/internal/sharing/`, `cli/domains/sharing/`, `ui/src/features/sharing/`, `packages/proto/schemas/personal-planner/v1/sharing/` |
| notifications | Own notification intents and preferences, publishing delivery through notification-hub. | Send a small number of useful, deduplicated notices with quiet hours and safe content, never leaking private reasons. | Notification intent, notification preference. | orchestration | service | NotificationIntent, Preference, DedupKey | `api/internal/notifications/`, `ui/src/features/notifications/`, `packages/proto/schemas/personal-planner/v1/notifications/` |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships `notes` as a worked CRUD vertical slice with a binary
upload exception. Copy its shape for the real domains above, then remove it.

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| notes | Provide the worked CRUD reference with attachment upload exception. | Demonstrate the expected vertical slice for a real domain. | Notes and attachment metadata. | crud | service | Note, Attachment | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/personal-planner/v1/notes/` |

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
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

### workspace

- Purpose: bind the authenticated subject to one private workspace and
  own the planning profile, availability, and appearance preferences.
- Primary archetype: service (with crud + validation traits).
- Owns: workspace identity binding; planning profile (person/resource
  identity, capacity caps, reserve policy, focus preferences); availability
  rules (weekday/date, local intervals, timezone, effective dates,
  priority); availability exceptions (protected/extra time); appearance
  settings (Auto/Day/Night policy and transition hours).
- Does not own: identity itself (owned by scenario-authenticator);
  schedule content (owned by `calendar`).
- Invariants enforced: INV-01 (all reads/writes scoped to the authorized
  subject); profile version changes invalidate affected proposals;
  protected time is never silently relaxed.
- Build order: P01 (foundation) and P04 (availability/reserve accounting).
- Related docs: [`DATA.md`](DATA.md), [`FLOWS.md`](FLOWS.md).

### work

- Purpose: own native work items and the effort model that planning
  schedules.
- Primary archetype: service (crud + mutation).
- Owns: work item (title, definition of done, status, priority, context,
  optional initiative/milestone link); effort revision (original / current
  / remaining, minutes or range, author, method); dependency edges
  (finish-to-start with optional lag, cycle-rejected); quick-capture inbox
  and the unscheduled backlog.
- Distinctions preserved: original estimate, revised estimate history,
  explicit remaining effort, and actual activity never collapse into one
  field (plan §4.2, INV-03).
- Does not own: source-app content (referenced as a projection through
  `integrations`); scheduled placement (owned by `calendar`).
- Build order: P01 (capture/edit) and P03 (dependencies).
- Related docs: [`DATA.md`](DATA.md).

### calendar

- Purpose: own the authoritative accepted schedule and the temporal model.
- Primary archetype: service (mutation + validation).
- Owns: native fixed events (timed or all-day, timezone, busy/free effect,
  recurrence reference); routine definitions (fixed recurrence *and*
  flexible frequency as distinct variants); routine occurrences with
  stable original-occurrence identity; allocations (timed with instants,
  or date-level with local date + effort quantity), each accepted/draft/
  cancelled with links to a work item or routine occurrence; temporal
  normalization (Instant/CivilDate/LocalTime/TimeZoneId, DST policy).
- Invariants enforced: INV-07 (date-assigned work consumes capacity, a
  linked timed session must not count twice); INV-13 (routine exceptions
  keep stable occurrence identity); exactly one temporal representation
  per event.
- Does not own: capacity math (owned by `capacity`); imported provider
  events (owned by `integrations`, referenced here as read-only busy
  facts).
- Build order: P02.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md).

### goals

- Purpose: connect time to outcomes and observable checkpoints.
- Primary archetype: service (crud).
- Owns: goals (name, purpose, progress method — manual/metric/milestone,
  baseline/target/observed values); lightweight native initiatives or
  references to a project owner; milestones (observable completion
  criteria, linked prerequisite work, optional target/commitment, derived
  or manual completion).
- Distinctions preserved: linked hours are effort exposure, not automatic
  outcome progress; a milestone does not consume time — its work does.
- Build order: P03.
- Related docs: [`DATA.md`](DATA.md).

### commitments

- Purpose: preserve explicit promises and their full history.
- Primary archetype: service (mutation + validation).
- Owns: commitments (promised result or availability, definition of done,
  promised boundary + timezone, optional beneficiary label, assumptions,
  scope exclusions, lifecycle proposed/active/fulfilled/cancelled);
  commitment revisions (original + every renegotiation, accepted-by,
  reason, optional acknowledgment status).
- Invariants enforced: INV-04 (forecasts/proposals cannot alter accepted
  commitment revisions); acknowledgment is unknown until actually recorded.
- Does not own: risk assessment (derived by `forecasts`); the schedule
  (owned by `calendar`).
- Build order: P03 (create/accept/revise) and P05 (risk/forecast link).
- Related docs: [`FLOWS.md`](FLOWS.md).

### capacity

- Purpose: compute how much usable time exists and whether work fits.
- Primary archetype: aggregation (scoring + query).
- Owns: no product tables; derives an interval set and scalar summaries
  from workspace availability, calendar busy/allocations, and work demand
  (plan §10). Overlapping busy events subtract their *union*; reserve is
  counted exactly once; remaining-today clips at now; unknown estimates
  are listed separately.
- Invariants enforced: INV-07, INV-08 (unknown effort/state never silently
  becomes zero), INV-16 (UI geometry, text, and API describe the same
  values).
- Build order: P04.
- Related docs: [`DATA.md`](DATA.md), [`FLOWS.md`](FLOWS.md).

### planning

- Purpose: propose and apply schedule changes deterministically.
- Primary archetype: orchestration (service + validation).
- Owns: immutable proposals (scope, base revisions/fingerprint,
  assumptions, candidate operations, unresolved work, expiry); proposal
  applications (one per proposal, atomic local write set, undo relation);
  the pure scheduling kernel and the five R1 what-if categories.
- Invariants enforced: INV-04, INV-06 (fixed commitments/protected time/
  source constraints/locks constrain placement), INV-11 (apply
  revalidates all inputs and commits atomically), INV-15 (forecast/
  proposal allocation is hypothetical, never accepted reserved time).
- Does not own: the accepted schedule rows it mutates (owned by
  `calendar`); source-domain changes (routed to `integrations`).
- Build order: P05.
- Related docs: [`FLOWS.md`](FLOWS.md).

### forecasts

- Purpose: show explainable completion outlooks and why they changed.
- Primary archetype: scoring (reporting).
- Owns: immutable forecast snapshots (subject/scope, input fingerprint,
  model version, generated time, central/cautious scenarios, limitations)
  and compact change records between successive material forecasts.
- Invariants enforced: INV-04, INV-15; no accepted allocations created;
  R1 uses labeled scenarios, never statistical probabilities.
- Build order: P05.
- Related docs: [`FLOWS.md`](FLOWS.md).

### focus

- Purpose: record working sessions and real elapsed effort as facts.
- Primary archetype: service (mutation).
- Owns: focus sessions (person identity, work reference, state, mode,
  device metadata, revision); session segments (work/break/interruption/
  unclassified with correction provenance); manual actuals (duration or
  interval, precision, reported-by); correction history.
- Invariants enforced: INV-03 (passing scheduled time, running a timer,
  and completing a task are separate events); INV-09 (at most one running
  exclusive session per person; breaks/pauses are not active effort).
- Build order: P06.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md).

### review

- Purpose: turn history into descriptive review and inspectable insights.
- Primary archetype: aggregation (reporting + classification).
- Owns: daily/weekly review records (confirmed corrections, carry-forward
  choices, optional reflection) and learning insights (rule/model version,
  eligible sample, evidence IDs, recommendation, status).
- Invariants enforced: INV-14 (learning never silently rewrites promises,
  hard constraints, or historical measurements); completed-only samples
  disclose their limitation.
- Build order: P07.
- Related docs: [`FLOWS.md`](FLOWS.md).

### integrations

- Purpose: bring authoritative external state in as read-only projections.
- Primary archetype: provider (service + mutation of projections only).
- Owns: source registrations (trusted app identity, capabilities, allowed
  actions, credential reference); source object projections (origin
  identity, revision, normalized kind, freshness, constraints); provider
  connections and calendars (opaque account ID, credential reference,
  cursor/query scope, busy policy, freshness); imported events (provider/
  series identity, occurrence identity, revision, tombstone); ICS import/
  export jobs.
- Invariants enforced: INV-02 (projections retain origin + revision),
  INV-10 (replayed events cannot create duplicate projections); external
  calendars are strictly read-only in R1; secrets live in the credential
  owner, never in records/logs/exports.
- Build order: P08.
- Related docs: [`INTEGRATIONS.md`](INTEGRATIONS.md), [`FLOWS.md`](FLOWS.md).

### sharing

- Purpose: expose selected commitment fields without leaking private data.
- Primary archetype: provider (validation + query).
- Owns: share grants (owner, recipient-bound viewer identity, explicit
  record set, field mask, expiry, revocation, version) and the server-side
  viewer projection layer that builds shared DTOs from the allowlist.
- Invariants enforced: INV-12 (shared views expose only explicitly allowed
  fields and records, including nested data and derived explanations);
  revocation applies on the next authorized fetch.
- Build order: P09.
- Related docs: [`FLOWS.md`](FLOWS.md), [`../internal/SECURITY.md`](../internal/SECURITY.md).

### notifications

- Purpose: send a small number of useful, deduplicated notices.
- Primary archetype: orchestration (service).
- Owns: notification intents (recipient scope, type, dedup key, relevance/
  expiry, deep link, minimum safe content) and per-type/channel
  preferences with quiet hours and lead times; publishes delivery through
  notification-hub and keeps delivery receipts with that owner.
- Invariants enforced: INV-10 (idempotent retry never sends duplicate
  human-facing messages); a private reason never reaches a shared viewer.
- Build order: P09.
- Related docs: [`INTEGRATIONS.md`](INTEGRATIONS.md).

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |
| Projection | A read-only copy of an external authoritative fact carrying origin identity + revision. | `integrations` / `sharing`. |
| Allocation | An accepted or draft reservation of effort (timed or date-level), separate from task identity, estimate, and actuals. | `calendar`. |
| Snapshot | An immutable derived result (forecast/proposal) computed from a fingerprinted input set. | `planning` / `forecasts`. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| collaboration | Household/team editing, shared availability aggregation, and assignment are an R3 product expansion; R1 is single-user (D03). | When OT-P2-005 is scheduled and multi-subject availability semantics are designed. |
| assistant | A conversational planning interface over the same bounded commands is R2; the core loop must work with no model. | When OT-P2-002 is scheduled and the command surface is stable. |
| calibration | Statistical, calibrated probability forecasts need enough data, a documented method, and held-out evaluation (R2). | When OT-P2-001 is scheduled and adequate history exists. |
| billing | Commercial metering/subscription is R3, gated behind proven retention. | When monetization moves past the free personal product. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/temporal/` — shared timezone/interval/recurrence helpers used by many domains (a library, not an owner).
- `api/internal/command/` — shared command envelope (actor/workspace/idempotency/revision) used by every domain.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/layout/` — library shell configuration (navigation data, mark, shell settings).
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
