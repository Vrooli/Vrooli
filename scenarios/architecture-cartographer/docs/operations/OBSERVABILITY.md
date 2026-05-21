# Observability — Architecture Cartographer

This document records logs, metrics, telemetry, health checks, and
business/product signals for the scenario.

Cartographer is a developer-facing tool, so most observability is
about agent/maintainer experience, not user funnel analytics. The
analytics event log (see `analytics` domain in
[`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)) carries the
load-bearing signals.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the scenario is healthy?
- What signals tell us cartographer is producing trustworthy verdicts?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain before deployment or monetization?

## Signals

| Signal | Type | Source | Purpose | Threshold |
|---|---|---|---|---|
| `/health` status | health | API | API and dependency reachability | healthy for local development |
| UI health endpoint | health | UI server | UI bundle/server reachability | responds during lifecycle health check |
| `go-code-graph` reachability | dependency-health | `arch-cart status` | Cartographer can extract Go graphs | reachable when needed |
| `typescript-code-graph` reachability | dependency-health | `arch-cart status` | Cartographer can extract TS graphs | reachable when needed |
| Build-green baseline status | correctness | `apply` domain | Apply operations land cleanly | baseline-compatible after every apply |
| `--force --note` rate | quality | analytics event log | Force-override usage indicates conflicts or build issues that aren't being properly addressed | < 5% of resolutions; if higher, investigate |
| Override rate by signal | quality / calibration | analytics event log | Per-signal frequency of being overridden by an agent — drives weight calibration | tracked over time; rising rate for a signal means downweight candidate |
| Conflict detection latency | performance | timing test | Time from graph extraction to conflict list | within budgets in [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) |
| Auto-place / suggest / conflict ratio | health | analytics event log | Distribution of verdict tiers — if `conflict` dominates, the manifest is underspecified | watched over time per scenario |
| test-genie result | validation | `make test` | Scenario correctness evidence | all required phases pass |

## Logs

| Log | Source | How To Read | Notes |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Request logging uses deterministic clock seam in tests; structured JSON in production. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |
| Analytics events | SQLite `events` table | `arch-cart history` and `arch-cart stats` | Append-only; never trimmed automatically. |
| Apply audit log | SQLite `apply_runs` table | `arch-cart history --filter applies` | Includes baseline status, post status, force notes. |
| Force-note audit | derived from analytics | `arch-cart history --filter force-notes` | Quarterly review per [`RUNBOOK.md`](RUNBOOK.md). |
| Calibration proposals | analytics + signal weights | `arch-cart calibrate --dry-run` | Monthly review per [`RUNBOOK.md`](RUNBOOK.md). |

## Metrics

| Metric | Status | Source | Notes |
|---|---|---|---|
| Migrations started | active (post-MVP) | `analytics.events` | Counted per scenario per month. |
| Migrations finalized | active (post-MVP) | `analytics.events` | Successful completion rate; abandonment rate is the inverse. |
| Conflicts detected per migration | active (post-MVP) | `analytics.events` | Per scenario, per conflict type. |
| Conflicts auto-resolved (via resolver) | active (post-MVP) | `analytics.events` | Mechanical resolution rate; rising = recipe candidate. |
| Conflicts manually resolved | active (post-MVP) | `analytics.events` | Inverse of above. |
| Auto-placement override rate per signal | active (post-MVP) | `analytics.events` | Highest-value calibration signal — see [`../internal/DECISIONS.md`](../internal/DECISIONS.md). |
| Apply build-green pass rate | active (post-MVP) | `analytics.events` | Should be near 100%; drops indicate either real regressions or `--force` overuse. |
| Recipe usage by type | active (when recipes ship in P1) | `analytics.events` | Drives recipe-identification pipeline (OT-P2-004). |
| Requirement coverage | active | requirements/* + test-genie | Tracked through requirements and test-genie coverage artifacts. |
| Performance budget compliance | active (post-MVP) | integration timing tests | Each budget in `PERFORMANCE.md` has a corresponding test. |
| Product activation / GTM | deferred (not-applicable in v1) | n/a | Cartographer is an internal tool in v1; user-funnel metrics are not in scope. |

## Alerts / Health

The lifecycle health check pattern is sufficient for local development.
For multi-developer or shared deployments (deferred), additional
alerts would be needed:

- Dependency-unreachable spike (cartographer calls `go-code-graph` or
  `typescript-code-graph` and gets transport failures) — alert at
  >5% failure rate over rolling 1h window.
- `--force --note` rate exceeds 10% of resolutions — alert
  immediately; this indicates a systemic problem.
- Build-green baseline check fails on >50% of applies — alert; likely
  a misconfigured build environment or a bad apply path.
- Override rate for a single signal exceeds 30% for a week — calibrate.

None of these alerts ship in v1; they are documented here so when the
deployment tier matures, the signals are already understood.

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| Cross-scenario calibration analytics | Per-scenario history is local SQLite. Aggregating across scenarios requires either each developer running `arch-cart history export` and combining manually, or a shared analytics resource. | When cartographer adoption crosses ~3 active scenarios. P2 in-scope. |
| Histogram of conflict-detection latency | Without per-run latency capture, performance regressions are caught only by deliberate timing tests. | Add when v1 ships and real workloads exist. |
| Recipe success rate when shipped | Required for the recipe-identification pipeline (OT-P2-004). | When the first recipe ships (P1). |
| Product / business telemetry | Cartographer has no GTM funnel in v1; if it later ships as a SaaS or shared offering, usage telemetry is needed. | See [`../business/MONETIZATION.md`](../business/MONETIZATION.md) revisit triggers. |
| External alerting integration (PagerDuty, Slack) | Local-only in v1; not applicable. | Required for multi-developer / shared deployments. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance budgets and measurements
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — analytics-as-P0 rationale
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — `analytics` domain ownership
