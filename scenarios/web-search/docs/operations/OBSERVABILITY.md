# Observability — Web Search

This document records logs, metrics, telemetry, health checks, and
business/product signals for the scenario.

> **Current status (2026-09-06):** Lifecycle health, capture delivery,
> research outcomes, evidence identity, and validation signals are
> implemented. Aggregate quality, source, cost, and organic operator cohorts
> remain explicitly limited or unknown where noted below.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the scenario is healthy?
- What signals tell us users are getting value?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain before deployment or monetization?

## Signals

| Signal | Type | Source | Purpose | Threshold |
|---|---|---|---|---|
| `/health` status | health | API | API + dependency reachability (SearXNG / Qdrant / Ollama / search-hub / SQLite) | healthy for local development |
| UI health endpoint | health | UI server | UI bundle/server reachability | responds during lifecycle health check |
| test-genie result | validation | `make test` | scenario correctness evidence | all required phases pass |
| **SearXNG reachability** | health (intended) | livesearch | Confirms the live-web path can serve; first thing to check on any live-search incident | reachable; alert if down (live degrades, learnings unaffected) |
| **Live-web cache hit-rate** | product/value | livesearch cache | Fixed-window owner events show cache reuse and avoided upstream calls | trend upward over time |
| **Budget governor remaining / exhaustion** | health + cost | livesearch governor | Fixed-window refusal events show bounded external demand; refusal returns without an external call | track exhaustion frequency; frequent exhaustion may mean mistuned window |
| **Findings index size + reconcile lag** | health (intended) | findings (Qdrant vs SQLite) | Index completeness and drift between the durable store and its semantic index | Qdrant rows ≈ active+disputed SQLite findings; low reconcile lag |
| **L3 run outcomes** | product/value (intended) | research | Iterative research-run success vs failure, briefs/findings emitted per run | runs complete and emit cited briefs (P1) |
| **Dispute-queue depth** | product/health (intended) | research/findings | Number of unresolved flagged contradictions; growing depth = curation gap | bounded; trend flat or down |

## Logs

| Log | Source | How To Read | Notes |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Request logging uses deterministic clock seam in tests. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |
| Live-search / governor logs | livesearch | `make logs` | SearXNG call outcomes, cache hit/miss, governor rate-limit events. |
| Findings mutation / audit (intended) | findings | `make logs` + audit log | Every supersede/flag/prune is audited (what/why/which brief). |
| Federation registration logs (intended) | federation | `make logs` | Provider self-registration with search-hub at boot; retry/re-register events. |

## Metrics

| Metric | Status | Notes |
|---|---|---|
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |
| Live-web cache hit-rate | implemented | Fixed-window owner events report hits/(hits+misses), or unknown for empty/truncated history. |
| Budget-governor exhaustion rate | implemented | Fixed-window owner events report refusal count, or unknown when no live-search observation exists. |
| Findings store size / status breakdown | deferred (intended) | active / disputed / superseded counts; growth over time. |
| Dispute-queue depth | deferred (intended) | Curation-health signal (OT-P1-005/007). |
| Product activation | deferred | Define after consuming scenarios/agents are routing to web-search. |
| Performance budgets | deferred | Define in `../internal/PERFORMANCE.md`. |
| Cost telemetry | deferred | Mostly local-runtime cost; external-engine rate-limit risk is the real driver. |

## Alerts / Health

The scaffold has lifecycle health checks for API and UI. The intended
operational alerts (not yet wired):

- **SearXNG unreachable** — `web-search.live` is degraded; learnings
  still serve. First check on any live-search incident.
- **Budget governor frequently exhausted** — may indicate a mistuned
  window or unexpected live-web load.
- **Reconcile lag / Qdrant unreachable** — findings recall falls back to
  text matching; reindex deferred until Qdrant returns.
- **Dispute-queue depth rising** — unresolved contradictions accumulating.

Add deployment-specific alerts only when the deployment target and
operator expectations are known.

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| Findings/index, dispute, and L3 aggregate measures | Those owner populations are still not fully exposed through fixed-window measures | Add when their cross-owner aggregation contracts are finalized. |
| Usage/effectiveness telemetry (P2) | Cannot prove which findings are actually surfaced/used, so telemetry-driven curation (OT-P2-001) is blocked | P2; once the store has meaningful volume. |
| Product usage telemetry | Cannot validate adoption or the deferred monetization hypothesis | Add before any external launch or monetization review. |
| Cost telemetry | Cannot evaluate hosted/SaaS unit economics | Add before managed deployment. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency degraded-behavior matrix
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
