# Data — Notification Hub

This document is the canonical data ownership and storage map for the
scenario. Update it when domains add tables, files, blobs, external
records, retention rules, migrations, imports, or exports.

## Purpose Of This Document

Use this document to answer:

- What data does the scenario persist?
- Which domain owns each data shape?
- Where is the source of truth?
- What is the retention/deletion story?
- How are schema changes handled?

## Storage Overview

Embedded SQLite through `modernc.org/sqlite`, and nothing else. The
lifecycle sets `SQLITE_PATH` through `.vrooli/service.json`, and the API
applies domain schemas on startup through `api-core/database`.

This is a capability decision, not a default accepted out of
convenience. The scenario must run unchanged on a macOS fleet node so it
can serve the Apple-only channels this host cannot reach. `resource-postgres`
and `resource-redis` are still acquired as OCI images and are recorded
`unsupported` on macOS and Windows in
[`docs/reference/platform-support.md`](../../../../docs/reference/platform-support.md);
depending on either would make the relay lane impossible to build.
See [`INTEGRATIONS.md`](INTEGRATIONS.md) and
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

The work queue, the retry schedule, and the rate-limit counters are
in-process and SQLite-backed for the same reason. At one owner's volume
they are a table and a ticker, not a broker.

## Data Ownership

Each domain owns its own tables and is the source of truth for its data.
The `health` domain owns no product data — it only probes configured
database reachability.

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Recipients | recipients | SQLite | `api/internal/recipients/schema.sql` | Until the operator removes the recipient. | A local projection of a scenario-authenticator subject. Holds no credential. |
| Devices | recipients | SQLite | `api/internal/recipients/schema.sql` | Until removed, or pruned after a long unseen period the operator sets. | A device is durable across address changes. |
| Channel addresses | recipients | SQLite | `api/internal/recipients/schema.sql` | Same lifecycle as the device. | Push topics, email addresses, and handles. Treated as sensitive. |
| Quiet windows and channel preferences | recipients | SQLite | `api/internal/recipients/schema.sql` | Same lifecycle as the recipient. | Read by `routing`; never enforced client-side. |
| Notifications | notifications | SQLite | `api/internal/notifications/schema.sql` | Operator-set window, 90 days by default. | Carries the body, so retention is a privacy control, not only a disk control. |
| Notification state transitions | notifications | SQLite | `api/internal/notifications/schema.sql` | Deleted with the parent notification. | Append-only; the audit answer to "why did this never arrive". |
| Deliveries | delivery | SQLite | `api/internal/delivery/schema.sql` | Deleted with the parent notification. | One row per attempt per channel, including the routing reason. |
| Channel health and capability | channels | SQLite | `api/internal/channels/schema.sql` | Latest observation only. | Small, overwritten; no history kept. |
| Node capability cache (P1) | relay | SQLite | `api/internal/relay/schema.sql` | Refreshed from vrooli-bridge; stale rows expire. | A cache, never a source of truth — the bridge registry is. |
| Ingress rules and receipts (P1) | ingress | SQLite | `api/internal/ingress/schema.sql` | Rules until removed; receipts share the notification window. | Receipts exist to make webhook redelivery idempotent. |

Credential values are never stored here. `channels` holds a reference to
a credential descriptor; the native credential authority holds the
secret. See [`../internal/SECURITY.md`](../internal/SECURITY.md).

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `recipients` | recipients | `api/internal/recipients/schema.sql` | recipients repository; read by routing |
| `devices` | recipients | `api/internal/recipients/schema.sql` | recipients repository; read by routing |
| `device_channels` | recipients | `api/internal/recipients/schema.sql` | recipients repository; read by routing and delivery |
| `quiet_windows` | recipients | `api/internal/recipients/schema.sql` | recipients repository; read by routing |
| `channel_preferences` | recipients | `api/internal/recipients/schema.sql` | recipients repository; read by routing |
| `notifications` | notifications | `api/internal/notifications/schema.sql` | notifications repository/service/handlers |
| `notification_transitions` | notifications | `api/internal/notifications/schema.sql` | notifications service; read by the timeline UI |
| `deliveries` | delivery | `api/internal/delivery/schema.sql` | delivery repository/service/handlers |
| `channel_state` | channels | `api/internal/channels/schema.sql` | channels repository; read by routing |
| `node_capabilities` (P1) | relay | `api/internal/relay/schema.sql` | relay repository; read by routing |
| `ingress_rules`, `ingress_receipts` (P1) | ingress | `api/internal/ingress/schema.sql` | ingress repository/handlers |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

### Shape notes that matter

These are the field-level decisions that would be expensive to change
later, recorded so the implementation plan does not have to rediscover
them.

- **`notifications.state`** is one of `accepted`, `held`, `routed`,
  `delivering`, `delivered`, `failed`, `cancelled`, `suppressed`.
  `suppressed` is distinct from `cancelled`: the former means duplicate
  suppression collapsed it, the latter means somebody stopped it. Both
  are terminal and both must stay visible, because a notification that
  silently vanished is the failure mode this scenario exists to prevent.
- **`notifications.sensitivity`** is `public`, `private`, or `secret`,
  and it is NOT NULL with no default. A caller that does not think about
  sensitivity should be made to think about it once.
- **`notifications.dedupe_key`** is scoped per recipient, not globally.
  Two people can be told the same thing.
- **`deliveries.node_id`** is nullable and NULL means "this host sent
  it". Present from the P0 schema so the P1 relay needs no migration.
- **`deliveries.routing_reason`** is a short machine-readable code plus
  a human string. This is the field that answers "why did it pick
  email", and omitting it is how delivery systems become unexplainable.
- **`device_channels.address`** is treated as sensitive. A push topic is
  a bearer secret: whoever knows it can send to the device.
- Timestamps are stored as RFC 3339 UTC strings, consistently, in every
  table. Mixed timestamp representations break `MAX()` comparisons in
  SQLite in ways that are painful to find later.

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships the `notes` domain as a worked CRUD slice with a
binary attachment-upload exception, showing how a real domain owns its
tables, metadata, and opaque blob bytes. Copy its shape, then remove it.

Its Data Ownership rows:

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Notes | notes | SQLite | `api/internal/notes/schema.sql` | Until deleted by future product behavior | Template reference data; remove with notes domain. |
| Attachment metadata | notes | SQLite | `api/internal/notes/schema.sql` | Until parent note or attachment is deleted by future product behavior | Metadata only; bytes are stored through BlobStore. |
| Attachment bytes | notes | Filesystem BlobStore by default | BlobStore implementation in notes handler module | Same lifecycle as metadata | Opaque bytes stay outside proto payloads. |

Its Schema Map row:

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| notes tables | notes | `api/internal/notes/schema.sql` | notes repository/service/handlers |

Its Retention And Deletion row:

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Template notes data | Domain removal or future product delete behavior | Local development data only | Real scenarios must define product-specific deletion semantics. |
<!-- EXAMPLE-DOMAIN:notes END -->

## Migrations And Compatibility

Idempotent schema bootstrap. Domain schema files use
`CREATE TABLE IF NOT EXISTS` and live beside the code that interprets
them.

Two forward-compatibility choices are deliberate and should be preserved
rather than "cleaned up":

- `deliveries.node_id` exists at P0 although only P1 writes it.
- The channel enumeration is stored as text, not an integer, so adding
  `imessage` or `webpush` is an insert rather than a migration.

For a change that needs a column drop, rename, or backfill, add a
migration plan to this section and record the tradeoff in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| Delivery timeline export | JSON lines | delivery | Planned with OT-P1-007; useful for answering "what did this system send me last month" outside the UI. |
| Device registry export | JSON | recipients | Deferred. Add when a second machine needs to be seeded without re-registering every device by hand. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Notifications and their transitions | Age past the retention window, or explicit operator delete. | Operator-set; 90 days by default. | The window is not yet configurable; the default is hard-coded until OT-P1-007 gives the UI a place to expose it. |
| Deliveries | Cascade from the parent notification. | Same as the notification. | None. |
| Recipients, devices, channel addresses | Explicit operator delete. | Kept until removed. | No automatic pruning of devices unseen for a long period. |
| Channel health | Overwritten on each probe. | Latest only. | None. |
| Node capability cache | Expired on staleness, refreshed from the bridge. | Short. | Arrives with the relay domain at P1. |

Deleting a recipient deletes their devices, addresses, preferences, and
notifications. That cascade is the whole deletion story, and it is the
reason recipient identity is a single column rather than being denormalised
across tables.

## Privacy Notes

This scenario stores personal data by construction: who you are, which
devices you own, how to reach each one, when you sleep, and the text of
every message sent to you. That makes it one of the more
privacy-sensitive scenarios in the fleet, despite having no customer
data in it.

Three consequences shape the design:

- **Body content is a liability, not an asset.** The house convention is
  that a notification says what happened and links back to the console
  for detail. That keeps bodies safe on a locked screen, in a shared
  room, and in a third-party push service's logs.
- **A channel address is a bearer secret.** A push topic name is enough
  to send to the device, and on public infrastructure it is enough to
  read what was sent. Treat it like a token.
- **Retention is a privacy control.** The notification table holds
  message text, so the retention window is not only about disk.

See [`../internal/SECURITY.md`](../internal/SECURITY.md) for the
sensitivity model and the threat table.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
