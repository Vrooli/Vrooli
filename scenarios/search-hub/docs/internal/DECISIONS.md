# Decisions — Search Hub

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation notes belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-06-03 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-08-11 | Record the ten durable retrieval and readiness decisions as one cross-domain contract. | The trustworthy-retrieval plan spans registration, routing, evals, scheduling, governance, and coverage. | Future changes must preserve: (D1) thin descriptor-driven routing; (D2) bounded scoped widening; (D3) provider-owned index age; (D4) exact leaf matching; (D5) direct and federated eval tiers over one corpus; (D6) no corpus content in Search Hub; (D7) generic breakers and demotion; (D8) registry-driven scheduling; (D9) Answer NOW requires active, reachable, fresh-eval signals; (D10) explicit-confirmation governance with provider files as SSOT. | Revisit only when a replacement contract is documented with migration and validation evidence. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history

## Honest-signals decisions

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-08-12 | Keep transport failure separate from graded emptiness. | A failed provider call must not become evidence that a corpus is empty. | Circuits expose transport outages; zero-yield demotion only consumes successful graded-empty responses. | Revisit reporting only if operators cannot explain either state. |
| 2026-08-12 | Treat fresh junk-leaking eval evidence as an automatic-routing hold. | Aggregate relevance alone can hide a provider that promotes gibberish. | The run id and gate reason are visible; explicit selection remains available for repair. | Revisit when the eval model adds a stronger calibrated junk discriminator. |
| 2026-08-12 | Use a short address cache with failure invalidation and preserve partial fan-out results. | Process-spawn cost and deadline cancellation made large federations look empty. | Re-ported endpoints converge within the TTL or first failure; completed provider groups are not discarded. | Revisit after measured fleet latency and active-provider count materially change. |
| 2026-08-12 | Run demoted-provider recovery as an unattended, provider-scoped probe. | Waiting for the next user query made recovery depend on traffic, while treating the probe as explicit traffic would erase honest zero-yield evidence. | A one-minute default loop checks expired demotions; graded hits restore automatic eligibility, and empty/unavailable probes restart decay without pretending transport failure is corpus evidence. | Revisit when provider-owned validation supplies a cheaper or stronger recovery signal. |

## End-to-end retrieval trust plan decisions

| Date | Decision | Tradeoff | Revisit Trigger |
|---|---|---|---|
| 2026-08-13 | Keep one federated eval tier and split its verdict into routing precision and retrieval recall. | Existing pass-rate history remains comparable, while attribution adds the ownership signal needed to diagnose misses. | Add another tier only if the two rates cannot distinguish shortlist loss from provider retrieval loss. |
| 2026-08-13 | Compose the router suite from registered provider suites. | The suite is self-maintaining and agnostic, but inherits weak or stale provider labels. | Add authored router cases only if composed evidence misses a demonstrated routing failure. |
| 2026-08-13 | Recovery fails open; adoption fails closed. | Degraded recovery probes release a stuck slot, while providers without evidence remain explicit-only. | Revisit when provider-owned health evidence can replace either state machine. |
| 2026-08-13 | Never remove an unproven provider; expose it as experimental/incubating. | The registry remains a durable adoption ledger and parked providers become visible operational work. | Revisit only with an explicit registry-retirement policy and migration. |
| 2026-08-13 | Validate corpora with descriptor-shape predicates, never provider identity. | Generic rules catch self-echoing and collapsed families without scenario-specific branches. | Add an opt-out only when a provider documents why identifier-shaped queries are its real contract. |
| 2026-08-13 | Bound reranking from the measured serving leg and prefer fast grouped degradation to an unbounded wait. | Slow fallback answers are truncated more often, but query latency and failure attribution remain honest. | Revisit when the active reranker leg or its measured tail changes materially. |
| 2026-08-13 | Sequence restoration, attribution, corpus/evidence governance, then scale and board projections. | Later metrics consume evidence established by earlier phases, so intermediate results remain interpretable. | Reorder only if a new dependency makes a phase validation impossible. |
| 2026-08-14 | Treat a generated empty BAS registry as non-mutating for file-isolation analysis. | Search Hub has no workflow that can write application files; demanding unused file seams would block validation without proving an additional safety property. Missing or malformed registries still fail closed. | Revisit when Search Hub adds a mutating file-backed workflow. |
| 2026-08-14 | Keep scenario-owned UI provider composition in the scenario test adapter. | `api-base` supplies generic Query/i18n plumbing, while Search Hub must provide its initialized catalog and ThemeProvider so component tests match production. | Revisit if api-base gains an explicit scenario-provider registration contract used by all UI scenarios. |
