# Observability — Audio Tools

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
| Turn diagnostic | user/operator recovery | Dictation Studio Record tab | Shows protocol state, durable-capture level, captured/processed chunk coverage, and terminal reason; the export is bounded metadata only (never PCM or transcript). | Any failed or incomplete turn must expose a terminal reason or retained recovery state. |

## Logs

| Log | Source | How To Read | Notes |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Request logging uses deterministic clock seam in tests. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |
| Safe turn export | Dictation Studio Record tab | Use **Turn details → Export safe diagnostic** after a failed/degraded turn. | Includes opaque session identity and state codes only; it excludes transcript and audio. |

## Metrics

| Metric | Status | Notes |
|---|---|---|
| Product activation | target, not joined | Voice attempt, route selected, first successful turn and recovery outcome per consented consumer cohort; no private speech in telemetry. |
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |
| Voice performance | instrument qualification open | Click/capture/session ready, first voiced sample, visible partial, committed interval position and final tail; clocks in PERFORMANCE.md, candidate bands in TESTING.md. |
| Owned service usage | implementation/qualification open | Correlate account, operation, session, meter/policy revision, durable delivery and shared ledger receipt; customer debit is not upstream cost. |
| Acceptance coverage | target inventory present; receipt joins missing | setpoint-read v2 lists all 15 PRD targets as unknown. Enumeration and collection success cannot imply acceptance. |

For each run retain build/source, engine/model, route, consumer, OS/browser,
device/acceleration, corpus/method/policy revision, cold/warm state and fake/live
lane. Preserve failed attempts, missing counts and timeouts. The acceptance owner
must refuse incompatible or stale joins rather than select the newest green run.

## Alerts / Health

The scenario has lifecycle health checks for API and UI. They do not establish
microphone/session readiness, deployed-source freshness, stream cadence, final
coverage or billing correctness. Add
deployment-specific alerts only when deployment target and operator
expectations are known.

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| Product usage telemetry | Cannot validate monetization or adoption. | Add before public launch or monetization review. |
| Cost telemetry | Cannot evaluate hosted/SaaS unit economics. | Add before managed deployment. |
| Current owner-backed outcome joins | Cannot accept the full development goal from the setpoint board. | Implement and negative-test before full-mandate completion. |
| Per-branch duration, early EOF and final-tail oracles | A fresh soak artifact can overstate STT qualification. | Repair before crediting long-duration runs. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
