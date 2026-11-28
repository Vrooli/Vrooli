# Git Control Tower rendering and subscription work

## Changes

- Moved approved-change and provenance polling into `FileList`, which owns the
  run attribution derived from those queries. The root `App` no longer
  subscribes to those updates or rebuilds the run index on every refresh.
- Moved the health query to the active status header. Health polling no longer
  invalidates the root workspace render; the `app-interactive` mark is emitted
  by the health hook when the first result arrives.
- Changed `VirtualizedFileSections` to store one compact range per section or
  custom header and resolve rows on demand. The DOM remains virtualized, and
  the backing model no longer allocates a wrapper object for every file.
- Kept group drill-in stageability and agent-run file selection on indexed
  membership paths, and retained the exact/prefix history lookup index from
  the preceding pass.
- Moved branch-list polling out of the root `App` subscription and into the
  visible header. Health and branch results now survive quick responsive
  remounts, and health reads consume the query cancellation signal.
- Changed bulk diff invalidation to scan the repo-scoped diff cache once and
  match changed paths, instead of doing three cache scans per staged path.
- Batched proposal base-tree lookup and canonical blob reads with bounded
  `cat-file --batch` streams. Per-file clean-filter hashing remains separate;
  literal paths and the existing 8 MiB per-file / 64 MiB capture limits remain
  in force.

## Evidence

- `pnpm type-check`, `pnpm build`, and API `go build ./...` passed. No tests
  were run.
- `make restart` completed healthy; API and UI health reported the same build
  identity (`sha256:d2acf4c43f66fe600b31912069efca39253602164931d936e5f0a37e5cd08323`).
- Current production UI entry: 1,390.49 KB (385.74 KB gzip). Vite still warns
  that the main chunk exceeds 500 KB.
- `performance-health audit run git-control-tower --workflow
  perf-grouped-changes` returned `AUDIT_OUTCOME_FAILED`: Browser Automation
  Studio requires an authenticated account. No before/after browser trace was
  available, so this change is not claimed as a measured trace improvement.

## Follow-up

Re-run the grouped Changes capture with an authenticated BAS account. Compare
FileList average and maximum commit time, dropped frames, Layout, and
UpdateLayoutTree against the 2026-09-07 audit before tightening budgets.
