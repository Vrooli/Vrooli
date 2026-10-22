# Native provenance and identity reconciliation

- Reviewer task: `2b4879be-5801-465c-8fad-218937574b99`; native `created_by` is the exact parent run `5d07e651-eaa6-48ec-89b4-5dc359059193`.
- Reviewer run: `b3b4692e-54e0-4395-80d9-ee56806dcea7`; native `parent_run_id` matches that parent; task, profile `b93d5233-4867-4569-8371-171bea404b7e`, COMPLETE status, exit 0, and APPROVED run-end state verified from Agent Manager.
- Selected native final candidate: `67f5c3ad-cf7b-4f5c-9468-a2faaaf54ac8`. The selected final below is preserved verbatim; SHA-256 before adding this provenance header: `5b3f08e572b4660868e49e4a9c90bf9a75909735010a0d48692399b1a2cae340`.
- The submitted Agent Manager task description contained an orchestrator-authored parent UUID typo. Native `task.created_by` and `run.parent_run_id` establish the correct lineage above; the historical submitted task description is retained unchanged. The local reviewer brief is accurate, and current goal-home handoffs use the verified parent ID.
- The reviewer read E1 hash `19f333c3faeea0e7dee0d8de7c0eb52f21cd5eb90869245e3a0f9fe9b8fc5140`. Subsequent orchestrator edits corrected handoff identity metadata and updated the slice status; they did not change calibration requirements or product evidence.

---

# Independent review verdict — calibration split-actual correction

**Verdict: PASS WITH LIMITS**

## Evidence identities and hashes

Reviewed the approved requirement at `docs/internal/goal/CONTRACT.md` item 2; the calibration slice and correction section in `docs/internal/goal/epochs/E1.md`; prior review `receipts/e1-review-calibration-20261020/independent-review.md`; worker brief `correction-brief.md`; correction baseline manifest and captured bytes; `correction-handoff.txt`; `focused-correction-test.txt`; and `epoch-check-correction.txt`.

Corrective worker identity recorded in the brief/handoff: task `530a6516-e90c-4d79-8e3f-da50c937d4d7`, run `5fbba55b-2582-4292-9bda-feee12d19052`, exact parent `5d07e651-eaa6-48ec-89b4-5dc359059193`, profile `4c19aef1-36a2-4cc5-8961-42648692883b`.

Hashes checked (SHA-256):

- `CONTRACT.md`: `8ea48135bcbfd758c0ca08f4235ece3b32b6869b307bd2fd7214a96a46627783`
- `E1.md`: `19f333c3faeea0e7dee0d8de7c0eb52f21cd5eb90869245e3a0f9fe9b8fc5140`
- Prior `independent-review.md`: `81a924e76fd04f5ea6ea99d5408ae1df14678f4d3127e4db583434a2d71492ac`
- `correction-brief.md`: `dd494a458a12703aa627a289cfe3fd5b1054c0c6b7f141f68c36c8ce6d030685`
- `baseline-correction/manifest.json`: `98d0351c210de1bec7ef8f55d08666d43b502596ea0c40bef0f527ed07d22846` (matches the expected manifest identity)
- `correction-handoff.txt`: `e876e8277bb62580fca5cb71085bbb728c7012f7eddd9a5eef7eea6c9ec990a0`
- `focused-correction-test.txt`: `17d6fb8bca67809ce80fe2bdaad8ce4d6e0e46c730429d61f26e6233ec836284`
- `epoch-check-correction.txt`: `c0975307a444733fd85ea4bde14d958003be25938307cf94f47b69e646e164c8`
- Current `api/internal/review/service.go`: `15b5b300263c55a6424d9765ebe1469e278a467ad6537c581b1a2421b2fd7753`, identical to the captured protected baseline bytes.
- Captured baseline `service_test.go`: `28436a2ba040a3454723f493d5c23a32430a90acb02126f297c46ff9902645e3`; current file: `1f4abbdc311329cfafa9386f2d035bf6bc4aa55d361569aa5dca37623fb04e40`.

## Finding

The fixture now has two 30-minute rows on distinct dates for allocation A (60 planned) and one 90-minute row for allocation B (60 planned). The assertions require two allocation samples, average error 25%, over ratio 0.5 and under ratio 0. The data makes B's +50% observation latest-contributing-date earlier than A's 0% observation, and the test checks response trend `[50, 0]`.

The same test reads the persisted `estimation_bias` row and compares sample count, average, ratios and timestamp with the response; it checks the empty category projection (`{}` and no response categories), decodes the persisted trend, and asserts the same `[50, 0]` values. These fixture and assertions directly establish the accepted split-actual regression behavior and response/persisted parity for this case.

The correction diff against the exact captured test baseline contains only the three intended fixture/expectation edits. `service.go` is byte-identical to its protected baseline. Goal-variance coverage remains in the current test file. The baseline itself already includes the separately accepted goal-variance and trend-order work; those pre-existing hunks are not attributed to this correction. The new fixture also continues to exercise the previously accepted trend ordering. No unrelated dirty bytes were edited by this review.

## Validation limits and disposition

The retained worker output reports `go test ./internal/review -count=1` passing (`ok personal-planner/internal/review 0.012s`). This is worker-run evidence; I did not rerun tests. The retained epoch-check reports exit 0 and no step-back, with the existing nonblocking legacy-gates warning. Neither is a substitute for full E1 acceptance.

This disposition accepts only the corrected split-actual regression slice. Equal latest-date tie-breaking, period-boundary semantics, timer actual eligibility, broader category coverage, full calibration/learning behavior, integrated journey, visual comparison and per-case LeaseStats remain outside this review. E1/Q1 and the full calibration criterion remain unaccepted. Recommendation: accept this bounded correction as PASS WITH LIMITS, subject to the parent's separate lineage/provenance check and mechanical acceptance gate.
