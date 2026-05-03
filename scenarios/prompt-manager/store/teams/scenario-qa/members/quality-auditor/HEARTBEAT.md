# Heartbeat: Quality Auditor

Before applying the rotation's current lens, read the paired strategic-canon doc at `docs/scenario-qa/audit-techniques/<current-rotation-slug>.md` (Available Skills table in [RESPONSIBILITIES.md](RESPONSIBILITIES.md) lists all seven). The PoR doc covers when the lens applies, when it backfires, and what the qa-contrarian will challenge.

If during the audit you observe a bug that's outside structural scope (broken code, regression, prompt confusion, data-shape mismatch, unexpected error), file it via the `report-bug` skill to `bug-inbox/*` — bugs go to the bug-investigator, not into the audit knowledge or backlog stream.

## Task Loop
1. Select one scenario from the review queue.
2. Check recent deep-audit knowledge for recency (per `team.json` `recencyWindowDays`).
3. Select the next audit-technique skill from rotation (`team.json` `taskParameters.skillRotation`).
4. Read the paired strategic-canon doc, then load the skill.
5. Inspect architecture docs, code structure, and tests.
6. Create an execute backlog item only for non-trivial structural findings.
7. Record the audit in `quality-audit/<scenario-id>/<skill-id>` knowledge.

## Handoff Shape
### Scenario audited
### Skill applied
### Findings
### Backlog item created
### Bugs filed (via report-bug)
### Knowledge entries written
