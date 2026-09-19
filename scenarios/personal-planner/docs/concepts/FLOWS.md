# Flows — Personal Planner

## Purpose Of This Document

This document is the canonical map of Personal Planner's user and system
workflows: their lifecycle states, transitions, retries, cancellation,
and the stale-completion risks that make honest planning hard. It is the
temporal companion to [`DOMAINS.md`](DOMAINS.md) (who owns what) and
[`DATA.md`](DATA.md) (what persists). Flows are derived from the
implementation plan §5 (journeys), §11 (planning/proposals), §13 (focus
sessions), §14 (review), and §25.3 (property invariants).

The through-line for every flow is that **facts, estimates, forecasts,
user reports, and AI suggestions stay distinguishable**, and three events
that products routinely conflate are kept separate (INV-03): *passing
scheduled time*, *running a timer*, and *completing a task*.

## Flow Inventory

| Flow | Trigger | Owning domain(s) | Writes accepted state? | Key risk it guards |
|---|---|---|---|---|
| First-use onboarding | New workspace | workspace | Yes (profile) | Inventing availability or fake commitments. |
| Quick capture | User adds a task | work | Yes (backlog) | Coercing a name-only task into false certainty. |
| Hybrid day planning | Assign/place work | calendar, capacity | Yes (allocations) | Double-counting demand on split/move. |
| Proposal → apply | User requests a plan | planning | Only on apply | Applying a stale proposal under new assumptions. |
| What-if | User explores a change | planning, forecasts | No | Side effects leaking into sources/commitments. |
| Forecast refresh | Material input change | forecasts | No | Presenting a stale forecast as current. |
| Commitment lifecycle | User promises/renegotiates | commitments | Yes | A forecast silently rewriting a promise. |
| Focus session | Start focus | focus | Yes (session/actuals) | Timer end interpreted as task completion. |
| Daily/weekly review | End of day/week | review | Yes (review, insights) | Treating unrecorded time as productive/zero. |
| Source intent ingestion | Source app submits intent | integrations, calendar | Yes (projection) | Duplicate/out-of-order events; source owning a second schedule. |
| Provider calendar sync | Poll / change notice | integrations | Yes (imported events) | A failed refresh read as newly free time. |
| Selective sharing | Owner grants a view | sharing | Yes (grant) | Private causes leaking to a viewer. |
| Notification dispatch | Due intent | notifications | Yes (intent state) | Duplicate or private-content delivery. |
| Export / import | User portability action | cross-cutting | Yes (new workspace) | Reactivating shares/credentials/timers on restore. |

## Flow Details

### First-use onboarding (plan §5.2)

Five short, skippable steps: confirm timezone + week start (detected as a
*suggestion*, not asserted); establish typical availability and protected
periods; ask what the user wants help with (overload, estimating,
starting, commitments, understanding time — priorities, not diagnoses);
offer read-only calendar connection and source discovery; capture one
meaningful task and produce a small **draft** day. The first draft
identifies assumptions and is applied only when the user explicitly
chooses "Apply to my day." A fresh workspace contains no fictional
commitments; an explicitly separate example workspace exists for demos.

### Quick capture (plan §7.2)

A name-only task is valid. Optional inline fields: estimate, desired day,
goal/project, due boundary, source link. Natural-language capture may
parse "draft tomorrow for 45 minutes" but shows interpreted fields before
any material change and works with no model. The inbox/backlog implies no
promise and consumes no capacity; moving an item to a day turns it into
planned demand (unestimated if no estimate), which lowers confidence in
that day's fit claim.

### Hybrid day planning (plan §8, §10.2)

Fixed events, timed sessions, date-level effort, and backlog coexist.
Splitting a 90-minute date allocation into a 30-minute timed session
leaves 60 minutes on the date; the timed session is a **child** of the
same planning demand (conserved quantity), not a clone (INV-07). Dragging
a flexible session creates a local preview with conflicts/ripples;
applying an isolated move never silently moves other accepted items.
Read-only provider events cannot be dragged into a mutation.

### Proposal → apply (plan §11.1–11.4)

`createProposal(snapshot, scope)` runs the deterministic kernel over a
validated, fingerprinted snapshot and returns candidate operations,
unresolved demand, conflicts, capacity/forecast deltas, and reason codes
— **without writing accepted records**. `apply` is a separate command
carrying the proposal ID, expected base revision/fingerprint, selected
operation IDs, and an idempotency key. Apply reloads the stored proposal,
revalidates the selected subset, commits local allocation changes + plan
revision + history + outbox atomically (INV-11), and returns the new
revision with undo availability. A materially stale proposal returns a
structured `PROPOSAL_STALE` conflict with reasons and a refresh action —
never a partial apply under new assumptions.

### What-if (plan §11.5)

Five R1 categories — add a task, reduce scope/remaining, move a target,
protect a day, change available hours — run inside a draft scenario
compared against the accepted baseline. They send no notifications and
write nothing to sources, commitments, shares, or the schedule. A
reduced-scope scenario that would change source-owned scope routes that
change to the source or requires the user to make it there.

### Forecast refresh (plan §12)

Forecasts derive every competing accepted item from **one** shared
personal resource snapshot, so independent projects cannot each claim the
same free hours (fixture F06). A forecast carries generated time,
freshness, input fingerprint, horizon, algorithm version, assumptions,
result state, and explanation. Central and cautious scenarios use
explicit effort ranges/assumptions and are labeled scenario ranges, never
"80% confidence." Refresh coalesces; while recomputing, the old forecast
is labeled "updating," never silently shown as current.

### Commitment lifecycle (plan §9.3, §12.2)

Lifecycle states: `proposed → active → fulfilled | cancelled`. Risk is a
*separate* assessment, and renegotiation is a *revision event*, not a
competing lifecycle state. The store keeps the original promise and every
revision; a forecast update is a derived event that never mutates
`promised_finish` or acceptance. Acknowledgment by another person is
recorded only if actually supplied, and is shown as unknown otherwise.

### Daily/weekly review (plan §14)

Review shows planned vs recorded activity with unrecorded periods
explicitly visible, asks only about material differences, and always
allows Skip. Unfinished work does not become an endless overdue queue;
mass rollover offers a capacity preview. Learning begins as descriptive
history; a recommendation changes a setting only on explicit acceptance,
which is recorded as an auditable setting command (INV-14).

### Source & provider flows (plan §15–16)

A source app submits normalized `SchedulingIntent`; the planner resolves
workspace/actor from authenticated identity (never the payload), validates
it, and stores a revisioned projection. Replays update the same object
(INV-10). Provider sync follows the provider's incremental model (full
read → durable cursor → changes incl. deletions), advances the cursor
only after a batch is durably processed, and on failure retains
last-known busy state with a visible freshness warning (fixture F09).

### Sharing & notifications (plan §17–18)

Sharing builds server-side DTOs from an allowlist bound to a recipient
identity; revocation applies on the next authorized fetch. Notifications
emit an intent (recipient scope, dedup key, relevance/expiry, deep link,
minimum safe content) through notification-hub; retries are idempotent and
never send duplicate human-facing messages, and a private reason never
reaches a shared viewer.

## State Machines

### Focus session (plan §13.3)

| State | Allowed next states | Segment treatment |
|---|---|---|
| Created | Running, Abandoned | No active work before an explicit start. |
| Running | Paused, Ended, Abandoned | Close the current segment on transition. |
| Paused | Running, Ended, Abandoned | Paused elapsed time is not active effort. |
| Ended / Abandoned | (terminal; correction remains available) | Further work starts a new session. |

Segment kinds: `work`, `break`, `interruption`, `unclassified`. Entering
a break closes the work segment and opens a break segment; ending the
break awaits explicit Resume by default. Task status
(`open/in_progress/blocked/done/cancelled/archived`) is tracked
**independently** of session lifecycle — ending a work interval does not
complete the task. At most one running exclusive session per person
across devices (INV-09), enforced by a database constraint.

### Commitment

`proposed → active → fulfilled | cancelled`. Revisions supersede one
another while preserving history; cancellation/renegotiation never count
as on-time fulfillment in summaries.

### Proposal

`draft → applied | rejected | expired | superseded`. Immutable after
apply; expiry is a usability/storage bound, not a substitute for revision
checks (a proposal can be stale one second after generation).

### Allocation

`draft → accepted → cancelled`. A cancelled allocation is not a completed
task; unscheduling a session does not delete its work item.

## Maturity Ladder

Flows are documented here at design maturity (L1: modeled, pre-code). The
ladder each flow climbs during implementation:

- **L0 — Unmodeled:** behavior exists only in code, no doc.
- **L1 — Modeled (current):** states, transitions, and guard invariants
  written here before code.
- **L2 — Implemented:** the state machine exists in a domain service with
  revision-checked transitions and unit tests.
- **L3 — Verified:** property/integration tests assert the invariants
  (idempotent transitions, no double-count, one-session guard) against a
  real database.
- **L4 — Formally checked:** where a flow warrants it (proposal apply,
  session concurrency), a checked model or exhaustive concurrency test
  covers the race space.

No flow is above L1 yet; this is a documentation-only initialization. The
first flows to reach L2–L3 are hybrid planning (P02/P04) and focus
sessions (P06), whose invariants are the highest-risk.

## Production Shape

- Every mutation is idempotent (idempotency key) and revision-checked; a
  competing command is serialized with a revision check and a
  database-enforced constraint, and the loser receives current state, not
  a win by incrementing a local timer.
- Long-running work (proposal generation, provider sync, forecast
  refresh) runs as coalescing background jobs with leases, durable
  cursors, visible health, and bounded retry — never a tight loop on
  invalid credentials or an unsupported source command.
- Cancellation is first-class: a superseded proposal, a revoked share, an
  abandoned session, and a dismissed insight are all explicit terminal
  states, not silent drops.
- **Stale-completion risk** is the flow-level hazard this product exists
  to prevent: passing an allocation's end never establishes progress;
  unconfirmed past work stays unresolved and still participates in
  forecasts (plan §10.3).

## Deferred / Unmodeled Flows

| Flow | Status | Revisit trigger |
|---|---|---|
| Automatic flexible rescheduling | Deferred (R3, opt-in) | When OT-P2-004 is scheduled; needs explicit user-authorized bounds. |
| Conversational planning turns | Deferred (R2) | When OT-P2-002 is scheduled; must reuse the same bounded commands. |
| Two-way provider writeback | Deferred (R3) | When OT-P2-006 is scheduled; needs conflict/ownership semantics. |
| Multi-person availability negotiation | Deferred (R3) | When collaboration (OT-P2-005) is scheduled. |
| Full offline multi-device timer ownership | Deferred | Beyond R1's online-authoritative model with pending local corrections. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — capability ownership
- [`DATA.md`](DATA.md) — persisted state and revisions
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — surfaces and boundaries
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — source/provider contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — test-substitutable boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md) — how flows are verified
