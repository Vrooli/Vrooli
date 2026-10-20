# Workarounds and validation constraints

Historical attempts, failures, tool limitations and exact receipts are preserved verbatim in [the archived snapshot](archive/WORKAROUNDS-through-20261018-before-A242-cleanup.md); the immediate pre-cleanup file is also retained under A242 preimages.

Current constraints:

- Preserve every prior Test Genie failure. The A176 changed-input G5 run `20261018-015558-f7565c16` is FAIL (four command blockers, 55 warnings); do not repeat it unchanged. A future G5 qualification requires repaired inputs and a fresh changed-input `unit` phase for each required scenario, with terminal execution and policy-projection receipts.
- The historical Oct 1 baseline attempts remain failures and do not substitute for a new changed-input qualification.
- Preserve E1 as PARTIAL/unaccepted and G7 as missing/unwaived. E2 is also unaccepted; its G1 full pipeline remains failed even though the selected catalog check, exact API test and G3 path-preservation check passed.


2026-10-18T15:47:54Z Agent Manager friction wake failed with HTTP 401 `verified operator required` during the D30 `agent-manager effort epoch-check`; `agent-manager run identity --json` had no current native run identity, and the assigned task still reads QUEUED. Fallback: retained the source/test evidence and returned the lane to the parent; did not retry the wake or substitute the task ID for a worker run ID. Retry only from a supported verified Agent Manager run context when the owner reconciles the queued task.
