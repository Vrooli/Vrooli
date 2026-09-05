# Performance — Architecture Cartographer

This document records performance budgets, current measurements, known
constraints, and regression procedures.

The cartographer's hot path is graph extraction and conflict
detection over potentially large target scenarios. Performance budgets
exist so an agent running `arch-cart` mid-conversation does not block
on multi-minute extractions.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

Targets are end-to-end (client invocation → result returned), assuming
the language-graph dependency scenarios are running and the SQLite
cache is warm where applicable.

| Surface | Workload | Budget | Measurement | Status |
|---|---|---|---|---|
| `arch-cart graph extract` (small) | ≤200 files | < 5s | Integration timing test against `bas/fixtures/small/` | budgeted |
| `arch-cart graph extract` (medium) | ≤2000 files | < 30s | Integration timing test against `bas/fixtures/medium-realistic/` | budgeted |
| `arch-cart graph extract` (cached, unchanged) | any size | < 500ms | Source-fingerprint cache hit returns before language adapters | active |
| Signal scoring (full ladder) | 2000-file graph, all candidate domains | < 2s | Unit benchmark | budgeted |
| `arch-cart conflicts list` | 2000-file graph, detectors complete | < 3s | Integration timing test | budgeted |
| `arch-cart conflicts show <id>` | one conflict + source range | < 200ms | Unit + integration | budgeted |
| `arch-cart apply <domain>` (50-file domain) | file moves + import rewrites + build-green verification | < 10s | Integration timing test | budgeted |
| `arch-cart analytics events` | any scenario history | < 500ms | SQLite query | budgeted |
| UI graph render | ≤200 nodes | < 1s | Vitest perf | budgeted |
| UI graph render | ≤2000 nodes | < 5s | Vitest perf | budgeted |
| UI build | full vite production build | accepted at 5–10 min | lifecycle/test-genie build logs | inherited |
| API health | `/health` | responsive under lifecycle health timeout | `/health` check | active |
| UI health | `/health` | responsive under lifecycle health timeout | `/health` check | active |
| Audit validation concurrency | concurrent `audit.Run` / scenario-validation requests | default 1 active request | `CARTOGRAPHER_VALIDATE_CONCURRENCY`, limiter tests | active |
| Signal scoring fan-out | per-request `ScoreBatch` workers | default `min(4, CPU count)`, max 8 | `CARTOGRAPHER_SIGNAL_WORKERS`, worker-cap tests | active |
| Signal graph indexes | batch scoring over repeated chunks | package, symbol, importer, test-coupling, and domain indexes are built once per scoring context | graphindex + signal package tests | active |
| `git-co-edit` history reads | batch scoring over repeated chunks | one `git log` parse per scoring context | `gitcoedit.TestScore_ReusesBatchGitHistoryCache` | active |
| Dev profiling | local CPU/memory incident reproduction | pprof disabled by default; `/debug/pprof/*` only when `CARTOGRAPHER_PPROF_ENABLED=true` | observability tests | active |
| Latest snapshot freshness check | prior-snapshot check before extraction | metadata-only, no payload decode | `LatestSnapshotMeta` repository test with invalid payload | active |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| Baseline `arch-cart-validation-cpu-hardening` | failed existing dirty-tree baseline: standards, unit, smoke failed; structure/workflows passed | `test-genie runs wait --json architecture-cartographer 20260619-021644-986de6d5` | 2026-06-19 |
| Focused config/audit/graph/signals/conflicts tests after limiter + cache slices | pass | `go test ./internal/config ./internal/audit ./internal/graph ./internal/signals ./internal/conflicts/...` | 2026-06-19 |
| Focused config/observability/signals/conflicts tests after indexing + git batching + pprof slice | pass | `go test ./internal/config ./internal/observability ./internal/signals/... ./internal/conflicts/...` | 2026-06-19 |

Measurements are added as integration tests land. Each row records the
benchmark's source file so regressions can be traced to a specific
commit.

## Known Constraints

- **Vite production builds may take several minutes** — inherited from
  the react-vite template. Cartographer's UI bundle is modest in v1
  (graph viewer + workbench) but full builds still cross 5+ minutes
  on the first run. This is not budget-violating, just to be expected.
- **Graph extraction time is bounded by the language-graph
  scenarios**, not by cartographer itself. If `go-code-graph` or
  `typescript-code-graph` becomes slow on a particular scenario, the
  fix lives there, not here. Cartographer's timing budgets are
  end-to-end totals.
- **Community detection scales with graph density**, not just node
  count. A scenario with very dense imports (many cross-domain
  references) will be slower at the `import-cluster` signal step.
  Mitigation: community detection runs once per graph snapshot and
  results are cached; the per-chunk scoring is a cheap lookup.
- **Build-green guard adds the target scenario's build time** to any
  apply operation. For Go scenarios this is typically sub-minute; for
  large TS scenarios `tsc --noEmit` can be slow. Mitigation:
  incremental TS builds where possible, and `--skip-build-check`
  available as an opt-out (with a `--note` requirement, logged in
  analytics, same as `--force`).
- **Signal scoring is bounded parallel work** because signals are pure
  functions over an immutable snapshot. The per-request worker cap is
  `CARTOGRAPHER_SIGNAL_WORKERS` (default `min(4, CPU count)`, max 8).
  Keep the validation-concurrency cap in mind: total runnable signal
  workers can approach `CARTOGRAPHER_VALIDATE_CONCURRENCY *
  CARTOGRAPHER_SIGNAL_WORKERS`.
- **Signal indexes are per-scoring-context.** `GraphContext.Caches`
  owns package lookup, domain-package, symbol, importer, test-coupling,
  import-cluster, and git co-edit caches. Batch scoring should reuse one
  `GraphContext`; constructing a new context per chunk is a performance
  regression.
- **Git co-edit is intentionally batched.** The signal parses one git
  history snapshot per scoring context and then scores each chunk from
  the parsed cache. Reintroducing per-chunk `git log -- <path>` calls is
  a host-saturation risk under validation batches.
- **Scenario validation is host-safety-first.** The default
  `CARTOGRAPHER_VALIDATE_CONCURRENCY=1` serializes audits and
  Test Genie delegated validation requests inside the cartographer
  process. Raise it only for controlled benchmark runs or hosts with
  spare CPU.

## Regression Procedure

1. **Reproduce locally**: re-run the affected fixture-based timing
   test against the version that ships the regression and the version
   that didn't.
2. **Profile**: use `go test -cpuprofile` for API hot paths;
   `pprof-flamegraph` for visual inspection.
3. **Bisect** with `git bisect` if the regression source is unclear.
4. **Fix or accept**: if the regression is unavoidable, raise the
   budget here with an explanation and a [`DECISIONS.md`](DECISIONS.md)
   entry. If acceptable, lower the budget back via the fix.
5. **Re-record measurement** in the table above with the new value,
   source file, and date.

For UI interaction regressions, use `ui/perf/README.md` and the
provided capture template (inherited from the react-vite template).

## Performance Anti-Patterns

- **Re-extracting unchanged graphs.** Always check the source cache
  first. Production graph extraction now computes a cheap
  `source_fingerprint` and checks it before calling language graph
  adapters. Bypassing that cache without a reason is a regression in
  itself.
- **Requesting facts the consumer does not use.** Cartographer's
  production Go adapter requests the `structural` go-code-graph profile.
  The returned profile and `omitted_information` metadata are persisted
  with the snapshot, so missing resolved facts are explainable rather than
  mistaken for extraction errors. Consumers needing type resolution or test
  variants must choose `semantic` or `full` explicitly.
- **Decoding graph payloads for metadata checks.** Audit freshness uses
  `LatestSnapshotMeta`; latest-snapshot existence checks must not load
  or decode the graph JSON payload.
- **Running signals inside a transaction or with side effects.**
  Signals are pure; if a signal needs to log, it logs through the
  analytics recorder seam, not inline.
- **Detecting conflicts inside a UI request handler.** Conflicts are
  computed by the API; UI requests fetch the precomputed result.
- **Loading the full analytics event log into memory for `arch-cart
  stats`.** Stats queries must use indexed SQL aggregations, not
  in-memory filtering.
- **Calling `go build ./...` per file move.** The build-green guard
  runs once after a domain's plan is executed, not after each
  operation.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and timing-test patterns
- [`SEAMS.md`](SEAMS.md) — caches and pluggable scoring substrates
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
