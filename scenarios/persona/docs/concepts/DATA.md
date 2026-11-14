# Data — Persona

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

The template default is embedded SQLite through `modernc.org/sqlite`.
The database path is resolved from the scenario id by `api-core/storage`, and
the API applies schemas on startup through `api-core/database`.

External storage resources should be introduced only when a real
domain needs them. Document those decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md) before editing
`.vrooli/service.json`.

## Data Ownership

Persona's storage design follows one rule: **hold bindings and policy,
never the sensitive payload.** Three classes of data are deliberately
referenced rather than copied, because holding them would make this
scenario a target without making it more capable.

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Persona records | personas | SQLite | `api/internal/personas/schema.sql` | Until retired; retired rows are tombstoned, never erased | Carries kind and immutable legal basis. |
| Addresses | personas | SQLite | `api/internal/personas/schema.sql` | Lifetime of the persona | Billing and postal; released under the same entitlement rule as documents. |
| ACL entries | access | SQLite | `api/internal/access/schema.sql` | Until revoked; revocations are retained | Who may act as, who may only propose. |
| Act-as sessions | access | SQLite | `api/internal/access/schema.sql` | 90 days, then pruned to journal rows | Short-lived; the durable record is the journal entry. |
| Emitted attestations | access | SQLite | `api/internal/access/schema.sql` | Until expiry plus 30 days | Signed delegation-chain tokens (P1). |
| Channel bindings | channels | SQLite | `api/internal/channels/schema.sql` | Lifetime of the persona | Address and adapter selection; **never the credential**. |
| Code-fetch records | channels | SQLite | `api/internal/channels/schema.sql` | 30 days | Records that a code was fetched, when, and by which run — **never the code value**. |
| Handoff records | handoffs | SQLite | `api/internal/handoffs/schema.sql` | 1 year after terminal state | Includes the checkpoint payload needed to resume. |
| Delivery attempts | handoffs | SQLite | `api/internal/handoffs/schema.sql` | 90 days | Relay outcomes; absent when no relay is configured. |
| Document bindings | documents | SQLite | `api/internal/documents/schema.sql` | Lifetime of the persona | A pointer into `document-manager` plus a class label. **No bytes.** |
| Release records | documents | SQLite | `api/internal/documents/schema.sql` | Permanent | Which document went into which handoff, when, on whose authority. |
| Action journal | journal | SQLite | `api/internal/journal/schema.sql` | Permanent, append-only | The scenario's durable memory; never edited or deleted. |
| Account links | accounts | SQLite | `api/internal/accounts/schema.sql` | Until the persona is retired *and* the link is closed | Site, login seam, recovery path (P1). |
| Obligations | accounts | SQLite | `api/internal/accounts/schema.sql` | Until cancelled plus 1 year | The identity half of a commitment; never an amount. |
| Staleness findings | accounts | SQLite | `api/internal/accounts/schema.sql` | Recomputed; latest only | Expiring documents, failing mailbox, dead code route. |

### Data this scenario deliberately does not hold

| Data | Held By | Why not here |
|---|---|---|
| Identity document bytes | `document-manager` | It already owns sensitivity classification, a fail-closed choke point, and an append-only custody journal. Duplicating that would create a second, weaker custody story. |
| Mailbox and provider credentials | `secrets-manager` | Credential custody is a solved, audited concern with one owner. A persona stores which credential to use, never the secret. |
| Run identity and scopes | `agent-manager` | The signing secret never leaves that scenario; verification is a live call that fails closed. |
| One-time code values | Nobody — transient | A code is fetched, returned to the caller once, and never persisted. Only the fact of the fetch is recorded. |
| Team and member rosters | `prompt-manager` | Grants are read from the org's existing source of truth rather than mirrored. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| personas, persona_addresses | personas | `api/internal/personas/schema.sql` | personas repository/service/handlers |
| persona_acl, act_as_sessions, attestations | access | `api/internal/access/schema.sql` | access repository/service/handlers |
| channel_bindings, code_fetches | channels | `api/internal/channels/schema.sql` | channels repository/service/handlers |
| handoffs, handoff_checkpoints, handoff_deliveries | handoffs | `api/internal/handoffs/schema.sql` | handoffs repository/service/handlers |
| document_bindings, document_releases | documents | `api/internal/documents/schema.sql` | documents repository/service/handlers |
| journal_entries | journal | `api/internal/journal/schema.sql` | journal repository/service/handlers |
| account_links, obligations, staleness_findings | accounts | `api/internal/accounts/schema.sql` | accounts repository/service/handlers |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

## Migrations And Compatibility

Domain schema files use `CREATE TABLE IF NOT EXISTS` and live beside the
code that interprets them, per the template default.

Two scenario-specific constraints govern any future migration:

1. **The journal is append-only across migrations too.** A migration may
   add columns to `journal_entries`; it may never rewrite, merge, or
   delete rows. If a shape must change incompatibly, write a new table
   and keep the old one readable — the audit value of the journal is
   entirely in its never having been edited.
2. **Legal basis and release records are immutable.** A migration that
   would rewrite either is a correctness bug, not a schema change. Both
   are load-bearing for attribution claims already made.

Record any migration tradeoff in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) before applying it.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| Journal export | JSON Lines | journal | **Planned P0** (`PSN-P0-011`). An operator must be able to take the attribution record with them; an audit trail that cannot leave the machine is not an audit trail. |
| Persona export | JSON | personas | **Planned P1.** Persona record, bindings, and links — pointers only, never document bytes or credentials. |
| Persona import | JSON | personas | **Deferred.** Importing a persona means importing claims about a legal identity, which needs a verification story that does not exist yet. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Persona record | Retirement | Tombstoned, never erased — journal rows reference it | Retirement is blocked while linked accounts lack a recovery path (`PSN-P2-002`). |
| Act-as sessions | Age | 90 days, then pruned; the journal row survives | Pruning job is P1; P0 keeps everything. |
| Code-fetch records | Age | 30 days | Code values are never stored, so this is metadata only. |
| Handoff records | Terminal state + age | 1 year after completion or cancellation | Checkpoint payloads may embed form data; the retention window is deliberately short for that reason. |
| Delivery attempts | Age | 90 days | Absent entirely when no relay is configured. |
| Document bindings | Persona retirement or explicit unbind | Removed; the release records survive | Unbinding here does not delete the document — that is `document-manager`'s decision. |
| Release records | Never | Permanent | The point of a release record is that it outlives the release. |
| Action journal | Never | Permanent, append-only | Growth is bounded by export-and-archive, not deletion. |
| Obligations | Cancellation + 1 year | Retained past cancellation so a re-bill can be disputed | Depends on `accounts` (P1). |

**Deletion of a person's data.** If an operator must erase a persona
representing a real person, the erasure covers the persona record,
addresses, bindings, and channel configuration. It cannot cover the
journal, whose rows are the evidence that actions were authorised. The
journal stores identifiers and verbs, not personal content, which is
what makes that split defensible — record any exception in
[`../internal/SECURITY.md`](../internal/SECURITY.md).

## Privacy Notes

This scenario handles **personal data by definition** — that is its
subject matter — so the template's "local development data" assumption
does not apply and the following are treated as binding.

- **Data minimisation is the architecture, not a policy.** Documents,
  credentials, and code values live elsewhere or nowhere. What remains
  here is a pointer, a policy, and a record — the smallest set that can
  still answer who acted as whom.
- **The controlled mailbox is the highest-value asset in the scenario**,
  because it is the recovery path for every account a persona created.
  It is treated as a credential of the highest class, held in
  `secrets-manager`, and never surfaced to an agent.
- **A one-time code is returned once and never persisted.** Only the
  fact, time, and requesting run are recorded.
- **The journal records identifiers and verbs, never payloads.** No
  document content, no code value, no form field ever enters it.
- **No data leaves the machine by default.** Every outbound path —
  relay delivery, attestation emission, paid handoff — is opt-in and
  named in [`INTEGRATIONS.md`](INTEGRATIONS.md).

See [`../internal/SECURITY.md`](../internal/SECURITY.md) for the threat
model these rules answer.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
