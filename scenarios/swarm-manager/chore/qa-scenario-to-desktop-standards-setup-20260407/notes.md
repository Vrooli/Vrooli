# QA Recommendation

targetScenario: scenario-to-desktop
problemOrOpportunity: Standards warning that `.vrooli/service.json` should run UI build before `show-urls` per production bundle guidance.
proposedAction:
- Update setup steps in `.vrooli/service.json` to run `build-ui` before `show-urls` (per docs/scenarios/PRODUCTION_BUNDLES.md).
- Re-run standards checks to confirm warning cleared.
evidence: evidence/gct-review.json (GCT review summary)
riskLevel: low
executionModeHint: manual
createdByTeam: scenario-qa
sourceRunId: ba6373e2-b77d-42ed-b430-d7e8d5015cd5
