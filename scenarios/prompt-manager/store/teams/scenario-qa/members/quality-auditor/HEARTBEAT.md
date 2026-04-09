# Heartbeat: Quality Auditor

## Deep Structural Audit

### Step 1: Select Scenario
Run: `swarm-manager scenarios review-queue --limit 1`

If zero scenarios returned, log "No scenarios for deep audit" and stop.

Check the knowledge log for recent deep-audit entries. Skip if this scenario was audited
in the last 7 days.

### Step 2: Select Steer Skill
Choose the next skill from the rotation that has NOT been applied to this scenario recently.

Rotation (always applicable):
1. `screaming-architecture-audit`
2. `boundary-of-responsibility-enforcement`
3. `seam-discovery-and-enforcement`
4. `invariant-discovery-and-enforcement`
5. `cognitive-load-reduction`
6. `decision-boundary-extraction`
7. `code-cleanup`

Check knowledge log for `[deep-audit] <scenario> via <skill>` entries to determine which
skills have already been applied. Pick the first unapplied skill from the list.

If all 7 have been applied to this scenario, pick the oldest one (longest since last audit).

### Step 3: Investigate
1. Read the steer skill: `prompt-manager skill read <skill-id>`
2. Read the scenario's architecture docs, code structure, and test suite
3. Apply the skill's methodology to identify structural quality issues
4. Document findings with specific file paths, line references, and evidence

### Step 4: Create Backlog Item
If findings warrant action (non-trivial structural improvements identified):
1. Create an `execute` backlog item via `swarm-manager backlog create`
2. Name pattern: `qa-deep-<scenario>-<skill-short-name>-YYYYMMDD`
3. Include in the description:
   - The steer skill used and what it revealed
   - Specific findings with file paths and evidence
   - Suggested approach for addressing the issues
4. Set `suggested_skills` to `["<skill-id>"]` (the steer skill used)
5. Set tags: `["deep-audit", "<scenario-name>"]`
6. Set acceptance_allow: `["scenarios/<scenario>/**"]`
7. Set priority: 6 (lower than the programmatic QA runner's automated findings)
8. Write a draft plan.md in the backlog item folder with the investigation findings
   structured as an implementation plan

### Step 5: Log
Record in knowledge log:
`[YYYY-MM-DD] Deep audit: <scenario> via <skill-id> — <one-line-summary>`
