# QA Recommendation

targetScenario: scenario-to-desktop
problemOrOpportunity: Standards warnings for missing test files in `runtime/infra` and `runtime/testutil`.
proposedAction:
- Add at least one `*_test.go` per package in `runtime/infra` and `runtime/testutil` to cover core behaviors.
- Re-run standards checks to confirm warnings cleared.
evidence: evidence/gct-review.json (GCT review summary)
riskLevel: medium
executionModeHint: manual
createdByTeam: scenario-qa
sourceRunId: ba6373e2-b77d-42ed-b430-d7e8d5015cd5
