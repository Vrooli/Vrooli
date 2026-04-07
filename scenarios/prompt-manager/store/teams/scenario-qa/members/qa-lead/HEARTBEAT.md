# Heartbeat: QA Lead

## Preemptive Scenario Quality Review

### Step 1: Get Priority Scenarios
Run: `swarm-manager scenarios review-queue --limit 3`

If zero scenarios are returned, log "No scenarios require preemptive review" to the
knowledge log and proceed to the Fallback section.

Skip any scenario where `cooldown_until` is in the future.

### Step 2: Run GCT Reviews
For each scenario from Step 1:
1. Run: `git-control-tower review-run <scenario_name> --json`
2. Wait for completion (the CLI polls automatically unless --no-wait)
3. Parse the readiness result and failing dimensions

### Step 3: Create Fix Items for Failing Scenarios
For each scenario with red or yellow readiness:
- Create a `fix` or `chore` backlog item via `swarm-manager backlog create`
- Name pattern: `qa-<scenario>-<failing-dimension>-YYYYMMDD`
- Required tags: `["preemptive-qa", "<scenario-name>"]`
- Required fields per swarm-manager-recommendations contract:
  - targetScenario, problemOrOpportunity (cite specific failing dimensions + scores),
    proposedAction, evidence (GCT review JSON), riskLevel, executionModeHint,
    createdByTeam: "scenario-qa", sourceRunId
- Set priority based on readiness: red=2, yellow=4
- Set acceptance_allow to `["scenarios/<scenario>/**"]`

#### Splitting Large Findings
When a GCT dimension has many violations:
- If ≤5 violations in a dimension: create one backlog item for that dimension
- If >5 violations: group by category (e.g., lint_issues, complex_functions, tech_debt_markers, missing_tests) and create one item per category with significant violations
- Each item should target roughly 5-10 files to keep work achievable in one focused session
- Name pattern for split items: `qa-<scenario>-<dimension>-<category>-YYYYMMDD`

### Step 3.5: Wire Dependencies on Related Backlog Items
For each fix/chore item created in Step 3:
1. Query existing non-terminal backlog items targeting the same scenario:
   `swarm-manager backlog list --scenario <scenario-name> --status backlog,researching,ready,queued --json`
2. From the results, exclude:
   - The fix/chore item itself (same kind/name)
   - Items tagged `preemptive-qa` (avoids circular dependencies between QA-created items)
   - Items that already list this fix in their `depends_on`
3. For each remaining item:
   a. Read its current `depends_on` array from the JSON response
   b. Append `<kind>/<fix-item-name>` to produce the merged list
   c. Update: `swarm-manager backlog update --kind <kind> --name <name> --data '{"depends_on":[<merged-list>]}'`
   Note: The update API replaces `depends_on` entirely — always merge with existing values.
4. Log each wired dependency:
   `[YYYY-MM-DD] Wired depends_on: <target-kind>/<target-name> → <fix-kind>/<fix-name>`

### Step 3.75: Create Execute Items for Code Quality Improvements
For each scenario where GCT review returns a codeQuality dimension with score below 70:
1. Create an `execute` backlog item via `swarm-manager backlog create`
2. Name pattern: `qa-<scenario>-code-quality-YYYYMMDD`
3. Set `suggested_skills` to `["refactor"]`
4. Include GCT code quality breakdown as evidence (categories + violation counts)
5. Set tags: `["preemptive-qa", "<scenario-name>", "code-quality"]`
6. Set priority based on score: <40 → priority 2, 40-60 → priority 4, 60-70 → priority 6
7. Set acceptance_allow to `["scenarios/<scenario>/**"]`
8. Apply the same depends_on wiring from Step 3.5 to these items

### Step 4: Track & Coordinate
- Record each reviewed scenario in the knowledge log:
  `[YYYY-MM-DD] Reviewed <scenario>: <readiness> (code_quality: X, tests: Y, standards: Z)`
- Record any created fix/execute items:
  `[YYYY-MM-DD] Created <kind>/<name> for <scenario>`
- If a scenario has critical (red) failures across 2+ dimensions, set fix item
  priority to 1 (critical) so it is workshopped and executed promptly

## Fallback
- Review any pending audit requests from team inbox
- Check if completed audits need follow-up (query knowledge log for recent entries)
- Monitor quality trends across scenarios
