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
| conversations | `api/internal/conversations/schema.sql` | `asks`, `answers`, `escalation_steps`, `ask_context_requests` |

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
- `quiet_windows` — per recipient, at most one row per weekday: a start and
  end time (24-hour `HH:MM`) in the row's timezone, plus whether critical
  urgency overrides it. A row belongs to the weekday it starts on. When
  `end_time` is earlier than `start_time` the row is overnight: it runs from
  the start on its weekday until the end on the next weekday in the same
  timezone, wrapping the week (a Saturday row ends Sunday morning, a Sunday row
  ends Monday morning). Equal start and end are refused. An overnight row can
  overlap the next weekday's own row; a critical notification passes only when
  every active row allows the override.
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
  Decision-subject columns: `subject_key`, `last_asserted_at`,
  `reassert_within_seconds`, `revision_count`, `superseded_by_ask_id` and
  `current_event_id`. A partial unique index admits one open ask per
  `(source, source_event_type, subject_key)` for a nonempty key. The index and
  the columns are added to an existing database by
  `Service.ensureAdditiveColumns` (the index only after the `ALTER`s), not by
  `schema.sql`.
  Answer-form column: `questions_json` (the ask's `AskQuestion` list as JSON,
  empty for none), added to an existing database by `ensureAdditiveColumns`.
  It is part of an ask's text, so a change updates the ask in place.
  `accepts_context_requests` (0/1) records that the requester declared it answers
  context requests; each revision restates it, and a change updates the ask in
  place.
- `ask_revisions` — what an updated ask said before each update (text,
  correlation, deadline, event id); the newest 50 per ask. The answer form is
  not kept per revision: it shows the current revision's questions.
- `ask_event_ids` — event ids an ask carried as `requester_ref` before an
  update or re-send moved it to the latest event's id; they still find the ask
  for redeliveries and withdrawals (newest 200 per ask).
- `decision_subject_withdrawals` — the last withdrawal a producer asserted per
  subject; a request asserted earlier is stale.
- `ask_attachments` — one row per attachment of an ask's *current* revision
  (`name`, `label`, `kind` file or folder, `role` evidence or explainer, the
  sender's `source_path` and `source_host`, and for a file its `sha256`,
  `size_bytes`, `media_type`; `preview_kind` and `truncated` as the Hub
  classified them when the ask was stored). A revision that changes the
  attachments replaces these rows. Deleted with the ask.
- `ask_attachment_entries` — the files of a folder attachment (`path`, `sha256`,
  `size_bytes`, `media_type`, `preview_kind`), in the sender's order, at most 500
  per folder. Deleted with the attachment.
- `attachment_blobs` — the inventory of the Hub's own copies: `sha256`,
  `size_bytes`, `media_type`, `stored_at` (first stored) and `last_touched_at`
  (last upload, confirmation by `StatAttachmentBlobs`, or reference by an ask).
  The bytes are not in SQLite: they live in the attachment blob store, keyed
  `sha256/<first byte>/<hash>`, under the scenario data root `attachments/`
  (storage entry `decision-attachments`). One blob serves every attachment with
  the same content.
  `asks.attachment_count` and `asks.attachments_digest` summarize the current
  attachments: the count feeds list rows, and the digest (names, labels, roles,
  sources and content hashes, never derived values) is how a re-sent subject-key
  ask is recognized as unchanged or as an in-place update.
- `answers` — the chosen answer, who gave it, and when. `responses_json` holds
  the operator's answers to the ask's questions (`AskQuestionResponse` list;
  column added by `ensureAdditiveColumns`).
- `ask_context_requests` — the recipient's "I need more context" questions about
  an open ask: `question`, `requested_by`, `requested_at`, `addressed_at` and
  `addressed_revision` (set when the requester revises the ask in place), `response`
  (the requester's written `context_response`, empty without one; kept, so each
  round of question and answer stays), and
  `published_at`, the outbox marker for `notification_hub.ask.context_requested.v1`.
  At most 5 are unaddressed per ask. Deleted with their ask.
- `escalation_steps` — each escalation taken for an unanswered critical
  ask, with the channel tried and the outcome.

## Deletes And Foreign Keys

Production opens SQLite with `foreign_keys` on (the `api-core/storage` DSN), and
the schemas declare `ON DELETE CASCADE` where a row has no meaning without its
parent. Code never depends on that: every delete removes its dependents
explicitly, in one transaction, so a connection that does not enforce foreign
keys (a test pool, a restored database) leaves the same result. `Retain` deletes
a notification's events, attempts, receipts and routing records with it, and a
resolved ask's answers, revisions, escalations, context requests and attachments
with the ask;
`RemoveDevice` deletes the device's addresses and subscriptions; replacing an
ask's attachments deletes their rows and entries first. Tests run each of these
with enforcement on and off.

## Where The Files Live

The data root for files (attachment blobs) is the directory the database lives
in, and `resolveScenarioStorageRoots` in `api/main.go` takes it from the same
`storage.SQLitePath` call that `database.Open` makes, so a packet's rows and the
files they point at cannot sit in different roots. Which directory that is depends
on how the instance was launched:

| Instance | Database and attachments |
|---|---|
| Live, launched by the lifecycle | `<scenario>/data/` (`SCENARIO_DATA_DIR`): `storage.SQLitePath` keeps a live instance's database where the lifecycle put it, so no fleet database had to move. |
| Shadow variant | `~/.vrooli/data/vrooli/notification-hub_<variant>/` |
| Test lease | The lease-owned root, chosen per request. |
| No lifecycle data directory | `~/.vrooli/data/vrooli/notification-hub/` |

Before this rule, a live instance wrote attachments to the class root
(`~/.vrooli/data/vrooli/notification-hub/attachments`) while its database sat in
`<scenario>/data/`. The `decision-attachments` entry in `.vrooli/service.json`
still names the class root as its owner path; storage-manager measures both
locations for a class-data entry (`storage.ResolveOwnerStorageLocations`).

The lifecycle setup step `cmd/schema-preflight` checks that same database. It
resolves its path through `storage.SQLitePath` with the lifecycle environment
completed for this scenario's own directory, so run by hand (with no
`SCENARIO_*` variables) it still reads the live Hub's database instead of an
empty one at the class root, and it never creates a database to check one.

Both live inside the repository tree or the runtime home, and the repository copy
is the weaker place for durable data: it is removed with the checkout. The class
root is the target. Relocating the live database there is a deliberate migration,
not a side effect: stop the scenario, copy `notification-hub.db` with its
`-wal` and `-shm` and the `attachments/` tree to
`~/.vrooli/data/vrooli/notification-hub/`, change `database.Open` to a DSN from
`storage.SQLiteDSNAt(fileRootPath(ctx, fileRoots, storage.ClassData, "notification-hub.db"), ...)`
(as react-component-library does), start, verify an ask and an attachment read,
then remove the old copies. Attachments follow automatically, because their root
is derived from the database's.

### Open question: who opens other scenarios' SQLite files

On 2026-11-13 something opened SQLite databases under `~/.vrooli/data/vrooli/`
that no scenario was serving: at 11:11:01 it rewrote the `-shm` file of 279
databases, including every leaked `treasury_cf2-*` variant, and at 21:04:51 it
touched two leaked `notification-hub_cf5-*` variants. At 21:05 and 21:07 the class
root `notification-hub/notification-hub.db` got an empty WAL and a rewritten
`-shm`. That database file is a 4096-byte empty file from 2026-10-17.

A search of storage-manager, `internal/scenarioruntime` and `internal/capacity`
found no code that opens per-scenario databases to measure or retain them. The only
cross-scenario opens are the orphaned-database provider's single configured
`quick_check` and this scenario's setup step `cmd/schema-preflight`, which no longer
reads the class root. The most likely cause is an operator or agent running
`sqlite3` over those paths by hand. That is not confirmed. The class root database is a
single page, so it holds no tables, and it can stay as it is until the migration
above.

## Migrations And Compatibility

Schema is applied through the shared `database.EnsureSchemas` seam with
per-domain providers. Each domain contributes its own embedded
`schema.sql`.

Two fields are required from the first proto version rather than added
later: `sensitivity_label` and `idempotency_key`. Adding a required field
after release is a breaking change across the API, the CLI, and the UI at
once, and the cost of carrying them from the start is one column each.

Timestamps are stored as fixed-width UTC text with a nine-digit fraction
(`2006-01-02T15:04:05.000000000Z`), written only through `storedTime` in
`internal/hub/stored_time.go`. SQLite compares them as text, and the trimmed
RFC 3339 form is not ordered by time within a second (`…:00Z` sorts after
`…:00.5Z`), which would reorder the inbox and its keyset cursors. At boot
`normalizeStoredTimes` rewrites rows written in the older form for the ask
domain and the notifications and receipts under it (`storedTimeColumns`): it is
idempotent, runs in batches, and leaves an empty value or one that is not a
timestamp alone. Rows already in other tables the Hub writes (delivery attempts, blobs, holds
and the like) keep the form they were written in; new rows use the fixed one, and
none of those is ordered against the ask domain.

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
| `notifications` | 90 days, except while an ask still references it. | The timeline is an operational surface, not an archive. An open ask never loses its notification. |
| `notification_events` | Deleted with the parent notification. | Events have no meaning without the record. |
| `delivery_attempts` | 90 days, with the notification. | Grows fastest. Retry multiplies rows per notification. A shorter 30 day rule was once written here; nothing applies it, so it is not promised. |
| `receipts` | 90 days, with the notification. | Evidence a delivery happened. |
| `routing_decisions` | 90 days, with the notification. | Needed to explain a past choice. |
| `holds`, `suppressions` | Deleted when released or expired. | Working state, not history. |
| `machine_channel_status` | Overwritten per observation; no history. | A cache. |
| `asks`, `answers`, `answer_identity`, `escalation_steps`, `ask_revisions`, `ask_event_ids`, `ask_context_requests` | 90 days after the ask was resolved. An open ask is never removed. | Audit of decisions taken through the fleet gate. |
| `ask_attachments`, `ask_attachment_entries` | Deleted with the ask. | A packet's attachments have no meaning without it. |
| `attachment_blobs` and the blob bytes | Swept 24 hours after `last_touched_at` when no ask references the blob. | Content nothing points at is only an upload the sender never used. A referenced blob lives as long as its ask; when retention removes the ask, the same pass sweeps the blobs that lose their last reference. Uploads no ask references yet are capped at 1 GiB. The `decision-attachments` entry declares a 10 GiB / 90 day budget; the Hub's own rules above are what enforce it (storage-manager's recovery never touches a non-regenerable entry). |
| `push_subscriptions` | Deleted immediately when the push service reports the endpoint gone. | A dead subscription that looks alive is the failure mode OT-P0-014 exists to prevent. |

The retention pass is `Service.Retain`, run by the maintenance loop at the
first tick after a start and then once a day; it logs how many asks and
notifications it removed. The attachment sweep runs on every tick and follows it.
A pass that removes nothing for a table that is growing is a signal, not a
success.

The pass also repairs historical orphans: rows in the tables that exist only for a
notification (events, delivery attempts, receipts, routing decisions, holds,
suppressions) or only for an ask (answers, steps, revisions, attachments and the
rest) whose notification or ask no longer exists. Older deletes that did not name
every dependent left some behind; the live database had six, in four tables. It
removes them whether or not the connection enforces foreign keys, logs the count
("removed N orphaned rows"), and is idempotent: with none left it deletes nothing.

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

Decision attachments can hold anything the requester chose to attach, which is
why the `decision-attachments` storage entry is marked sensitive and why the Hub
serves only its own copies, only to the ask's recipient, never a host path. The
sender's `source_path` is shown to the operator and read only to compare with the
stored copy; it is never opened for serving.

Delivery attempt error reasons are written to the timeline and may be
shown in the UI. An adapter must not put body content into an error
reason.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — which domain owns each table.
- [`FLOWS.md`](FLOWS.md) — the order these tables are written in.
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — threat model and
  secret handling.
- [`../../PRD.md`](../../PRD.md) — OT-P0-013 and OT-P0-014.
