# Progress — React Component Library

Dated observations and design decisions below are historical evidence, not current readiness claims. Unchecked recommendations are **design intent**. See the [behavior claim register](TESTING.md#behavior-claim-register).

This log keeps recent dated milestones. It does not certify current readiness.
Read [PROBLEMS.md](PROBLEMS.md) for tracked gaps and the owning Plan Manager
record for live execution status.

## Earlier history

The complete pre-cleanup log, including intermediate results and unresolved
handoffs, is preserved byte-for-byte at the protected runtime-home location:

```text
<runtime-home>/plan-artifacts/docs-history-20260907/scenarios/react-component-library/docs/internal/PROGRESS.md
```

SHA-256: `dbdb6a6442dd2ab221fbef10bc243eca4fcb630f7b5e6d5b3f9413d5c6d947a8`. Restore or read it before relying on an
older completion claim; archival does not close any outstanding work. The
[project preservation record](../../../../docs/internal/PROGRESS.md#documentation-cleanup--2026-09-07)
records ownership and recovery.

## Recent milestones

| 2026-08-31 | codex | partial | Completed the two-surface derivation migration: the live corpus has one canonical version shape across 281 versions, catalog generation/typecheck/export-boundary tests pass, release hash and immutability measurements are zero after the explicit canonical migration refresh, and I21 adoption depth is now 3.3057% (above the 3.1% baseline) across three building pilot consumers. Moved all package tooling to `packages/react-component-library/tooling/`, removed obsolete ledger repair targets and stale probes, added retired-tier storage ownership plus overdue-quarantine invariant I26, moved gate performance validation to persisted evidence, and removed scenario scratch output. Focused Go suites and lifecycle restart pass; full Test Genie, gate-cost timing measurements, and the inherited documentation/population gaps remain open and are not represented as complete.
| 2026-08-31 | codex | partial | Completed the governed retention reap after migrating 114 linked adoption records from stale exact-version metadata to major-line paths. The independent reachability probe contained all 35 Go cleanup candidates; the exact plan hash retired 35 versions atomically. Current corpus-report is I1=3, I2=0, I3=0, I4=0, I5=285, I6=10%, I7=0, I10=0, I11=0, I12=0, I18 machine-minus-manual=15, and I20=1. Cleanup has 0 eligible rows, `versions doctor` is empty, reconciliation is idempotent at 1,098 unchanged, and ledger audit is 746 matching / 0 mutated / 0 missing. `offer-desk`, `web-console`, and `git-control-tower` all pass `make build` after fixing idempotent selector composition. Added the read-only `components republish-plan --json` command and its CLI tests. Full Test Genie/readiness and repository-wide UI lint remain open inherited validation debt. |
| 2026-08-31 | codex | partial | Revalidated after rebuild/setup: the filesystem-only dependency-lock validator now accepts provenance-backed cold releases while still rejecting fabricated targets, with regression coverage. `make lint-go`, `make build`, and lifecycle restart pass. Setup re-warmed five retired versions; governed reconciliation evicted those five and the following preview was idempotent (0 evictions / 0 rematerializations / 1,100 unchanged). Fresh matrix evidence is 4,158 cells: 2,480 pass, 200 attributable failures, 1,478 unmeasured with runner messages, and 0 finding-less failures. The current corpus remains I5=320 and I6=23%; readiness remains not ready because the declared population/evidence run is incomplete. |

## New entries

Append a dated milestone with its outcome, remaining constraint, and durable
owner reference. Store detailed command output with the producing owner.
