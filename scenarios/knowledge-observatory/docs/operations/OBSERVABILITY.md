# Observability

## Signals

Important signals are API health, Qdrant availability, search latency,
documentation health findings, and agent job terminal states. Scenario metadata
and workflow records persist in SQLite; PostgreSQL is not a runtime dependency.

## Logs

Scenario logs should be read through lifecycle commands rather than direct
process output.

## Metrics

The scenario exposes quality metrics for knowledge collections and stores job
review state for asynchronous workflows. agent-manager reports execution state.

## Telemetry Gaps

Documentation health scoring is deterministic but should continue to be tuned
as manifest contracts become richer.
