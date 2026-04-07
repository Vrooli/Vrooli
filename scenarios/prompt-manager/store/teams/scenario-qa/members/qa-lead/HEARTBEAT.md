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

### Step 4: Track & Coordinate
- Record each reviewed scenario in the knowledge log:
  `[YYYY-MM-DD] Reviewed <scenario>: <readiness> (code_quality: X, tests: Y, standards: Z)`
- Record any created fix items:
  `[YYYY-MM-DD] Created <kind>/<name> for <scenario>`
- If a scenario has critical (red) failures across 2+ dimensions, message the
  debug team lead for deeper investigation
- If test failures look like regressions, message test-strategist for analysis

## Fallback
- Review any pending audit requests from team inbox
- Check if completed audits need follow-up (query knowledge log for recent entries)
- Monitor quality trends across scenarios
