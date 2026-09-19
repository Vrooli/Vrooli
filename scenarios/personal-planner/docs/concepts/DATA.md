# Data — Personal Planner

## Purpose Of This Document

This document is the canonical map of what data Personal Planner stores,
which domain owns each table, how temporal and effort values are typed,
how the data migrates, imports, exports, is retained, and is deleted, and
which privacy rules govern it. It is the storage companion to
[`DOMAINS.md`](DOMAINS.md) (ownership) and [`FLOWS.md`](FLOWS.md)
(lifecycle). The persistence contract is derived from the implementation
plan §19 (persistence model), §19.3 (temporal/numeric types), §23.4–23.5
(export/restore, deletion/retention).

The single most important storage rule is **one authoritative owner per
mutable fact** (INV-02). Read models are disposable projections, never a
second authority. Facts, estimates, forecasts, user reports, and AI
suggestions are always distinguishable in storage, not merged for
convenience.

## Storage Overview

- **Engine:** the scenario-owned SQLite substrate is the default. Schema
  is embedded next to the code that interprets it (per-domain
  `schema.sql` + `SchemaProvider`/`EnsureSchemas`), and repository
  interfaces hide the engine from services. A relational resource (e.g.
  Postgres) is adopted only if measured scale or a concurrency guard that
  SQLite cannot express (see the partial-unique focus-session constraint
  below) justifies it — recorded through the Scenario Dependency Analyzer,
  never hand-edited (see [`INTEGRATIONS.md`](INTEGRATIONS.md)).
- **State + history:** each domain stores authoritative current state
  plus a revisioned history of meaningful edits, and an append-only
  change/outbox log that supports explanations and reliable event
  delivery. Rebuilding every entity from events is **not** required.
- **No exotic stores for R1:** no graph database, distributed
  event-sourcing platform, or analytical warehouse. Dependencies are an
  ordinary directed graph of edge rows; forecasts and learning are
  bounded queries plus background jobs (plan §19.1).
- **Derived domains own no tables:** `capacity` is computed on demand;
  `forecasts` and `planning` persist only immutable snapshots, never
  accepted schedule rows.

## Data Ownership

Every logical entity has exactly one owning domain. The table below maps
the plan's logical entity catalog (§19.2) to this scenario's domains.
Names are logical entities, not a mandate to create one physical table
per row when a normalized representation is simpler.

| Entity | Owning domain | Integrity / lifecycle notes |
|---|---|---|
| Workspace | workspace | Ownership comes from auth; no client-supplied owner reassignment. |
| Planning profile | workspace | Version changes invalidate affected proposals; accepted setting history preserved. |
| Availability rule / exception | workspace | Date exceptions override normal rules deterministically; protected time is never silently relaxed. |
| Appearance preference | workspace | Manual Auto/Day/Night override persists per user; theme never alters scheduling timezone or data. |
| Native work item | work | Simple tasks need no initiative; revision-checked edits. |
| Effort revision | work | Original estimate remains identifiable; unknown is nullable, never zero. |
| Dependency edge | work | Same-workspace scope, no self-edge, no cycle; unknown release stays unknown. |
| Goal | goals | Manual outcome updates and work-derived indicators are visibly distinct. |
| Initiative reference | goals | Referenced source content is not copied into a second editable project system. |
| Milestone | goals | Date-only checkpoints carry timezone and an explicit deadline-boundary policy. |
| Native event | calendar | Exactly one temporal representation per event (timed **or** all-day). |
| Routine definition | calendar | Fixed recurrence and flexible frequency are separate variants. |
| Routine occurrence | calendar | Moving an occurrence keeps its stable original identity. |
| Allocation | calendar | Timed/date assignments conserve demand; a cancelled allocation is not a completed task. |
| Commitment | commitments | Risk is a separate derived value; recipient identity is not inferred from a free-text label. |
| Commitment revision | commitments | Original and renegotiated promises preserved; acknowledgment recorded only if actually supplied. |
| Proposal | planning | Draft/applied/rejected/expired/superseded; immutable after apply. |
| Proposal application | planning | One per proposal; atomic local write set; undo relation. |
| Forecast snapshot | forecasts | Immutable derived snapshot; no accepted allocations created. |
| Change record | forecasts | Separates observed change (evidence) from reported cause (interpretation). |
| Focus session | focus | At most one running exclusive session per person; bounded device metadata. |
| Session segment | focus | No negative intervals; exclusive work segments cannot overlap for one person. |
| Manual actual | focus | Duration-only entries do not invent clock times; overlaps require resolution. |
| Review | review | Completing a review is never required to use the next day. |
| Learning insight | review | Inspect/accept/dismiss/expire; accepting creates a separate setting command. |
| Source registration | integrations | Installation is privileged; an ordinary task cannot register callbacks. |
| Source object projection | integrations | Unique origin identity; only the verified adapter updates source-owned fields. |
| Provider connection / calendar | integrations | Secrets stored in the credential owner, not UI records or exports; one cursor per query scope. |
| Imported event | integrations | Identity is provider/series/occurrence, not title+start matching; deletion follows provider semantics. |
| Share grant | sharing | Revocable; no recursive grants through linked private objects. |
| Notification intent / preference | notifications | Current permission + preference checks before delivery. |
| Integration receipt / outbox record | integrations / cross-cutting | Replays return the prior outcome or safely resume; retained until safely delivered. |
| Import / export job | cross-cutting | No credentials or reusable share tokens in exports; download authorization enforced. |

## Schema Map

Schemas live next to the domain that interprets them
(`api/internal/<domain>/schema.sql` + `schema.go`). The map below is the
target shape; it describes intended tables, not files that exist yet.

- **workspace:** `workspaces`, `planning_profiles`, `availability_rules`,
  `availability_exceptions`, `appearance_preferences`.
- **work:** `work_items`, `effort_revisions`, `dependency_edges`,
  `capture_inbox`.
- **calendar:** `events`, `routines`, `routine_occurrences`,
  `allocations` (with a `parent_allocation_id` lineage column so a
  date-level parent and its timed children share a conserved quantity).
- **goals:** `goals`, `initiative_refs`, `milestones`,
  `milestone_prerequisites`.
- **commitments:** `commitments`, `commitment_revisions`.
- **planning:** `proposals`, `proposal_operations`,
  `proposal_applications`.
- **forecasts:** `forecast_snapshots`, `change_records`.
- **focus:** `focus_sessions`, `session_segments`, `manual_actuals`,
  `actual_corrections`.
- **review:** `reviews`, `learning_insights`.
- **integrations:** `source_registrations`, `source_projections`,
  `provider_connections`, `provider_calendars`, `imported_events`,
  `integration_receipts`, `outbox`.
- **sharing:** `share_grants`.
- **notifications:** `notification_intents`, `notification_preferences`.

### Temporal and numeric typing

The plan mandates that these value kinds never collapse (§4.2, §19.3).
Even though the transport uses strings and integers, the domain models
them explicitly and validates at the boundary:

- `Instant` — RFC 3339 instant with offset, normalized for storage; used
  for real timed intervals (session start/end, timed allocations).
- `CivilDate` — `YYYY-MM-DD`, **never** parsed as a UTC-midnight instant;
  used for all-day events and date-level allocations.
- `LocalTime` + `TimeZoneId` (IANA) — for recurring wall-clock schedules.
- `Revision` — opaque equality token, **not** lexically sortable; never
  compared numerically/lexically to guess which is newer.
- `Effort` — a discriminated union of `unknown` / `point{minutes}` /
  `range{low,high}`; unknown is a first-class value, never coerced to
  zero (INV-08). Validate `low <= high` and a supplied central value
  within the range.
- `TemporalExtent` / `AllocationPlacement` — discriminated `timed` vs
  `all_day` / `date` variants; a fixed event uses exactly one.

Preserve seconds (or finer) for actual session timestamps and round only
for display or a documented scheduling grid (5-minute UI snapping by
default); never accumulate rounding error by rounding every timer tick.

### Revisions, indexing, and the one-session guard

- Maintain per-entity revisions and a **workspace planning revision** for
  mutations that affect planning. A proposal fingerprint is derived from
  the relevant entity/policy/source/freshness/algorithm-version inputs,
  not a hash of every field — a cosmetic label change need not invalidate
  a placement, but an estimate, date, dependency, availability, or busy
  interval change must.
- Recommended indexes: workspace + interval start/end (allocations/
  events); workspace + date (dated demand); source identity tuple;
  provider/calendar/remote-occurrence tuple; predecessor/successor edges;
  person + session state; pending job schedule; recipient + grant state.
- **One current exclusive focus session per person** is protected by a
  database constraint, not a preflight `SELECT`. Where Postgres is used,
  a partial unique index on person where `(state IN ('running','paused'))
  AND attention = 'exclusive'` is the final guard; under SQLite an
  equivalent transactional guard applies. This is the primary
  storage-level reason a relational resource could later be justified.

## Migrations And Compatibility

- Use the repository's established migration tooling. The three-tier
  migration story applies: **declarative greenfield** while there is no
  production data (the co-located `schema.sql` + `EnsureSchemas` is the
  source of truth); **out-of-tree scripts** for greenfield-with-data;
  **versioned migrations only** once real production schema evolution
  earns them.
- **Legacy note:** the unused, undeveloped `calendar` scenario was
  removed before this scenario was generated, so there is **no live
  scheduling data to migrate** at R0. If legacy or external data appears
  later, follow plan §3.3: back up and validate a restore first; produce
  a mapping for events, recurrences, reminders, task refs, timezones, and
  completion fields; use stable ID mappings, repeatable commands, dry-run
  counts, per-record errors, and a migration version; mark migrated
  provenance; and never treat old scheduled events as actual activity.
  Ambiguous timezone or completion status must produce an import issue,
  not an invented fact.

## Import / Export

- **Native versioned export (JSON):** format version, export time,
  timezone/policy metadata, stable IDs, provenance, native records,
  allocations, meaningful revisions, and actuals — enough to interpret
  and restore. **Excludes** credentials, access/session tokens, reusable
  share secrets, and unnecessary private infrastructure IDs. Source
  projections are included only under an explicit documented policy and
  retain their owner identity.
- **CSV summaries** for actuals/estimates default to current effective
  actuals; native backup exports include the selected history needed for
  restoration and label that choice.
- **ICS import/export** is a manual fallback distinct from a live
  connection: import previews with duplicate detection, provenance, and
  bounded recurrence expansion, and never fetches arbitrary URLs embedded
  in descriptions; export covers selected native events/accepted sessions
  with privacy choices and never exports private notes through a
  commitment-share route.
- **Native import** validates the whole package first (counts/conflicts/
  warnings + preview), defaults to importing into a **new private
  workspace**, and is idempotent — reimporting the same package does not
  duplicate objects. Restored connections require reconnection; restored
  shares are inactive until re-granted; a restored running session
  becomes "needs review," never assumed-productive elapsed work.

## Retention And Deletion

Defaults are engineering choices to align with the host, not legal
retention claims (plan §23.5, §28.1):

| Data | Default retention |
|---|---|
| User-authored planning history | Retained until the user deletes it. |
| Unapplied proposal payloads | 30 days unless explicitly saved. |
| Applied proposal changes | Remain in plan history after preview-payload cleanup. |
| Routine diagnostic logs | 30 days. |
| Delivered notification/job diagnostics | 30 days. |
| Interactive idempotency results | 7 days (durable entity uniqueness protects older replays). |
| Downloadable export artifacts | 24 hours (regenerable, authenticated). |

- Provide ordinary archive/delete with undo where useful, plus an
  explicit **permanent-deletion** path. Permanent deletion removes or
  anonymizes associated personal history and recomputes affected derived
  data (dependency, forecast, learning references), retaining only a
  minimal, disclosed, non-identifying structural tombstone where required
  for integrity.
- **Source receipt cleanup cannot allow an old replay to resurrect
  deleted work** — keep durable origin uniqueness/tombstones. Deletion
  documentation distinguishes removal from live views vs eventual backup
  expiry (backups follow the host policy).

## Privacy Notes

- INV-01: all reads, writes, jobs, projections, and caches are scoped to
  the authorized workspace and subject. Private titles and details never
  cross workspace boundaries merely to reserve time.
- INV-12: shared views expose only explicitly allowed fields and records,
  including nested data and derived explanations. A private event that
  delays a shared commitment yields "Available time changed," never its
  title, attendees, or location.
- Secrets (provider tokens, tokenized share URLs) live in the credential
  owner and never appear in records, logs, analytics, frontend bundles,
  or exports. Redact query parameters where tokens could appear.
- Imported calendar descriptions and task titles are untrusted text:
  sanitize or render as plain text; never treat them as instructions.
- Learning uses only the user's own authorized records; optional usage
  telemetry flows solely through the host consent model and is never a
  dependency of personal learning.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — capability ownership
- [`FLOWS.md`](FLOWS.md) — lifecycle and state transitions
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external data and resources
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — data sensitivity and threats
- [`../reference/configuration.md`](../reference/configuration.md) — storage configuration
