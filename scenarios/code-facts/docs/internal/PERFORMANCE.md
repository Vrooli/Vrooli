# Performance — Code Facts

This document records performance budgets, current measurements, known
constraints, and regression procedures.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| UI build | 5-10 minutes accepted for current Vite module graph | lifecycle/test-genie build logs | inherited |
| API health | responsive under lifecycle health timeout | `/health` check | active |
| UI health | responsive under lifecycle health timeout | `/health` check | active |
| Repeated selective describe | Provider extraction avoided after identical cached request | API unit test with counting provider | active |
| `file_domain` scenario scan | 16 bounded `ScoreChunk` calls in parallel, deterministic output | `internal/facts/filedomain.go` and focused tests | active |
| Graph invalidation | One touched parse-unit root changes only that unit's graph key | `TestUnitFingerprintInvalidatesOnlyTouchedParseUnit` | active |

## Phase 5 Cost Profile

The original `file_domain` deadline failure was reproduced by tracing one
`ScoreChunk` RPC per source file: the implementation waited for those calls in
sequence, so total latency grew linearly with the number of files and was
dominated by network round trips. The confirming experiment was to retain the
same RPC payloads and verdict ordering while running them through a bounded
worker pool. The resulting implementation uses 16 workers, preserves input
ordering in the returned facts, and sorts warnings before publication. This
establishes the root cause as fan-out serialization rather than classifier
scoring or protobuf decoding.

The report cache remains whole-report for compatibility, while graph cache
entries are now keyed by the individual parse-unit source/config fingerprint.
Consequently, an edit in one language surface invalidates that surface's graph
entry without forcing extraction of unrelated parse units. The regression test
touches one file and asserts the other unit's fingerprint is unchanged.

| Family | Target | Cold budget | Warm budget | Measurement contract |
|---|---|---:|---:|---|
| `symbols`, `imports`, `references`, `calls` | one parse unit | 2,000 ms | 500 ms | provider extraction + graph cache |
| `proto_adoption`, `endpoint_proofs` | scenario | 3,000 ms | 750 ms | proof synthesis over cached graph |
| `file_domain` | `scenario:search-hub` | 10,000 ms | 3,000 ms | bounded Cartographer fan-out |
| fleet project scan | `project:` | 60,000 ms | 20,000 ms | bounded scenario workers |

These are enforceable budgets, not claims that every host currently meets
them. `AssertFamilyCost` is intentionally tested with a lowered synthetic
budget so a future benchmark cannot silently accept a regression.

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| Repeated `DescribeCodeFacts` cache reuse | second identical request uses report cache and makes zero provider calls | `internal/facts` unit test | 2026-06-12 |

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- Cache fingerprinting walks bounded parse-unit roots and prunes dependency/build directories.
- Performance budgets for proof synthesis should be revisited after Phase 11 exposes larger operator UI workflows and Phase 12 adds the first external consumer.

## Regression Procedure

1. Run `make test`.
2. Capture relevant API/UI command timing.
3. For UI interaction regressions, use `ui/perf/README.md` and the
   provided capture template.
4. Record persistent findings in this document or
   [`PROBLEMS.md`](PROBLEMS.md) depending on whether they are accepted
   constraints or unresolved debt.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
