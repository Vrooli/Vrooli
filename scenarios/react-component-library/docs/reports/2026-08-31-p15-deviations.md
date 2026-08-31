# Phase 15 completion comparison

The final live artifacts are `2026-08-31-p15-final-corpus-report.json` and
`2026-08-31-p15-final-matrix.json`. The full matrix completed successfully in
33.28 seconds and recorded 26,416 inspected files, 2,183 findings, and 2,680
runner errors. Findings and runner errors remain visible; they were not
removed to improve the score.

## Delivered targets

- The shape census reports one live version shape.
- Metadata duplication is zero.
- Authored-derived drift is zero after an ordinary generator check.
- The overdue retired-tree count is zero.
- Adoption depth is 3.3054%, above the 3.1% captured baseline.
- All executable gates have implementation files under the 400-line gate
  threshold, and the facts index uses one batch analyzer entry point.

## Deviations that remain measurable

- I7 (failing cells without findings) is now `0`. The corpus report invokes the
  repository-owned gate-matrix producer and counts cell-level `fail` verdicts
  whose `finding_count` is zero; the live matrix observed 230 failing cells and
  none without finding attribution.
- I8 is 34 seconds, above the 30-second target. The timing includes the
  lifecycle CLI and evidence path, not only individual gate functions.
- I9 is 12 seconds, above the 10-second target. The scoped type gate inspects
  only Button's two source files, but TypeScript startup/checker cost remains.
- I5 is 283 version directories, below the historical 290 comparison point;
  seven historical live directories were not recreated from incomplete
  evidence. I6 is 12%, within its 15% ceiling. No releases were fabricated.
- I18 passes comparatively at machine-minus-manual `15` against target `1`.
  I19 remains an honest `225` declared-without-implementation gap, with dated
  targets rather than an exemption file; the documentation gate still reports
  inherited documentation findings. The I19 backlog is filed as
  `knw-1788207660532952583`.
- The focused live catalog-schema test still reports 96 findings for retained
  legacy declarations; filed as `knw-1788207580530713531`.
- The scoped cycle remains above ten seconds; filed as
  `knw-1788207594258255614`.
- The required full scenario suite completed with 21/27 phases passing and six
  inherited provider/contract/branding/documentation/execution phases failing;
  filed as `knw-1788207603244031494`.

These deviations are recorded rather than hidden. The final matrix itself is
the durable evidence for the remaining findings and runner errors.

The authoritative cell matrix is additionally recorded at
`2026-08-31-p15-final-cell-matrix.json`: 1,701 of 5,079 cells are unmeasured
(33.5%, above the 10% target), while all 230 failing cells have findings. This
unmeasured-cell deviation is covered by the full-suite debt report
`knw-1788207603244031494`.
