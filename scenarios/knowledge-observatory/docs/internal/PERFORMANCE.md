# Performance

## Budgets

Documentation health checks should be filesystem-bound and deterministic.
Search and graph operations should cap result sizes and avoid unbounded vector
collection scans.

## Current Measurements

No current performance budget is enforced in code for documentation validation.

## Known Constraints

Large repositories can make global documentation search expensive. Agent-backed
search and healing are intentionally asynchronous.

## Regression Procedure

Add focused tests or benchmarks before widening validation to expensive content
analysis.
