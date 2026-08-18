# Observability — Notification Hub

This document records logs, metrics, telemetry, health checks, and
business/product signals for the scenario.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the scenario is healthy?
- What signals tell us users are getting value?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain before deployment or monetization?

## Signals

| Signal | Type | Source | Purpose | Threshold |
|---|---|---|---|---|
| `/health` status | health | API | API and dependency reachability | healthy for local development |
| UI health endpoint | health | UI server | UI bundle/server reachability | responds during lifecycle health check |
| test-genie result | validation | `make test` | scenario correctness evidence | all required phases pass |
| Channel availability | health | `channels` domain | Whether each enabled channel can actually reach its provider | at least one channel healthy; zero healthy channels is an outage even though `/health` is green |
| Delivery success rate | product | `deliveries` table | The one number that says whether the scenario is doing its job | investigate below 95% over a day |
| Time from accept to delivered | product | `notifications` + `deliveries` | Whether notifications are timely enough to act on | p95 under 30s for non-held, non-relayed deliveries |
| Held count and age | product | `notifications` where state is `held` | Detects a quiet window misconfigured into an indefinite hold | no notification held past its staleness bound |
| Terminal failure count by reason | product | `deliveries` | Distinguishes a broken address from a broken provider | any repeated terminal reason for one device is actionable |
| Critical-rate by caller | product | `notifications` grouped by requester | Detects a caller that marks everything critical and defeats quiet hours | any caller over ~10% critical warrants a look |
| Suppression rate | product | notifications settling `suppressed` | High suppression means callers are over-sending, not that dedupe is working well | trend, not a threshold |

**`/health` deliberately does not include channel health.** A reachable
database does not mean a reachable push provider, and folding the two
together produces the exact failure this scenario was regenerated to
eliminate: every signal green while nothing is delivered. Channel
availability is its own signal, owned by the `channels` domain, and the
UI surfaces it separately.

## Logs

| Log | Source | How To Read | Details |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Request logging uses deterministic clock seam in tests. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |

## Metrics

| Metric | Status | Details |
|---|---|---|
| Product activation | deferred | Define after PRD users and workflows are real. |
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |
| Performance budgets | deferred | Define in `../internal/PERFORMANCE.md`. |

## Alerts / Health

The generated scenario has lifecycle health checks for API and UI. Add
deployment-specific alerts only when deployment target and operator
expectations are known.

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| No delivery-rate signal until the P0 core exists | The single most important signal is unavailable during the first slice. | Ships with OT-P0-004. |
| No end-to-end proof that a notification reached a human | Delivery to a provider is not delivery to a person. The provider accepting a push says nothing about whether the phone showed it. | Partially closed by OT-P2-004 (acknowledgement). Until then, the honest position is that the scenario measures dispatch, not receipt, and the release checklist compensates with a manual real-device gate. |
| Relay latency unmeasured | Cannot tell a slow node from a stuck one. | Ships with OT-P1-001. |
| Cost telemetry | Only matters if a paid channel (SMS) is enabled. | OT-P2-002. |
| Product usage telemetry | Not applicable — this scenario has no adoption funnel to instrument. | Would only apply if the capability were ever packaged for others. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
