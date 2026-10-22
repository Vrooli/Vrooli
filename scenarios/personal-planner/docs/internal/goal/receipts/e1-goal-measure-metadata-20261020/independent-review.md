# Independent review — E1 measurable-goal metadata slice

**Verdict: PASS WITH LIMITS**

This verdict applies only to the bounded slice admitted by `ORCH-E1-GOAL-MEASURE-METADATA-20261020`. Its admitted gate is met for one-time and recurring monthly USD target definitions through the available REST, CLI, and UI create/read paths. It is not E1 acceptance and does not change E1/Q1's existing NOT ACCEPTED disposition.

## Scope and identity checks

I reviewed the authority in `GOAL.md`, `CONTRACT.md`, `ACCEPTANCE.md`, `TARGETS.md`, the selected-slice section and directive in `epochs/E1.md`, and the receipt `README.md`. I inspected all 19 admitted source paths listed in the request: API goal types/service/repository/SQLite/schema and tests; goals REST/Connect handlers and handler tests; CLI goal handlers/registration/test and manifest; UI goal API/tests, GoalCard, GoalsPage and its tests.

The SHA-256 of `baseline-original/manifest.json` is exactly `16f8fd4b0fe9579819e2ffd901311f3f8085c4dfa3ea06772788d0b70e4a7d98`. I verified every captured baseline byte copy against its manifest hash and independently hashed every current admitted path against `current-identities.txt`: all 19 current entries match in hash, byte count and line count. `handlers_test.go` is the sole path recorded as absent at baseline and is now present as the admitted new test. The identity comparison is relative to the receipt's captured baseline, which correctly includes earlier dirty target-date work; it does not overwrite or reinterpret unrelated working-tree changes.

## Findings

- **Metadata and round trips:** The domain type and SQLite `goal_measures` row preserve positive target amount in USD minor units, currency code, `after_costs_before_tax` basis, recurrence/period, and target date. The one-time form is constrained to `one_time`/`once`; recurring monthly is constrained to `recurring`/`month`. The REST POST and GET representation carries these values. CLI `goals create` and `goals list` use that existing scenario REST route; UI create/read uses the same representation. The UI tests cover both examples and render their target meaning.
- **Validation and compatibility:** Service validation rejects missing/nonpositive amount, unsupported currency/unit/basis, missing target date, and contradictory or unsupported recurrence/period combinations. Generic goals without measure remain on the existing path; their manual progress is still returned and updated. No contrary behavior was found in the inspected code or tests.
- **Unknown progress and updates:** Measured-goal REST responses set `progress_state: unknown` and omit `progress_basis_points`. The UI renders an explicit unknown state and omits the generic progress slider for measured goals; CLI output says `progress unknown`. SQLite rejects manual progress writes for measured goals, including the Connect-backed update service path. Its unmeasured progress remains available. The UI transport mapper internally defaults an absent `progress_basis_points` to zero, but carries `progressState: unknown`, and measured cards branch on the measure and do not render that zero as progress. This is an implementation detail worth preserving in future consumers.
- **Persistence safety:** Goal and optional measure rows are inserted through one SQLite transaction with deferred rollback and commit after both writes. A failed measure insert therefore cannot leave a goal-only row. The reviewed tests exercise persisted measure readback, measured progress rejection, and generic progress update. This establishes atomicity for this write path; it is not a broader storage/backup qualification.
- **Boundary:** Changes remain in the admitted 19 paths. The inspected CLI evidence receipt records a refreshed manifest and only the `goals create`/`goals list` primitive identity changes; its freshness/consistency check passed. The feature adds no income observations, provider ingestion, forecast behavior or Proto/generated contract changes. Existing Connect `GoalsService` messages cannot carry or return measure metadata under the explicit no-Proto boundary. Therefore measured-goal metadata is not claimed on Connect create/read; other Connect goal behavior remains available. The bounded REST/CLI/UI outcome must not be generalized to full E1 surface parity.
- **Tests:** Added tests are relevant: service validation tests cover valid forms and invalid/incomplete/contradictory metadata; SQLite tests cover durable round trip and progress restrictions; handler tests cover REST create/read and rejection; CLI uses a deterministic local HTTP server to test create/read/rendering; UI API/page tests cover request and rendered round trips. These cover distinct layers and are not materially duplicative. No test run was repeated for this review; retained outputs were inspected as requested.

No additional product-code defect was identified within the admitted behavior. Two evidence limitations are material and recorded in the verdict: (1) Connect cannot expose the new metadata without the explicitly excluded Proto change; (2) the README reports UI `18 tests`, but `ui-goals-tests.log` records **19 tests** (6 API + 13 GoalsPage). The retained log is the direct command output; the summary count should be corrected in any later receipt amendment. This mismatch does not negate the observed passing test output.

## Retained command and growth evidence

The retained outputs independently inspected were:

- `api-domain-handler.log`: both `go test ./internal/goals ./handlers/goals -count=1` packages report `ok`.
- `cli-goals.log`: `go test ./domains/goals -count=1` reports `ok`.
- `ui-goals-tests.log`: Vitest reports 2 files and 19 tests passed.
- `ui-typecheck.log`: empty output, consistent with the recorded successful `tsc --noEmit` exit.
- `cli-manifest-check.log`: manifest valid JSON.
- `cli-primitive-evidence.log`: CLI package freshness/consistency check reports `ok`.
- `scoped-diff-check.log` and `cli-evidence-diff-check.log`: empty outputs, consistent with successful scoped whitespace checks; `untracked-test-whitespace-check.log` says no trailing whitespace in the new handler test.
- `epoch-check-status.txt` and `epoch-check.txt`: check exited 0; cumulative runtime growth is +1,706 versus +2,100 ceiling, with no growth step-back. Test growth is +2,145, explicitly disclosed as 45 above the 2,100 expected-size comparison; the check did not emit a test-growth step-back. The receipt's admitted-path inventory independently totals +288 runtime lines and +332 test lines. Runtime slice growth is 108 above its +180 estimate, justified in the admission/receipt by using the existing REST representation because Connect metadata would require excluded Proto work. Test growth is also above the +140 estimate and is itemized by the current identity deltas; focused tests cover validation, persistence, route transport and UI behavior. Remaining cumulative runtime headroom is 394 lines.

## Gate and unresolved limits

The bounded gate is met: both accepted currency target forms preserve amount, unit/currency, basis, recurrence/period and target date through the admitted REST/CLI/UI create/read routes, invalid metadata is rejected, generic goals retain their prior progress behavior, and measured progress remains explicitly unknown. The slice does not prove Connect measure readback, actual-income observation, evidence-derived progress, broader UI visual/accessibility criteria, the integrated planning journey, storage/restore qualification, or any other full E1 exit gate. E1/Q1 remains NOT ACCEPTED as before.
