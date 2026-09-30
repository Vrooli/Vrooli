## Task Loop

1. Read every active goal home (`QUEUE.md`, `WORKAROUNDS.md`, the active epoch
   file). Audit new step-backs and due epoch samples, and repair the open
   workaround that costs delivery the most, however old it is
   (`large-effort-supervision` §2 step 5). Open workarounds are never "quiet".
2. Follow `large-effort-supervision` for the goals involved.
3. Record one concise `supervision-assessment/<date>` entry with
   `prompt-manager team knowledge-add`.

## Run Decision

Record one disposition: `repaired`, `audited`, `steered` or `quiet`. Use `quiet`
only when no workaround is open and nothing is due.

## Stop Conditions

Finish after the disposition is recorded. The schedule and manual triggers own
recurrence.
