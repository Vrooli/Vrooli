# Integrations — Personal Planner

## Purpose Of This Document

This document is the canonical dependency contract for Personal Planner:
which Vrooli resources it uses, which scenarios it depends on, which
third-party services it integrates, and how each fails. It is the
authority the "Dependency Decisions" gate (START-HERE Gate 4) reconciles
against [`../../.vrooli/service.json`](../../.vrooli/service.json).
Configuration keys for every integration live in
[`../reference/configuration.md`](../reference/configuration.md).

The governing principle is **reuse over rebuild** (plan §3): use shared
Vrooli owners first, then a scenario CLI/resource, then a direct API.
Personal Planner is itself the **single scheduling authority** — other
scenarios submit scheduling intent and read accepted schedules through
its contracts; a shared database is never the integration API (plan §15).

## Dependency Inventory

| Dependency | Kind | Required | Startup policy | Purpose |
|---|---|---|---|---|
| SQLite (in-process) | Storage substrate | Yes | n/a (embedded) | Authoritative per-domain persistence. |
| scenario-authenticator | Scenario | Yes | must_start | User identity, private-workspace isolation, recipient-bound viewer identity for sharing. |
| notification-hub | Scenario | No | try_start | Reminder + risk-notice delivery with quiet hours and receipts. |
| Shared event transport | Platform | No | reuse | Domain event envelope + receipts for outbox delivery. |
| Credential storage owner | Platform | Conditional | reuse | Read-only external-calendar provider tokens; never in logs/exports. |
| External calendar provider (Google fallback) | Third-party | No (optional feature) | on-connect | Read-only import of external obligations into capacity. |
| Daily / Cadence / agent work | Scenario (consumers) | No | contract | Submit scheduling intent; read accepted schedule. Personal Planner is their dependency, not vice versa. |

No mandatory Vrooli *resource* (Postgres/Redis/Qdrant/Ollama) is declared
for R1. Any future resource is added only when measured need justifies it,
through the Scenario Dependency Analyzer.

## Vrooli Resources

- **SQLite (in-process):** the default storage substrate; no external
  resource process. See [`DATA.md`](DATA.md) for the schema model and the
  single documented condition (the partial-unique one-exclusive-session
  guard at scale) under which a relational resource could later be
  adopted.
- **No AI resource in R1:** the deterministic scheduling/capacity/forecast
  core requires no model. If an optional assistant is added (R2), it is
  obtained through the repository's role/router mechanism, treated as
  data, and never made a prerequisite of the core workflow (plan §18.3).
- **Third-party packages** (npm/go/pip) are governed by the Scenario
  Dependency Analyzer, not hand-edited into
  `.vrooli/dependencies/approved-dependencies.json`. Notably a mature
  timezone/recurrence library (RFC 5545 semantics) will be needed for the
  `calendar` domain and must be found + installed through SDA.

## Scenario Dependencies

### scenario-authenticator (required)

- **Why:** Personal Planner is multi-user with strictly private
  workspaces (D03); every read/write/job/projection/cache is scoped to
  the authenticated subject (INV-01). It also supplies the recipient-bound
  viewer identity that selective sharing needs — if guest identity is
  unavailable, the sharing gate stays explicitly blocked rather than
  silently downgrading to a public link (plan §17.2).
- **Contract:** resolve subject/workspace/capabilities from authenticated
  context; never trust a payload-supplied tenant. The API uses the passive
  shared authentication seam; a Cloudflare Access profile can be layered
  in `.vrooli/service.json` without changing domain code.
- **Degraded behavior:** the scenario refuses to start without an identity
  provider — a workspace cannot be scoped to a subject.

### notification-hub (optional)

- **Why:** deliver a small number of useful notices — upcoming fixed
  commitments, latest-safe-stopping-time, review reminders, material
  deadline-risk changes, provider reconnection, and failed queued owner
  updates — with per-type/channel controls, quiet hours, and lead times.
- **Contract:** the `notifications` domain emits an *intent* (recipient
  scope, dedup key, relevance/expiry, deep link, minimum safe content);
  delivery + receipts belong to notification-hub. Retries are idempotent
  and never produce duplicate human-facing messages; a private reason
  never appears in a message directed to a shared viewer.
- **Degraded behavior:** notices remain visible in-app only; no external
  channel delivery. The product stays fully usable.

### Daily / Cadence / agent work (consumers, not upstream deps)

These scenarios *consume* Personal Planner's shared scheduling contracts;
Personal Planner does not depend on them. Three surfaces are provided
(plan §15.1): a source registration/capability contract, a work/constraint
projection contract, and a schedule command/query contract. A source
submits a normalized `SchedulingIntent` (human_work / fixed_event /
marker / passive_wait, with effort ranges, timing windows, attention, and
split policy); the planner owns availability and reservations, while each
source owns its content, eligibility, and authoritative completion state.

Where a consumer scenario is absent or lacks an API, ship the **exact
contract + a fixture adapter** and keep that live-integration gate
explicitly open — do not fabricate a live integration or build an
unrelated substitute source (plan §15.8, §21.4). Examples:

| Source | Submits | Planner response | Source stays responsible for |
|---|---|---|---|
| Daily | Meal-prep effort, meal window, batch prerequisites | Accepted prep allocation or alternatives | Recipe, dietary rules, ingredients, consumption status. |
| Cadence | Draft/review work, content milestone, publication marker | Human work sessions + conflict/forecast info | Draft body, channel versions, publication status. |
| Agent work | Review/check-in tasks + passive run milestones | Human attention reservations + nonblocking run markers | Agent execution, run results, artifacts. |

A dinner with 10 minutes active prep and 30 minutes unattended cooking is
**not** 40 minutes of exclusive attention; a publish-at-09:00 automation
may reserve zero human minutes while its review reserves 20 earlier. The
adapter supplies that distinction explicitly.

## Third-Party Services

### Read-only external calendars (optional feature, R1)

- **Provider:** provider-neutral adapter interface; **Google Calendar is
  the recommended first implementation/fallback target**, not an assertion
  the user uses it. Microsoft Graph and Apple/CalDAV are later adapters.
- **Access:** strictly **read-only**, enforced at both requested scopes
  and adapter implementation — there is no provider mutation path behind
  the UI in R1. Tokens are stored in the credential owner and never pasted
  into logs, bundles, exports, or docs.
- **Sync model:** adapters list calendars, do initial + incremental reads,
  normalize identities/time semantics, report deletions, refresh
  credentials through the owner, and expose health. One durable cursor per
  connection/calendar/query scope, advanced only after a batch is durably
  processed. Google's incremental flow (full read → retained sync token →
  changes incl. deletions; invalidated tokens force a scoped full
  resync) and Graph's calendar-view delta model are followed per their
  current official docs; a moved recurring instance keeps a stable
  original-instance key (never merged into a second appointment).
- **ICS** import/export is a separate manual fallback (see
  [`DATA.md`](DATA.md)).

The first provider adapter seam now exists in
`api/internal/integrations/google.go`. It is read-only, accepts a credential
from the credential owner, paginates calendar and event reads, normalizes
timed/all-day/transparent events, captures the final incremental sync token,
and reports HTTP 410 token invalidation as a scoped full-reset signal. The
adapter is covered by HTTP fixtures; it is not yet wired to OAuth, durable
cursor storage, or a live provider account.

## Failure Modes

| Dependency | Failure | Behavior |
|---|---|---|
| scenario-authenticator | Unavailable | Scenario does not start; no workspace can be scoped. |
| notification-hub | Unavailable/degraded | Notices remain in-app; delivery retried idempotently; no duplicate messages. |
| Credential owner | Token expired/revoked | `PROVIDER_REAUTH_REQUIRED`; last-known busy facts retained with a reconnect path. |
| External provider | Outage / rate limit | Bounded exponential backoff with jitter; retain stale busy intervals with a freshness warning; feasibility marked uncertain. Never treat provider failure as new free time. |
| External provider | Cursor invalidated | Scoped full resync into a replacement snapshot, switched in only after success; native tasks/schedules/history untouched. |
| Consumer source (Daily/Cadence) | Absent or no API | Ship contract + fixture adapter; keep the live-integration gate open; local manual use stays complete. |
| Consumer source | Constraint revision changed mid-plan | Apply detects the mismatch, leaves accepted allocations unchanged, and requests refreshed inputs (`SOURCE_CONSTRAINT_STALE`, fixture F08). |
| SQLite | Corruption / write failure | Command returns a failed state; user input preserved with retry/export; never claims saved. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — domains that own each integration
- [`DATA.md`](DATA.md) — projections, cursors, import/export
- [`FLOWS.md`](FLOWS.md) — source/provider/notification flows
- [`../reference/configuration.md`](../reference/configuration.md) — configuration keys for every integration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — credential and token handling
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md) — operating integrations
