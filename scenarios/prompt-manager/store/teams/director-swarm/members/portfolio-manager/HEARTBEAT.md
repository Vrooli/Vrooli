# Run Task: Portfolio Manager

## Resume Protocol

On the first heartbeat after the team is re-enabled (or after any long pause), re-baseline before proposing anything: read `path:scenarios/swarm-manager/docs/concepts/OPERATOR-JOURNEYS.md` for the current operator loop, take a fresh `swarm-manager goals context` snapshot into `goal-portfolio-record/YYYY-MM-DD`, and verify which approved work items were applied while paused. Triage open work items whose context no longer exists in `team.json`: supersede each one, re-raising it under the current work item vocabulary only if its evidence still holds. Raise no other new work items on that pass.

## Task Loop
1. Review accepted portfolio work items and mark supported applications.
2. Inspect current goal and backlog state.
3. Diff live goal names (`swarm-manager goals list`) against the `goal:` references in `path:docs/director-swarm/strategy/ROADMAP.md`. Any delta — a listed goal that no longer exists, or a live goal with no theme row — is drift; propose one bounded `goal-portfolio` work item covering the delta.
4. Check scenario-scoped proposals against existing goal coverage before drafting anything new.
5. Identify the smallest portfolio correction that needs approval.
6. Record portfolio markers or snapshots.
7. Propose corrections when they are not duplicative. `goal-proposal` and `goal-portfolio` work items carry the prediction block required by the Outcomes Charter.

## Run Decision

Record durable continuity in your declared Source Ledger topics. Choose one disposition: existing-action-reference, new-action-candidate, cli-backlog, capability-work-item, prune, improve, graduate, or no-action; state the evidence for the choice. Preserve any narrower lane-specific decisions stated in the task loop.
