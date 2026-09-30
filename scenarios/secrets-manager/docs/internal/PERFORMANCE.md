# Performance — Secrets Manager

## Purpose Of This Document

This document records performance constraints for operational scans and posture queries.

## Budgets

Health and posture queries should complete promptly under a healthy lifecycle. Security scanning must respect file and context limits.

## Current Measurements

Use Test Genie performance evidence and scenario-local tests as the source of current measurement; do not freeze derived performance counts in this document.

## Known Constraints

Full filesystem scans and broad test suites can take substantial time. Desktop SQLite uses a constrained connection pool, so nested queries must close result cursors before issuing another query.

## Regression Procedure

Run the server-owned scenario suite and inspect the terminal Test Genie artifacts. Investigate increased runtimes before altering timeouts.

## Cross-References

- [Testing](TESTING.md)
- [Troubleshooting](../guides/troubleshooting.md)
