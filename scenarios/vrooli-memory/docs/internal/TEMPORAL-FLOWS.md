# Temporal Flows — Vrooli Memory

## Flow Index

| Flow ID | Domain | Risk | Model Status | Source of Truth | Tests | Remaining Gaps |
|---|---|---|---|---|---|---|
| `journal-write` | journal | high: permanent append and degraded inference | Level 5 checked model, generated artifacts, and replay | `path:api/internal/journal/flow/flow.json` | `path:api/internal/journal/flow/flow_test.go`, `path:api/internal/journal/service_test.go` | None for the write flow. |
| `harness-import` | harness | high: background work, retries, interruption | Level 2 explicit persisted state | `path:api/internal/harness/runs.go` | `path:api/internal/harness/import_test.go` | Add formal transition spec once the verifier root defect is repaired. |
| `forest.compaction.pass` | forest | high: derived-tree mutation after fallible inference | Level 5 checked model, generated artifacts, and replay | `path:api/internal/forest/flow/flow.json` | `path:api/internal/forest/flow/flow_test.go`, `path:api/internal/forest/service_test.go` | Flow verifier rejects the plan's hyphenated id and its scaffold falsely reported success; tracked as `knw-1785298130564326686`. |

## Checkpoint Flows

Harness import is an asynchronous, durable flow:

`queued → running → completed | completed_with_errors | failed`

`RunImport` returns a run ID immediately. `GetImportStatus` (and the `import-status` CLI command) reports the source total, processed/imported/existing/failed counters, last checkpoint, and terminal state. Each source is persisted before the next checkpoint; callers must use the terminal state, not HTTP duration, to determine whether import finished.

Interrupted runs remain inspectable. Re-running import is safe because each source is content-addressed before inference and journal append.

## Audit Notes

- 2026-07-27: The journal contract uses two terminal outcomes: `appended` for a fully enriched write and `queued` for a persisted unclassified entry plus retry work. This avoids representing immutable append as a mutable state.
- 2026-07-28: Flow Verifier now resolves the client scenario root correctly. `flow-verifier verify run --root . --flow journal-write`, `verify check`, generated replay, and the journal persistence/retry invariant all pass.
