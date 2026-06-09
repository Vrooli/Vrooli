# Performance — Web Search

This document records performance budgets, current measurements, known
constraints, and regression procedures.

> **Scaffold status (2026-06-09):** The budgets below are *intended*
> targets derived from `PRD.md` / requirements. None are implemented or
> measured yet — they are the design targets the implementation should
> be validated against, not observations.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## What Performance Matters Here

web-search has two latency-critical paths with very different cost
profiles:

- The **live** face (L0/L1) reaches the rate-limited external web via
  SearXNG — every external call is expensive (rate-limit budget) and
  slow (network + engine aggregation), so the design goal is to reach it
  *as little as possible*.
- The **learnings** face serves the local findings corpus from Qdrant
  via aisearch-go — fast, local, and the path a default federated query
  should take.

The headline performance objective is therefore not just raw speed but
**call avoidance**: a default federated query should issue **zero
external HTTP calls** and answer from the learnings corpus + cache.

## Budgets

These are intended targets (not yet measured). Concrete numeric SLOs are
set when each level lands and a real corpus exists.

| Surface | Intended Budget | Measurement | Status |
|---|---|---|---|
| Default federated query (learnings) | **Zero external HTTP calls** — served entirely from the findings index + cache. | request-trace: external-call count == 0 on non-gated queries | intended (OT-P0-004) |
| Findings semantic recall | Low p95 (local Qdrant + reranker; target on the order of single-digit-hundreds of ms p95) — fast enough to join default routing without slowing the blend. | aisearch-go query timing; per-query p95 | intended (OT-P0-005) |
| L0 live web search | Bounded live-search latency (dominated by SearXNG aggregation + network); cache hit short-circuits it. | livesearch request timing; cache-hit vs miss split | intended (OT-P0-001) |
| Live-web cache hit-rate | High hit-rate target on repeated/near-repeated queries — the cache is the first line of external-call avoidance. | cache hit/miss counter exposed in the ops panel | intended (OT-P0-007) |
| External QPS (budget governor) | Capped per time window by a token-bucket; on empty bucket → graceful "rate-limited, try later" with **no** external call. | governor token-bucket state; rate-limited-response count | intended (OT-P0-007) |
| L1 snippet synthesis | **Additive, never blocking** — raw hits return regardless; synthesis is an optional overlay that abstains on conflict/thin sources. | synthesis latency measured independently of raw-hit return | intended (OT-P0-002) |
| L2/L3 research runs | Long-running and asynchronous (browserless fetch + multi-pass synthesis / agentic loop); not on any interactive query path. Budget-ordered: answer first, curate as a bounded post-step. | run-duration + per-phase timing | intended (P1) |
| UI build | 5–10 minutes accepted for the current Vite module graph | lifecycle / test-genie build logs | inherited |
| API / UI health | responsive under lifecycle health timeout | `/health` check | active (scaffold) |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured yet. | n/a | n/a | 2026-06-09 |

Measurements land as each level (L0 → L3) is implemented and exercised
against a real corpus and a live SearXNG resource.

## Known Constraints

- **External calls are the scarce resource.** SearXNG / upstream engine
  rate limits cap how often the live path can run; the cache + budget
  governor exist specifically to ration them. Performance work that
  reduces external calls (better cache, better learnings recall) is
  worth more than shaving live-call latency.
- **L1 synthesis must stay off the critical path.** It is always
  additive — it can be slow or abstain without ever delaying raw hits.
- **Semantic recall depends on the embeddings/reranker chain.** Cold
  Ollama or a missing reranker degrades both latency and ranking; the
  fallback is raw dense order / text match (see
  [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md)).
- Vite production builds may process thousands of modules and take
  several minutes (inherited template constraint).

## Regression Procedure

1. Run `make test`.
2. Capture the external-call count for a default federated query — it
   must remain **zero** (this is the load-bearing invariant, not just a
   latency number).
3. Capture live-search latency (cache hit vs miss) and findings recall
   p95 against the test corpus.
4. For UI interaction regressions, use `ui/perf/README.md` and the
   provided capture template.
5. Record persistent findings in this document or
   [`PROBLEMS.md`](PROBLEMS.md) depending on whether they are accepted
   constraints or unresolved debt.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry (cache hit-rate, budget remaining)
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md) — budget-governor and research-run flows
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
