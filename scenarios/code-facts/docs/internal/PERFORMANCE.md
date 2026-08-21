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
| Persistent exact search, current fixture | p95 at most 100 ms over 32,000 documents | `TestSQLiteExactSearchPerformanceBudgets/current` | active |
| Persistent exact search, three-times fixture | p95 at most 200 ms over 96,000 documents | `TestSQLiteExactSearchPerformanceBudgets/three-times` | active |
| Idle process RSS | at most 150 MiB | `/health` `rss_mb` plus lifecycle process status | active |
| Query RSS delta | at most 50 MiB | before/after `/health` `rss_mb` around a fixed query workload | active |
| Index RSS high-water | at most 500 MiB | `/health` `rss_high_water_mb` during a lifecycle-managed rebuild | active |
| Expensive-work admission | 16 weighted units, 64 queued callers, 2 s maximum wait | `/health` `admission_*` metrics and focused cancellation tests | active |

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

SQLite report headers and facts are stored separately; paged cache hits read
only the requested fact rows. Graph cache entries are binary protobuf under
gzip and are keyed by the individual parse-unit source/config fingerprint.
Consequently, an edit in one language surface invalidates that surface's graph
entry without forcing extraction of unrelated parse units. The regression test
touches one file and asserts the other unit's fingerprint is unchanged.

One FIFO weighted admission controller governs fallback scans (weight 4),
embedding (4), graph extraction (8), fleet discovery (8), and indexing (12).
Query coordination uses weight 1. Identical graph misses and generation-keyed
queries coalesce; final results use a 64-entry/8 MiB defensive-copy LRU.
Cancellation removes queued work immediately and capacity release is idempotent.

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
| Cold `imports` describe for `scenario:go-code-graph` | Go parse units completed in 16-231 ms each with the structural profile; 1.5 s on the subsequent whole-report cache hit | lifecycle-managed live request and provider access logs | 2026-08-20 |
| Idle API after startup cache sweep | 25 MB RSS / 2.6 MB live heap, down from 3.92 GB RSS / 2.21 GB heap with the eager resident index | lifecycle-managed process status and `/health` | 2026-08-20 |
| Streaming project search, `provider demotion`, limit 5 | 30.6 s; post-request 44 MB RSS / 5.1 MB live heap | `code-facts facts search` plus lifecycle-managed process status | 2026-08-20 |
| Execution-start project search, `provider demotion`, limit 5 | 21.67 s wall; API CPU increased by about 27 s; API `syscr` increased by 505,924; API `read_bytes` increased by 255,668,224; API RSS changed from 72.5 MiB to 67.4 MiB after completion | lifecycle-managed API PID plus `/usr/bin/time -v`, `/proc/<pid>/io`, and `ps`; baseline run `20260820-225937-4625b831` | 2026-08-20 |
| Execution-start SQLite storage | 286,285,824-byte primary database plus 28,872-byte WAL | lifecycle-managed API open-file inventory | 2026-08-20 |
| Whole-project cache status | 2.9 s with metadata-only SQL projection; payload bodies were not materialized | `code-facts cache status project:<repo-root>` | 2026-08-20 |
| Persistent exact search, 32,000 documents | 7.24 ms p50; 11.96 ms p95; 14.18 ms p99; 91 allocations/query | deterministic in-memory SQLite scale proof | 2026-08-20 |
| Persistent exact search, 96,000 documents | 43.04 ms p50; 72.51 ms p95; 106.44 ms p99; 91 allocations/query | deterministic in-memory SQLite three-times scale proof | 2026-08-20 |
| Embedding profile smoke bake-off | 384-dim and 768-dim `nomic-embed-text` both achieved recall@5 1.00 and MRR@3 1.00 on five installed-host cases | `internal/retrieval/testdata/model-bakeoff-v1.json` | 2026-08-20 |

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- Cache fingerprinting walks bounded parse-unit roots and prunes dependency/build directories.
- Go provider extraction is capability-aware: imports, symbols, and proto adoption request `structural`; references, calls, and endpoint proofs request `semantic`. The profile is part of the graph cache identity so a structural result cannot satisfy a semantic request.
- The serving persistent retrieval package reads normalized catalog, external-content FTS5, and freshness rows only. The legacy project/repository path still streams source roots until the cutover phase wires the new package into public handlers and removes the old implementation.
- Cache status and inspection select metadata only; compressed graph/report bodies are loaded exclusively by exact cache reads.
- Cache hits never update SQLite. Trigger-maintained per-scope counters make health and budget checks constant-time. The global byte ceiling is `CODE_FACTS_CACHE_MAX_BYTES` (2 GiB default); graph data may use 75% and report data 50%, allowing unused capacity to be borrowed while preventing either kind from monopolizing storage.
- Startup cleanup deletes at most 100 stale rows and 100 entries older than seven days, then uses passive WAL checkpointing and bounded incremental vacuum. Repeated startup sweeps converge without an unbounded request-path `VACUUM`.
- `/health` reports current OS RSS, RSS high-water, process CPU seconds, Go heap, catalog-derived generation counts, storage bytes, active jobs, degraded reasons, queue depth/high-water, rejection/cancellation totals, and admission wait p50/p95/p99. Cache rows are not treated as indexed corpus counts.
- Performance budgets for proof synthesis should be revisited after Phase 11 exposes larger operator UI workflows and Phase 12 adds the first external consumer.

## Regression Procedure

1. During implementation, run focused package tests and the relevant single
   benchmark. Reserve `make test` and Test Genie suites for authored phase or
   final gates.
2. Run the legacy current-corpus comparison exactly once with
   `go test ./internal/facts -run '^$' -bench '^BenchmarkLegacyProjectSearchCurrentCorpus$' -benchtime=1x -count=1`.
3. Set `CODE_FACTS_BENCH_3X_ROOT` to the prepared deterministic scale fixture
   and run `BenchmarkLegacyProjectSearchThreeTimesCorpus` with the same
   `-benchtime=1x` constraint.
4. Run the persistent exact-search gate with `CODE_FACTS_PERF_ASSERT=1 go test ./internal/retrieval -run '^TestSQLiteExactSearchPerformanceBudgets$' -count=1 -v`. The test creates 32,000- and 96,000-document fixtures, records p50/p95/p99 and allocations, and enforces the 100 ms and 200 ms p95 budgets.
5. Capture relevant API/UI command timing. Record wall time, CPU, allocations,
   live heap, RSS high-water mark, source files and bytes opened, candidate
   counts, database bytes, and stage timings. Metrics unavailable from the Go
   benchmark must come from the lifecycle-managed API process and operating
   system counters.
6. For UI interaction regressions, use performance-health with a registered
   performance workflow.
7. Record persistent findings in this document or
   [`PROBLEMS.md`](PROBLEMS.md) depending on whether they are accepted
   constraints or unresolved debt.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
