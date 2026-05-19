# GCT readiness evidence: vrooli-onboarding

Fresh commands run on 2026-05-18 UTC:

- `vrooli scenario completeness score calculate vrooli-onboarding --json`
- `vrooli scenario completeness score get vrooli-onboarding --json`
- `vrooli scenario completeness score validation vrooli-onboarding --json`
- `vrooli scenario completeness score recommend vrooli-onboarding --json`

## Result

- Score: 94
- Classification: nearly_ready
- Calculated at: 2026-05-18T04:01:26Z
- Validation penalty: 0
- Validation issues: none

## Passing evidence

- Requirements: 14/14 passing, 20/20 quality points
- Operational targets: 6/6 passing, 15/15 quality points
- Tests: 64/64 passing, 15/15 quality points
- Test coverage ratio: 4.57x, 8/8 coverage points
- UI: 25/25 points

## Yellow readiness evidence

- Coverage depth: avg_depth=1.0, only 2/7 depth points
- Target quantity: 6 targets, threshold=below, 2/3 quantity points
- Overall classification remains nearly_ready despite clean validation and green pass rates.

## Success target

Rerun GCT calculate/get/validation. The fix is complete when the scenario reaches production_ready, depth_score avg_depth improves from 1.0 toward 3.0+, target quantity no longer reports below threshold, and validation_analysis.has_issues remains false.
