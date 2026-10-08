# Infrastructure workarounds

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

## 2026-10-01 — E1 Agent Manager child run stopped advancing

- Tool/owner: Agent Manager interactive worker run `ac2aa69a-6efc-49ff-969b-422f35593893`, task `08351232-917c-499c-9226-06ae9204b0fd`, workload `rcl/E1`.
- Observation: repeated child-producer timer wakes found the run still `RUN_STATUS_RUNNING`, but its last heartbeat remained `2026-10-01T14:14:23Z`, last update `15:02:54Z`, and transcript had no new worker message after `15:02:10Z`. The bounded D4 closure directive was typed into this exact interactive session; the next reconciliation still showed no new activity.
- Fallback: after repeated timer wakes and no new activity following the bounded D4 message, stopped the stale child through Agent Manager and verified it as `RUN_STATUS_CANCELLED`. Dispatched one successor from E1 using the same worker profile/model, workload key, and parent run: `65f94472-38ba-4116-b356-6599d49e707a`, task `6059e5a4-06b3-435c-a4d2-b13d8a1d233f`; it was `RUN_STATUS_PENDING` at admission. Do not create another successor unless this run reaches a terminal state and the epoch's recovery rules still allow it.
- Affected evidence: E1 remains unaccepted. Focused Heading integration check and explicit normal-entry-to-governed-adoption handoff are still outstanding; independent §4.4 acceptance has not run.
- Recurrence: repeated across successive timer wakes after D4. The supervision heartbeat trigger returned `deduplicated`; a direct cross-team inbox message was rejected because this agent is not a member of `effort-supervision`, so no direct message was delivered. Current owner state is recorded in QUEUE.md; the supervisor heartbeat may read this goal home.
- Cleanup: none.
