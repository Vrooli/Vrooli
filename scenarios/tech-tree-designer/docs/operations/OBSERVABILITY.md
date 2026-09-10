# Observability — Tech Tree Designer

## Purpose Of This Document

Expose product correctness, progress and resource cost without treating health checks as evidence of design readiness.

## Signals

Current lifecycle health establishes process/API/UI reachability. Target product signals include source freshness/coverage, bounded query results, immutable review identity, validation applicability and per-entry apply/recovery state.

## Logs

Use lifecycle logs. Target structured records correlate proposal, revision, operation, entry, owner and validation run IDs. Record denied effects, conflicts, retries and recovery decisions without logging secret values or full sensitive drafts. Preserve durable receipts independently of transient logs.

## Metrics

Measure query/diff/apply latency distributions, source/cache age, visited/returned nodes and bytes, layout stalls, CPU/RSS/heap, disk IO, queue depth, retained unique content, pinned revisions and recovery backlog. See PERFORMANCE for cohort requirements and pending thresholds.

Product-value measures include author-to-review effort, artifacts re-authored after approval, conflict/recovery frequency and successful fresh-agent target discovery. These are evaluation hypotheses, not measured improvements.

## Alerts / Health

Separate process health, dependency availability, stale source coverage, validation gaps and unresolved effects. Configure thresholds from qualified cohorts and operator expectations. An empty graph or completed HTTP request is not proof that the full operation succeeded.

## Telemetry Gaps

The expanded proposal/scale sensors and alert thresholds are unqualified. Establish owner, invocation, bounded result schema, freshness and evidence storage for each sensor before claiming autonomous-development acceptance. Missing sensors must remain visible.

## Cross-References

- [Runbook](RUNBOOK.md)
- [Performance](../internal/PERFORMANCE.md)
- [Testing](../internal/TESTING.md)
