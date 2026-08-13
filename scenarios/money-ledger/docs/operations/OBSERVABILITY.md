# Observability — Money Ledger

This document records logs, metrics, telemetry, health checks, and
business/product signals for the scenario.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the scenario is healthy?
- What signals tell us users are getting value?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain before deployment or monetization?

## The rule that shapes this document

`OT-P1-004` — *adapter health is honest* — is written in the PRD as a product requirement,
but it is an **observability requirement wearing a product hat**. "An adapter that cannot
run is reported unavailable with a reason and an age; it is never reported as zero" is only
deliverable if the signals below exist. Everything here follows from making that claim
checkable rather than aspirational.

The corollary constrains what may be emitted: **no financial value may appear in a metric
label, a log line at info level, or a health payload.** Observability must prove the system
is honest without becoming a second, less protected copy of the ledger.

## Signals

| Signal | Type | Source | Purpose | Threshold |
|---|---|---|---|---|
| `/health` status | health | API | API and dependency reachability | healthy for local development |
| UI health endpoint | health | UI server | UI bundle/server reachability | responds during lifecycle health check |
| test-genie result | validation | `make test` | scenario correctness evidence | all required phases pass |
| Adapter availability | health | `ingest` | Per-adapter: `available` / `unavailable` / `never-run`, each with a reason code | any `unavailable` is surfaced, never suppressed |
| Adapter last-success age | freshness | `ingest` | How stale the newest successful sync is, per adapter | compared against that adapter's declared freshness window |
| Position completeness | integrity | `position` | Whether the current position had every contributing adapter available | `partial` is a first-class result, not a failure |
| Basis mix | integrity | `journal` | Share of events by basis: `authoritative` / `derived` / `operator-asserted` | no threshold; a *trend* toward operator-asserted is the signal |
| Ingestion idempotency | correctness | `ingest` | Duplicate postings suppressed on overlapping re-runs | must be zero duplicates (`OT-P0-007`) |
| Reversal rate | product | `journal` | Reversing entries as a share of postings | a rising rate means data entry is going wrong upstream |

**`never-run` is deliberately distinct from `unavailable`.** An adapter that has never
executed and one that executed and failed are different facts, and collapsing them is the
same class of error as reporting a missing figure as zero.

## Logs

| Log | Source | How To Read | Details |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Request logging uses the deterministic clock seam in tests. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |
| Adapter run log | `ingest` | `make logs` | One entry per run: adapter, window, outcome, counts, duration. **Counts, never amounts.** |
| Audit trail | `journal` (database, not a log file) | CLI query | Append-only; the durable record. Not a log stream and not subject to log retention. |

**Prohibited in logs at any level below debug:** posting amounts, account identifiers,
counterparty names, credential values or handles, and file paths of imported statements.
An adapter failure logs the reason class and the adapter, never the payload that failed.

## Metrics

| Metric | Status | Details |
|---|---|---|
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |
| Adapter availability by adapter | required | The metric behind `OT-P1-004`. Label by adapter id and reason class only. |
| Time since last successful sync | required | Per adapter. Enables the freshness display the UI contract depends on. |
| Partial-position rate | required | Share of position reads that were partial. Directly measures how often the honesty path is exercised — and if it is never exercised, the path is untested in production. |
| Basis distribution over time | required | Rising `operator-asserted` share means adapters are decaying and the user is compensating by hand. |
| Duplicate-suppression count | required | Non-zero suppression is healthy (re-runs happen); non-zero *duplicates* is a defect. |
| Reversal rate | recommended | Product-quality signal for data entry. |
| Product activation | deferred | Define after the first real user. Must not require exfiltrating financial data — count events, never sum amounts. |
| Performance budgets | deferred | Define in `../internal/PERFORMANCE.md`. Position is computed at read time, so its latency is a real budget, not an afterthought. |
| Cost telemetry | deferred | Only meaningful if AI-assisted categorisation ships and consumes gateway tokens. |

## Alerts / Health

The generated scenario has lifecycle health checks for API and UI. Beyond those:

- **An adapter transitioning to `unavailable` is a user-facing state, not an alert.** It
  belongs on the adapters surface with its reason and age. Paging an operator for a
  routinely flaky third-party API trains them to ignore it.
- **A position read that is partial is not an error.** It must not increment an error
  metric or produce an error-level log, or normal operation will look like an outage.
- **Duplicate postings surviving ingestion is a real alert.** It means idempotency is
  broken and the journal is accruing corruption that only reversal can undo.

## What "users are getting value" looks like here

Deliberately narrow, because the honest answer is that most value signals for a finance
product require data this scenario should not emit:

| Question | Signal | Notes |
|---|---|---|
| Is the ledger trusted? | Reversal rate falling; basis mix stable | A user who trusts it stops correcting it. |
| Is the honesty path working? | Partial-position rate is non-zero and adapter failures resolve | A permanently zero partial rate means either perfect upstreams or a broken detector. Assume the latter. |
| Is manual entry viable? | Manual-adapter event count | If sources without an API are genuinely first-class, this number is healthy, not embarrassing. |

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| No adapter availability metric yet | `OT-P1-004` is undeliverable — the claim cannot be checked. | Build with the `ingest` domain, not after. |
| No freshness window declared per adapter | "Stale" has no definition, so the stale UI state cannot be entered. | Required before the adapters surface ships. |
| Product usage telemetry | Cannot validate monetization or adoption. | Before public launch or monetization review, and only in a form that emits no amounts. |
| Cost telemetry | Cannot evaluate unit economics. | Before managed deployment or AI-assisted categorisation. |
| No signal distinguishing "no events" from "no adapter ran" | An empty journal and a broken pipeline look identical — a silent-zero failure at the system level. | Before the first non-manual adapter. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — what must never be emitted
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
