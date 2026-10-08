# Domains — Offer Desk

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The shipped implementation has one persisted product domain, `catalog`, and
one operator-facing composition domain, `operator-ui`. The API transport and
shared UI primitives remain non-domains. The board and gate capabilities are
implemented as slices of the catalog service rather than as undeclared runtime
packages.

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
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/offer-desk/v1/shared/health.proto` |
| catalog | Own the offer graph — typed nodes, edges, release ladder projections, and the lifecycle the API enforces on them. | One place every sellable thing lives with a state that cannot be set illegally, while release urgency and readiness remain derived views. | Nodes, edges, status history, audit entries. | crud | policy, reporting | Offer, Variant, Channel, RevenueLine, Deliverable, Ramp, Stream, Audience, Edge | `api/internal/catalog/` |
| offers | Provide the typed transport and composition boundary for catalog, gate, board, space, and Money Ledger reads. | Keep cross-domain orchestration at the service edge while persisted offer truth remains in `catalog`. | No independent product data; composes catalog-owned state and external ledger reads. | provider | integration, reporting | CatalogService, GatesService, BoardService, SpaceService | `api/handlers/offers/` |
| operator-ui | Compose catalog, gate, board, proposal, and space projections into a responsive operator console. | Make lifecycle state, evidence, trigger freshness, proposal authority, and source degradation legible at the point of action. | Browser preferences only; no catalog or financial facts. | interface | query, review | Dashboard, Offer, Trigger, Proposal, Space | `ui/src/pages/`, `ui/src/layout/`, `ui/src/consts/` |

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
| `compliance` | The source catalog records legal and platform obligations as prose per revenue line — disclosure requirements, marketplace terms, tax treatment. Real, but it needs an offer graph to hang from. | OT-P2-002, once the catalog is green and an offer approaches launch. |
| `books` (multi-tenant offer graphs) | Scoping every query by book is cheap to add later and speculative now, with one operator. | OT-P2-003, when a second operator's graph must coexist. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.
- The Money Ledger read client — a typed caller, not a bounded context.
- The migration importer — a one-time tool owned by `catalog`.

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

### catalog

- **Owns**: the offer graph. Node kinds are `offer`, `variant`, `channel`, `revenue-line` and `deliverable`; edges are typed and directional (a channel *feeds* a line, an offer is *sold at* a variant, a deliverable *belongs to* an offer, an offer *requires* a parent offer).
- **Release typing**: deliverables carry `MARKETED` or `ENABLING` class plus an orthogonal `finish_bar`. Marketed nodes alone carry operator release ranks; enabling nodes are kept off the schedule and derive urgency from their acyclic `enables` closure and earliest scheduled marketed opener. Streams use the generated monetization meter inventory as their name vocabulary. The ladder reports unscheduled marketed work and excludes retired nodes by default.
- **Mapping from the source canon** (see `../internal/DECISIONS.md`, 2026-08-13): a **delivery tier is a `variant`** — an offer *sold at* a variant is "this bundle, delivered this way", which preserves the orthogonality the source document asserts and makes pricing a property of the edge. An **add-on is an `offer` with a `requires` edge** to its parent bundle, so many-to-many membership survives and the "no add-on before the parent has paying users" guardrail becomes a transition precondition. The **funnel is deliberately not modelled** — its stages are measurements, not records with a lifecycle.
- **Key rule**: one status vocabulary across every node kind. The replaced documents used four separate vocabularies across four file families, which is precisely why no cross-kind view was possible.
- **Key rule**: the lifecycle is a state machine owned by the service. An illegal transition is refused with an error naming the rule and the transitions that *would* be legal. Prose that describes a lifecycle can refuse nothing — that is the defect being replaced.
- **Audit**: every transition writes an append-only entry with actor, timestamp, prior status and reason. Corrections are new entries.
- **Targets**: OT-P0-001, OT-P0-002, OT-P0-007. **Requirements**: GRAPH-001…GRAPH-004.

### Catalog capability slice: gates

- **Owns**: trigger declarations, the fact registry they evaluate against, evaluation runs, and promotion proposals.
- **Key rule**: a node cannot enter or remain in `candidate` without a parseable trigger. The source canon states this with the word "must" and nothing able to enforce it.
- **Key rule**: an unknown fact evaluates to **unknown, not false**. A candidate whose trigger references a missing fact stays put and the run reports the gap. Treating unknown as false would keep candidates asleep forever, which is the exact failure this scenario exists to end.
- **Key rule**: promotion to `active` requires an operator role. An agent may create a proposal and nothing more. The source canon says "agents never self-promote"; this makes it a permission rather than an instruction.
- **Deliberate limit**: the trigger language admits declared facts, comparison operators and boolean composition — nothing more. A richer language becomes an unmaintainable rules engine, and the P2 target for scenario-sourced facts is the intended growth path.
- **Market benchmarks are facts** (`../internal/DECISIONS.md`, 2026-08-13). A competitor comp is a fact with an observation date and a dimension-derived staleness window — pricing 90d, retention and activation 180d, channel-cac 120d, everything else 365d. Past its window it becomes **unknown rather than false**, which is the behaviour this domain already guarantees and the reason benchmarks belong here rather than in a registry of their own. A trigger may therefore reference a comp directly, and a trigger resting on a stale comp reports the gap instead of firing on an old number.
- **Targets**: OT-P0-003, OT-P0-004, OT-P0-005, OT-P1-005, OT-P2-004. **Requirements**: GATE-001…GATE-008 (GATE-007 lives in the console module, with the P2 target it serves).

### Catalog capability slice: board

- **Owns**: nothing. Every entry is computed at read time from catalog, gates, and two Money Ledger reads.
- **Key rule**: each entry names its source, and a source that cannot be read becomes a visible availability entry while healthy sources continue to rank. An unavailable ledger is never rendered as zero earnings — that would be indistinguishable from an offer that genuinely earned nothing.
- **The instrument role**: this domain is the `monetization` team's single address (`../internal/DECISIONS.md`, 2026-08-13). That is what makes the second Money Ledger read necessary: a team with one address needs its financial member served by the same surface as its catalog members, so the board carries **runway, goal verdicts with sustain-window progress, and the default-alive gap** alongside the per-offer actuals join.
- **Surfaced, never owned**: posture rows carry no stored amount and no computation of one. The board asks Money Ledger for a figure and renders it with its source and age. The PRD non-goal excluding accounting and balances is intact — reporting someone else's number under attribution is not owning it.
- **Fallback is legal**: per the §5 degradation contract in `path:docs/agent-system/TARGET_MODEL.md`, the board makes the good path cheap and never makes the manual path illegal. A consumer that cannot reach the board reads `money-ledger` directly and says so.
- **Targets**: OT-P1-002, OT-P1-003. **Requirements**: INT-002, INT-003, INT-005.

### operator-ui

- **Owns**: route composition, responsive presentation, operator-only
  affordances, evidence labels, and explicit empty/degraded states.
- **Does not own**: offer lifecycle truth, trigger evaluation, proposals, or
  money-ledger figures.
- **Surfaces**: `ui/src/pages/` and the shared shell in `ui/src/layout/`.

## Build Order

The catalog capability slices share one persisted store and one service
composition boundary. The operator UI reads the public API and never owns
catalog, trigger, proposal, or financial state.

```
catalog (graph + gates + board)  →  operator-ui
```

Build the catalog as one vertical slice across proto, API, CLI, and
persistence; then compose it in the operator UI. The migration importer goes
through the same catalog state machine.

## Deliberately Excluded

| Excluded | Why |
|---|---|
| Pricing, billing, subscriptions, entitlements | The commerce scenario upstream owns all of it. Reimplementing any part here would be the worst available outcome. |
| Money, balances, revenue figures | Money Ledger owns them. This scenario reads actuals and financial posture for the board and never stores an amount, never computes one, and never corrects one. Rendering another scenario's figure under attribution is not ownership; the moment a figure is cached or adjusted here, it is. |
| Marketing execution — campaigns, drafts, publishing | A different clock and a different failure mode. Adjacency is not a boundary. |
| Strategy — whether to sell something, and at what price | The scenario records a decision and never makes one. Promotion is an operator action by construction. |
| PRD and requirements conformance | An existing scenario already owns that for every scenario in the fleet. |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
