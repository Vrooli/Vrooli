# E1 planning and domain journey checkpoint — 2026-10-19

## Owner and serving target

Original Planner owner: run `58257550-4062-40d8-aa7d-ffa4f01d279f`, task `096c52a1-8735-4654-abd5-4c1c1e96de11`, profile `prompt-manager/delivery-orchestrator`, effort `effort:personal-planner-I18`. No replacement task or leader was created.

The supported restart completed healthy. `personal-planner@live` served API `17604` and UI `20003`, build `sha256:a72a19f92e57e849f1a6fb016c89d87769a5e33c846de8dd4b68d61d302ced23`; `/goals` returned HTTP 200. Test Genie was healthy/readiness true and eligibility returned `routed:true`. The current CLI rebuilt from source fingerprint `3b0cea02135eacaa4bf5949e2a0d9d6b7ecab67a6f7bc55114ef1021d0944064` after the test-mode transport fix below.

## Demonstrated SQLite fix

Before the live restart, `goals milestones` timed out because `ListMilestones` kept its result rows open while loading prerequisite rows on the same SQLite pool. Production config has `MaxOpenConns:1`. `api/internal/goals/sqlite.go` now scans and closes the list rows before loading prerequisites. The existing prerequisite test now pins the one-connection pool and bounds the read to two seconds. Focused test `TestSQLiteMilestonePrerequisitesBlockCompletionAndRollUpProgress` passed; the restarted live API returned the linked milestone read promptly.

## Isolation fault and retained evidence

The first lease attempt exposed an independent routing bug before the new journey could be credited. Planner CLI `goals`, `work`, and `focus` handlers use `cliapp.NewConnectHTTPClient` without adding `X-Vrooli-Test-Mode`; before this change `NewApp` configured no global header source. The API routes only requests carrying that header to the installed pool. Therefore the first attempt's CLI writes were unleased. A test-mode REST work-completion request could not find the CLI-created item, and the BAS goals observer captured “No goals yet” in its isolated view.

Retained first-attempt server-generated IDs and mutations:

- Goal `bfb9dc64-b2f8-42bb-9050-5ab68ada1360`, revision 1.
- Work item `491777b9-e155-4cff-be26-790ed2b337e7`.
- Milestone `42e44a44-7fdf-4844-88fd-be5118d001ac`, advanced from revision 1 to 2 by the CLI completion command.
- Manual actual `3d25fefa-6d8e-481d-b3fd-0ce0eba8d404`.

The first seed session was run `1479aabc-fcfe-4dd7-8ab1-69906a545a4b`; exact-token cleanup returned `status: cleaned`. That result covers the seed owner's routed lease checks; it does not account for the untagged CLI requests. Treat these first-attempt records as possible primary-store writes. No manual delete, rollback, or other primary write was attempted. This is a boundary failure, not E1 evidence.

## Narrow CLI routing repair

`cli/app.go` now adds `X-Vrooli-Test-Mode: 1` to all outgoing CLI Connect requests only when `PERSONAL_PLANNER_TEST_MODE=1`. The default remains header-free. Focused CLI tests cover both paths and passed with `go test -v -count=1 .` from `cli/`. Before the isolated journey, the new routed lease's CLI goals read returned zero goals and did not expose the first-attempt goal ID, confirming the fresh test pool was distinct from those unleased records.

## Fresh isolated journey

Seed apply run `01c465f9-d596-4eb2-8e37-712bffa4a8c6` installed the new routed lease. All Planner CLI calls used the rebuilt CLI with `PERSONAL_PLANNER_TEST_MODE=1`; the work completion REST call sent `X-Vrooli-Test-Mode: 1` to the same live API.

Server-generated records and readbacks:

- Goal `21aaaf0a-5815-44ab-985b-03a216575ad8`: created through GoalsService/CLI at revision 1; final list showed `complete`, `10000` basis points, revision 2.
- Work item `30c6a8a2-a274-46c5-8637-cd5cf4eaa0eb`: created through WorkService/CLI; supported `/api/v1/work/{id}/complete` returned HTTP 204. Connect `work get` read back the same ID and description. The current WorkItem response omits status/remaining minutes, so those are not claimed from that response.
- Milestone `c91250f9-6178-4315-b5fe-27868d753d18`, linked to that goal and work item: `ListMilestones` read it open at revision 1; after work completion, the same Connect list returned `complete` at revision 2.
- Approximate manual actual `cc284397-66fa-4db1-ba63-b282d651ebb3`, linked to that work item, 15 minutes: `focus actuals --date 2026-10-19` returned the exact ID and values.
- Forecast read returned current `no_known_work`, risk `unknown`, and explicitly explained that no remaining work with known effort was in the shared forecast input. Daily review returned 15 recorded active minutes.

Seed cleanup for run `01c465f9-d596-4eb2-8e37-712bffa4a8c6` returned `status: cleaned` with exit 0. Numeric per-case test-pool/request/root-write counters are unavailable and are not reported as zero. Raw command outputs and cleanup-token-bearing owner files are retained privately under `/home/matthalloran8/.vrooli/state/vrooli/agent-manager/runs/58257550-4062-40d8-aa7d-ffa4f01d279f/runtime/codex/receipts/planner-domain-journey-20261019/`; no cleanup token is copied here.

## Remaining original criteria

This qualifies only a routed CLI/Connect plus supported REST goal→milestone→work→actual→completion→forecast→review slice. It does not qualify browser UI parity for these same IDs, authenticated actor/workspace ownership, or the retained PP-UX/domain correctness findings. The existing CLI manifest and Planner Proto schemas expose no Project service/entity; a goal is not being relabeled as a project. Per-case LeaseStats remain unavailable. E1/Q1 remains unaccepted; the independent judgment remains NOT ACCEPTED; owner review remains NEEDS_REVIEW/APPROVAL_PENDING; F8 remains NEEDS_REWORK with historical 78-file attribution missing. R27 app-wide latency remains open; no speedup or interactive-paint conclusion is claimed.

## Root containment reconciliation — 2026-10-19

The earlier boundary note above reported the first retained untagged attempt and described four records as possible primary writes. That was an incomplete inventory, not the full incident count. Root's supported owner reads found 12 primary records across three attempts, including the exact IDs and crosslinks in [the A249 incident review](../../../../../../docs/internal/delivery-reliability-research/sources/A249logs/planner-primary-record-containment-20261019/incident-review.md). Exact per-request header attribution for the three attempts remains unavailable; preserve that uncertainty and the original failed attempts.

Root stopped Planner, captured and verified managed backup `0367b467-8e46-405b-a7ad-69eb7060bbe2`, snapshot `c9531b0d0a462a7cf868f67c9b33fe8d`, and verification `9ee25123-be60-4a11-8e06-0528b6d1a130`. A peer-reviewed fixed-target transaction deleted exactly the 12 records and three links. Its committed report states all 40 unrelated table digests and schema were unchanged and integrity/foreign-key checks passed. Independent post-repair native API/CLI owner reads found all 12 IDs absent; Planner restarted healthy. The full structured receipt is [postrepair-independent-receiving.json](../../../../../../docs/internal/delivery-reliability-research/sources/A249logs/planner-primary-record-containment-20261019/postrepair-independent-receiving.json). The incident is resolved; do not repeat the old “possible primary writes remain” disposition as current state. The original source preimage and per-request attribution were not recovered.

Root also qualified the Goals/Work/Focus CLI transport regressions for enabled/default-off test mode, identity preservation and no-leak behavior. Historical controls reproduce the old three missing headers and the SQLite two-second `ListMilestones` deadlock. The temporary archived `.go` fixture that caused unit run `20261019-191703-fe17be86` was relocated without changing evidence bytes. Current direct CLI checks passed 17 tests in 12 packages; scoped Test Genie unit run `20261019-192311-d18b38cc` passed in 33s (0 failed/skipped, 52 warnings). Do not rerun that unchanged suite. Proto is fresh. The separately owned RCL candidate preflight degradation remains unaccepted and is not evidence of a new Planner artifact.

## Single post-SQL responsiveness observation — 2026-10-19

One selected Workflow Health run `6b8979ca-becd-4ec5-9931-52aa709d369e` completed setup plus the corrected responsiveness case, both passing, under lease `6b8979ca-becd-4ec5-9931-52aa709d369e`; counters were 452 test-pool requests and zero primary requests, primary-root writes, or test-root writes. Lease install/heartbeat/clear errors were empty. It recreated the same fixture fingerprint and counts as baseline `d5b9983f-f1fd-43fc-a122-361c5bccf037`; the 390×844 viewport and corrected case hash were unchanged. Detailed samples, source/build identity, comparability limits, and raw terminal artifact references are in [the A249 R27 measurement summary](../../../../../../docs/internal/delivery-reliability-research/sources/A249logs/planner-primary-record-containment-20261019/r27-postsql-receiving/measurement-summary.json).

Admission mistakenly recorded parent run `58257550-40d8-40d8-aa7d-ffa4f01d279f` instead of this owner's actual parent `58257550-4062-40d8-aa7d-ffa4f01d279f`. The provider run is not claimed as correctly parent-linked; it was not canceled or replayed, and the original owner/task/profile remained the executor. Provider and inner case results are retained as observed. Baseline was `@shadow` build `701de714…`; the observation used `@live` build `0bc54676…`, so the builds/variants are not like-for-like. Event Timing and React Profiler entries were unavailable; route paint is not established. The mixed three-sample medians do not demonstrate an app-wide improvement or identify a repeatable optimization. R27 remains open and no optimization was selected.

This reconciliation does not replace the receipt's historical attempt data. E1/Q1 remains unaccepted; independent review remains NOT ACCEPTED; owner review remains NEEDS_REVIEW/APPROVAL_PENDING; F8 remains NEEDS_REWORK with historical 78-file attribution missing.
