# Observability

## Signals

Important signals are API health, Qdrant availability, PostgreSQL availability,
search latency, documentation health findings, and agent job terminal states.

## Logs

Scenario logs should be read through lifecycle commands rather than direct
process output.

## Metrics

The scenario exposes quality metrics for knowledge collections and stores job
state for asynchronous workflows.

## Telemetry Gaps

Documentation health scoring is deterministic but should continue to be tuned
as manifest contracts become richer.
