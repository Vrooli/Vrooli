# Observability

## Purpose Of This Document

Document health signals and operational telemetry.

## Signals

Key signals are API health, UI health, upstream fact-service health, graph error counts, and drift finding counts.

## Logs

Lifecycle logs are under `.vrooli/logs` and scenario test artifacts are under `coverage/logs`.

## Metrics

Current metrics are mostly phase/test outputs and API health payloads.

## Alerts / Health

Health endpoints should report degraded state when SQLite or upstream fact services are unavailable.

## Telemetry Gaps

Dedicated graph build latency metrics and upstream fact freshness metrics remain future work.

## Cross-References

- `RUNBOOK.md`
- `../internal/TESTING.md`
