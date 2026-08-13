# Domains — Money Ledger

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`health` is the one real domain the scaffold ships. Add your scenario's
domains to the inventory below as you build them. The scaffold also ships
one clearly fenced worked example domain (never product scope) as a
copyable reference; `template-manager detemplate <scenario>` removes every
fenced example once your real domains are green.

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
| books | Own accounting entities and the accounts inside them. | Personal and business money share one system without being mixed; a solo operator's finances genuinely overlap, so the separation is a boundary and a filter rather than a second installation. | Books, accounts, inter-book transfer links. | crud | policy | Book, Account, AccountKind, Transfer | `api/internal/books/` |
| journal | Own money events as dated signed postings, the append-only audit trail, and the reversing-entry correction path. | One place every money event lands, in one shape, that cannot be edited after the fact. | Events, postings, audit entries, reversal links. | crud | policy | Event, Posting, Reversal, AuditEntry | `api/internal/journal/` |
| ingest | Own the money-event contract and the adapters that satisfy it, including manual entry and file import. | Every source — an API, a file, a person typing a number — enters through exactly one door, which is what makes any upstream pluggable. | Adapter registrations, sync cursors, ingestion receipts. | integration | service | Adapter, MoneyEvent, Provenance, Basis, Cursor | `api/internal/ingest/` |
| position | Compute balances, cash flow, runway, statements, and goal verdicts from the journal. Owns no financial facts. | Every figure is a query, so a stale number is structurally impossible. | Goal declarations and position snapshots only; never a balance. | reporting | policy | Position, Goal, Threshold, Runway, Statement | `api/internal/position/` |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships `notes` as a worked CRUD vertical slice with a binary
upload exception. Copy its shape for your own domains, then remove it.

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| notes | Provide the worked CRUD reference with attachment upload exception. | Demonstrate the expected vertical slice for a real domain. | Notes and attachment metadata. | crud | service | Note, Attachment | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/money-ledger/v1/notes/` |

- Purpose: demonstrate the expected vertical slice for a real domain.
- Primary archetype: CRUD / entity.
- Secondary traits: binary/blob attachment upload, upload workflow.
- Owns: note records, attachment metadata, note validation, note
  service/repository seams, UI note interactions, CLI notes commands.
- Does not own: product scope for a generated scenario.
- API: `api/internal/notes/`, `api/handlers/notes/`.
- CLI: `cli/domains/notes/`.
- UI: `ui/src/features/notes/`, `ui/src/api/notes.ts`.
- Storage: domain-owned SQLite schema in `api/internal/notes/schema.sql`.
- Requirements: template starter only; replace with PRD-specific
  requirements.
- Tests: repository, service, handler, CLI, UI, accessibility, and
  workflow tests.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).
<!-- EXAMPLE-DOMAIN:notes END -->

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

### books

- **Owns**: books (accounting entities), accounts, and the account-kind vocabulary. Every account belongs to exactly one book.
- **Does not own**: any amount. A book knows its accounts; it does not know what is in them, because balances are derived (see `position`).
- **Key rule**: money moving between books is an explicit paired posting — an owner draw or contribution — never two unrelated events. Without the pairing, money leaving one book vanishes and both books read wrong.
- **Targets**: OT-P0-001. **Requirements**: JRNL-001, JRNL-002.

### journal

- **Owns**: the append-only event and posting store, the audit trail, and reversal links.
- **Key rule**: nothing is ever edited or deleted. A correction is a new reversing entry that references what it reverses. This is what replaces the history and reviewability that version control provided before this state moved into a database.
- **Shape decision**: postings are signed and account-scoped from the first commit — the double-entry *data shape* — while the accounting reporting UX is deliberately excluded. The shape is nearly free to adopt now and painful to retrofit; the reports are the opposite.
- **Targets**: OT-P0-002, OT-P0-007. **Requirements**: JRNL-003, JRNL-004.

### ingest

- **Owns**: the money-event contract, the adapter registry, per-adapter sync cursors, and ingestion receipts.
- **Key rule**: an adapter may only emit events through the contract. It may not write a balance, alter a goal, or reach any other domain. Adapters are the most volatile part of the system — auth expires, APIs drift, syncs half-complete — so confining them to one verb keeps that volatility away from state.
- **Deliberate design**: manual entry and file import are ordinary adapters with no special handling. Several real revenue sources have no API on any platform, so a design in which manual entry is a fallback fails for exactly those sources.
- **Targets**: OT-P0-004, OT-P0-005, OT-P0-006, OT-P0-007, OT-P1-005. **Requirements**: CTR-001…CTR-005, POS-005.

### position

- **Owns**: goal declarations and cached position snapshots. It owns no financial fact and no balance.
- **Key rule**: every figure it returns names its inputs, each with a source and an age. A number without a basis is not reportable.
- **Deliberate design**: a goal is a declared threshold with a comparison, a sustain window, and an optional buffer multiple. The "default-alive" rule that motivated this scenario is one instance of that shape rather than a rule the code knows about.
- **Targets**: OT-P1-001, OT-P1-002, OT-P1-003, OT-P1-004, OT-P1-006. **Requirements**: POS-001…POS-004, POS-006.

## Build Order

Domains that read other domains decide the order. `position` reads `journal`; `journal` reads `books`; `ingest` writes through `journal`. Nothing reads `position`, and no two domains read each other.

```
books  →  journal  →  { ingest (writes), position (reads) }
```

Build `books` and `journal` as one vertical slice, then `ingest` with the manual and file adapters, then `position`. Do not build every API, then every CLI, then every UI — finish one domain across proto, API, transport, CLI and UI before starting the next.

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
