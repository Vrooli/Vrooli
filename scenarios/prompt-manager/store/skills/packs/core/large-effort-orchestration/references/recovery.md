# Recovery: lost runs, runner limits and route changes

The goal home is the state. Every recovery resumes from it; none copies state
elsewhere or resets an epoch.

**One orchestrator.** Never create a second orchestrator while the first may be
live. Reconcile a timed-out or uncertain start through Agent Manager with the
original run identity; a network error is not proof of rejection. Until
reconciled, mark the dispatch uncertain in the epoch file and work elsewhere.

**Worker ended early.** Stop the old run if it still shows as running, then spawn a
successor that resumes from the epoch file. A repeated early end with no new
slice-log line is a step-back signal, not a reason to respawn again.

| Observation | Recovery |
|---|---|
| Context or output limit | Continue the same session or start a successor from the epoch file. Not quota exhaustion. |
| Session or weekly subscription allowance exhausted | Park until the observed reset time. With no reset time, park on a long timer; a second profile on the same pool is not fresh quota. |
| Transient rate limit or overload | Honor retry-after with backoff; one timer, no polling. |
| API credits or spending limit exhausted | Stop paid dispatch for that pool. Never buy credits or enable top-up. |
| Authentication failure or refusal | Record the remedy the owner needs. Another launcher is not a remedy. |
| Normal completion | Read the result once. Do not restart a finished worker. |

**Route change.** Delivery teams dispatch through Agent Manager only. The
control-plane coding-agent launcher or a direct harness run is a fallback only
when the goal explicitly authorizes its weaker guarantees. The epoch and its gate
do not change; record the lost guarantees (telemetry, sandbox tracking, native
goal support) as a `WORKAROUNDS.md` entry. There is no universal `--goal` flag;
check the runner's contract.
