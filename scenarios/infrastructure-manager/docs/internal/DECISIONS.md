# Decisions — Infrastructure Manager

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
| 2026-08-17 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-08-17 | **Build the instrument as a new scenario rather than extending an existing one.** | `INSTRUMENTATION_ROADMAP.md` core principle 1 is "extend before creating; new scenarios are a last resort", so this needed a real justification. No existing scenario can hold the board: `system-monitor` and `app-monitor` are *inside the plant* with their own uptime targets, so a board hosted there goes down with what it measures; `vrooli-autoheal` owns the check registry, so hosting there is deviation `D6`. | A new thin scenario is the only seat that is neither in the plant nor in the setpoint. | If a future scenario emerges that is neither supervised nor a setpoint owner and could host the board more cheaply. |
| 2026-08-17 | **Hold no roster; derive every set at read time.** | `INSTRUMENTATION_ROADMAP.md` Gap 11 states "No central capability-health scenario — that would recreate the roster this PoR forbids", and operating-model rule 6 forbids enumeration anywhere in the team's surfaces. | `targets` and `supervision` own no tables. The supervised set is computed per read from the core-set closure ∪ load-bearing declarations. A `supervision` cache would be an architectural defect, not an optimization. | Never, while rule 6 stands. If the derivation becomes too slow, the fix is a faster derivation, not a cache. |
| 2026-08-17 | **Store readings; never store a band verdict. Trust verdicts are the one exception and are stored.** | Diverges from `meta-optimization-manager`, which never stores a numerator so a stale board is structurally impossible. But reliability targets are inherently historical — "99.5% over 30d" cannot be computed from a live probe — and Gap 11 names the cost of not storing: an outage becomes indistinguishable from missing data after the fact. The band/trust split is the line that makes the rule enforceable: a band verdict describes the *target* and is recomputable against the current deadband, while a trust verdict describes the *observation* and is not — nothing can reconstruct after the fact whether a check was saturated at the moment of the read. | Band evaluation is recomputed at query time, so tightening a target re-grades its own history. Trust verdicts persist on the reading row. The invariant's intent survives: no conclusion is ever served that was not just computed. | If reading history is ever consumed as a conclusion rather than re-evaluated, the deviation has failed and storage must be reconsidered. If an integrity rule ever becomes retroactively computable, its verdict moves out of storage. |
| 2026-08-17 | **Trust is a first-class axis with its own model document.** | This instrument's inputs are alarms on live processes, not registry joins. The 2026-07-23 decomposition measured 4,624 critical events in 24h with ~92% from ghost and saturated checks — a board reporting that at face value would have called a healthy platform critically unhealthy. | `TRUST-MODEL.md` exists as a peer to `SETPOINT-MODEL.md`; every reading carries a closed-vocabulary verdict; untrusted readings are excluded from aggregates and routed to the instrument rather than the plant. | If the fleet grows a shared sensor-integrity contract, collapse this into a pointer to it. |
| 2026-08-17 | **The setpoint stays in the team's plan of record, not in this scenario.** | Diverges from the four space docs, which live with capability owners. Here the deadbands *are* operator judgment, changed only through an approved `reliability-target-update` under documented hysteresis rules. | `docs/infra-health/strategy/RELIABILITY_TARGETS.md` remains the setpoint and is read at query time, never cached. The scenario cannot lower its own bar. | If reliability targets ever become owner-declared per plant element, the read moves to those owners. |
| 2026-08-17 | **No actuation path, ever — the watchdog is a separate decision.** | The genuinely critical class (autoheal down, agent-manager unable to spawn, shared-package breaks blocking core starts) needs a seconds-clock response, but operating-model rule 3 is "supervise, don't operate" and an instrument that acts is a controller with a bad boundary. | That class is deferred to a watchdog tier with its own operator authorization, an enumerated action set, and claim-suppression with mandatory expiry. This scenario would only *supervise* it (`OT-P2-003`). | Operator authorizes a watchdog tier. Even then, this scenario measures it and never becomes it. |
| 2026-08-17 | **Documentation-first: author the charter, requirements and concept models before any domain code.** | Mirrors how `meta-optimization-manager` was built; its `ARCHITECTURE.md` records the same initial state. Operator directed this sequence explicitly over authoring a plan-manager plan first. | PRD validates at L3 with zero findings; four domains, five axes and two model documents are authored with no product code. The plan and implementation follow and can be refined against a real contract. | None — this is a completed sequencing choice, recorded so it is not mistaken for an incomplete scaffold. |
| 2026-08-17 | **Generated from a template reporting `quarantined` status.** | Both scenario templates report `quarantined` in `template-manager registry list`, while Template Manager's own progress log records react-vite 1.6.5 passing deep validation on 2026-07-12 with the registry active. Generation succeeded; the scaffold validates. | Accepted as an upstream status discrepancy rather than a property of this scenario, and recorded here so it is not rediscovered. | Resolve with Template Manager before the first vertical slice. If the quarantine is real rather than stale, re-evaluate the template choice. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
