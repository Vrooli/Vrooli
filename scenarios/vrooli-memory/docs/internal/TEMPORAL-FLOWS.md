# Temporal Flows — Vrooli Memory

## Flow Index

| Flow ID | Domain | Risk | Model Status | Source of Truth | Tests | Remaining Gaps |
|---|---|---|---|---|---|---|
| `journal-write` | journal | high: permanent append and degraded inference | Level 5 artifacts/replay generated | `path:api/internal/journal/flow/flow.json` | `path:api/internal/journal/flow/flow_test.go` | Flow-verifier check has an absolute-root lint propagation defect; generated replay and Go test pass. |
| `harness-import` | harness | high: background work, retries, interruption | Level 2 explicit persisted state | `path:api/internal/harness/runs.go` | `path:api/internal/harness/import_test.go` | Add formal transition spec once the verifier root defect is repaired. |

## Checkpoint Flows

Harness import is an asynchronous, durable flow:

`queued → running → completed | completed_with_errors | failed`

`RunImport` returns a run ID immediately. `GetImportStatus` (and the `import-status` CLI command) reports the source total, processed/imported/existing/failed counters, last checkpoint, and terminal state. Each source is persisted before the next checkpoint; callers must use the terminal state, not HTTP duration, to determine whether import finished.

Interrupted runs remain inspectable. Re-running import is safe because each source is content-addressed before inference and journal append.

## Audit Notes

- 2026-07-27: The journal contract uses two terminal outcomes: `appended` for a fully enriched write and `queued` for a persisted unclassified entry plus retry work. This avoids representing immutable append as a mutable state.
- 2026-07-27: A relative flow-verifier root scans the verifier project rather than this scenario; absolute-root generation succeeds, but check-time lint then drops the discovered test directory. Evidence is recorded in the Plan Manager ledger.
