# Data — Token Economy

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

Embedded SQLite through `modernc.org/sqlite`, via the routed scenario storage
seam so test isolation is honored per request. The path is resolved by `api-core/storage` from the scenario id
through `.vrooli/service.json`, and the API applies schemas on startup through
`api-core/database`.

**SQLite is the terminal choice here, not a starting point.** Volume is
inherently low — a household, not a market — and every mutation is
single-writer under a row lock. The one condition that would revisit it is the
P2 real-chain adapter (`TKE-P2-001`), which introduces concurrency the scenario
does not otherwise face. That decision is recorded in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

**The journal is the source of truth for every quantity.** Balances exist as a
projection table for read performance, but they are derived and rebuildable;
if projection and journal ever disagree, the journal wins and the projection is
rebuilt. No code path may treat a projection row as authoritative
(`TKE-P0-004`).

External storage resources should be introduced only when a real
domain needs them. Document those decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md) before editing
`.vrooli/service.json`.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Token types | mints | SQLite | `api/internal/mints/schema.sql` | Retired, never deleted — events reference the type forever | Carries supply policy, transfer policy, display identity. Carries **no** monetary field, by contract (`TKE-P0-014`). |
| Minter authority bindings | mints | SQLite | `api/internal/mints/schema.sql` | Until the authority is revoked; revocation is a row, not an erase | Binds a token type to a `scenario-authenticator` subject. Never stores a credential. |
| Journal events | journal | SQLite | `api/internal/journal/schema.sql` | **Permanent. Never compacted, never deleted.** | Append-only. The repository exposes no update or delete, asserted structurally by test (`TKE-P0-010`). |
| Balance projections | journal | SQLite | Derived — rebuilt from journal events | Rebuildable at any time; safe to truncate | A cache, never an authority. Reconstruction equality is tested (`TKE-P0-004`). |
| Actor provenance | journal | SQLite (columns on the event row) | `api/internal/journal/schema.sql` | Same lifetime as its event | Records actor identity and verification status, resolved through `packages/cli-core` against agent-manager claims (`TKE-P0-011`). |
| Grants | grants | SQLite | `api/internal/grants/schema.sql` | Retained after expiry; expiry is a state, not a delete | The mandate-shaped object. Field set is parity-tested against the `treasury` mandate. |
| Grant rules | grants | SQLite | `api/internal/grants/schema.sql` | Same lifetime as the grant | Declared conditions from a closed vocabulary. Never caller-supplied code (`TKE-P1-002`). |
| Grant schedules | grants | SQLite | `api/internal/grants/schema.sql` | Until cancelled; cancellation stops future issuance only | P1. Carries the catch-up policy for a missed window. |
| Holders | holders | SQLite | `api/internal/holders/schema.sql` | Tombstoned on removal, never erased — their events must remain readable | Binds to an authenticator subject. See Privacy Notes. |
| Earning submissions | earning | SQLite | `api/internal/earning/schema.sql` | Permanent at household scale | Adapter-scoped dedup keys make retries idempotent forever. The row stores a payload summary and reason digest, not the raw payload. |
| Catalog entries | catalog | SQLite | `api/internal/catalog/schema.sql` | Retired, never deleted — redemptions reference them | Carries availability, quantity and approval posture. |
| Redemptions | redemption | SQLite | `api/internal/redemption/schema.sql` | Permanent | Carries the idempotency key, decision, decider and reason. |
| Reservations | redemption | SQLite | `api/internal/redemption/schema.sql` | Until released or consumed | One mechanism, two uses: pending redemptions and P1 savings goals. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `token_types`, `minter_authorities` | mints | `api/internal/mints/schema.sql` | mints repository/service/handlers; read by every other domain |
| `journal_events`, `balance_projections` | journal | `api/internal/journal/schema.sql` | journal repository; written only through grants/redemption service paths |
| `grants`, `grant_rules`, `grant_schedules` | grants | `api/internal/grants/schema.sql` | grants repository/service/handlers |
| `holders` | holders | `api/internal/holders/schema.sql` | holders repository; scoping filter applied by every holder-facing read |
| `earning_submissions` | earning | `api/internal/earning/schema.sql` | earning repository/service/handlers |
| `catalog_entries` | catalog | `api/internal/catalog/schema.sql` | catalog repository/service/handlers |
| `redemptions`, `reservations` | redemption | `api/internal/redemption/schema.sql` | redemption repository/service/handlers |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

## The append-only contract

This is the load-bearing storage decision and it constrains every future
change:

- `journal_events` rows are **inserted only**. The repository exposes no
  update and no delete, and a structural test asserts that.
- A mistake is corrected by appending a **compensating event** that references
  the original and carries a reason (`TKE-P0-010`). Both remain visible in a
  holder's history, which is what makes the economy teachable rather than
  arbitrary.
- Debit and event write commit in **one transaction under a row lock**, keyed
  by a caller-supplied idempotency key. A repeated key is a successful no-op;
  distinct keys are independent (`TKE-P0-009`). This follows the
  `landing-page-business-suite` credit-wallet invariant, which is the proven
  in-house pattern for exactly this problem.
- The failure that would corrupt this store — a debit without its event, or an
  event without its debit — has **no repair verb by design**, so it is tested
  against induced failure rather than assumed.

## Migrations And Compatibility

Idempotent schema bootstrap: domain schema files use
`CREATE TABLE IF NOT EXISTS` and live beside the code that interprets them.

Because `journal_events` is permanent and never compacted, its shape is the
most expensive thing in the scenario to change. Two rules follow:

1. **Event columns are additive only.** A new event kind or a new column is
   fine; changing the meaning of an existing column is not, because historical
   rows cannot be reinterpreted.
2. **Projections absorb change instead.** If a derived quantity needs to be
   computed differently, change the projection and rebuild it — the events do
   not move.

For any migration needing a column drop, rename, or backfill, add a
scenario-specific migration plan here and record the tradeoff in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| Journal export | Versioned JSON matching API event semantics | journal | Planned (`TKE-P1-010`). Holder exports are isolation-scoped to their own events. |
| Import | — | — | Deliberately absent. An importable journal would let a caller assert history the system never observed, which defeats the audit property the scenario is built on. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Journal events | None. | Permanent, uncompacted. | Growth is bounded by household volume; revisit only if a non-household deployment appears. |
| Balance projections | Rebuild. | Derived; safe to truncate at any time. | None — truncation is a supported operation. |
| Holders | Tombstone on removal. | Record retained so their events stay attributable. | Hard-delete semantics for a departing holder are undefined; see Privacy Notes. |
| Token types, catalog entries | Retire, never delete. | Retained because events and redemptions reference them. | None. |
| Earning submissions | None. | Permanent dedup outcome plus a non-reversible reason digest; the raw payload is not retained. | Household volume makes permanent keys safer and simpler than an expiry window. Revisit only with non-household adapter volume. |
| Grants, redemptions, reservations | None (state transitions only). | Permanent. | None. |

## Privacy Notes

**This scenario stores data about children, and that deserves stating plainly
rather than being discovered later.** A household deployment records a minor's
name (or nickname), what they did to earn, what they chose to redeem, and when
— a behavioral record over time.

What makes the posture defensible:

- **Nothing leaves the machine.** No third-party processor, no analytics
  vendor, no cloud sync. The self-hosted deployment is the privacy story, and
  it is the same wedge `document-manager` makes for documents.
- **No regulated identifiers.** A holder record binds to a
  `scenario-authenticator` subject and a display name. No date of birth, no
  address, no contact details, no payment instrument — because the scenario has
  no use for any of them.
- **No monetary value** (`TKE-P0-014`), so no financial record about a minor
  exists to protect in the first place.
- **The isolation boundary is enforced at the repository layer**
  (`TKE-P0-006`), not only at the handler, so a future handler written without
  the check still cannot leak one holder's history to another.

Open questions to resolve before any non-self-hosted deployment is
contemplated:

- **Hard deletion for a departing holder.** Tombstoning keeps the journal
  intact, which is correct for audit and wrong for erasure. If this scenario is
  ever offered as a hosted product, an erasure path that preserves aggregate
  integrity must be designed first — not retrofitted.
- **Export scope.** A holder's export contains their full behavioral history;
  the P1 export must be isolation-scoped and should not be reachable by a
  shared link.

If deployment assumptions change, update this section and
[`../internal/SECURITY.md`](../internal/SECURITY.md) **before** implementation
expands.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — storage and retention decisions
