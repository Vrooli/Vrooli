# Data — Switchboard

## Purpose Of This Document

This document is the canonical description of what this scenario stores, which
domain owns each piece, how long it is kept, and what happens on deletion. Use
it to answer:

- Where does data live, and under whose schema?
- Which domain owns a given table, so two domains never both write it?
- What is sensitive, and what is deliberately *not* stored?
- What is retained, for how long, and what does deletion actually remove?

## Storage Overview

| Store | Engine | Scope | Rationale |
|---|---|---|---|
| Scenario database | SQLite through `api-core/storage` | All domain state | Resolved from the scenario identity, never from the environment. No server engine at all, so this scenario runs on the Mac fleet node the iMessage lane depends on without relying on macOS support this repo has not proven |
| Channel descriptors | Versioned JSON files under `data/channels/` | Channel definitions | **The file is the source of truth; the database table is a cache rebuilt at boot.** Adding a channel must never require a migration |
| Media | Filesystem under the scenario storage root, referenced by row | Attachments in both directions | Blobs do not belong in the row store; the row holds a reference, a content hash, a declared type and a size |
| Credentials | **Not stored here** | — | Only a reference to the credential authority is persisted. A write carrying a credential value is rejected |

Per-domain schemas are embedded next to the code that interprets them, through
the `SchemaProvider` / `EnsureSchemas` substrate. No domain reads another
domain's tables directly; it calls that domain.

## Data Ownership

| Domain | Owns | Never writes |
|---|---|---|
| `channels` | `channel_descriptor_cache`, `channel_availability`, `adapter_connection` | anything about threads, agents, or people |
| `conversations` | `thread`, `message`, `participant`, `media_ref`, `ingress_dedupe` | tiers, scopes, bindings |
| `agents` | `agent_binding`, `agent_ref` | agent descriptors themselves — those live in `prompt-manager` |
| `trust` | `contact`, `trust_assignment`, `scope_resolution_log` | messages, threads |
| `turns` | `turn`, `budget_window`, `spend_ledger`, `approval_request` | anything above it in the chain |

## Schema Map

Illustrative shape, not a migration. Column sets will be settled by the domain
slice that builds each one.

| Table | Owner | Key columns | Notes |
|---|---|---|---|
| `channel_descriptor_cache` | `channels` | `channel_id` PK, `version`, `content_hash`, `loaded_at` | Rebuilt at boot from files. Never authoritative |
| `channel_availability` | `channels` | `channel_id` PK, `state`, `reason`, `probed_at` | `state` ∈ live, unavailable, unimplemented, degraded. `reason` is required whenever state is not live |
| `adapter_connection` | `channels` | `channel_id` PK, `state`, `last_error`, `connected_at` | Runtime; safe to lose on restart |
| `thread` | `conversations` | `id` PK, `channel_id` FK, `remote_thread_id`, `is_group`, `agent_binding_id` FK | `(channel_id, remote_thread_id)` unique |
| `message` | `conversations` | `id` PK, `thread_id` FK, `remote_message_id`, `author_kind`, `author_address`, `body`, `created_at` | **`(channel_id, remote_message_id)` unique — this is the idempotency key.** `author_kind` ∈ human, agent, system, and is what makes the loop breaker possible |
| `participant` | `conversations` | `thread_id` FK, `contact_id` FK, `joined_at` | The roster the thread ceiling is computed from |
| `media_ref` | `conversations` | `id` PK, `message_id` FK, `path`, `content_hash`, `mime`, `bytes` | Reference only; blob on disk |
| `ingress_dedupe` | `conversations` | `channel_id`, `remote_message_id`, `seen_at` | Composite PK. Retained 30 days, then pruned |
| `agent_binding` | `agents` | `id` PK, `agent_id`, `channel_id` FK, `address`, `state` | `(channel_id, address)` unique — an address binds to exactly one agent |
| `agent_ref` | `agents` | `agent_id` PK, `display_name`, `appearance_json`, `cached_at` | **A cache for rendering only.** `prompt-manager` is authoritative; a stale cache degrades display, never permission |
| `contact` | `trust` | `id` PK, `channel_id` FK, `address`, `label` | `(channel_id, address)` unique. An address is channel-scoped and never globally unique |
| `trust_assignment` | `trust` | `contact_id` PK, `tier`, `assigned_by`, `assigned_at` | `tier` ∈ owner, trusted, known, stranger |
| `scope_resolution_log` | `trust` | `id` PK, `turn_id`, `sender_tier`, `room_ceiling`, `granted`, `withheld`, `decided_at` | The audit record of every permission decision. Append-only |
| `turn` | `turns` | `id` PK, `thread_id` FK, `state`, `run_id`, `started_at`, `ended_at` | `state` per the FLOWS state machine |
| `budget_window` | `turns` | `thread_id`, `window_start` PK, `turns_used` | Hourly window, derived from the clock seam |
| `spend_ledger` | `turns` | `id` PK, `thread_id` FK, `turn_id` FK, `units`, `recorded_at` | Local record of metered spend. **LPBS remains the wallet authority**; this is a mirror for display and for cap enforcement, never a second ledger of record |
| `approval_request` | `turns` | `id` PK, `turn_id` FK, `scope`, `state`, `expires_at` | Expiry is mandatory, not nullable |

## Migrations And Compatibility

Greenfield, so the three-tier story applies at its first tier: **declarative
schemas via `EnsureSchemas`**, no versioned migration machinery until production
schema evolution earns it. Out-of-tree scripts if a greenfield-with-data
situation appears. Versioned migrations only when there is production data whose
shape must change.

Descriptors are the exception and have their own compatibility rule: a descriptor
carries `schemaVersion`, and an unrecognised version **fails boot loudly rather
than being coerced**. A channel that loads with a misread capability set is worse
than a channel that does not load, because the first silently violates a limit.

## Import / Export

| Direction | Scope | Notes |
|---|---|---|
| Export — threads | A thread or a date range, as JSON with media referenced by hash | The owner's conversations are the owner's. There must be no lock-in on the one data type here that is genuinely personal |
| Export — descriptors | Already plain files under `data/channels/` | Copyable between installs as-is |
| Import — descriptors | Drop a validated file in and restart | This *is* the extensibility mechanism |
| Import — threads | **Not supported** | Importing history would let a forged thread manufacture a trust relationship. Deliberately absent |
| Export — contacts and tiers | JSON | Tiers are the owner's policy and must be portable |

## Retention And Deletion

| Data | Retention | On deletion |
|---|---|---|
| `message`, `media_ref` | Kept until the owner deletes | Row and blob both removed. A deletion that leaves the blob is a privacy defect, not a cleanup task |
| `ingress_dedupe` | 30 days | Pruned. The only risk of pruning early is re-answering a very old redelivery |
| `scope_resolution_log` | 1 year, append-only | **Not deleted with a thread.** It is the audit record of what was permitted, and it must outlive the conversation it describes |
| `spend_ledger` | 1 year | Retained for reconciliation against LPBS |
| `budget_window` | 7 days | Pruned |
| `agent_ref` | Cache | Safe to drop at any time; refetched from `prompt-manager` |
| `contact`, `trust_assignment` | Until the owner deletes | Deleting a contact resets that address to `stranger` — it never leaves a dangling grant |
| Deleting an agent binding | — | Threads survive, marked orphaned. Deleting a binding must never silently delete conversation history |
| Deleting a channel descriptor | — | Its bindings become `unavailable` with a stated reason. Threads and messages are retained |

**Storage accounting.** Every entry is declared in the scenario's storage
manifest with an honest `regenerable` flag: descriptor caches, availability, and
`agent_ref` are regenerable; threads, messages, media, tiers, and the resolution
log are not.

## Privacy Notes

This scenario holds the most sensitive data of any in the ecosystem, and the
design has to be read with that in front of it.

- **Message bodies are private correspondence**, frequently involving people who
  never agreed to any of this and cannot consent — the other participants in a
  group thread. They are stored on the owner's own machine and are never
  transmitted to a third party by any default path. This is the single promise
  that justifies the scenario over a hosted competitor, and any feature that
  weakens it is a product decision, not a technical one.
- **A hosted relay would break that promise**, which is why it is rejected as a
  default and may only ever exist as an explicitly labelled fallback that states
  its trade-off at the point of purchase.
- **Credentials are never stored** — only a reference to the credential
  authority. A write carrying a credential value is rejected rather than
  scrubbed.
- **The resolution log is deliberately durable and deliberately minimal.** It
  records *what was permitted*, not *what was said*: tiers, scopes granted and
  withheld, and a turn identifier. It must not become a second copy of message
  content.
- **Media may carry more than intended** — location in photo metadata being the
  obvious case. Inbound media is stored as received because stripping it would
  destroy evidence the owner may want, so this is called out here rather than
  silently decided.
- **Deletion must be real.** Removing a message removes its blob. The one
  deliberate exception is the resolution log, and it is an exception because an
  audit trail that a participant can erase is not an audit trail.

## Cross-References

- `docs/concepts/DOMAINS.md` — the domain that owns each table
- `docs/concepts/FLOWS.md` — when each row is written
- `docs/internal/SECURITY.md` — the threat model over this data
- `docs/concepts/INTEGRATIONS.md` — why no server database engine is used
- `docs/internal/DECISIONS.md` — the storage decisions and their reasons
