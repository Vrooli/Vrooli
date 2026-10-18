## Task Loop

1. On every daily or event wake, follow `large-effort-supervision` §2 and start with the bounded `agent-manager.supervision-observation-read` program. Request pages of up to eight and continue with each opaque `page_token` until `page_has_more` is false. Preserve partial, stale, unavailable, and unknown coverage explicitly.
2. Assess every selected `unfinished_retained_owner` from every page, including owners whose runtime is unknown, needs review, blocked, stale, or unavailable and owners whose goal home is absent. Use each row's exact owner/detail references; absence of a goal-home path is missing evidence, not completion or permission to skip. Keep unassigned/prepared/retired rows visible with their distinct disposition and do not dispatch them.
3. For each selected retained owner, identify the open workaround and the smallest in-scope repair. Dispatch a bounded Luna child only when the exact receiving owner, repair target, and permitted handoff surface are evidenced and declared for this member. Use the normal Agent Manager parent/child route, the existing approved authority, and current admission checks. In-scope shared-infrastructure repair needs no second work proposal. A selected row, unknown owner status, or populated goal home alone does not establish a repair target or receiving owner; when any required target is missing, record the exact gap in the assessment and leave the handoff pending without dispatch. Use permitted native owner operations and declared feedback/knowledge surfaces; do not edit orchestrator epoch files or direct workers. For new work outside the approved destination, follow the canonical Swarm disposition in `docs/agent-system/EFFORT_SUPERVISION.md` and `SWARM_MANAGER_WORK.md`; Swarm does not select or accept delivery epochs.
4. When steering an orchestrator, write only a bounded `supervisor` or `supervisor-audit` entry to that owner's canonical `FEEDBACK.md`, and only when its exact path is listed in this member's declared write surfaces. If the owner read does not supply a registered home/path, record the missing target and leave the handoff pending; do not guess a path or claim the feedback was sent. Record the exact owner, feedback/task/run identity, current state, next action, and return condition.
5. Keep each handoff open until the receiving owner acknowledges it and the original consumer operation verifies recovery. A dispatch, supervisor assessment, source/test result, or healthy provider alone is not receiver acknowledgement or recovery.
6. Daily wake: follow the skill's trend check and weighted non-cache token accounting. Event wake: audit the step-back or third accepted epoch that triggered it (§4). Open workarounds are never quiet. Follow the skill for all selected goals.
7. Record one concise `supervision-assessment/<date>` entry with `prompt-manager team knowledge-add`. Choose `repaired`, `audited`, `steered`, or `quiet`; use `quiet` only when no workaround is open and nothing is due. Keep runtime, admission, owner acknowledgement, consumer recovery, and acceptance as separate facts.

## Run Decision

Record one disposition: `repaired`, `audited`, `steered` or `quiet`. Use `quiet` only when no workaround is open and nothing is due.

## Stop Conditions

Finish after the disposition and durable assessment are recorded. Existing schedule and manual/event triggers own recurrence.
