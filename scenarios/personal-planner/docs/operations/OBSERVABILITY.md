# Observability — Personal Planner

## Purpose Of This Document

This document defines the health, log, and metric signals Personal Planner
should expose, and the product-success signals that tell us the tool is
actually helping. It is design-stage: the only running code today is the
`/health` endpoints and the worked-example `notes` domain, so **no domain
metrics are instrumented yet**. The signal catalog below is the target
instrumentation from the implementation plan (§23.7), described so that
each metric is built the right way the first time — not a report that any
of it is being collected now. The single "Telemetry Gaps" entry records
that reality honestly.

Use this document to answer:

- What tells us the scenario is up and its integrations are current?
- What metrics should exist, and how must they avoid leaking personal content?
- How is a user-visible request traced to its internal command/job trail?
- What tells us users are getting value — measured from their own records?
- What telemetry is not yet instrumented?

## Signals

| Signal | Type | Source | Status | Purpose |
|---|---|---|---|---|
| API `/health` | health | Go API | **live** | API + dependency reachability (critical lifecycle check). |
| UI `/health` | health | Node UI server | **live** | UI bundle/server reachability. |
| test-genie result | validation | `make test` | live | Correctness evidence; R1 acceptance suite T01–T36. |
| Integration health panel | product/ops | integrations domain | planned | Owner-facing: last successful refresh, current problem, next retry/reconnect action, affected data scope. |
| Domain/job metrics | metric | API + jobs | planned | See Metrics — not yet instrumented. |

## Logs

| Log | Source | How to read | Notes |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Should carry a correlation/request ID that links a user-visible request to its internal command and job trail. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |
| Job diagnostics | background jobs | `make logs` (planned per-job status) | Per-job attempts, next retry, last error class, terminal unresolved state. Delivered notification/job diagnostics retain for 30 days. |

Logging rules (binding as domains land): never log credentials, provider
tokens, or tokenized share URLs; redact query parameters where a token
could appear; treat imported calendar descriptions and task titles as
untrusted text, never as log-format instructions.

## Metrics

Target metric catalog (plan §23.7). **Keep personal content out of every
metric label** — no titles, no recipient identities, no free-text.
Aggregate by safe code and scope, not by content. All of these are
**planned, not yet instrumented.**

| Metric | What it measures | Why it matters |
|---|---|---|
| Command errors by safe code | Failed commands grouped by stable error code | Detect broken paths without exposing what the user was doing. |
| Conflict / stale-proposal rate | Apply-time conflicts and proposals superseded before apply | High rate signals churn or fingerprinting/freshness problems. |
| Proposal computation time | Time to compute a planning proposal | Latency budget for the core planning loop. |
| Provider freshness | Age of the last successful provider refresh per scope | Drives the freshness warning; distinguishes stale-but-known from failing. |
| Sync failures | Provider sync errors, backoff, cursor invalidations | Health of read-only external-calendar import. |
| Pending outbox age | Oldest undelivered outbox record | Stuck delivery (notification-hub down, stranded lease). |
| Job retries | Retry counts and terminal unresolved states per job | Detect tight loops on invalid credentials / unsupported source commands. |
| Forecast age | Age of the latest forecast snapshot | Stale forecast after a material input revision. |
| Unexpected timer reconciliation | Focus-session/timer corrections that were not expected | Timer/actual-activity integrity; end-of-timer is not task completion. |

Correlation: a **user-visible request ID** must map to the internal
command and job trail so an operator can follow one user action end to
end through commands and any jobs it triggers — without reconstructing it
from personal content. A developer diagnostic report carries versions,
configuration categories, and redacted IDs only.

## Alerts / Health

- **Live:** lifecycle health checks poll `/health` on both surfaces
  (`api_endpoint` is critical; `ui_endpoint` is non-critical) on a
  ~30s interval with a startup grace period. `make status` surfaces the
  result.
- **Planned:** an owner-facing integration health panel (last successful
  refresh, current problem, next retry/reconnect action, affected data
  scope), plus alert thresholds derived from the Metrics catalog —
  pending outbox age, provider freshness, forecast age, sync failure rate,
  and job terminal-unresolved counts. Define concrete alert thresholds
  when a deployment tier beyond local development exists and operator
  expectations are known; do not fabricate alerting before it can fire.

## Product Success Signals

Product value is measured **from the user's own authorized records**, not
by surveillance. Learning uses only the user's data; optional usage
telemetry flows solely through the host's consent model and is never a
dependency of personal learning. Target signals:

- **Fewer unrecognized overloads** — the plan flags over-commitment
  before the user hits it, rather than the user discovering it after the
  fact.
- **Earlier risk identification** — material deadline-risk changes surface
  with useful lead time instead of at the deadline.
- **Better calibration** — the gap between estimated effort and effective
  actuals narrows over time as learning analysis produces versioned
  recommendations (never silent preference writes).

These are computed from the user's records inside their own workspace
(INV-01 scoping), and remain honest signals of usefulness — not
engagement metrics harvested across users.

## Telemetry Gaps

| Gap | Impact | Revisit trigger |
|---|---|---|
| **Nothing instrumented yet (design stage)** | Only `/health` and test-genie results exist; none of the Metrics-catalog signals, the integration health panel, request-ID correlation, or product-success signals are collected. Adoption, reliability, and value cannot yet be measured. | Instrument each signal as its owning domain/job lands; wire alert thresholds and the health panel before any tier beyond local development is released. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures and incident response
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — release gates and readiness
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependencies and failure modes
- [`../concepts/DATA.md`](../concepts/DATA.md) — retention of diagnostics, privacy rules
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — redaction and token handling
- [`../reference/configuration.md`](../reference/configuration.md) — configuration keys
