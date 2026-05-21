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
| `arch-cart graph extract` (cached, unchanged) | any size | < 500ms | Cache hit returns the persisted snapshot | budgeted |
| Signal scoring (full ladder) | 2000-file graph, all candidate domains | < 2s | Unit benchmark | budgeted |
| `arch-cart conflicts list` | 2000-file graph, detectors complete | < 3s | Integration timing test | budgeted |
| `arch-cart conflict show <id>` | one conflict + source range | < 200ms | Unit + integration | budgeted |
| `arch-cart apply <domain>` (50-file domain) | file moves + import rewrites + build-green verification | < 10s | Integration timing test | budgeted |
| `arch-cart history` | any scenario history | < 500ms | SQLite query | budgeted |
| UI graph render | ≤200 nodes | < 1s | Vitest perf | budgeted |
| UI graph render | ≤2000 nodes | < 5s | Vitest perf | budgeted |
| UI build | full vite production build | accepted at 5–10 min | lifecycle/test-genie build logs | inherited |
| API health | `/health` | responsive under lifecycle health timeout | `/health` check | active |
| UI health | `/health` | responsive under lifecycle health timeout | `/health` check | active |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured yet — implementation pre-flight. | n/a | n/a | 2026-05-21 |

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
- **Signal scoring is parallelizable** because signals are pure
  functions over an immutable snapshot. Reach for goroutine
  parallelism only if benchmarks show single-threaded scoring exceeds
  budget; default is sequential for simplicity.

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

- **Re-extracting unchanged graphs.** Always check the content-hash
  cache first. Bypassing the cache without a reason is a regression
  in itself.
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
