# Performance — TypeScript Code Graph

This document records performance budgets, current measurements, known constraints, and regression procedures.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| Extract — small project | <5 seconds end-to-end including Connect-RPC + sidecar IPC, for any project with ≤200 files | CI performance regression test against `bas/fixtures/ts-small/` | required, OT-P0-012 / REQ-P0-012 |
| Extract — medium project | <30 seconds end-to-end including Connect-RPC + sidecar IPC, for any project with ≤2000 files | CI performance regression test against `bas/fixtures/ts-medium/` | required, OT-P0-012 / REQ-P0-012 |
| Extract — large project | best-effort; document p50/p95 if it exceeds 30s | scheduled benchmark | informational |
| Sidecar spawn + handshake | <2 seconds | startup probe in lifecycle health check | required |
| Sidecar heartbeat response | <500 milliseconds | supervisor heartbeat probe | required |
| Rewrite plan | <1 second for ≤100 operations | unit benchmark | informational |
| Rewrite apply | scales linearly; ts-morph save typically <50ms per file | integration benchmark | informational |
| IPC round-trip latency | <10ms p50 over stdio for small payloads | sidecar microbenchmark | informational |
| API health | responsive under lifecycle health timeout | `/health` check (includes sidecar reachability) | active |
| UI build | 5–10 minutes accepted for current Vite module graph | lifecycle/test-genie build logs | inherited |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured yet. | n/a | n/a | 2026-05-23 |

Measurements will be captured during implementation against `bas/fixtures/`.

## Known Constraints

- **`ts-morph` Project initialization**: each `Extract` constructs a fresh Project from the target `tsconfig.json`. This is the dominant cost. Caching Projects across calls is intentionally deferred (see [`DECISIONS.md`](DECISIONS.md) — no internal cache in v1).
- **Node sidecar IPC overhead**: each call has at least one round-trip (request + response) plus any streaming progress messages for `RewriteApply`. JSON serialization of large graphs is the second-largest cost driver.
- **Per-path serialization (both layers)**: two concurrent `Extract` calls for the same path serialize. Different paths run in parallel, bounded by `GOMAXPROCS` on the Go side and Node's event loop on the sidecar side.
- **No internal cache**: every `Extract` re-parses. `graph_hash` lets callers detect "no change".
- **Sidecar restart cost**: a sidecar crash triggers respawn + handshake (~500ms-2s depending on backoff). In-flight requests during the gap return `SidecarUnavailable`.
- **Memory growth from large projects**: `ts-morph` holds an AST in Node memory. Very large projects (>10k files) may exceed default Node heap. Document the failure mode rather than try to handle it transparently.

## Regression Procedure

1. Run `make test`.
2. Run the dedicated performance suite: `go test ./api/internal/graph -run=Performance -bench=. -benchtime=5x`.
3. Compare results against the budgets table above. Small + medium regressions beyond budget are CI failures (OT-P0-012 is gating).
4. For IPC-overhead profiling, the sidecar exports its own profile via `sidecar/bench/`.
5. For UI interaction regressions, use `ui/perf/README.md`.
6. Record persistent findings here or in [`PROBLEMS.md`](PROBLEMS.md).

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
- [`../../PRD.md`](../../PRD.md) — OT-P0-012 Performance SLA
