# Data — Treasury

This document is the canonical map of what this scenario stores, which
domain owns it, where its schema lives, and how long it is kept.

Two rules govern everything below and are worth stating before the tables.
**Nothing here is a balance.** Headroom, position and outcome are queries
over authorization and settlement records, never mutable fields, because a
stored balance is a number that can silently disagree with its own history.
**No credential material is stored.** Card numbers, keys and facilitator
secrets live in `secrets-manager` and are resolved by reference at use
time; this scenario persists the reference and never the value.

## Purpose Of This Document

Use this document to answer:

- What does this scenario persist, and which domain owns it?
- Which schema file is the source of truth for a given table?
- How long is each shape kept, and what removes it?
- What is deliberately *not* stored here, and where does it live instead?

## Storage Overview

Embedded SQLite through `modernc.org/sqlite`. The lifecycle sets
`SQLITE_PATH` through `.vrooli/service.json`, and the API applies schemas
on startup through `api-core/database`.

**Why SQLite and not a shared database.** Custody is single-operator and
authorization volume is low, so the shared-Postgres pattern used by
higher-traffic billing surfaces is not warranted. The *discipline* from
that pattern carries over unchanged: lock the row, require a
caller-supplied idempotency key, and make a retried debit a successful
no-op after first commit. SQLite serialises writers, which satisfies the
locking requirement directly.

**The one declared migration trigger.** Inbound x402 metering
(`TRS-P1-002`) is the only traffic this scenario does not control — many
external callers may pay concurrently, and SQLite's single-writer lock is
the constraint that would bind first. `TRS-P1-002` carries a performance
validation whose purpose is to observe that boundary rather than to assert
a throughput number. If it binds, the migration is this scenario's storage
seam only; no other decision in this document changes. Recorded in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

External storage resources should be introduced only when a real
domain needs them. Document those decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md) before editing
`.vrooli/service.json`.

## Data Ownership

Each domain owns its own tables and is the source of truth for its data.
The `health` domain owns no product data — it only probes configured
database reachability.

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Books | book | SQLite | `api/internal/book/schema.sql` | Life of the scenario; archived, never deleted while any charge references it. | Holds the single beneficiary identity. The schema admits exactly one, which is where `TRS-P0-010` is enforced. |
| Budgets | budget | SQLite | `api/internal/budget/schema.sql` | Life of the scenario; archived on retirement. | Caps and gating intent. Never holds a spent-to-date figure. |
| Budget scope entries | budget | SQLite | `api/internal/budget/schema.sql` | With parent budget. | Allow and deny entries. Deny outranks allow for the same counterparty. |
| Mandates | mandate | SQLite | `api/internal/mandate/schema.sql` | Retained after expiry; never deleted while evidence references them. | The signed grant. Immutable after issue — a change is a new mandate. |
| Mandate templates | mandate | SQLite | `api/internal/mandate/schema.sql` | Until the operator deletes one. | Editing a template never alters mandates already issued from it. |
| Authorizations | authorization | SQLite | `api/internal/authorization/schema.sql` | Append-only; retained with evidence. | Carries the verdict and the refusing constraint. |
| Holds | authorization | SQLite | `api/internal/authorization/schema.sql` | Released on settle, decline or expiry. | Reserves headroom between decision and settlement so concurrent planners cannot double-count. |
| Approval requests | approval | SQLite | `api/internal/approval/schema.sql` | Append-only; retained with evidence. | Resolution and resolver are recorded, not just the outcome. |
| Relay attempts | approval | SQLite | `api/internal/approval/schema.sql` | 90 days. | Notification-hub delivery attempts. A failure here never changes an approval outcome. |
| Instruments | instrument | SQLite | `api/internal/instrument/schema.sql` | Until revoked; record retained after revocation. | Stores a credential *reference*, never credential material. |
| Rails | rail | SQLite | `api/internal/rail/schema.sql` | Life of the scenario. | Adapter registration and non-secret configuration. |
| Charges | settlement | SQLite | `api/internal/settlement/schema.sql` | Append-only; retained with evidence. | The exactly-once execution record. |
| Idempotency keys | settlement | SQLite | `api/internal/settlement/schema.sql` | 180 days after terminal outcome. | Caller-supplied and required. Retention must outlive any plausible client retry window. |
| Evidence records | evidence | SQLite | `api/internal/evidence/schema.sql` | Indefinite; append-only, never rewritten. | Joins mandate, approval, request, rail response and receipt. Covers declines and expiries equally. |
| Ledger emissions | ledger | SQLite | `api/internal/ledger/schema.sql` | Indefinite. | Emission log with its own idempotency, so a retry cannot double-post downstream. |

Deliberately **not** stored here:

| Data | Lives In | Why not here |
|---|---|---|
| Card numbers, API keys, facilitator secrets | `secrets-manager` | Keeping credentials out of this scenario is what preserves `money-ledger`'s no-credential-storage non-goal across the pair. |
| Identity documents (photo ID, incorporation papers) | `document-manager` | It already owns sensitivity classification, custody receipts and an append-only custody journal. A passport is a document, not a payment object. |
| Persona, legal entity, addresses, contact channels | `persona` | Transactional identity has a multi-year lifetime and a different verifier. |
| Agent workload identity, delegation chain, scope attenuation | `agent-manager` | Already built and ecosystem-wide. This scenario verifies through it and caches nothing. |
| Financial position, runway, categorisation, tax treatment | `money-ledger` | Two authorities over the same numbers is the failure this split exists to prevent. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `books` | book | `api/internal/book/schema.sql` | book repository/service/handlers |
| `budgets`, `budget_scope_entries` | budget | `api/internal/budget/schema.sql` | budget repository/service/handlers; authorization evaluator (read-only) |
| `mandates`, `mandate_templates` | mandate | `api/internal/mandate/schema.sql` | mandate repository/service/handlers; authorization evaluator (read-only) |
| `authorizations`, `holds` | authorization | `api/internal/authorization/schema.sql` | authorization repository/service; budget headroom (read-only) |
| `approval_requests`, `relay_attempts` | approval | `api/internal/approval/schema.sql` | approval repository/service/handlers |
| `instruments` | instrument | `api/internal/instrument/schema.sql` | instrument repository; rail adapters (read-only) |
| `rails` | rail | `api/internal/rail/schema.sql` | rail registry |
| `charges`, `idempotency_keys` | settlement | `api/internal/settlement/schema.sql` | settlement repository/service |
| `evidence_records` | evidence | `api/internal/evidence/schema.sql` | evidence store; every domain writes through its seam |
| `ledger_emissions` | ledger | `api/internal/ledger/schema.sql` | ledger emitter |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

**Cross-domain reads are one-way and read-only.** The authorization
evaluator reads `mandates` and `budgets`; budget headroom reads
`authorizations` and `holds`. No domain writes another's tables. Where a
read crosses a boundary it goes through the owning domain's repository
interface, not through raw SQL, so the seam stays substitutable in tests.

## Migrations And Compatibility

Domain schema files use `CREATE TABLE IF NOT EXISTS` and live beside the
code that interprets them.

Two constraints are stronger here than in an ordinary scenario:

- **Evidence and emissions are append-only.** A migration may add a column
  or a table. It may not drop, rename in place, or backfill a value into an
  existing evidence row, because the point of the record is that it was not
  edited afterwards. Where a shape must change, add the new column, write
  new records with it, and leave old records readable as they were.
- **Idempotency keys must survive a migration.** Dropping or rewriting
  `idempotency_keys` converts every in-flight client retry into a potential
  double charge. A migration touching that table needs an explicit entry in
  [`../internal/DECISIONS.md`](../internal/DECISIONS.md) with its
  reasoning, not just a schema diff.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| Evidence export | JSON, one record per attempt with its full decision path | evidence | Planned. Needed the first time an operator has to answer an external question about an automated payment. |
| Budget and mandate configuration export | JSON | budget, mandate | Planned. Makes an operator's policy portable between instances and reviewable outside the console. |
| Rail statement import | Rail-defined | reconciliation (deferred) | Deferred with `TRS-P2-001`; there is nothing to match until an automated rail has settled real charges. |

No import path may create an evidence record retroactively. Imported data
is either configuration or a statement to reconcile against — never
history this scenario claims to have observed.

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Evidence records | None. | Indefinite, append-only. | No operator-facing purge exists by design. If a storage bound is ever needed, it must be an explicit decision, not a default. |
| Ledger emissions | None. | Indefinite. | — |
| Authorizations, holds, charges | None while referenced by evidence. | Retained with their evidence record. | — |
| Idempotency keys | Age. | 180 days after terminal outcome. | The window is a judgement, not a measurement. Revisit once real client retry behaviour is observed. |
| Relay attempts | Age. | 90 days. | Delivery telemetry, not evidence; safe to age out. |
| Mandates | None. | Retained after expiry while evidence references them. | — |
| Mandate templates | Operator deletion. | Until deleted. | Deleting a template never affects mandates already issued from it. |
| Books, budgets, rails | Operator retirement. | Archived, not deleted, while any charge references them. | — |
| Instruments | Operator revocation. | Record retained after revocation; the credential reference is cleared. | Revocation must also revoke at the rail, which is adapter-specific. |

**Deletion is deliberately weak here.** A scenario whose job is to prove
what was authorized cannot also offer to erase the proof. Storage growth is
bounded by authorization volume, which is low by construction for a
single-operator instance. If that assumption breaks, the answer is an
export-then-archive path with an explicit decision record — not a purge.

## Privacy Notes

This scenario stores **financial and operational data**, which raises the
bar above the template default. Specifically:

- **Counterparty names and amounts** are business-sensitive. They are not
  regulated personal data on their own, but together with book identity
  they describe an operator's spending in detail.
- **No cardholder data is stored.** No PAN, CVV, expiry or magnetic-stripe
  equivalent enters this scenario. Instruments carry a reference resolved
  through `secrets-manager` at use time. This keeps the scenario outside
  cardholder-data scope by construction rather than by policy.
- **No identity documents, ever.** They belong to `document-manager`. See
  the not-stored-here table above.
- **Evidence is the sensitive artefact.** An evidence record joins who
  authorized, what was bought, from whom, and for how much. Access to it is
  operator-realm only; the agent-facing service can read its own attempt
  outcomes and nothing else.
- **Emitted money events carry provenance, not narrative.** The payload to
  `money-ledger` names adapter, external id, fetch time, amount and basis.
  It does not carry the mandate's reasoning or the approval conversation.

Update this document and
[`../internal/SECURITY.md`](../internal/SECURITY.md) together whenever a
new data shape is added.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`FLOWS.md`](FLOWS.md) — how records are created and transitioned
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — storage and retention decisions
