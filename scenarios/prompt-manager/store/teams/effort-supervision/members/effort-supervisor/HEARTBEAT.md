## Task Loop

1. Daily check: follow `large-effort-supervision` §2 for each active goal home.
   That means reading and reporting only, flagging stalls, and starting one
   repair (usually a bounded Luna run) for the costliest open workaround.
   Event wakes: audit the step-back or the 3rd acceptance that triggered the wake
   (§4). Open workarounds are never "quiet".
2. Follow `large-effort-supervision` for the goals involved.
3. Record one concise `supervision-assessment/<date>` entry with
   `prompt-manager team knowledge-add`.

## Run Decision

Record one disposition: `repaired`, `audited`, `steered` or `quiet`. Use `quiet`
only when no workaround is open and nothing is due.

## Stop Conditions

Finish after the disposition is recorded. The schedule and manual triggers own
recurrence.
