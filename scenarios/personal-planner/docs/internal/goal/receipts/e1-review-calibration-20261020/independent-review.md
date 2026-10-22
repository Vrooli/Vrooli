## Parent provenance

Native reviewer task `ce7eba11-6800-457b-9cc8-c82b494e02a5`, run `b4d260df-fc9a-4152-b693-1294d6cb31cf`, profile `b93d5233-4867-4569-8371-171bea404b7e`, exact parent `5d07e651-eaa6-48ec-89b4-5dc359059193`; Agent Manager confirms COMPLETE, exit 0, `gpt-6-luna` medium, active I18 revision. Selected native final candidate `e9083a73-03ed-4161-a21e-0760a66fd97a` says the receipt is complete and returns FAIL for this bounded slice. The reviewer body below is preserved verbatim. Parent reconciled the current full-file delta against separately accepted goal-variance and trend-order snapshots; no accepted changes are reverted.

---

# Independent review — Planner I18 E1 review-calibration slice

- Reviewer run/task: independent reviewer, run/task identity not exposed in this workspace.
- Review date: 2026-10-20.
- Worker run/task reviewed: `7da64fde-e151-4f35-bf6b-5198078e0963` / `21a489ee-b480-43e3-afc6-2df33a82508c`.
- Verdict: **FAIL** for this bounded slice.

## Authority and evidence checked

Reviewed accepted authority at `docs/internal/goal/AUTHORIZATION.md`, `docs/internal/goal/GOAL.md`, and `docs/internal/goal/CONTRACT.md`; directive and full slice-log entry at `docs/internal/goal/epochs/E1.md` (`ORCH-E1-REVIEW-AGGREGATION-20261020`, including its `2026-10-20T14:50:00-04:00` entry); and these retained receipts:

- `docs/internal/goal/receipts/e1-review-calibration-20261020/baseline-original/manifest.json` — SHA-256 `bec4607ae371756c5d61009ea15d2f07b83f84be174f6ad74537aade71b00cda`.
- `docs/internal/goal/receipts/e1-review-calibration-20261020/focused-review-tests.txt` — reports `go test ./internal/review` passed, exit code 0.
- `docs/internal/goal/receipts/e1-review-calibration-20261020/epoch-check.txt` — `epoch_check_exit_code=0`.

The worker's final result was not available as a separate retained artifact in the inspected workspace; the E1 slice-log entry is the retained worker result used here. No `FEEDBACK.md` or signal-log approval was used.

## Source comparison

Compared current files byte-for-byte against the retained baseline paths under `docs/internal/goal/receipts/e1-review-calibration-20261020/baseline-original/bytes/scenarios/personal-planner/api/internal/review/`.

| File | Baseline SHA-256 (manifest) | Current SHA-256 | Lines baseline → current |
| --- | --- | --- | --- |
| `api/internal/review/service.go` | `cd5003af7aa8ae9682773267db37c5a2935bcabff46005e8d3fe9d922cae37c0` | `15b5b300263c55a6424d9765ebe1469e278a467ad6537c581b1a2421b2fd7753` | 555 → 557 (+2) |
| `api/internal/review/service_test.go` | `08719ec6905d90246a027ce6dfc408b1d39258f995d5bdfa2a60eddc021c3a59` | `28436a2ba040a3454723f493d5c23a32430a90acb02126f297c46ff9902645e3` | 305 → 338 (+33) |

The changed paths are the two admitted files. However, the measured changes do not match the E1 receipt's stated +1 runtime line and +16 test lines. `service.go` also changes `GoalVariances` completion filtering, and the test adds goal-variance assertions. That is outside this calibration directive. The recorded source/test receipt is therefore not accurate for the current files.

## Findings

The calibration SQL change itself groups eligible linked manual actual rows by allocation ID, sums minutes, and then computes the sample and ratios from one grouped row per allocation. Allocation IDs remain distinct. Its date predicate remains on individual actual rows before grouping, preserving the existing period-filter placement. The focused test output reports a pass.

The regression does not exercise the accepted 30+30 actual minutes against a 60-minute allocation. Its fixture inserts 30 and 60 minutes for `allocation-1` (planned 60), plus 30 minutes for `allocation-2` (planned 60), and expects trend values `[-50, 50]`. It does assert two samples, returned average/over/under values, and persisted projection fields/trend. Because its input does not match the accepted two-30-minute repro, it does not demonstrate the specific desired behavior (one 60-minute aggregate sample with 0% error, alongside a separate allocation yielding two samples overall, 25% average error, 0.5 over ratio and zero under ratio as directed in the slice record). The reported test pass cannot repair this fixture mismatch.

SQL grouping is compatible with the existing schema: the allocation table has unique `id`, the join links `manual_actuals.allocation_id` to that ID, and grouping also includes the planned duration. Material residual: for `last_30d`/`last_90d`, filtering rows before aggregation means only actual rows inside the requested window contribute to an allocation's sum. Whether that period-boundary behavior is intended remains unverified and is not a required correction in this slice.

## Required corrections

1. Replace the fixture/assertions so they actually cover two 30-minute manual actuals on one 60-minute allocation and a distinct allocation with the accepted expected sample size, error, ratios, trend values, returned projection, and persisted projection parity.
2. Remove the unrelated goal-variance production and test changes from this slice, or have them admitted and reviewed under the appropriate separate directive. Reconcile the baseline/line-count receipt with the resulting exact source bytes.

## Nonblocking residuals

Period-boundary semantics remain unverified. Timer actuals, categories, trend ordering, integration journey, visual comparison, and per-case LeaseStats are explicitly outside this slice and remain open. This verdict concerns only the bounded calibration slice; it does not alter the full E1 destination, visual gate, F8, or the original independent E1 criterion judgment. E1/Q1 remains unaccepted, and this receipt does not declare acceptance.
