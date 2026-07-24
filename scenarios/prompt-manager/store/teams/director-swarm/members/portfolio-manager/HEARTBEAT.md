# Heartbeat: Portfolio Manager

## Resume Protocol

On the first heartbeat after the team is re-enabled (or after any long pause), re-baseline before proposing anything: read `path:scenarios/swarm-manager/docs/concepts/OPERATOR-JOURNEYS.md` for the current operator loop, take a fresh `swarm-manager goals context` snapshot into `goal-portfolio-record/YYYY-MM-DD`, and verify which accepted decisions were applied while paused. Triage pending decisions whose context no longer exists in `team.json`: supersede each one, re-raising it under the current decision vocabulary only if its evidence still holds. Raise no other new decisions on that pass.

## Task Loop
1. Review accepted portfolio decisions and mark supported applications.
2. Inspect current goal and backlog state.
3. Diff live goal names (`swarm-manager goals list`) against the `goal:` references in `path:docs/director-swarm/strategy/ROADMAP.md`. Any delta — a listed goal that no longer exists, or a live goal with no theme row — is drift; propose one bounded `goal-portfolio` decision covering the delta.
4. Check scenario-scoped proposals against existing goal coverage before drafting anything new.
5. Identify the smallest portfolio correction that needs approval.
6. Record portfolio markers or snapshots.
7. Propose corrections when they are not duplicative. `goal-proposal` and `goal-portfolio` decisions carry the prediction block required by the Outcomes Charter.

## Handoff Shape
### Portfolio state
### Accepted decisions applied
### Coverage checks
### Proposed corrections
### Decisions raised
### Knowledge entries written
