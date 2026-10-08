# Performance

## Purpose Of This Document

Track performance budgets and known constraints.

## Budgets

UI pages should meet configured Lighthouse thresholds and keep graph interactions responsive for normal local scenario counts.

Dependency freshness targets are a warm `freshness --touched` run below 1 s,
a cold-cache run below 6 s, and `deps reconcile --all --json` below 10 s.
The repeatable measurement command is:

```bash
bash scenarios/scenario-dependency-analyzer/test/perf/freshness-bench.sh \
  --label after scenarios/scenario-dependency-analyzer/coverage/perf
```

## Current Measurements

Scenario test artifacts under `coverage/logs` hold the latest Lighthouse measurements.

## Known Constraints

D3 force layouts can become expensive for large graphs.

## Regression Procedure

Run the full scenario test and inspect performance artifacts when thresholds fail.

## Cross-References

- `../operations/OBSERVABILITY.md`
- `../guides/troubleshooting.md`
