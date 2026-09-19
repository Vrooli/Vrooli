# Monetization — Web Search

This document records how the scenario could create revenue or support
a monetizable Vrooli capability. Keep it honest: `not-applicable` is
better than inventing a commercial story.

> **Status (2026-06-09):** Not implemented yet. The value framing below
> is the *intended* role drawn from `PRD.md`. Direct external
> monetization is an explicit deferred hypothesis, not a committed plan.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component,
  add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

## Role In Vrooli

Per Vrooli's model, scenarios serve triple duty — **product**,
**validation**, and **capability**. web-search is, first and foremost,
the third:

- **Internal capability (primary):** web-search adds the missing "web"
  dimension to Vrooli's federated search (search-hub) and turns
  expensive, throwaway web research into compounding local knowledge.
  Two registered providers — `web-search.live` (SCOPE_EXTERNAL,
  rate-safe live web) and `web-search.learnings` (SCOPE_PROJECT, the
  self-curating findings corpus) — become permanent tools every other
  scenario and agent can consume. It is also the foundation for an
  in-Vrooli deep-research capability (L2/L3).
- **Validation:** it is a reference implementation of the search-hub
  provider registration + scope-aware routing contract and the
  aisearch-go learnings-corpus pattern (mirroring cli-health /
  knowledge-observatory), proving these patterns out for future
  scenarios.
- **Direct product:** deferred hypothesis (see below).
- **SKU/bundle candidate:** deferred — could later anchor a
  "research-as-a-service" / deep-research answer-engine offering.
- **Revenue line:** none committed; deferred hypothesis.

The compounding effect is the core value: as findings accumulate, the
system answers more from what it has already internalized and reaches
the rate-limited live web progressively less. That lowers external-call
cost and latency for *every* consuming scenario over time — value that
accrues internally rather than as a line item.

## Customer / Buyer

- **Primary consumer (today):** Vrooli agents operating mid-task who
  need current or external information, the search-hub federation
  itself, and human operators using unified search. These are internal
  consumers, not paying customers.
- **Hypothetical external buyer (deferred):** teams wanting a
  privacy-respecting search layer (SearXNG-backed, local, no third-party
  query leakage) and/or a deep-research answer engine that returns
  honest, citation-forward briefs with explicit uncertainty.
- **Pain:** live web search is rate-limited and ban-prone, raw results
  lack synthesis, and one-off research is discarded instead of
  accumulating. web-search addresses all three (cache + budget governor,
  cited synthesis, persistent findings store).
- **Existing alternatives:** hosted answer engines (Perplexity-style)
  and raw search APIs — none of which keep the knowledge local,
  privacy-respecting, and compounding inside the operator's own stack.

## Packaging

| Packaging Option | Status | Notes |
|---|---|---|
| Internal capability / microservice | **primary, active intent** | Consumed by search-hub federation, agents, and future deep-research. This is the committed role. |
| Standalone app | deferred | Revisit if a privacy-respecting search / research front-end shows external demand. |
| Bundle component | deferred | Could ship inside a "research" or "knowledge" SKU once deep-research (L3) is proven. |
| Add-on | deferred | Natural add-on to a unified-search or knowledge product if one is packaged. |
| Service/consulting assist | deferred | Deep-research briefs could accelerate done-for-you research delivery. |

## Pricing Hypothesis

- **Model:** deferred. Internal capability has no price; an external
  "research-as-a-service" model would most plausibly be usage- or
  seat-based, but no model is committed.
- **Comparable products:** hosted answer/deep-research engines; none
  benchmarked yet.
- **Willingness-to-pay evidence:** none captured yet.
- **Cost drivers:** local runtime by default (SearXNG, Ollama, Qdrant,
  reranker, browserless are local resources). The dominant operational
  cost/risk is external-engine rate limits — deliberately bounded by the
  TTL cache and token-bucket budget governor (OT-P0-007) rather than by
  paid API spend. A hosted offering would add hosting, auth, and
  per-tenant isolation cost.

## Validation Plan

- **Demand signal needed (internal):** consuming scenarios/agents
  actually routing to `web-search.learnings` and live-web cache hit-rate
  rising over time (findings displacing live calls). See the intended
  signals in [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md).
- **Demand signal needed (external, deferred):** explicit interest in a
  privacy-respecting search / deep-research answer engine.
- **Channel:** define in [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- **Success threshold:** internal — agents/search-hub depend on
  web-search and live-web reliance trends down as findings grow.
- **Revisit trigger:** revisit the external/direct-product hypothesis
  once L3 deep research (OT-P1-002) ships and the findings store reaches
  meaningful volume with demonstrated reuse.

## Current Status

`internal-capability` (intended) — not yet implemented. Direct external
monetization is a deferred hypothesis pending L3 deep research and
evidence of external demand. No revenue line is committed.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- [`../../../../docs/monetization/README.md`](../../../../docs/monetization/README.md) — project-level monetization strategy
