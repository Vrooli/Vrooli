# Heartbeat: Programmatic QA Runner

Cross-link: [`docs/scenario-qa/readiness-checks/README.md`](../../../../../../../docs/scenario-qa/readiness-checks/README.md) (stub; future strategic-canon home for individual readiness checks).

If during a review you observe a bug that's outside the readiness-review scope (broken code, regression, prompt confusion, data-shape mismatch, unexpected error), file it via the `report-bug` skill to `bug-inbox/*` rather than queuing it as a backlog item — bugs go to the bug-investigator for triage, not into the readiness backlog.

## Task Loop
1. Query the scenario review queue.
2. Skip scenarios that are missing or still cooling down.
3. Run GCT readiness review for each selected scenario.
4. Create evidence-rich backlog items for red or yellow findings.
5. Split large findings by category when needed.
6. Wire dependencies on related backlog items.
7. Record reviewed scenarios and created work in knowledge.

## Handoff Shape
### Scenarios reviewed
### Findings converted to backlog
### Dependencies wired
### Skipped scenarios
### Bugs filed (via report-bug)
### Knowledge entries written
