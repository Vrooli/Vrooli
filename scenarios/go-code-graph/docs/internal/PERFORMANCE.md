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
| Full profile, go-usage-facts fixture | 657ms load / 0.21ms normalize | `GOMAXPROCS=8 GOWORK=off go test ./internal/graph -run '^$' -bench '^BenchmarkExtractProfiles$' -benchtime=1x -count=1` | 2026-08-20 |
| Semantic profile, go-usage-facts fixture | 587ms load / 0.21ms normalize | same benchmark | 2026-08-20 |
| Structural profile, go-usage-facts fixture | 104ms load / 0.10ms normalize | same benchmark | 2026-08-20 |
| Warm in-process cache hit, go-usage-facts fixture | ~0.12ms total / ~0.068ms fingerprint | `GOWORK=off go test ./internal/graph -run '^$' -bench '^BenchmarkExtractCacheHit$' -benchtime=1x` | 2026-08-20 |
| Warm in-process cache hit, API module | 15ms total / 1.7ms fingerprint | `GO_CODE_GRAPH_BENCH_LARGE=1 GOWORK=off go test ./internal/graph -run '^$' -bench '^BenchmarkExtractCacheHitLarge$' -benchtime=1x` | 2026-08-20 |

These are relative local measurements, not the CI SLA gate. They show the structural profile removes the dominant type-checking cost for consumers that do not need semantic facts.

Consumers must treat profiles as explicit information contracts. Every successful
response returns its effective profile and typed `omitted_information` metadata;
an empty resolved-facts field under `structural` is therefore an intentional
omission, not an unexplained failure. Architecture-cartographer currently uses
`structural` for its production Go inventory path and preserves the omission
metadata in its snapshots.

## Known Constraints

- **`go/packages` load mode**: the default full load mode (`NeedFiles | NeedImports | NeedTypes | NeedSyntax | NeedTypesInfo | NeedName | NeedDeps`) is CPU-heavy because it triggers full type checking. The explicit structural profile omits type checking and is the quick path when semantic facts are unnecessary.
- **Profiles and scope**: `full` is the compatibility default; `semantic` retains type analysis but excludes test variants; `structural` returns structural facts only. `package_patterns` accepts module-relative Go patterns such as `./api/...` to avoid loading unrelated packages.
- **Per-path serialization**: two concurrent `Extract` calls for the same path serialize. By design — the second call would do duplicate work otherwise. Different paths can run in parallel up to the global extraction cap.
- **CPU containment**: production extraction is capped at one concurrent module load by default (`GO_CODE_GRAPH_MAX_CONCURRENT_EXTRACTS=1`) and the lifecycle sets `GOMAXPROCS=8`. This prevents one fleet-wide request from multiplying type-checking work across modules or consuming every host core. Raise either limit only from measured host capacity; per-path locking still provides the correctness boundary independently of the global throughput cap.
- **Content-fingerprint cache**: successful extractions are reused when source content, profile, scope, vendor setting, Go runtime, and loader-affecting environment are unchanged. Fingerprinting still walks the module, so a future optimization should make fingerprinting package-aware or incremental. The cache is bounded and disposable: 8 in-process entries/128 MiB, with a 512 MiB disk limit by default.
- **Vendored deps**: by default `vendor/` is excluded (REQ-P1-003). Including vendor can double parse time on modules that vendor heavily; the `--include-vendor` flag exists for callers that need it.
- **Vite production builds** may process thousands of modules and take several minutes (inherited template constraint).

## Regression Procedure

1. Run `make test`.
2. Run the dedicated performance suite from `scenarios/go-code-graph/api`: `GOWORK=off go test ./internal/graph -run '^$' -bench '^BenchmarkExtractProfiles$' -benchtime=5x`.
3. Compare results against the budgets table above. A regression beyond budget is a CI failure for `Extract — small` and `Extract — medium` (OT-P0-010 is gating).
4. For UI interaction regressions, use `ui/perf/README.md` and the provided capture template.
5. Record persistent findings here (as a row in **Current Measurements**) or in [`PROBLEMS.md`](PROBLEMS.md) depending on whether they are accepted constraints or unresolved debt.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
- [`../../PRD.md`](../../PRD.md) — OT-P0-010 Performance SLA
