# Go To Market — Web Search

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

> **Status (2026-06-09):** Not implemented yet. The adoption path below
> is internal-first by design. External go-to-market is an explicit
> deferred hypothesis (see [`MONETIZATION.md`](MONETIZATION.md)).

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Audience And Positioning

- **Audience (today):** internal consumers — the search-hub federation,
  Vrooli agents needing current/external information mid-task, and human
  operators using unified search. The deep-research foundation (L2/L3)
  later serves any scenario that needs a cited brief.
- **Positioning:** "the web dimension of unified search that gets
  cheaper the more you use it" — live web search reachable rate-safely,
  plus a self-curating, citation-backed findings store that compounds.
- **Main claim:** a default federated query surfaces internalized
  findings (SCOPE_PROJECT) *without firing a rate-limited live web call*;
  live web (SCOPE_EXTERNAL) joins only on explicit `--type web`/`--all`
  or fallback escalation. Answers are honest about uncertainty (abstain
  / "sources disagree") rather than falsely confident.
- **Proof needed:** dual provider registration works, scope-aware
  blending keeps live web off the default path, and findings reuse rises
  over time (live-web cache hit-rate up, external reliance down).
- **External positioning:** deferred — a privacy-respecting search +
  deep-research answer engine, contingent on L3 and external demand.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| search-hub federation (internal) | Registering `web-search.live` + `web-search.learnings` makes web-search discoverable to every search consumer with no per-scenario integration. | `.vrooli/search.json` provider descriptors; idempotent self-registration at boot. | Other scenarios/agents route queries to `web-search.learnings`; live web reached only when explicitly requested. |
| Agent tooling (internal) | Agents adopt L2/L3 endpoints + `web-search` CLI as research tools. | L2 API endpoints (P1), `web-search research` CLI, agent-manager integration (P1). | L3 runs emit cited briefs; findings auto-captured and reused. |
| External (privacy-respecting search / deep research) | deferred | n/a until external demand is evidenced. | Add when the external monetization hypothesis is revisited. |

## Launch Motion

Rollout mirrors the PRD launch sequencing — internal capability first,
deep research next, broader/external last:

1. **P0 — internal capability.** Confirm the SearXNG resource is healthy
   and standards-current on this host, then ship L0 live search, L1
   cited snippet synthesis, both search-hub providers, the findings
   store + management CLI, the cache + budget governor, and the core UI.
   Goal: web-search is a registered, scope-aware provider the federation
   and agents can rely on.
2. **P1 — deep research.** Ship L2 (fetch top-N via browserless →
   extract → cited synthesis), the L3 agent-manager research-and-
   reconcile loop, finding auto-capture, contradiction handling +
   trust/freshness metadata, and the dispute review queue UI. Goal:
   web-search becomes the foundation for in-Vrooli deep research.
3. **P2 — broaden.** Ship usage-telemetry-driven curation, classifier
   auto-routing of web-shaped queries to live web, and the periodic
   full-store consistency GC. Goal: the store self-maintains at scale
   and routing gets smarter.
4. **External GTM — deferred.** Only after L3 is proven and the findings
   store shows real reuse, decide whether to package an external
   privacy-respecting search / research-as-a-service offering.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "Unified search can finally reach the live web, rate-safely." | search-hub consumers, agents | Cache + budget governor keep external calls bounded (OT-P0-007). | planned (P0) |
| "Research stops being throwaway — valuable findings persist and self-curate." | agents, operators | Findings store + reconcile loop (OT-P0-005, OT-P1-003). | planned (P0/P1) |
| "Honest answers: cited, dated, and explicit when sources disagree." | all consumers | Mandatory citations, dispute-with-warning, abstain-on-conflict. | planned (P0/P1) |
| External privacy-respecting / deep-research offering. | external buyers | n/a | deferred |

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Dual provider registration + scope routing | search-hub | Default query surfaces findings with zero live-web calls; live web reachable only on explicit request. | Gate P0 viability (OT-P0-003/004). |
| Findings reuse over time | observability signals | Live-web cache hit-rate rises / external reliance falls as findings accumulate. | Confirms the compounding-knowledge thesis. |
| L3 cited brief quality | agent tooling | L3 runs emit verifiable, cited briefs and reconcile cleanly against existing findings. | Gate the deep-research / external hypothesis. |
| External demand probe | deferred | n/a | Add only after L3 ships. |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging and pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
