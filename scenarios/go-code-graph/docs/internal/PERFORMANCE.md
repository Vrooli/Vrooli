# Performance — Go Code Graph

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
| Extract — small module | <5 seconds end-to-end including Connect-RPC transport, for any module with ≤200 files | CI performance regression test against `bas/fixtures/go-small/` | required, OT-P0-010 / REQ-P0-010 |
| Extract — medium module | <30 seconds end-to-end including Connect-RPC transport, for any module with ≤2000 files | CI performance regression test against `bas/fixtures/go-medium/` | required, OT-P0-010 / REQ-P0-010 |
| Extract — large module | best-effort; document p50/p95 if it exceeds 30s | scheduled benchmark, not gating | informational |
| Rewrite plan | <1 second for ≤100 operations | unit benchmark | informational |
| Rewrite apply | scales linearly with operation count; <1 second per 10 file moves on a warm filesystem | integration benchmark | informational |
| API health | responsive under lifecycle health timeout | `/health` check | active |
| UI build | 5–10 minutes accepted for current Vite module graph | lifecycle/test-genie build logs | inherited |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured yet. | n/a | n/a | 2026-05-23 |

Measurements will be captured during implementation against `bas/fixtures/`.

## Known Constraints

- **`go/packages` load mode**: the fixed load mode (`NeedFiles | NeedImports | NeedTypes | NeedSyntax | NeedTypesInfo | NeedName | NeedDeps`) is CPU-heavy because it triggers full type checking. There is no lighter mode that preserves import resolution and type info. This is the dominant cost driver for `Extract`.
- **Per-path serialization**: two concurrent `Extract` calls for the same path serialize. By design — the second call would do duplicate work otherwise. Different paths run in parallel, bounded by `GOMAXPROCS`.
- **No internal cache**: every `Extract` call re-parses. Cartographer caches snapshots at its layer. If profiling later shows the bottleneck is repeated identical extractions, revisit the no-cache decision.
- **Vendored deps**: by default `vendor/` is excluded (REQ-P1-003). Including vendor can double parse time on modules that vendor heavily; the `--include-vendor` flag exists for callers that need it.
- **Vite production builds** may process thousands of modules and take several minutes (inherited template constraint).

## Regression Procedure

1. Run `make test`.
2. Run the dedicated performance suite: `go test ./api/internal/graph -run=Performance -bench=. -benchtime=5x`.
3. Compare results against the budgets table above. A regression beyond budget is a CI failure for `Extract — small` and `Extract — medium` (OT-P0-010 is gating).
4. For UI interaction regressions, use `ui/perf/README.md` and the provided capture template.
5. Record persistent findings here (as a row in **Current Measurements**) or in [`PROBLEMS.md`](PROBLEMS.md) depending on whether they are accepted constraints or unresolved debt.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
- [`../../PRD.md`](../../PRD.md) — OT-P0-010 Performance SLA
