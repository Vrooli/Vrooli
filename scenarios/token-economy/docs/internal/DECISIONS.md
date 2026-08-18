# Decisions — Token Economy

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-08-18 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-08-18 | **A grant is a mandate.** The grant contract is authored congruent with the `treasury` mandate shape — subject, amount, scope, expiry, provenance — from the first commit. | The scenario was scoped in the same workshop as `treasury` and `persona`. A token grant carrying spend rules *is* structurally the object that authorizes real spend. | The two scenarios can share one policy model. The eventual real-value rail becomes a new adapter rather than a rewrite. Cost: a parity test (`TKE-P0-002`) and the discipline of recording every divergence in this table. | Divergence pressure that cannot be recorded as an intentional exception, or `TKE-P2-004` extracting the shared contract. |
| 2026-08-18 | **`treasury` is a contract sibling, not a dependency.** Neither scenario calls the other and this one must function fully with `treasury` absent. | Congruence could have been achieved by depending on a shared package. | Independent deployability preserved; each scenario is separately useful and separately sellable. Cost: congruence is maintained by test and discipline rather than by the compiler. | `TKE-P2-004` — extract a shared contract only once both have shipped enough to know what is genuinely common rather than coincidentally similar. |
| 2026-08-18 | **Balance is a projection, never a stored truth.** Every quantity is derived from an append-only journal; a cache may exist but the journal wins. | A stored total is an assertion nobody can audit. `money-ledger` made the same call for financial position. | The economy is fully explainable to a holder, which is the product's central promise. Corrections are compensating events. Cost: every read path must go through projection, and rebuild equality must be tested. | Never, under the current product shape. A performance problem would be solved by better projection, not by trusting a stored total. |
| 2026-08-18 | **Authority separation is a service boundary, not a permission check.** Two Connect services: a minter service and a holder service. The holder service has no mint/grant/rule method at all. | A runtime role check is one bug away from being bypassed; an absent RPC is not. This mirrors `treasury`'s autonomy-toggle posture. | A holder with a valid token still cannot mint. The control is visible in the proto and in codegen. Cost: two service surfaces to maintain and two client sets. | Never. If a legitimate holder-initiated mint appears, it is modelled as a *request* on the holder service that the minter service fulfils. |
| 2026-08-18 | **No real value, enforced by absence.** No price field, no payout path, no external transfer anywhere in the contract. | A policy check could be relaxed quietly in a later change; a missing capability cannot. This is also what keeps multi-holder balances clear of money transmission. | Adding a monetary path requires a visible contract change and a recorded decision here. Structurally asserted by `TKE-P0-014`. | `TKE-P2-001` only, and only after a recorded custody and regulatory decision. The structural test must be *amended*, never deleted. |
| 2026-08-18 | **SQLite, terminally — not as a starting point.** No shared resource of any kind. | Household volume with single-writer mutations. A household economy that requires infrastructure is not a household economy. | Laptop-runnable with nothing else started. Row-lock plus caller-supplied idempotency key follows the proven `landing-page-business-suite` credit-wallet pattern. | Concurrent external writers — realistically only `TKE-P2-001`. Nothing in P0/P1 produces contention. |
| 2026-08-18 | **`scenario-authenticator` is a hard, fail-closed dependency.** | The multi-holder isolation boundary is the failure that would end trust in a household product. Without a verifiable identity it is cosmetic. | Authenticated surfaces refuse rather than degrading when the authenticator is unreachable. Cost: the scenario is not usable standalone in the strictest sense. | Never. A client-side role flag is not an acceptable alternative at any point. |
| 2026-08-18 | **Isolation is enforced at the repository layer, not only the handler.** | A future handler written without the check would otherwise leak one holder's history to another. | Cross-holder queries return empty regardless of handler behavior; refusals do not disclose whether the other holder exists. Cost: every repository read carries a scoping parameter. | Never. |
| 2026-08-18 | **Rule programs use a closed condition vocabulary; caller-supplied code is never executed.** | A household economy that can execute caller code is a remote-execution surface wearing a reward-app costume. | Rule effects are explainable and refusals can name the rule that refused (`TKE-P0-003`). Cost: expressiveness is bounded by the vocabulary. | Never for arbitrary code. New conditions are added to the vocabulary instead. |
| 2026-08-18 | **No inference anywhere in the product.** Rule evaluation is deterministic. | An LLM-decided refusal cannot be explained to a child, which contradicts the transparency the scenario is built on. | Zero marginal cost, which is why the free/metered/gated split resolves to free for the whole core product. | Never for rule evaluation. A non-decisional convenience (suggesting catalog wording) could be reconsidered separately. |
| 2026-08-18 | **Operator entry is a first-class earning adapter, not a fallback.** | Several real earning sources will never have an API — a chore done offline is reported by a person. | The operator path and the programmatic path traverse identical code, asserted by test. Cost: no shortcuts for the UI path. | Never. This mirrors `money-ledger`'s treatment of manual entry. |
| 2026-08-18 | **Two audiences, one token contract.** The minter console is dense and operational; the holder view is sparse and child-legible. They share colors, type scale, status semantics and motion rules, and differ only in composition. | Forking the token contract would make the product feel like two products; forcing one density would make the holder view unusable by its actual reader. | The holder view *exceeds* the accessibility floor rather than relaxing it. Recorded in `DESIGN.md`. | Adoption of a different design kit. |
| 2026-08-18 | **Import of journal history is deliberately absent.** Export exists; import does not. | An importable journal would let a caller assert history the system never observed, defeating the audit property. | Migration from another tool is manual. | Never, while the journal is the audit authority. |
| 2026-08-18 | Requirement validation refs for unwritten tests live in `notes`, not `ref`. | `vrooli scenario requirements validate` errors on a `ref` pointing at a non-existent file, so a planned test cannot be declared through `ref`. | Registry validates clean at documentation stage; intended paths are preserved and recoverable. Cost: a manual move from `notes` to `ref` as each test lands. | The validator gaining a planned-ref affordance. Tracked in `PROBLEMS.md`. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
