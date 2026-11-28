# Workarounds

## E18 worker token and receipt query

- Failed: `agent-manager run tokens`, `run receipts`, and `run get` could not reach the Agent Manager API at `http://localhost:18800`; the API health endpoint refused connections.
- Fallback: retained the current worker ID from `agent-manager run list --status running`, completed the work in the shared tree, and ran the required `effort epoch-check`, which reported no open directives. At the outage checkpoint, weighted spend and receipt finality were unverified.
- Retry when: the Agent Manager API is running and its health endpoint is available. Do not infer token spend or receipt finality from the worker's model profile or the epoch check.

## E18 Agent Manager query recovery

- Recovery: a later read-only `vrooli scenario status agent-manager --json` showed the Agent Manager API healthy after a successful lifecycle restart. No restart was initiated by this worker.
- Retry result: `agent-manager run tokens 4de81f69-070e-49df-8020-bcc340783720` succeeded with weighted non-cache spend 701,745. `run receipts` returned status `unobserved` and no observations because the run was still active.
- Remaining check: receipt finality can only be established after the worker run becomes terminal; do not infer it from the successful epoch check.

## E26 shadow lifecycle logs unavailable

- Failed: `vrooli scenario logs browser-automation-studio --instance shadow --runtime --tail 120` reports that `/home/matthalloran8/.vrooli/logs/scenarios/browser-automation-studio@shadow` does not exist, so the J04 `bringToFront()` stall cannot be traced from service logs.
- Fallback: kept the active `bas-goal` shadow untouched; checked API and driver health, the driver's empty session inventory, the J04 failure trace, and the tab handler's `bringToFront()` call. No additional journey was run while another Agent Manager worker was active.
- Retry when: the shadow's lifecycle log is provisioned by its owner or the owner restores runtime log routing. Do not restart the live BAS or infer a root cause from healthy health endpoints.
