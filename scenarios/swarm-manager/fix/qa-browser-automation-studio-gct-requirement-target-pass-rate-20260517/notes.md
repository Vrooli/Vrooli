# GCT evidence

Fresh command sequence on 2026-05-17T04:02:10Z:

- `vrooli scenario completeness score calculate browser-automation-studio`
- `vrooli scenario completeness score get browser-automation-studio --json`
- `vrooli scenario completeness score validation browser-automation-studio --json`
- `vrooli scenario completeness score recommend browser-automation-studio --json`

Score result: score 38, base_score 46, validation_penalty 8, classification `foundation_laid`.

Pass-rate readiness evidence:

- requirements: 1/63 passing, rate 0.016, 0/25 quality points
- targets: 1/31 passing, rate 0.032, 0/25 quality points
- tests: 6/9 passing, rate 0.667, 10/15 quality points
- GCT recommendation priority 1: increase test pass rate to 90%+
- GCT recommendation priority 2: increase requirement pass rate to 90%+
- GCT recommendation priority 3: increase operational target pass rate to 90%+

Success check: rerun calculate/get and verify requirement, target, and test pass rates reach at least 90%, and score improves from 38 foundation_laid.
