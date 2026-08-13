# Decisions — Money Ledger

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
| 2026-08-13 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-08-13 | The scenario is a **committed internal capability and a candidate direct product**, and those two roles are tracked separately. | The `financial-tracker` member needs this regardless of any commercial outcome; the standalone product thesis is real but unvalidated. | The internal role ships without waiting on market evidence. A failed product hypothesis does not invalidate the scenario. Detail in `../business/MONETIZATION.md`. | The product revisit trigger fires, or the internal role is superseded. |
| 2026-08-13 | **No price is stated anywhere** until comparable products are captured. | A price written into a document becomes canon by accident and then anchors every later discussion. | `market-validator` owns capturing comps through a `validation-inbox/*` entry. Pricing sections say what is rejected (usage-based pricing on money volume) rather than what is chosen. | Comps are captured and a pricing decision is taken at the vision walk. |
| 2026-08-13 | **Integrity outranks confidentiality** in the security posture. | The dataset is non-regenerable and its primary failure mode is a silently wrong figure rather than a disclosed one. | Append-only journal, reversal-only correction, and basis-on-every-figure are security controls, not just product features. At-rest encryption is deferred; audit-trail integrity is not. See `SECURITY.md`. | A deployment where the host is not solely the operator's own machine. |
| 2026-08-13 | Positioning leads with **mixed provenance**, not with privacy or local-first. | A competitive scan found SenticMoney selling the local-first, no-bank-login, runway-for-irregular-income posture at ~$39/yr, with Actual Budget and Firefly III giving the same posture away free. Privacy is the price of entry to this segment, not a differentiator. | "We never ask for your bank password" is demoted to a supporting message. The lead claim becomes that we admit automatic and hand-entered sources through one contract and always say which is which — the one thing a fully-automatic or fully-manual competitor has no reason to model. Detail in `../business/GO-TO-MARKET.md`. | The mixed-source claim fails demand testing too — at which point drop the direct-product hypothesis rather than reposition a third time. |
| 2026-08-13 | **Not building envelope budgeting, investment performance analysis, or automated bookkeeping as a headline.** | Each surfaced in the competitive scan as a common feature. Each is owned decisively by an incumbent (YNAB/Actual, Beancount, every aggregator) and each pulls toward machinery the PRD non-goals already exclude. | Keeps the product a *record* rather than a *planning tool*, and keeps comparison off integration count — the axis we cannot win. Rationale table in `../business/GO-TO-MARKET.md`. | A user need appears that none of the existing model can express, rather than a competitor feature we lack. |
| 2026-08-13 | The four scan expansions become **operational targets `OT-P2-006`…`OT-P2-009`**, amended into the PRD's P2 list. | Operator decision: feature findings should be tracked as real operational targets rather than as prose candidates, so they inherit requirement linkage, the traceability matrix, and checkbox sync. | PRD P2 list extended by four; matching requirements `RCR-001`, `CAT-001`, `ATT-001`, `CUR-001` added to `requirements/04-expansion/`. Coverage stays at 22/22 and `business-health validate scenario` stays PASSED. Market reasoning stays in `../business/GO-TO-MARKET.md`; the commitment now lives in the PRD. | A target is built, dropped, or re-prioritised out of P2. |
| 2026-08-13 | Amend the PRD's P2 list in place rather than **regenerating** it. | The canonical template says to regenerate when business intent changes. Regeneration via `business-health wizard` rebuilds the document from interview answers and would discard the Provenance, generalisation-decision, and sibling-scenario reasoning in the appendix — the most load-bearing content in the file. | Amendment is confined to appending to P2, which is the future/expansion list; no existing target was altered and no narrative section was rewritten. The amendment is recorded in the PRD appendix so it is visible to a later reader. Validation confirms the contract still conforms. | An amendment would need to touch P0/P1 or any narrative section — at which point regenerate instead, and port the appendix by hand. |
| 2026-08-13 | The experience contract is authored to **L4 before the UI exists**, and its `bindings` are a forward contract. | Claim selectors constrain component structure, so authoring them after the UI means retrofitting the spec to whatever was built. | `testid` values in `experience/pages/*.json` are binding on implementation. Each state's `setup.query.fixture` names a fixture the UI must serve or the claim never runs. | A claim type proves unimplementable against the real component tree. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
