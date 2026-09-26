# Observability — Treasury

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
| Refusal rate by constraint | product | authorization records | Shows which cap or scope entry actually binds, which is the signal that tells an operator their policy is wrong. | no threshold; it is diagnostic, not an alert |
| Settlements in `unknown` | product | settlement records | The one state that always needs a human. | **any non-zero count is actionable** |
| Ledger emission backlog | product | emission log | Money moved but the journal does not know. | non-zero for more than one retry cycle |
| Identity verification failures | security | identity seam | Distinguishes an outage from an attack; a spike with valid-looking callers is the latter. | any sustained rate |
| Pending approval age | product | approval queue | An approval nobody answers is a stalled agent. | oldest pending exceeding its budget's expiry window |
| Approval decision latency | product | approval records | Consistently sub-second resolutions suggest the gate is ceremonial rather than real. | diagnostic |

## Logs

| Log | Source | How To Read | Details |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Request logging uses deterministic clock seam in tests. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |

## Metrics

| Metric | Status | Details |
|---|---|---|
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |
| Mandates issued, and mandates actually used | planned | The activation signal named in [`../business/GO-TO-MARKET.md`](../business/GO-TO-MARKET.md). A high issue rate with a low use rate means operators are configuring but not trusting. |
| Charges by rail | planned | Shows whether the manual rail remains dominant after automated rails land, which would indicate the automation is not trusted. |
| Refusals by constraint | planned | The most useful operator-facing diagnostic in the scenario. |
| Earnings collected inbound | planned | Independent adoption signal for the metering half. |
| Performance budgets | planned | Defined in `../internal/PERFORMANCE.md`; measured after implementation. |

## Alerts / Health

The generated scenario has lifecycle health checks for API and UI.

Three conditions warrant a real alert rather than a dashboard row, because
each means money is in an unresolved state:

1. **A settlement sitting in `unknown`.** Whether money moved is not known
   and cannot be resolved by waiting.
2. **Ledger emission backlog persisting past one retry cycle.** Money moved
   and the journal does not reflect it, so financial position is wrong.
3. **A freeze that did not take effect**, evidenced by any authorization
   succeeding after a freeze timestamp. This would mean the kill switch is
   not a kill switch.

Everything else — refusals, declines, expiries — is normal operation.
A scenario that alerts on refusals would train its operator to ignore it.

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| Nothing is implemented, so no signal above is emitted yet. | Every threshold is a proposal. | Implementation. |
| Product usage telemetry | Cannot validate the activation hypothesis in `../business/GO-TO-MARKET.md`. | Before monetization review. |
| Cost telemetry | Cannot evaluate hosted-facilitator or card-issuing unit economics. | Before the hosted tier is offered. |
| No signal distinguishes an identity *outage* from an identity *attack*. | Both look like verification failures; the responses differ completely. | Before the first automated rail. |
| Approval-fatigue signal is proposed but unvalidated. | Decision latency is a proxy for attention, and a weak one. | After real queue depth is observable. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
