# Domains — Money Ledger

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The shipped implementation has one persisted product domain, `ledger`, and
one operator-facing composition domain, `operator-ui`. The API transport and
shared UI primitives remain non-domains. This map follows the code that is
actually shipped; the ledger capability slices below are not separate runtime
packages or ownership boundaries.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/money-ledger/v1/shared/health.proto` |
| ledger | Own the complete append-only money capability: books, accounts, postings, reversals, derived position, statements, runway, and goals. | Every source enters through one contract and every figure is derived from the immutable journal, so stale or fabricated numbers remain visible as caveats rather than becoming facts. | Books, accounts, postings, audit entries, and goals. | service | policy, reporting | Book, Account, Posting, Reversal, Position, Goal | `api/internal/ledger/`, `cli/domains/ledger/` |
| ingest | Provide the adapter/provider boundary that admits manual and external money events into the ledger contract. | Keep volatile source credentials, cursors, receipts, and failure reasons at the edge while the ledger remains the financial authority. | Adapter registrations, sync cursors, ingestion receipts, and operator-input findings. | provider | integration, service | Adapter, MoneyEvent, Provenance, Basis, Cursor | `api/internal/ingest/`, `api/handlers/ingest/` |
| operator-ui | Compose the ledger's API surfaces into the responsive operator console and preserve state, provenance, and correction affordances at the point of use. | A financial figure is useful only when its basis, age, missing source, and sanctioned correction path are visible beside it. | Browser preferences only; no financial facts. | interface | query, review | Dashboard, Journal, Statement, Goal, AdapterStatus | `ui/src/pages/`, `ui/src/layout/`, `ui/src/consts/` |

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `reconciliation` | Needs an authoritative feed to reconcile *against*; reconciling against nothing is not a capability. | OT-P2-001, once a bank or processor adapter is live. |
| `inventory` | Per-item cost basis for resale. Only a lot *reference* is in scope; running a resale loop belongs to whatever capability actually does that, and none exists. | OT-P2-005, when a resale capability exists to need it. |
| `adapters` (as its own scenario) | The contract is the boundary today and the code lives here. | OT-P2-004 — a third live adapter, independent sync clocks, or credential rotation. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.
- The secret-store client used by authenticating adapters — a typed caller, not a bounded context.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Domain Details

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Requirements: starter scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

### Ledger capability slice: books

- **Owns**: books (accounting entities), accounts, and the account-kind vocabulary. Every account belongs to exactly one book.
- **Does not own**: any amount. A book knows its accounts; it does not know what is in them, because balances are derived (see `position`).
- **Key rule**: money moving between books is an explicit paired posting — an owner draw or contribution — never two unrelated events. Without the pairing, money leaving one book vanishes and both books read wrong.
- **Targets**: OT-P0-001. **Requirements**: JRNL-001, JRNL-002.

### Ledger capability slice: journal

- **Owns**: the append-only event and posting store, the audit trail, and reversal links.
- **Key rule**: nothing is ever edited or deleted. A correction is a new reversing entry that references what it reverses. This is what replaces the history and reviewability that version control provided before this state moved into a database.
- **Shape decision**: postings are signed and account-scoped from the first commit — the double-entry *data shape* — while the accounting reporting UX is deliberately excluded. The shape is nearly free to adopt now and painful to retrofit; the reports are the opposite.
- **Targets**: OT-P0-002, OT-P0-007. **Requirements**: JRNL-003, JRNL-004.

### Ledger capability slice: ingest

- **Owns**: the money-event contract, the adapter registry, per-adapter sync cursors, and ingestion receipts.
- **Key rule**: an adapter may only emit events through the contract. It may not write a balance, alter a goal, or reach any other domain. Adapters are the most volatile part of the system — auth expires, APIs drift, syncs half-complete — so confining them to one verb keeps that volatility away from state.
- **Deliberate design**: manual entry and file import are ordinary adapters with no special handling. Several real revenue sources have no API on any platform, so a design in which manual entry is a fallback fails for exactly those sources.
- **Targets**: OT-P0-004, OT-P0-005, OT-P0-006, OT-P0-007, OT-P1-005. **Requirements**: CTR-001…CTR-005, POS-005.

### Ledger capability slice: position

- **Owns**: goal declarations and cached position snapshots. It owns no financial fact and no balance.
- **Key rule**: every figure it returns names its inputs, each with a source and an age. A number without a basis is not reportable.
- **Deliberate design**: a goal is a declared threshold with a comparison, a sustain window, and an optional buffer multiple. The "default-alive" rule that motivated this scenario is one instance of that shape rather than a rule the code knows about.
- **Its main external consumer**: Offer Desk's `board` reads this domain for the `monetization` team's single address — per-offer actuals, plus runway, goal verdicts and the default-alive gap. Two consequences bind the read API. A partial position must remain **legibly partial to a caller**, because a caveat this domain attaches and a consumer cannot see is a caveat that will be dropped at the surface the operator actually reads. And a goal verdict must carry its sustain-window progress in the response rather than only in the UI, since the consumer renders the verdict and cannot re-derive the window.
- **Targets**: OT-P1-001, OT-P1-002, OT-P1-003, OT-P1-004, OT-P1-006. **Requirements**: POS-001…POS-004, POS-006.

### operator-ui

- **Owns**: route composition, responsive presentation, provenance labels,
  empty/degraded states, and operator correction affordances.
- **Does not own**: books, postings, balances, goals, or adapter state.
- **Key rule**: an unavailable or incomplete ledger source is rendered as an
  explicit state with its reason and age; it is never converted to zero.
- **Surfaces**: `ui/src/pages/` and the shared shell in `ui/src/layout/`.

## Build Order

The ledger capability slices are one implementation domain, so their order is
an internal build concern rather than a cross-domain dependency. Ingest writes
through the journal admission method; position and statements read it. The
operator UI reads the public API and never owns ledger state.

`position` does have an external reader — Offer Desk's `board` reads it for the actuals join and for financial posture — but that is a consumer of this scenario's public read API, not an internal dependency, and it does not affect build order. The direction is one-way: nothing here reads Offer Desk (see [`INTEGRATIONS.md`](INTEGRATIONS.md) and `../internal/DECISIONS.md`, 2026-08-13).

```
ledger (books + journal + ingest + position)  →  operator-ui
```

Build the ledger as one vertical slice across proto, API, CLI, and persistence;
then compose it in the operator UI. Do not create a second domain package for
an implementation slice unless it acquires an independent ownership boundary.

## Deliberately Excluded

These are real parts of managing money and are **not** domains here. Each is excluded for a stated reason, so a later reader can tell a decision from an omission.

| Excluded | Why |
|---|---|
| Tax computation and filing | Jurisdictional, regulated, and a liability. This scenario tags deductibility categories and never computes an amount. |
| Payroll | Regulated, and nothing about it composes with the rest of this scenario. |
| Invoicing and accounts receivable | A separate product shape. Subscription billing belongs to the commerce scenario upstream. |
| Accrual accounting and revenue recognition | Cash basis is correct for the intended user and far simpler. |
| FX gain/loss | Every amount carries a currency; there is no rate engine and no realised-FX modelling. |
| Direct bank credential storage | Holding bank logins is a serious liability. Aggregator APIs, file import, or nothing. |
| Inventory management | Only a cost-basis *lot reference* is in scope (P2). Running a resale loop belongs to whatever capability actually does that, and none exists. |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
