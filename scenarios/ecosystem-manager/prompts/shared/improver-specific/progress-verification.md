# Progress & Regression Verification

## Core Philosophy
- PRD checkboxes must reflect reality; test proof before updating them.
- Improvements only count when previously working features still work.
- Capture evidence so the next agent can trust your handoff.

## Canonical Checklist Reference
- Apply the "When to Check/Uncheck" rules defined in the `prd-protocol` section when evaluating each requirement.
- Record partial progress notes using the PRD Protocol's example format; link test evidence instead of redefining criteria here.

## Verification Workflow
### 1. Baseline before changes
```bash
# Scenario baseline (run from the scenario directory)
make test > /tmp/${SCENARIO}_baseline_tests.txt
make status > /tmp/${SCENARIO}_baseline_status.txt

# Resource baseline (run from the resource directory)
./lib/test.sh > /tmp/${RESOURCE}_baseline_tests.txt
vrooli resource status ${RESOURCE} --json > /tmp/${RESOURCE}_baseline_status.json
```

### 2. Validate every PRD check
```bash
# Start the target with its lifecycle tools
make run                                   # Scenario (preferred)
# OR
vrooli scenario start ${SCENARIO}          # Scenario via CLI
# OR
vrooli resource start ${RESOURCE}          # Resource lifecycle

for item in $(grep "✅" PRD.md); do
  echo "Testing: $item"
  # Execute the command, curl, or script that proves the requirement works
done
```
Uncheck anything that fails immediately—partial notes beat dishonest checkmarks.

### 3. Monitor while implementing
- Re-run focused suites after each change (`make test-smoke`, `make test-api`, or `vrooli scenario test ${SCENARIO}`).
- Watch runtime output via `make logs` (scenario Makefile) or `vrooli scenario logs ${SCENARIO} --follow`.
- For resources, poll `vrooli resource status ${RESOURCE} --json` to confirm the service stays healthy.

### 4. Post-change confirmation
```bash
# Scenario regression pass
make test                                  # Preferred
# OR
vrooli scenario test ${SCENARIO}           # CLI alternative

# Resource regression pass
./lib/test.sh
vrooli resource status ${RESOURCE} --json

# Optional UI evidence
vrooli resource browserless screenshot --scenario ${SCENARIO} --output /tmp/${SCENARIO}_ui.png
```
Compare results with the baseline artifacts and screenshots.

### 5. If a regression appears
- Stop immediately, capture the failing evidence (failing `make test` output, CLI test logs, screenshots).
- Follow the handoff steps in the **Failure Recovery** section and wrap up using **Documentation & Finalization** guidance.
- Hand the task back (status: broken/degraded) and wait for further direction. Do **not** attempt to self-recover via git or continue development.

## Regression Safeguards
- Keep the regression test suite broad: health endpoints, prior workflows, CLI commands, integrations, and performance checks.
- Confirm compatibility: API contracts, database schemas, config formats, and backward compatibility must remain intact.
- Use the baseline artifacts to detect drift; differences mean you owe an explanation.

### Regression indicators
```markdown
🔴 Critical: Health checks fail, service will not start, API 500s, lost data connections
🟡 Moderate: >20% performance drop, >30% memory increase, new warnings, flaky tests
🟠 Minor: Style violations, deprecation warnings, missing coverage
```

### Root-cause checklist
1. When did we introduce the regression?
2. What functionality broke?
3. Why did it break (true root cause)?
4. What is impacted downstream?
5. What remediation is required (for the next agent)?
6. How do we prevent a repeat?

## Reporting & Metrics
```yaml
real_progress:
  - P0_requirements_completed
  - Tests_passing_percentage
  - Integration_points_working
  - UI_features_visual_verified

not_progress:
  - Lines_changed
  - Files_touched
  - Comments_added

net_progress = features_working - features_broken - debt_added
```

### Progress report template
```markdown
## [Date] Progress

### Verified Complete
- [Feature]: [test command proving it works]

### Partial Progress
- [Feature]: [what works] / [what does not]

### Regressions
- [What broke]: [error details]

### Net Progress
- Added: X features
- Broken: Y features
- Net: X - Y
```

### Impact scoring & golden rule
```
impact = users_affected * feature_criticality * downtime_risk
# Block deployment if impact > 50, require review if > 30, add warning if > 10
```
**Golden Rule:** If it worked before your change, it must work after your change.

## Quick reference
- Never hide regressions—document, escalate, and stop.
- Update the PRD the moment reality changes.
- Lean on **Documentation & Finalization** whenever work cannot finish cleanly.
- Preserve baseline evidence so future agents can trust your work.
