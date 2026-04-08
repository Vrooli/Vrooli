# QA Recommendation

targetScenario: prompt-manager
problemOrOpportunity: GCT tests failed (6/11 passed). Failures include standards gate (critical violations), docs validation, UI unit tests, CLI unknown-command behavior (should return non-zero), and missing playbooks registry.
proposedAction:
- Run `scenario-auditor audit prompt-manager --standards-only --timeout 60` and resolve HIGH+ standards issues impacting tests.
- Fix docs validation failures and rerun docs checks.
- Investigate UI unit test failures under `scenarios/prompt-manager/ui` and fix.
- Update CLI install handling to return non-zero for unknown commands.
- Regenerate `scenarios/prompt-manager/bas/registry.json` via playbook builder.
evidence: evidence/gct-review.json (GCT review summary)
riskLevel: high
executionModeHint: manual
createdByTeam: scenario-qa
sourceRunId: cc10bf7c-1ad0-4bb7-b66d-bae886f6384a
