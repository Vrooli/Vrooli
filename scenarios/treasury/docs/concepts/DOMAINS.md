# Domains — Treasury

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The domain boundaries follow the life of one authorization: a **book**
contains a **budget**, a budget backs a **mandate**, a mandate is tested by
**authorization**, an authorization may wait on **approval**, an approved
charge draws an **instrument** through a **rail**, **settlement** makes that
movement exactly-once, **evidence** records what happened, and **ledger**
tells `money-ledger` about it. Each of those is a boundary you could delete
and name what stopped working, which is the test for whether a domain is
real.

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
| book | Contain custody and separate personal from business. | Keep one operator's contexts from borrowing each other's instruments, budgets, or approvers, and hold the operator-funds-only boundary. | Books, beneficiary identity. | crud | policy | Book, Beneficiary | `api/internal/book/` |
| budget | Hold caps, counterparty scope, gating intent, and freeze state. | Give an operator one place to say how much may be spent, on what, and whether a human must agree first. | Budgets, caps, allow/deny entries, freeze state. | crud | policy, reporting | Budget, Cap, Headroom, Freeze | `api/internal/budget/` |
| mandate | Issue and expire the scoped grant that authorizes a spend. | Make authority a signed object with a cap and an expiry rather than a credential handed over. | Mandates, recurrence, templates. | crud | workflow | Mandate, Standing Mandate, Scope, Expiry | `api/internal/mandate/` |
| authorization | Decide whether one proposed charge sits inside a live grant. | Keep the decision server-side and away from the agent that read untrusted content. | Authorization records, verdicts, holds. | policy | service | Authorization, Verdict, Hold | `api/internal/authorization/` |
| approval | Hold charges awaiting a human and resolve them. | Own the gate in-scenario so approval never depends on another scenario being up. | Approval requests, resolutions, relay attempts. | workflow | service | Approval Request, Resolution | `api/internal/approval/` |
| instrument | Hold payment instruments and scope them to a mandate. | Ensure what reaches a counterparty is already bounded, so a compromised flow cannot overspend. | Instrument records and credential references. | crud | gateway | Instrument, Scoped Instrument | `api/internal/instrument/` |
| rail | Define the execution contract and host its adapters. | Let a new payment mechanism arrive as an adapter rather than a rewrite. | Rail registrations, adapter config. | gateway | service | Rail, Adapter, Facilitator | `api/internal/rail/` |
| settlement | Execute an authorized charge exactly once. | Make retry safe so a network failure never becomes a double charge. | Charges, idempotency keys, outcomes. | service | workflow | Charge, Idempotency Key, Outcome | `api/internal/settlement/` |
| evidence | Retain one replayable record per spend attempt. | Make every decision reconstructable afterwards, including the ones that refused. | Append-only evidence records. | reporting | query | Evidence Record, Replay | `api/internal/evidence/` |
| ledger | Emit settled movement to `money-ledger`. | Keep the journal complete without giving it any knowledge of this scenario. | Emission log, emission idempotency. | gateway | service | Money Event, Basis, Emission | `api/internal/ledger/` |
| identity | Verify the calling agent and resolve its authority. | Refuse spend that cannot be attributed, rather than recording it at a weaker grade. | No durable data; caches nothing across requests. | gateway | policy | Verified Claims, Fail Closed | `api/internal/identity/` |
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/treasury/v1/shared/health.proto` |

## Domain Details

### book

- Purpose: contain custody, and keep personal and business contexts from
  borrowing each other's resources.
- Primary archetype: CRUD / entity, with a policy trait.
- Owns: book records; the beneficiary identity that every instrument and
  balance resolves against; the schema-level constraint that admits exactly
  one beneficiary so a third-party balance has nowhere to be stored.
- Does not own: what may be spent (that is `budget`) or who authorized it
  (that is `mandate`).
- Why it exists as its own domain: `TRS-P0-010` is enforced at a schema
  boundary rather than by a check that a later handler could forget. That
  boundary needs an owner.
- API: `api/internal/book/`, `api/handlers/treasuryadmin/`.
- CLI: `treasury books list|show|create`.
- UI: book switcher in the shell; book settings surface.
- Storage: `books` table; see [`DATA.md`](DATA.md).
- Requirements: `TRS-P0-010`, `TRS-P1-004`.
- Tests: isolation, custody boundary, schema constraint.

### budget

- Purpose: give an operator one place to state how much may be spent, on
  what, and whether a human must agree first.
- Primary archetype: CRUD / entity, with policy and reporting traits.
- Owns: cap dimensions (total, periodic, per-transaction); counterparty
  allow and deny entries; the `requires_approval` declaration; freeze state
  at budget scope; headroom computation from this scenario's own records.
- Does not own: the evaluation itself (that is `authorization`), or
  financial position (that is `money-ledger`, permanently).
- Headroom rule: computed, never stored as a mutable field. Pending
  authorizations reduce headroom before they settle, so two agents planning
  concurrently cannot both believe the same money is available.
- API: `api/internal/budget/`.
- CLI: `treasury budgets list|show|create|freeze|thaw|headroom`.
- UI: budget list, budget detail with cap meters, freeze control.
- Storage: `budgets`, `budget_scope_entries`; see [`DATA.md`](DATA.md).
- Requirements: `TRS-P0-003`, `TRS-P1-006`, `TRS-P1-007`.
- Tests: per-dimension refusal, deny-outranks-allow, period rollover,
  headroom under pending holds, freeze scope.

### mandate

- Purpose: make authority an object rather than a credential.
- Primary archetype: CRUD / entity, with a workflow trait for standing
  mandates.
- Owns: mandate records with authorizer, cap, counterparty scope, expiry
  and required evidence; recurrence for standing mandates; mandate
  templates; the rule that expiry binds from the mandate's own timestamp
  with no sweep job required.
- Does not own: whether a specific charge passes (that is `authorization`).
- Expiry rule: evaluated at authorization *and* at settlement, because a
  mandate can expire in the gap between them.
- API: `api/internal/mandate/`.
- CLI: `treasury mandates issue|list|show|revoke|templates`.
- UI: mandate detail, standing-mandate list with next charge date and
  one-action cancel.
- Storage: `mandates`, `mandate_templates`; see [`DATA.md`](DATA.md).
- Requirements: `TRS-P0-001`, `TRS-P0-012`, `TRS-P1-005`, `TRS-P2-006`.
- Tests: construction validation, expiry at both checkpoints, standing
  recurrence, template non-widening.

### authorization

- Purpose: decide whether one proposed charge sits inside a live grant.
- Primary archetype: policy / rules engine.
- Owns: the evaluator; authorization records and their verdicts; the hold
  that reserves headroom between decision and settlement.
- Does not own: configuration (`budget`, `mandate`) or execution
  (`settlement`).
- Trust rule: the evaluator reads only stored state. Caller-supplied
  verdict, allowance or override fields are ignored when present rather
  than rejected, so a probing caller learns nothing from the difference.
- Why it is separate from `budget`: budget is configuration with an
  operator lifecycle; authorization is an event with a request lifecycle.
  Merging them would put a hot path inside a CRUD surface.
- API: `api/internal/authorization/`, `api/handlers/agentspend/`.
- CLI: `treasury authorize` (agent-facing), `treasury authorizations list`.
- UI: authorization history with verdict and refusing constraint.
- Storage: `authorizations`, `holds`; see [`DATA.md`](DATA.md).
- Requirements: `TRS-P0-002`, `TRS-P0-005`.
- Tests: evaluator determinism, injected-payload resistance, hold
  lifecycle.

### approval

- Purpose: own the human gate in-scenario.
- Primary archetype: workflow / state machine.
- Owns: approval requests and their states; resolution by an operator;
  request expiry; relay attempts and their outcomes.
- Does not own: notification transport. `notification-hub` is called
  through a seam and its failure is recorded without changing the approval
  outcome.
- Why the relay is optional: an approval surface that stops working when
  another scenario is down is not a gate, it is a dependency. The console
  queue is always sufficient.
- API: `api/internal/approval/`.
- CLI: `treasury approvals list|approve|decline`.
- UI: the pending-approval queue — the console's landing surface.
- Storage: `approval_requests`, `relay_attempts`; see [`DATA.md`](DATA.md).
- Requirements: `TRS-P0-006`.
- Tests: queue lifecycle without a relay, relay failure isolation,
  console approve and decline.

### instrument

- Purpose: hold payment instruments and hand out only mandate-scoped ones.
- Primary archetype: CRUD / entity, with a gateway trait.
- Owns: instrument records; the mapping from a mandate to the scope an
  issued instrument carries; references to credentials held elsewhere.
- Does not own: credential material. Card numbers and keys live in
  `secrets-manager` and are resolved by reference at use time. They never
  enter this scenario's storage or any API response body.
- Scoping rule: scope is derived from the mandate, never from caller input.
- API: `api/internal/instrument/`.
- CLI: `treasury instruments list|show`.
- UI: instrument list with scope and status; never displays credentials.
- Storage: `instruments`; see [`DATA.md`](DATA.md).
- Requirements: `TRS-P1-003`, supports `TRS-P0-010`.
- Tests: scope derivation, credential non-leakage.

### rail

- Purpose: define one execution contract and host every adapter behind it.
- Primary archetype: gateway / adapter.
- Owns: the rail interface; the adapter registry; per-adapter
  configuration. Adapters: `manual`, `x402`, `card`, and later `checkout`.
- Does not own: idempotency or retry (that is `settlement`).
- Contract rule: no operational target names a vendor, and the interface
  carries no vendor-specific type. The manual adapter is an ordinary rail
  with the same evidence and emission path, differing only in basis.
- Sub-packages: `api/internal/rail/manual/`, `.../x402/`, `.../card/`.
- CLI: `treasury rails list|show`.
- UI: rail status panel; facilitator health for x402.
- Storage: `rails`; see [`DATA.md`](DATA.md).
- Requirements: `TRS-P0-001`, `TRS-P0-007`, `TRS-P1-001`, `TRS-P1-002`,
  `TRS-P1-003`, `TRS-P2-003`.
- Tests: contract conformance across all registered adapters, no-mandate
  refusal, manual-rail parity.

### settlement

- Purpose: execute an authorized charge exactly once.
- Primary archetype: service, with a workflow trait.
- Owns: charge records; idempotency keys and their resolution; the
  settle/decline/expire outcome; row-level locking around the mutation.
- Does not own: the decision (`authorization`) or the mechanism (`rail`).
- Idempotency rule: a key is required, not defaulted. A repeated key
  returns the first commit's outcome; a distinct key under the same mandate
  is an independent charge. This is the pattern already proven in the
  ecosystem's credit-wallet surface, carried over without the shared
  database that surface needed.
- API: `api/internal/settlement/`.
- CLI: `treasury charges list|show`.
- UI: charge history with outcome and rail.
- Storage: `charges`, `idempotency_keys`; see [`DATA.md`](DATA.md).
- Requirements: `TRS-P0-011`, `TRS-P0-012`.
- Tests: repeated key, distinct key, absent key, concurrent submission.

### evidence

- Purpose: retain one replayable record per spend attempt.
- Primary archetype: reporting, with a query trait.
- Owns: append-only evidence records joining mandate, approval, request,
  rail response and receipt; replay reconstruction.
- Does not own: any mutation. The store refuses update and delete by
  construction, not by convention.
- Coverage rule: declines and expiries are recorded with the same fidelity
  as approvals. A decline is usually the system working, so it is the more
  interesting record.
- API: `api/internal/evidence/`.
- CLI: `treasury evidence show|replay`.
- UI: attempt detail showing which constraint refused.
- Storage: `evidence_records`; see [`DATA.md`](DATA.md).
- Requirements: `TRS-P0-009`.
- Tests: terminal-outcome coverage, immutability, replay across all four
  outcome kinds.

### ledger

- Purpose: tell `money-ledger` what moved, without it knowing this
  scenario exists.
- Primary archetype: gateway.
- Owns: the money-event emitter; emission idempotency; the emission log.
- Does not own: financial position, runway, categorisation, or any
  interpretation of the numbers. Duplicating those would create two
  authorities over the same figures.
- Direction rule: strictly one-way and fire-and-record. `money-ledger`
  admits this scenario as one anonymous adapter through the contract every
  source satisfies.
- API: `api/internal/ledger/`.
- CLI: `treasury ledger emissions list`.
- UI: emission status on a settled charge.
- Storage: `ledger_emissions`; see [`DATA.md`](DATA.md).
- Requirements: `TRS-P0-008`.
- Tests: contract conformance, basis selection, replay safety.

### identity

- Purpose: verify the calling agent and refuse what cannot be attributed.
- Primary archetype: gateway, with a policy trait.
- Owns: the verification seam onto `agent-manager`; the fail-closed rule;
  translation of verified claims into the authority this scenario reads.
- Does not own: agent workload identity, the delegation chain, scope
  attenuation, or persona resolution. `agent-manager` already owns the
  first three and this scenario consumes them; `persona` owns the fourth.
- Fail-closed rule: transport failure, expired token, invalid signature and
  absent token all produce refusal. None produces a degraded-grade
  acceptance. The accepted cost is that `agent-manager` becomes a hard
  runtime dependency on the spend path.
- Caching rule: nothing is cached across requests. A cache would convert a
  revocation into a delayed refusal, which is the wrong failure for money.
- API: `api/internal/identity/`.
- CLI: none; it is an internal seam.
- UI: verification status shown on an authorization record.
- Storage: none.
- Requirements: `TRS-P0-005`.
- Tests: each failure mode refuses; unreachable authority refuses and
  records the cause.

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

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |
| Mandate | A signed, scoped, expiring grant naming authorizer, cap, counterparty scope and expiry. | `mandate` domain. |
| Book | The custody container that separates personal from business and holds the beneficiary identity. | `book` domain. |
| Headroom | Remaining spend under a budget, computed from authorization records and never stored. | `budget` domain. |
| Basis | How much an emitted money event can be trusted — `authoritative` or `operator-asserted`. | Vocabulary owned by `money-ledger`; used by `ledger`. |
| Fail closed | Refusing an action when its precondition cannot be verified, rather than proceeding at a weaker grade. | `identity` domain. |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `reconciliation` | `TRS-P2-001` only. Matching against a rail statement needs at least one automated rail settling real charges before the matcher has anything to match. | The first automated rail has been live long enough to produce a statement. |
| `pricing` | `TRS-P2-004`. Inbound prices are declared locally until `offer-desk` integration is wanted; a domain for one config value would be premature. | Inbound x402 metering is live and a second surface needs the same price. |
| `signals` | `TRS-P2-005`. Anomaly detection over a budget's own history needs a history. | A budget has enough settled charges for a baseline to mean anything. |
| `interop` | `TRS-P2-002`. AP2 mandate translation belongs beside `mandate` until a real counterparty needs it. | A counterparty Vrooli wants to transact with speaks AP2. |
| Non-money actuation policy | The policy engine here and `device-control`'s lease model are the same shape. Two instances is not a pattern, and generalising now would be speculative. | A third bounded-actuation surface appears, or `device-control` independently needs what this engine grew. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

Specific to this scenario:

- **Currency and money arithmetic** — a shared value type, not a domain. It
  has no lifecycle and owns no decision.
- **Persona and legal identity** — belongs to the `persona` scenario. A
  passport is not treasury's concern and identity documents must never
  enter this scenario's storage.
- **Financial position, runway, categorisation** — permanently
  `money-ledger`'s. If this scenario starts answering "how much have I
  spent this quarter", two authorities now disagree about the same numbers.
- **What to buy** — no scenario here decides that. Treasury answers only
  whether a proposed charge sits inside a grant a human made.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — durable choices and their reasons
