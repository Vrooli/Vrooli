# Domains — Token Economy

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The scenario has **seven product domains** plus `health`. They are listed
below in dependency order, which is also the build order in the PRD's launch
sequencing: nothing exists without a declared type (`mints`), no verb may run
before the journal exists to record it (`journal`), and the console is last
because the loop it presents must be real first.

The scaffold also ships one clearly fenced worked example domain (never
product scope) as a copyable reference; `template-manager detemplate
token-economy` removes every fenced example once the real domains are green.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## The two-service split

Before reading the inventory, note the structural rule that cuts across every
domain: **there are two Connect services, not one.** A *minter* service owns
mint, grant, catalog and rule mutation. A *holder* service owns view, redeem
and request. The split is visible in the proto and enforced by codegen — a
holder presenting a valid token cannot mint, because the RPC is not on the
service they can reach (`TKE-P0-005`). Every domain below states which service
exposes it.

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/token-economy/v1/shared/health.proto` |
| mints | Own token types, supply policy, minter authority, and the no-monetary-value constraint. | Nothing in the economy exists outside a declared type; this domain is that declaration. | Token types, supply policy, minter authority bindings. | registry | policy | TokenType, SupplyPolicy, MinterAuthority | `api/internal/mints/`, `api/handlers/mints/`, `cli/domains/mints/`, `ui/src/features/mints/`, `packages/proto/schemas/token-economy/v1/mints/` |
| journal | Own the append-only event store and the balance projection over it. | A balance is a query, never an assertion; this domain is the only authority on what happened. | Events, projection cache, provenance. | ledger | query | Event, Projection, Reversal, Provenance | `api/internal/journal/`, `api/handlers/journal/`, `cli/domains/journal/`, `ui/src/features/journal/`, `packages/proto/schemas/token-economy/v1/journal/` |
| grants | Own the one typed grant that admits tokens to a holder, and the rules that survive it. | The mandate-shaped object that makes every credit auditable and every redemption checkable. | Grants, rules, schedules, expiry policy. | policy | service | Grant, Rule, RuleProgram, Schedule | `api/internal/grants/`, `api/handlers/grants/`, `cli/domains/grants/`, `ui/src/features/grants/`, `packages/proto/schemas/token-economy/v1/grants/` |
| holders | Own holder identity, the isolation boundary, and peer transfer. | Multiple people share one instance; this domain is why they cannot see each other. | Holder records, authentication bindings, transfer policy. | registry | policy, query | Holder, Isolation, Transfer | `api/internal/holders/`, `api/handlers/holders/`, `cli/domains/holders/`, `ui/src/features/holders/`, `packages/proto/schemas/token-economy/v1/holders/` |
| earning | Own the one inbound contract through which anything can grant tokens for work done. | Any Vrooli scenario becomes an earning surface without this scenario knowing it exists. | Earning submissions and their dedup keys. | gateway | service | EarningEvent, Adapter, Submission | `api/internal/earning/`, `api/handlers/earning/`, `cli/domains/earning/`, `ui/src/features/earning/`, `packages/proto/schemas/token-economy/v1/earning/` |
| catalog | Own what a token buys, declared by the minter with availability and approval posture. | The product ships no built-in redeemables; the household defines its own economy. | Catalog entries, availability, approval posture. | crud | policy | CatalogEntry, Availability, ApprovalPosture | `api/internal/catalog/`, `api/handlers/catalog/`, `cli/domains/catalog/`, `ui/src/features/catalog/`, `packages/proto/schemas/token-economy/v1/catalog/` |
| redemption | Own settlement, reservation, and the approval queue. | Turning balance into a thing received, exactly once, with the minter in the loop where declared. | Redemptions, reservations, approval state. | workflow | service | Redemption, Reservation, ApprovalQueue, IdempotencyKey | `api/internal/redemption/`, `api/handlers/redemption/`, `cli/domains/redemption/`, `ui/src/features/redemption/`, `packages/proto/schemas/token-economy/v1/redemption/` |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships `notes` as a worked CRUD vertical slice with a binary
upload exception. Copy its shape for your own domains, then remove it.

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| notes | Provide the worked CRUD reference with attachment upload exception. | Demonstrate the expected vertical slice for a real domain. | Notes and attachment metadata. | crud | service | Note, Attachment | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/token-economy/v1/notes/` |

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

## Domain Details

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior.
- Service exposure: both (unauthenticated readiness only).
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Requirements: scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

### mints

- Purpose: declare token types so nothing in the economy can exist without one.
- Primary archetype: registry.
- Secondary traits: policy (supply enforcement, structural constraints).
- Owns: token type records (identifier, display identity, symbol, color),
  supply policy (unbounded / capped / fixed), the named minter authority, and
  the **absence** of any monetary field — the no-real-value constraint is this
  domain's responsibility because it is enforced by what the type cannot carry.
- Does not own: who holds tokens (`holders`), how many exist right now
  (`journal`), or what they buy (`catalog`).
- Service exposure: minter service only. There is no holder-reachable mutation.
- API: `api/internal/mints/`, `api/handlers/mints/`.
- CLI: `cli/domains/mints/` — type create, list, show, retire.
- UI: `ui/src/features/mints/` — token type management in the minter console.
- Storage: `api/internal/mints/schema.sql`; token types and authority bindings.
- Requirements: `TKE-P0-001`, `TKE-P0-005`, `TKE-P0-014`; later
  `TKE-P1-001` (non-fungible), `TKE-P1-007` (multi-type), `TKE-P2-005`
  (delegation), `TKE-P2-006` (reputation tokens).
- Tests: supply-policy boundary, implicit-creation refusal, structural
  no-monetary-value assertion, service-surface authority assertion.
- Related docs: [`DATA.md`](DATA.md), [`../internal/SECURITY.md`](../internal/SECURITY.md).

### journal

- Purpose: hold every event and answer every balance question from them.
- Primary archetype: ledger.
- Secondary traits: query (projection), reporting (export).
- Owns: the append-only event table, the projection cache, actor provenance on
  each event, compensating-event semantics, and export. It is the **only**
  authority on balance; every other domain asks rather than asserts.
- Does not own: authorization to create an event (`grants`, `redemption`), or
  what an event means to a user (`holders` view, `catalog`).
- Service exposure: minter service (full journal); holder service (own events
  only, enforced by `holders` isolation).
- API: `api/internal/journal/`, `api/handlers/journal/`.
- CLI: `cli/domains/journal/` — events list, balance show, export.
- UI: `ui/src/features/journal/` — the minter's audit surface.
- Storage: `api/internal/journal/schema.sql`. Append-only: the repository
  exposes no update or delete for event rows, asserted by test.
- Requirements: `TKE-P0-004`, `TKE-P0-010`, `TKE-P0-011`; later
  `TKE-P1-008` (expiry materialization), `TKE-P1-010` (export),
  `TKE-P2-007` (analytics).
- Tests: projection-equals-cache replay, append-only structural assertion,
  reversal referencing, provenance status matrix.
- Related docs: [`DATA.md`](DATA.md), [`FLOWS.md`](FLOWS.md).

### grants

- Purpose: own the single typed object that admits tokens to a holder, and the
  rules that outlive the grant.
- Primary archetype: policy.
- Secondary traits: service (rule evaluation), workflow (schedules, P1).
- Owns: grant records (holder, amount, scope, expiry, provenance), rule
  definitions, server-side rule evaluation, recurring schedules and catch-up
  policy (P1), expiry and decay policy (P1), and declared rule programs (P1).
- Does not own: the events those grants produce (`journal`), whether a
  redemption is approved (`redemption`), or who may grant (`mints` authority).
- Service exposure: minter service for mutation; holder service may read the
  rules that apply to them so a refusal is explainable.
- API: `api/internal/grants/`, `api/handlers/grants/`.
- CLI: `cli/domains/grants/` — grant issue, list, revoke, schedule.
- UI: `ui/src/features/grants/` — grant issuance and rule authoring.
- Storage: `api/internal/grants/schema.sql`.
- Requirements: `TKE-P0-002`, `TKE-P0-003`; later `TKE-P1-002` (rule
  programs), `TKE-P1-003` (recurring), `TKE-P1-008` (expiry),
  `TKE-P2-004` (treasury contract unification).
- Tests: single-inbound-shape reachability, treasury contract parity, rule
  evaluation table, client-asserted-authorization rejection.
- Related docs: [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — the
  treasury congruence decision and every recorded divergence.

### holders

- Purpose: represent the people who share this instance and keep them apart.
- Primary archetype: registry.
- Secondary traits: policy (transfer), query (isolation-scoped reads).
- Owns: holder records, the binding to `scenario-authenticator` identities, the
  isolation boundary enforced at the repository layer, and peer transfer policy
  (P1).
- Does not own: balances (`journal`), what a holder may redeem (`grants`,
  `catalog`), or authentication itself (`scenario-authenticator`).
- Service exposure: minter service reads all; holder service reads self only.
- API: `api/internal/holders/`, `api/handlers/holders/`.
- CLI: `cli/domains/holders/` — holder add, list, show, transfer.
- UI: `ui/src/features/holders/` — holder list (minter) and the holder's own
  balance-and-history view.
- Storage: `api/internal/holders/schema.sql`.
- Requirements: `TKE-P0-006`, `TKE-P0-012`; later `TKE-P1-004`
  (transfer), `TKE-P2-003` (marketplace).
- Tests: repository-layer scoping independent of handler, cross-holder refusal
  that does not disclose existence, two-session BAS isolation case.
- Related docs: [`../internal/SECURITY.md`](../internal/SECURITY.md),
  [`INTEGRATIONS.md`](INTEGRATIONS.md).

### earning

- Purpose: be the one door through which work becomes tokens.
- Primary archetype: gateway.
- Secondary traits: service (dedup, provenance capture).
- Owns: the inbound earning contract, submission dedup keys, and the mapping
  from an earning event to a grant request. Operator entry is an ordinary
  satisfier of this contract, not a separate path.
- Does not own: the grant itself (`grants`), or any knowledge of which scenario
  is calling — a privileged earner would defeat the contract's purpose.
- Service exposure: a distinct adapter-facing surface authenticated per
  adapter; never the holder service.
- API: `api/internal/earning/`, `api/handlers/earning/`.
- CLI: `cli/domains/earning/` — earning submit, list.
- UI: `ui/src/features/earning/` — operator entry in the minter console.
- Storage: `api/internal/earning/schema.sql`; submissions and dedup keys only.
- Requirements: `TKE-P0-007`; later `TKE-P1-009` (first real adapter).
- Tests: identical code path for operator and programmatic satisfiers, replay
  idempotency, provenance preserved to the journal.
- Related docs: [`INTEGRATIONS.md`](INTEGRATIONS.md), [`FLOWS.md`](FLOWS.md).

### catalog

- Purpose: hold what tokens buy, as declared by the minter.
- Primary archetype: crud.
- Secondary traits: policy (availability, approval posture).
- Owns: catalog entries (title, cost in a named type, availability window or
  quantity, approval posture), and server-side availability enforcement.
- Does not own: whether a specific holder can afford or is permitted an entry
  (`grants` rules), or the act of redeeming (`redemption`).
- Service exposure: minter service for mutation; holder service reads the
  entries available to them.
- API: `api/internal/catalog/`, `api/handlers/catalog/`.
- CLI: `cli/domains/catalog/` — catalog add, list, update, retire.
- UI: `ui/src/features/catalog/` — catalog authoring (minter) and the
  browsable reward list (holder).
- Storage: `api/internal/catalog/schema.sql`.
- Requirements: `TKE-P0-008`, `TKE-P0-013`; later `TKE-P1-007`
  (type-scoped catalogs).
- Tests: availability and window enforcement independent of UI filtering,
  direct-request refusal for unavailable entries.
- Related docs: [`FLOWS.md`](FLOWS.md).

### redemption

- Purpose: turn balance into a thing received, exactly once.
- Primary archetype: workflow.
- Secondary traits: service (settlement, reservation).
- Owns: redemption records, idempotent settlement under a row lock, balance
  reservation while a redemption is pending or saved toward, the approval queue
  and its state transitions, and denial with a recorded reason.
- Does not own: what may be redeemed (`catalog`), whether the rules permit it
  (`grants`), or the resulting events (`journal`).
- Service exposure: holder service requests; minter service approves or denies.
- API: `api/internal/redemption/`, `api/handlers/redemption/`.
- CLI: `cli/domains/redemption/` — redeem, approvals list, approve, deny.
- UI: `ui/src/features/redemption/` — the approval queue (minter) and the
  redeem action plus pending state (holder).
- Storage: `api/internal/redemption/schema.sql`.
- Requirements: `TKE-P0-009`, `TKE-P0-013`; later `TKE-P1-005`
  (goals and reservations), `TKE-P2-003` (marketplace listings).
- Tests: idempotency-key no-op, failure injection between debit and event,
  reservation prevents double-spend while pending, approval works with
  `notification-hub` absent.
- Related docs: [`FLOWS.md`](FLOWS.md), [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md).

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |
| Minter | The declared authority for a token type; the only role that may mint, grant, or change rules. | `mints`. |
| Holder | A person on this instance who can earn, hold and redeem, and who can see only themselves. | `holders`. |
| Grant | The one typed object that admits tokens to a holder, carrying rules that outlive it. Congruent with a `treasury` mandate by design. | `grants`. |
| Event | An append-only journal record. Balance is a projection over events, never a stored total. | `journal`. |
| Reservation | Balance made unspendable while a redemption is pending or a goal is saved toward. One mechanism, two uses. | `redemption`. |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `rails` | A real-value or cross-instance settlement rail (`TKE-P2-001`, `TKE-P2-002`) would be a genuine bounded context, but opening it re-introduces custody and regulatory questions P0 deliberately closed. | A recorded custody and regulatory decision in `../internal/DECISIONS.md`, never merely a feature request. |
| `marketplace` | Listings and offers (`TKE-P2-003`) are a distinct context from peer transfer, but building a market over an economy nobody trades in produces a surface with no users. | Observed transfer usage in a real household after `TKE-P1-004` ships. |
| `analytics` | Trend reporting (`TKE-P2-007`) is a projection over `journal` and may never need its own context. | Only if analytics acquires its own storage or its own vocabulary; otherwise it stays a `journal` read model. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.
- **Notification delivery** — relayed through `notification-hub` when present;
  this scenario owns the approval *queue*, never a delivery channel.
- **Authentication** — owned by `scenario-authenticator`; `holders` binds to an
  identity but never stores a credential.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — durable choices,
  including treasury contract congruence
