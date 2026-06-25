# Observability — Plan Manager

This document records logs, metrics, telemetry, health checks, and
business/product signals for plan-manager.

## Purpose Of This Document

Use this document to answer:

- What signals tell us plan-manager is working and valuable?
- Where do logs and metrics come from?
- What health checks and alerts exist?
- What telemetry is still missing?

## Signals

- Product success signal: per-plan velocity — time, tokens, and iterations
  to author and execute a plan. This is the core measure of whether
  planning is getting cheap enough for local models, and it is emitted to
  meta-optimization-manager as trial data.
- Health-ish signals: plan staleness (plan→code references drifting from
  the tree) and validation status (results from test-genie /
  scenario-validation). These describe plan quality/freshness rather than
  process liveness.

## Logs

- Standard scenario logging from the Go api server and CLI, viewable via
  `make logs`.
- Logs should record plan lifecycle events, integration calls that
  degraded (soft integrations failing closed), and staleness/validation
  outcomes. plan-manager does NOT log agent transcripts — it neither reads
  transcripts nor spawns agents.

## Metrics

- Primary metric: per-plan velocity (time/tokens/iterations), forwarded to
  the meta-optimization velocity sink as trial data.
- Secondary metrics (intended): counts of stale plans, validation
  pass/fail, and candidate-finding volume.
- Run attribution: where an agent-manager run id is present
  (`VROOLI_AGENT_MANAGER_RUN_ID`), velocity can be attributed to that run
  via the run-id attribution contract.
- This scenario is pre-implementation, so exact metric names and emission
  cadence are deferred until the first vertical slice exists.

## Alerts / Health

- Health check: the standard Vrooli scenario health endpoint reports
  process liveness; this is what the lifecycle and status commands consume.
- Staleness and failing validation are surfaced as plan-level signals in
  the UI/CLI rather than as paging alerts. Dedicated alerting thresholds
  are deferred until real measurements exist.

## Telemetry Gaps

- No measurements yet — the scenario is documentation-first, so velocity,
  staleness, and validation telemetry are designed but not yet collected.
- Cross-scenario velocity correlation (linking plan velocity to downstream
  outcomes) depends on meta-optimization-manager being available and is a
  known gap when that integration is absent.
- Detailed per-phase tracing is not yet defined.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — runtime and release context
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system architecture
