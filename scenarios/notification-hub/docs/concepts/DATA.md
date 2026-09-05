# Data — Notification Hub

Storage model: what is persisted, which domain owns it, how long it is
kept, and what must never be written here.

## Purpose Of This Document

This document fixes the data shape before the tables exist. Two decisions
in particular are cheap now and expensive later: the retention rule, and
which fields are required at ingress. Both are recorded here rather than
discovered after the first million rows.

## Storage Overview

SQLite through the `api-core` storage seam. One database file, WAL mode,
owned by the scenario.

There is no PostgreSQL and no Redis. This is a capability decision, not a
convenience one, and the reason is macOS evidence rather than Docker.
Neither resource is Docker-backed in the start path any more — both declare
`driver: managed-service`, and the Linux `postgres` path stages a
digest-pinned filesystem tree with no container runtime — though both are
still *acquired* as OCI images
(`managed_service.acquisition.kind: "oci-image"`).

What rules them out is that neither is `supported` on macOS.
`path:docs/reference/platform-support.md` records `postgres` there as
`build-verified` with **no macOS hardware run performed**, and its two
tables disagree about `redis` on macOS: the generated resource matrix says
`build-verified` while the narrative capability table says `unsupported`.
Depending on either would put this scenario on unproven — possibly
unsupported — footing on the macOS fleet node that cross-node delivery
exists to reach.

The queue, the retry schedule, the dedupe counters, and the deadline
sweeper are all in-process and SQLite-backed. At single-owner volume — tens
to hundreds of notifications a day — this is not a compromise.

## Data Ownership

Each domain owns its own schema file and no domain reads another's tables
directly. Cross-domain reads go through a service seam.

| Domain | Schema file | Owns |
|---|---|---|
| recipients | `api/internal/recipients/schema.sql` | `recipients`, `devices`, `channel_addresses`, `push_subscriptions`, `quiet_windows`, `escalation_chains` |
| notifications | `api/internal/notifications/schema.sql` | `notifications`, `notification_events` |
| routing | `api/internal/routing/schema.sql` | `routing_decisions`, `holds`, `suppressions` |
| delivery | `api/internal/delivery/schema.sql` | `delivery_attempts`, `receipts`, `machine_channel_status` |
| conversations | `api/internal/conversations/schema.sql` | `asks`, `answers`, `escalation_steps` |

## Schema Map

### recipients

- `recipients` — one row per verified identity. Keyed by the
  `scenario-authenticator` subject claim. No password, no API key, no
  profile. Created on first authenticated sight.
- `devices` — a named endpoint belonging to a recipient: a phone, a
  laptop, a Mac node. Carries an optional `machine_id` when the device is
  a fleet machine.
- `channel_addresses` — one row per `(device, channel)`. The address shape
  depends on the channel: an email address, a phone number, or a machine
  reference for a host-bound channel.
- `push_subscriptions` — endpoint, `p256dh`, `auth`, and the browser and
  origin it was created against. One row per browser per recipient. This
  table is expected to churn; see Retention.
- `quiet_windows` — per recipient, a weekday and time range in the
  recipient's timezone, plus whether critical urgency overrides it.
- `escalation_chains` — ordered channel preference used when a critical
  notification is not acknowledged.

### notifications

- `notifications` — the durable record returned to every caller. Required
  at ingress and never nullable: `sensitivity_label`, `urgency`,
  `idempotency_key`, `requested_by`. A unique index on
  `(requested_by, idempotency_key)` is what makes OT-P0-012 true in the
  database rather than in a handler.
- `notification_events` — append-only state transitions with a timestamp
  and a reason. This is what the timeline reads. A notification's current
  state is derived from its last event, so no state is lost to an update.

### routing

- `routing_decisions` — the chosen channels, the machine each was routed
  to, and the rule that selected them. Recorded once and never
  recomputed, so the timeline can explain a past choice even after
  preferences change.
- `holds` — a notification waiting for a quiet window to close, with the
  release time. The sweeper reads this table.
- `suppressions` — dedupe keys and their windows, recording which
  notification a suppressed request collapsed into.

### delivery

- `delivery_attempts` — one row per attempt, not per notification. Carries
  channel, machine, outcome, error reason, and attempt number. **This is
  the table that grows fastest**, because retry multiplies rows.
- `receipts` — the terminal record for a successful delivery.
- `machine_channel_status` — the cached answer to "can machine M serve
  channel C", with the disposition, the stated reason, and when it was
  observed. This is a cache with a stated lifetime, not a source of
  truth; the source is the remote instance's own
  `notification-hub channels status`; delivery itself uses the cataloged
  `notification-hub notifications relay` verb.

### conversations

- `asks` — a question, its allowed answers, its deadline, and its state.
  Durable so a restart between question and answer does not strand a
  blocked caller.
- `answers` — the chosen answer, who gave it, and when.
- `escalation_steps` — each escalation taken for an unanswered critical
  ask, with the channel tried and the outcome.

## Migrations And Compatibility

Schema is applied through the shared `database.EnsureSchemas` seam with
per-domain providers. Each domain contributes its own embedded
`schema.sql`.

Two fields are required from the first proto version rather than added
later: `sensitivity_label` and `idempotency_key`. Adding a required field
after release is a breaking change across the API, the CLI, and the UI at
once, and the cost of carrying them from the start is one column each.

## Import / Export

The delivery timeline is exportable as JSON through the CLI for offline
analysis. There is no import path: a notification's history is a record of
what this scenario did, and accepting a foreign history would make the
timeline unfalsifiable.

Push subscription material is never exported. It is credential-equivalent:
anyone holding an endpoint plus its keys can push to that browser.

## Retention And Deletion

Retention is defined before the tables exist, because an unbounded attempt
table is a known failure mode elsewhere in this repository, not a
hypothetical one.

| Table | Rule | Reason |
|---|---|---|
| `notifications` | 90 days | The timeline is an operational surface, not an archive. |
| `notification_events` | Deleted with the parent notification. | Events have no meaning without the record. |
| `delivery_attempts` | 30 days | Grows fastest. Retry multiplies rows per notification. |
| `receipts` | 90 days, with the notification. | Evidence a delivery happened. |
| `routing_decisions` | 90 days, with the notification. | Needed to explain a past choice. |
| `holds`, `suppressions` | Deleted when released or expired. | Working state, not history. |
| `machine_channel_status` | Overwritten per observation; no history. | A cache. |
| `asks`, `answers`, `escalation_steps` | 90 days | Audit of decisions taken through the fleet gate. |
| `push_subscriptions` | Deleted immediately when the push service reports the endpoint gone. | A dead subscription that looks alive is the failure mode OT-P0-014 exists to prevent. |

A retention pass runs on a schedule and records how many rows it removed.
A pass that removes nothing for a table that is growing is a signal, not a
success.

## Privacy Notes

Notification bodies can carry anything a caller puts in them, which is why
`sensitivity_label` is required at ingress rather than inferred. Only the
caller knows whether a body is safe on a locked screen.

Web Push payloads are encrypted end to end under RFC 8291 with keys held
only by this scenario and the recipient's browser. No push service can
read a body. This is stronger than keeping bodies on an owner-run server
that the phone must then reach, and it is why the self-hosted relay design
was rejected.

Push subscription rows are credential-equivalent and must be treated as
secrets: never logged, never exported, never included in a support bundle.

Delivery attempt error reasons are written to the timeline and may be
shown in the UI. An adapter must not put body content into an error
reason.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — which domain owns each table.
- [`FLOWS.md`](FLOWS.md) — the order these tables are written in.
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — threat model and
  secret handling.
- [`../../PRD.md`](../../PRD.md) — OT-P0-013 and OT-P0-014.
