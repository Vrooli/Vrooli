# Run Task: Portfolio Manager

## Resume Protocol

On the first heartbeat after the team is re-enabled (or after any long pause), re-baseline before proposing anything: read `path:scenarios/swarm-manager/docs/concepts/OPERATOR-JOURNEYS.md` for the current operator loop, take a fresh `swarm-manager goals context` snapshot into `goal-portfolio-record/YYYY-MM-DD`, and verify which approved work items were applied while paused. Triage open work items whose context no longer exists in `team.json`: supersede each one, re-raising it under the current work item vocabulary only if its evidence still holds. Raise no other new work items on that pass.

## Task Loop
1. Review accepted portfolio work items and mark supported applications.
2. Inspect current goal and backlog state.
3. Read Swarm Manager's canonical item staleness verdict from `swarm-manager backlog list --status backlog --archived false --json`. Select only rows where `.stale == true`, sort by `.updated` ascending, and inspect at most the 6 oldest items. For each selected item, assign `keep`, `refresh`, or `supersede` and cite its age, whether its plan reference resolves, and whether its acceptance paths still exist.
4. Separately select active goals whose `.goal.updated` is at least 14 days old from `swarm-manager goals list --json`, sort oldest first, and inspect at most the 3 oldest goals. For each selected goal, assign `keep`, `refresh`, or `supersede` and cite its age, acceptance criteria, milestone state, and target state.
5. Write one bounded staleness-verdict section to today's `goal-portfolio-record/YYYY-MM-DD` snapshot on every run. Include the inspected counts, each typed item or goal reference, its verdict, and its evidence. Record an explicit empty batch when no stale entity qualifies.
6. Treat every staleness verdict as a proposal. Never apply it: do not import backlog changes, change item status, archive a goal, or start consequential work from this lane. Human disposition is the only path from this batch to mutation. Do not use the interactive `operations-sweep-staleness` job.
7. Diff live goal names (`swarm-manager goals list`) against the `goal:` references in `path:docs/director-swarm/strategy/ROADMAP.md`. Any delta — a listed goal that no longer exists, or a live goal with no theme row — is drift; propose one bounded `goal-portfolio` work item covering the delta.
8. Check scenario-scoped proposals against existing goal coverage before drafting anything new.
9. Identify the smallest portfolio correction that needs approval.
10. Record portfolio markers or snapshots.
11. Propose corrections when they are not duplicative. `goal-proposal` and `goal-portfolio` work items carry the prediction block required by the Outcomes Charter.

## Run Decision

Record durable continuity in your declared Source Ledger topics. Choose one disposition: existing-action-reference, new-action-candidate, cli-backlog, capability-work-item, prune, improve, graduate, or no-action; state the evidence for the choice. Preserve any narrower lane-specific decisions stated in the task loop.
