# Infrastructure workarounds

## 2026-10-17 — Web Console CLI had stale saved API base

- Tool/owner: Web Console CLI session read for the dispatched D6 receiver.
- Attempt: `web-console session get` resolved the saved API base `http://127.0.0.1:1` and failed to connect; its diagnostic identified the live API at `http://localhost:16382`.
- Fallback: made one read-only session lookup with `web-console --api-base http://localhost:16382 session get <assigned-session-id> --json`; it returned the expected persistent programmatic `rcl/E1` receiver owned by Agent Manager. Do not retry using the stale saved base; use the detected owner endpoint for this read if needed.
- Affected evidence: no D6 browser action or E1 gate; this qualified the assigned interactive receiver only.
- Cleanup: none; no config or service changes.

## 2026-10-16 — Search Hub combined-read helper unavailable

- Tool/owner: `program-runtime.prepare-operation` returned `/bin/bash: program-runtime.prepare-operation: command not found` for the `browser-automation-studio.capture-surface` discovery result.
- Fallback: stopped retrying the helper; continued only with read-only control-plane scenario status, running-process environment, source contract and health GETs. No BAS capture operation was invoked.
- Affected evidence: none of the RCL gates; this missing helper did not prevent current read-only BAS/RCL qualification.
- Cleanup: none.

## 2026-10-01 — Test Genie unit phase failures at baseline

- Tool/owner: Test Genie scenario unit phases for `react-component-library` and `prompt-manager`.
- Attempt: one server-owned run per scenario; RCL run `20261001-142109-f9c0e41d`, Prompt Manager run `20261001-142109-e90f1ef9`. Both reported `TEST_EXECUTION_FAILURE`; RCL also reported `UNIT_POLICY_PROJECTION_DRIFT`. These failures preceded E1 product edits.
- Fallback: do not retry the same Test Genie phase during E1. Use focused package/module checks for the files E1 actually changes and record their command/results in the epoch log.
- Affected evidence: Test Genie unit phase verdicts remain unverified; the worker separately recorded passing RCL package and Prompt Manager UI builds before edits.
- Recurrence: after D2 explicitly prohibited retries, the worker logged post-edit Test Genie runs RCL `20261001-143431-8091e80e` and Prompt Manager `20261001-143432-b2908144`. The latest epoch handoff records both as failed with the same API execution/projection findings as baseline. This repeated attempt did not produce a passing unit verdict; do not start another run.
- Cleanup: none; no tools installed or processes started by this workaround.

## 2026-10-01 — RCL component story report endpoint unavailable

- Tool/owner: RCL component-test story evidence through BAS CaptureService.
- Attempt: one component-test report, `ctr_b5819efe7cd32400`, failed because the configured CaptureService endpoint `127.0.0.1:17116` refused the connection.
- Fallback: normal-entry BAS desktop/mobile journey executions and four isolated story-sheet captures completed and were visually inspected. These prove the retained browser steps/captures only; they do not substitute for the failed component-test report.
- Affected evidence: durable catalog story-run evidence remains unverified. E1's other journey and build gates retain their separate evidence.
- Recurrence: first observation in this revision.
- Cleanup: no external tool installed; no test-only service started by this fallback.

## 2026-10-05 — RCL consumer CaptureService route still unavailable

- Tool/owner: Browser Automation Studio CaptureService route consumed by RCL `componenttests.NewBASCaptureExecutor`.
- Attempt: BAS shadow status was healthy at API port 15372 with build identity `sha256:e16bbd1f5ed9d79d12ba591912b24b08962bc4c4fca8512238948b7fa304cbd`; RCL source then used a guessed `17116` fallback. The worker reported a GET to that historical address refused. The X-Dry-Run POST was denied by outer review before execution; no POST occurred.
- Fallback: stop D6 before keyboard navigation; no flow was run or saved workflow changed. Do not retry the denied POST, use another capture route to bypass review, or start/restart/deploy BAS from the RCL owner.
- Affected evidence: E1 D6 keyboard movement/activation and visible-focus proof remains unverified; the existing component-story report remains separately unverified. Shadow health and old screenshots do not qualify the currently selected RCL consumer or prove keyboard behavior.
- Resume: see the superseding 2026-10-06 owner handoff below; the old `17116` fallback is not a current route.
- Correction: 2026-10-06 feedback `RCL-CAPTURE-DISCOVERY-20261005` establishes that the POST was denied before execution because deployed dry-run semantics were not established. The former “POST failed before HTTP response” wording was not a completed request result. `SUP-OVERNIGHT-20261006-RCL` supersedes the fixed-port return condition.
- Cleanup: no service/process started and no external tool installed.

# 2026-10-18 — discovery CLI host-memory admission

- Tool/owner: `search-hub query` recall/discovery, with one required `prompt-manager discover` fallback.
- Attempt: Search Hub's CLI health/provider circuit was degraded; the single fallback was denied by CLI build admission because host memory PSI `some.avg10` was `13.67%`, above the `10.00%` substrate/SB20 setpoint.
- Fallback: stop retrying discovery CLIs during this pass. For the requested existing component report, used the source-confirmed read-only RCL Connect Get/List endpoints; both returned HTTP 200 and exact report ID `ctr_650bcdb38eed362b_09a1adc97fa9c48dfbdfcbc9`. This resolves report discovery only, not the report's failed verdict.
- Effect: no product, runtime, or catalog state changed through discovery. The owning report remains failed on CaptureService screenshots/readiness. Resume discovery CLI use after host-capacity admission returns to setpoint.

## 2026-10-01 — E1 Agent Manager child run stopped advancing

- Tool/owner: Agent Manager interactive worker run `ac2aa69a-6efc-49ff-969b-422f35593893`, task `08351232-917c-499c-9226-06ae9204b0fd`, workload `rcl/E1`.
- Observation: repeated child-producer timer wakes found the run still `RUN_STATUS_RUNNING`, but its last heartbeat remained `2026-10-01T14:14:23Z`, last update `15:02:54Z`, and transcript had no new worker message after `15:02:10Z`. The bounded D4 closure directive was typed into this exact interactive session; the next reconciliation still showed no new activity.
- Fallback: after repeated timer wakes and no new activity following the bounded D4 message, stopped the stale child through Agent Manager and verified it as `RUN_STATUS_CANCELLED`. Dispatched one successor from E1 using the same worker profile/model, workload key, and parent run: `65f94472-38ba-4116-b356-6599d49e707a`, task `6059e5a4-06b3-435c-a4d2-b13d8a1d233f`; it was `RUN_STATUS_PENDING` at admission. Do not create another successor unless this run reaches a terminal state and the epoch's recovery rules still allow it.
- Affected evidence: E1 remains unaccepted. Focused Heading integration check and explicit normal-entry-to-governed-adoption handoff are still outstanding; independent §4.4 acceptance has not run.
- Recurrence: repeated across successive timer wakes after D4. The supervision heartbeat trigger returned `deduplicated`; a direct cross-team inbox message was rejected because this agent is not a member of `effort-supervision`, so no direct message was delivered. Current owner state is recorded in QUEUE.md; the supervisor heartbeat may read this goal home.
- Cleanup: none.

## 2026-10-06 — Requalify repaired RCL-to-BAS consumer after owner recovery

- Tool/owner: BAS live runtime owner supplies the selected native BAS instance; RCL `NewBASCaptureExecutor` resolves it through api-core/discovery, while RCL lifecycle owner serves the repaired source.
- Observation: read-only control-plane status reports BAS live `start-failed` with unresolved `API_PORT` after a dependency build failure at `vrooli-memory`; only `browser-automation-studio@shadow` is registered on API port 15372. RCL remains healthy on pre-repair build `74e037ce6595e63c651fe3dbd52e2bd27a8faac1c7d68d39c70a22f6d1212a06`. The 2026-10-05 source repair is applied and package-tested, but runtime and receiver acknowledgement are open.
- Refusal: outer review denied the X-Dry-Run CaptureService POST before execution because deployed dry-run semantics were not established and the request exceeded the retained read-only scope. No POST occurred and no alternate capture route was attempted. Do not bypass the denial.
- Fallback: park D6. Do not register a guessed port, choose the healthy shadow by convenience, change endpoint overrides/instance selection, mutate old captures, or restart/deploy either runtime without existing qualified owner authority.
- Affected evidence: selected consumer route/build and receiver acknowledgement are unverified; D6 keyboard evidence and E1 acceptance remain open.
- Resume: BAS owner recovery or separately qualified existing-instance selection, followed by supported RCL lifecycle refresh onto the repaired source and read-only proof of the exact current selected consumer route/build and receiver acknowledgement. CaptureService use still requires its actual native authority.
- Cleanup: none; no service/process started or external tool installed.
- New read-only qualification at 2026-10-06: `vrooli scenario status browser-automation-studio --json` reports live `start-failed`, unresolved `API_PORT`, and operation `startop-3d93dff58891a3f54380e9e4ed367bc8` failed at dependency build `vrooli-memory` (17:50:08Z–17:50:29Z). The observed operation was not initiated by this orchestrator. The running RCL remains healthy on build `74e037ce6595e63c651fe3dbd52e2bd27a8faac1c7d68d39c70a22f6d1212a06`.
- Effect: no current selected consumer route/build or receiver acknowledgement; D6 remains unverified. Existing keyboard task/run and its terminal handoff are retained; do not duplicate dispatch while the prerequisites are absent.
- Resume: BAS live owner resolves the dependency/start failure or an existing BAS instance is independently qualified under its actual selection authority; then the supported RCL lifecycle owner refreshes repaired source and read-only evidence proves the exact route/build and receiver acknowledgement. CaptureService mutation remains refused unless actual native authority is established; no retry or alternate route.
- New read-only qualification at 2026-10-07: control-plane status now reports live BAS API_PORT 17116 and runtime `running` with health `healthy`, although latest start operation `startop-14cf21597674f838e31a5780ab17877c` failed during replacement UI artifact preparation. Direct GET `http://127.0.0.1:17116/health` succeeded with readiness true and build `sha256:c14f426cf40f15c5e0c2c5a51bd3710390850caa9de226629b2e12f309c7678f`. Generic BAS CLI `status` targets shadow port 15372 and does not qualify the selected live consumer route. RCL remains healthy on pre-repair build `74e037ce6595e63c651fe3dbd52e2bd27a8faac1c7d68d39c70a22f6d1212a06`.
- Effect: the bounded read-only qualification run `0f7844e0-a3a5-47e7-b9eb-144898bfb871` completed under the original task/profile/parent. It verified the BAS and RCL health/build identities and the repaired discovery/override contract, but could not prove the selected CaptureService base from any supported read-only route because current RCL still runs pre-repair bytes. No POST, workflow invocation, restart, deployment, or route substitution occurred. Run tokens reported 103,194 weighted non-cache and 1,179,648 cache-read (`tokens_known=false`); receipts remain unobserved. Task remains queued; D6/E1 remain unaccepted.
- Resume: BAS owner recovery or separately qualified existing-instance selection, then supported RCL lifecycle refresh onto repaired source and read-only proof of exact selected consumer route/build plus receiver acknowledgement. Continue D6 under original task/profile after those prerequisites and actual native authority for any CaptureService use. The earlier X-Dry-Run POST was denied by outer review before execution because deployed dry-run semantics were unestablished; do not retry it or use an alternate route. Do not restart/deploy by assumption.
- 2026-10-07 follow-up: fresh status at 08:37Z is unchanged: BAS live remains `start-failed` for replacement UI artifacts while its old API listener is healthy; RCL remains on pre-repair build. `browser-automation-studio drills list` exposes only controlled driver-unavailable, partial-initialization, capacity, and expiry drills; none repairs this replacement artifact failure. The managed BAS CLI projection refreshed itself from source during that read; recorded in INSTALLS.md. Stop repeating the drill-list command for this failure. No service or product state changed.
- 2026-10-07 diagnostics fallback: `vrooli scenario logs browser-automation-studio --lifecycle --tail 80` returned `no lifecycle log found`; `--step ui --tail 80` returned that the scenario log directory does not exist. Use the already available control-plane operation status as evidence (`startop-14cf21597674f838e31a5780ab17877c`, component `ui`, replacement artifact preparation, exit status 1); do not retry the missing log paths until the BAS owner restores log output. This leaves the detailed build cause unobserved.

## 2026-10-08 — Prompt Manager team inbox CLI unavailable

- Tool/owner: `prompt-manager team message-list rcl-prompt-manager-ux-design-campaign rcl-delivery-orchestrator`.
- Attempt: one read attempt failed while auto-rebuilding the Prompt Manager CLI: `scenarios/prompt-manager/cli/teams/finite_leader.go:69:41: syntax error: unexpected *, expected name`; build exited non-zero. The CLI also reported its binary fingerprint did not match the current sources.
- Fallback: stop retrying this CLI in the current pass. FEEDBACK.md and the native child/run state remain readable; team inbox state is unknown, not empty.
- Affected evidence: no new supervisor or operator message can be ruled out by inbox evidence in this pass. D6 remains parked on the already recorded BAS owner recovery / selected-route qualification conditions.
- Cleanup: no install or service action; use no further Prompt Manager CLI calls until its owner repairs the build.

## 2026-10-11 — Agent Manager auto-rebuild blocked during park

- Tool/owner: `agent-manager run park` for the original RCL owner run.
- Attempt: the park command auto-rebuild failed because Go reported `updates to go.mod needed; to update it: go mod tidy`. The installed binary fingerprint was `414c0b89191fd664897ad69a9f500395ce98cbb16db162ab1a4c5e4befa192b4`, while sources fingerprinted `14d9ae33fac2e096aec7993154ac095a636eac3d26b71932f208bc8e1bb902aa`. Despite the rebuild failure, the requested park completed with the installed binary.
- Fallback: stop retrying the rebuild. `packages/cli-core/cliutil/stalechecker.go` supports `CLI_CORE_REBUILD_FINGERPRINT`; setting it to the current source fingerprint makes the stale-check loop guard skip auto-rebuild while allowing the existing binary to run the park operation. Resume automatic rebuild only after the Agent Manager owner resolves the Go module state.
- Follow-up: first guard attempt used stale source fingerprint `14d9ae33fac2e096aec7993154ac095a636eac3d26b71932f208bc8e1bb902aa`; sources had changed to `1ab766519bea848c66c6a36a0fd35eed9aa49e0f223d2ea30151c97efb497922`, so auto-rebuild was attempted and failed again. Park still succeeded with the installed binary. Use the latest source fingerprint in the guard; do not retry the Go rebuild.
- Effect: no product or service state changed; the run remains parked through the existing owner operation.
- Recovery: a later park rebuild completed successfully at source fingerprint `4c4e6da66f179f7eb9849265c78d16149e4cdd7936c75a4ffe5fbdcbb549118f`; no dependency install or service lifecycle action occurred. The previous Go module rebuild failure is not active on the latest observed invocation.

## 2026-10-17 — Agent Manager child status read stalled

- Tool/owner: `agent-manager run get 9622ac09-1b5a-4734-91e0-aaeb6588bf36 --json` (Agent Manager).
- Attempt: after the child-status read produced no output for 10 seconds, the shell session was interrupted with exit 130; no owner error or terminal child status was returned. The last native wake had reported the child RUNNING.
- Fallback: stopped retrying Agent Manager status reads. Direct filesystem inspection confirms D21's exact G3 Get/List evidence line was appended to `epochs/E1.md` at 2026-10-17T17:10:34Z. The worker's terminal status remains uncertain; the row write does not prove task/run completion.
- Effect: no product/service operation was performed in the fallback. Do not infer terminal status, native accounting, or acceptance from the E1 row. Resume with native run/task reconciliation only after Agent Manager status reads respond; preserve the recorded E1 evidence and avoid repeating G3, D6, Get/List, or the worker's write.

## 2026-10-17 — Agent Manager child status and continuation still stall

- Tool/owner: Agent Manager exact child run `9622ac09-1b5a-4734-91e0-aaeb6588bf36`, task `b324009f-9f28-4fb7-b70f-cc4605b69895`.
- Attempt: `agent-manager task get b324009f-9f28-4fb7-b70f-cc4605b69895 --json` returned `TASK_STATUS_QUEUED`. `agent-manager run get 9622ac09-1b5a-4734-91e0-aaeb6588bf36 --json` rebuilt the CLI repeatedly (fingerprints `8910e4d63a3032afc61847d8495ae32814f1a9a11db4db3e2fcf1d16a3159c40`, `e32ed6ce5a507bbfacb822fd9bad2ce65c073f33084f98bf678be9406f3cae13`, `69321d3b99ec58281d04681647089bfbb842282addc1c88ff5fa9a64c67f5131`) without returning run JSON and was interrupted (exit 130). A log-only `agent-manager run continue ...` for this same child also produced no response during repeated rebuild and was interrupted (exit 130). Last child wake reported RUNNING; final native run status remains unknown.
- Fallback: the task read is the only responding native status artifact; preserve `QUEUED` and the run's last-known `RUNNING` wake without inferring a terminal state. The E1 evidence row exists, but no typed G3 cell was appended. Do not retry run status/continue until Agent Manager owner resolves the rebuild/status path; do not repeat the report, Get/List, capture, D6, or product action.
- Return condition: Agent Manager owner restores a responsive exact `run get` and continuation for this child; reconcile its terminal event/accounting, then have the same worker append the typed G3 pass cell from already retained evidence. No changed criterion or new grant is needed.
- Effect: no RCL product or runtime state changed. E1 remains unaccepted; exact gate and next action are in QUEUE.md.
- Recovery: child wake at 2026-10-17T18:00:11Z reported D21 ended complete. Exact `agent-manager run get` then returned `RUN_STATUS_COMPLETE`, exit 0, and confirmed task `b324009f-9f28-4fb7-b70f-cc4605b69895`; task read still says `TASK_STATUS_QUEUED`. The worker appended the typed G3 pass cell with prewrite manifest hash `a5a9c02c9f1301061067e7b6ace4d27b38d44d994759b7657e37ce6d10dd8a1f`. A new bounded D19 review is pending under existing task `dee77aff-8548-4481-b547-f12159f977b4`, run `25b3d98c-fd1d-4a32-9e2e-4dd6b23948e0`; the earlier run-get/continue stalls remain historical and their accounting uncertainty is preserved.

## 2026-10-17 — Prompt Manager report-bug skill read unavailable

- Tool/owner: `prompt-manager skill read report-bug` (Prompt Manager).
- Attempt: the CLI rebuilt successfully, then reported `No local prompt-manager API was detected` and `No running API port was detected for prompt-manager`; it suggested starting Prompt Manager.
- Fallback: did not start another scenario or retry the CLI. The Test Genie failure and exact return condition are recorded in the RCL QUEUE handoff for now; the report-bug filing remains unsubmitted.
- Effect: no Prompt Manager lifecycle or product state changed. Resume filing only after the Prompt Manager owner restores its API or provides the supported offline report route.

## 2026-10-17 — E1 owner park refused after D22 worker return

- Tool/owner: `agent-manager effort park scenarios/react-component-library/docs/internal/goal --run 2b16a068-2bc1-4a41-8629-54c4992c93c6` (Agent Manager effort owner).
- Attempt: first call with `--timeout 1h --json` returned exit 4: `{"allowed":false,"rule":"admissible-slice","reason":"no live direct child and nothing under ## Needs operator. Admissible-slice rule: admit the next [ready] slice (or plan one), or record a decision with options under ## Needs operator, then park"}`. The D22 run had just completed; its run detail reads `parent_run_id=null` although create supplied the original owner ID, so the linkage/termination timing is unresolved. After recording the required root heavy-lane return under `## Needs operator`, a second effort-park call returned exit 1: `api error (401): verified agent caller required`.
- Fallback: no direct `run park`, credential substitution, new identity, grant, or duplicate worker was attempted. D22 source work is complete and independently diff-checked; the original owner remains the authoritative run. Keep E1 handoff current and resume parking only through the same owner's supported effort route when the existing verified caller is available.
- Effect: no additional product, test, build, lifecycle, or saved-data operation occurred. The E1 worker returned; G5 and G7 remain unverified and E1 remains unaccepted.
- Return condition: Agent Manager accepts the existing original owner caller for the effort-park operation. The next action after root's heavy-lane release is the changed-input G5 unit qualification; G7 still requires exact D12/D13 admitted baseline bytes/attribution.

## 2026-10-18 — Agent Manager API stopped during D28 reconciliation

- Tool/owner: Agent Manager run Get and effort owner API for the original RCL orchestrator.
- Attempt: after D28 task/run creation and Queue handoff succeeded, Run Get returned API connection refused because Agent Manager was stopped. One supported `agent-manager --auto-start run get a7cadca3-2995-4e89-ac98-0484825c7378` attempt attached to an in-flight lifecycle start; it had not completed when checked. `vrooli scenario status agent-manager` reported stopped, restart in progress at dependency vrooli-events (3/4), PID 1587545.
- Fallback: stopped further Agent Manager calls while its owner lifecycle start is in progress. Preserve the exact pending task/run/session IDs in E1 and the Queue handoff; use only file reads during the outage. Resume native run reconciliation and effort parking when Agent Manager reports healthy.
- Effect: no RCL product edits, tests, builds, E1/E2 acceptance, or additional child/run creation occurred during this outage.
- Return condition: Agent Manager lifecycle start completes and its API is healthy; then read the exact D28 run/task and park this same parent through effort owner.
