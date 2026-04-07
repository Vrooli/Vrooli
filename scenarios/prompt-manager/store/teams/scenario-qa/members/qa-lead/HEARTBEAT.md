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

### Step 4: Track & Coordinate
- Record each reviewed scenario in the knowledge log:
  `[YYYY-MM-DD] Reviewed <scenario>: <readiness> (code_quality: X, tests: Y, standards: Z)`
- Record any created fix items:
  `[YYYY-MM-DD] Created <kind>/<name> for <scenario>`
- If a scenario has critical (red) failures across 2+ dimensions, set fix item
  priority to 1 (critical) so it is workshopped and executed promptly
- If test failures look like regressions, message test-strategist for analysis

## Fallback
- Review any pending audit requests from team inbox
- Check if completed audits need follow-up (query knowledge log for recent entries)
- Monitor quality trends across scenarios
