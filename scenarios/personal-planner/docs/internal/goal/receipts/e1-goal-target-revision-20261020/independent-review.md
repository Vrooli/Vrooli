# Parent provenance

Materialized by orchestrator from the selected native final output after same-run continuation. Reviewer task `80e28b63-d186-4ce8-8490-d50bffec9f88`; run `dd13d0e7-3ad0-41f6-96df-6badefe8c60a`; profile `b93d5233-4867-4569-8371-171bea404b7e`; exact parent `5d07e651-eaa6-48ec-89b4-5dc359059193`; selected final candidate `6eeb8722-1626-4953-89bb-7ebe6a37ec7c` (sequence 63). Native reviewer body says it did not independently read a terminal worker status; orchestrator separately verified worker task `68f3b53b-3ac5-415c-bc98-de29ef0d2baf`, run `9ee3e864-21c2-4698-b39f-1f75458783df`, exact parent `5d07e651-eaa6-48ec-89b4-5dc359059193`, status COMPLETE.

---

# Independent review — ORCH-E1-GOAL-TARGET-REVISION-20261020

**Verdict: PASS WITH LIMITS**

This is the complete independent review for the bounded target-date revision guard. It is not acceptance of target-date history, the broader product journey, or E1/Q1.

## Verdict by review question

1. **Stale revisions reject without changing target date/revision; current revisions succeed and advance revision — PASS.** The SQLite implementation checks the stored revision inside a transaction and rejects a mismatch before writing. Its conditional update requires the expected revision; successful updates advance the revision and commit with the history insert. The retained SQLite test checks that a stale update leaves the date and revision at `2026-11-01` and `3`, then that current-revision updates succeed and advance revisions to `4` and `5`.

2. **UI sends the loaded revision and refreshes the goals query after save — PASS.** `GoalCard` submits `goal.revision` as `expectedRevision`. On success it invalidates the `["goals"]` query, as well as the related target-date history, drift, and variance queries. The retained UI API test checks expected-revision serialization.

3. **Admitted paths and measured deltas supported; later history changes distinguished — PASS WITH LIMITS.** The target-revision baseline manifest admits ten paths. All ten hashes in the later target-history pre-history snapshot match the target-revision post-slice identities below. Live files have since changed; those changes are recorded separately below and are not attributed to this revision guard. The worker reports **+16 runtime lines / +48 test lines**. The snapshots support the scoped path boundary; the retained files do not independently expose a line-count calculation, so those totals are treated as worker-reported measurements.

4. **Retained focused Go/UI results supported — PASS WITH LIMITS.** The retained Go log reports the goals package passing and the handler package compiling with no test files. The retained Vitest log reports 2 files and 15 tests passing. No tests were rerun for this review. **The target-date HTTP 409 mapping has no route-level test in the pre-history snapshot; it was inspected and compiled, but no HTTP request exercised the mapping.**

5. **Corrections, limits, and acceptance scope — PASS WITH LIMITS.** No required correction was identified for the bounded guard. The HTTP 409 route-test gap is a material limit. This review makes no target-date-history, broader journey, or E1/Q1 acceptance claim and infers no approval from other signals.

## Source identities and scope

Target-revision baseline manifest SHA-256: `aa1126e53013c2222f96ada095a666b42a154ea2e4f57fe3980d1ef47b298367`.

The table gives each admitted source’s **baseline → post-revision/pre-history → live** SHA-256:

| Admitted path | Baseline | Post-revision/pre-history | Live |
|---|---|---|---|
| `scenarios/personal-planner/api/internal/goals/service.go` | `25be6f2c4cd0fd9c77d6e6b066e3ffc2483c2643669ae9e1580717da24d8f224` | `d4c95740b49149326a2638dd4456abb74a790e53cc55994a8511c55a60cfc8e7` | `f739e8ffac486d7c8bc50aff3775cc31232e47ae254bb0f6913888370a2c0d9d` |
| `scenarios/personal-planner/api/internal/goals/repository.go` | `5af44ecf1ea66ad4ab06fc0b9672f8d3ec62bb7bcb73f051fb50a32fe40835a6` | `52fea020664a100552cdd6a5f2f3ac5529183a91191ce74afe4c591f1a147310` | `df3b04b266ec0a34f59eee00056be2f7aa12cb776ff0454985b038208d354ba2` |
| `scenarios/personal-planner/api/internal/goals/sqlite.go` | `706ab920341ad070062c1bf011995696408ba8c43dccff6e1e9ac93df767d03f` | `c56787d3626619e086bceb49b2e07c6b0c96958cecb16dd1ff8dd21032a9ba18` | `c02b7d0b7e69258080fcb0b91fcd2441f732a081d011316a534f30240074d56e` |
| `scenarios/personal-planner/api/internal/goals/service_test.go` | `acd3c8ae9a8375c2ba14da544c2f16bc648615a97dde4e2e362f846f049dcc55` | `ecbb364e79d475e7b9267a2df02f9191f900d6ce9c301d66d74fcdd3ac4146a8` | `5de324c1c332e554da02ff2d08a3ff902b482ae28f9001e3d65601ccda687a4d` |
| `scenarios/personal-planner/api/internal/goals/sqlite_test.go` | `f7aceda1d8e619a57da75f126a4efa84009fc0b11ebc5e612c74c8576a171c2f` | `f8b743850e17f76c5add915b170fafa48b26c140595d0407d5ae0839910e5ea1` | `941b1db91fd0cfeb2cf37b203781df3ce9cc0ec64725ce9080ac76daf4000eb2` |
| `scenarios/personal-planner/api/handlers/goals/module.go` | `4c47c0542cb06bc0369c35e3dd5b186b4cff90083cb8cf74275a8b729297d782` | `f2a7a3f0f5a1393422a9effa2f3a9623c0e6b0991a4f19f04ee7e118647161a7` | `0b05cd8fbf6e7e447429eec034ac7cf26213baa9bbc432b82ee239b46934188e` |
| `scenarios/personal-planner/ui/src/api/goals.ts` | `d008b56e6dfca108e4a17bb8618816b2a358393c7d5ee348efff66daec45adf2` | `b869f363a6acdebebd0ddfeac50a06df0788e22a9ca629f8f66e9720c3bc0da1` | `39ad60fc43e7e9ee8a0566c5190adcad3268b3a3bfb9238c0631b791a97e374e` |
| `scenarios/personal-planner/ui/src/api/goals.test.ts` | `a8558178a4265e4b1fd3b2f5372c37967043e5aab2e994d125fcb36f07cca4f0` | `745bbea5e3e1968f157cce5d35fa7dbbca5c63ce00c0fb387abbc4faeee8b248` | `a423117df50aa16291c7ebe94c2098b74493947a39863591b9a13bb2e922f625` |
| `scenarios/personal-planner/ui/src/components/GoalCard.tsx` | `7115b3a9f45439ec8272e467da1d520aff2f2f6c84398eb4801369ffd813bdf8` | `8dbc64f8482853679b7ddcd13b9f94713454c801fc17d58657fa53a474bd490c` | `0e5ac5e730b6838f23431f65e0830a7408c711f866e991a98a1f5c3353aeaeef` |
| `scenarios/personal-planner/ui/src/pages/GoalsPage.test.tsx` | `1b7725541ed03b56e52e2c8853e3aab76402f399017cc3cb463971162737e0a9` | `276bd4cc0abfec838e126881ee77d8518260c67f5b3e76b569ec1a752bbe63cc` | `03693f7dcad9c748d0e30f1db9ce275621b8376b06415844362d90462091c573` |

The later target-history snapshot manifest SHA-256 is `2b156c07de78866d2d61d393dd23913028af0f96d60726948b7d06b30542af17`. Its ten common-path hashes match the post-revision/pre-history column exactly. That later manifest also includes `api/internal/goals/schema.sql`, `api/internal/goals/types.go`, and `api/handlers/goals/module_test.go` (the route-test file was absent in that snapshot). Live identities differ from the later snapshot on all ten common paths, consistent with subsequent edits. These later changes are not attributed to the revision guard, and their absence is not required for this review.

## Retained test evidence

- `docs/internal/goal/receipts/e1-goal-target-revision-20261020/go-test.log` — SHA-256 `099445dc268719290aed691131eeb76c40dee96a3dca0d52e80dde8052560589`. Output: `ok personal-planner/internal/goals`; `? personal-planner/handlers/goals [no test files]`.
- `docs/internal/goal/receipts/e1-goal-target-revision-20261020/ui-vitest.log` — SHA-256 `8da553e6965eb06b53114746cc5d46cce580290234376971e8d743e35250b76b`. Output: 2 test files passed, 15 tests passed (4 API, 11 GoalsPage).
- `docs/internal/goal/receipts/e1-goal-target-revision-20261020/epoch-check.txt` — SHA-256 `f154ae17840fd47d1c04543b90b48f1358a2e1311b8d2ec5f40e2d70b0852322`. Its final recorded exit metric is “bounded goal target-date optimistic concurrency and current UI revision propagation verified”; open directives: none. The check reports no flatline, overrun, or growth trigger; its legacy-gates warning says acceptance is not checked against that gate.

## Provenance

- **Reviewer task:** `80e28b63-d186-4ce8-8490-d50bffec9f88`
- **Reviewer run:** `dd13d0e7-3ad0-41f6-96df-6badefe8c60a`
- **Exact parent:** `5d07e651-eaa6-48ec-89b4-5dc359059193`
- **Worker task:** `68f3b53b-3ac5-415c-bc98-de29ef0d2baf`
- **Worker run:** `9ee3e864-21c2-4698-b39f-1f75458783df`
- **Worker parent:** `5d07e651-eaa6-48ec-89b4-5dc359059193`
- **Worker status evidence:** the retained epoch check records the bounded exit metric and no open directives; retained Go/UI logs contain the stated focused results. A separate native terminal-status readback was not among the inspected artifacts, so I do not assert a native terminal status beyond those retained results.

No product files were edited and no tests were rerun.
