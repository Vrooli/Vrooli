# GCT evidence

Fresh command sequence on 2026-05-16T22:01:32Z:

- `vrooli scenario completeness score calculate brand-manager --json`
- `vrooli scenario completeness score get brand-manager --json`
- `vrooli scenario completeness score validation brand-manager --json`
- `vrooli scenario completeness score recommend brand-manager --json`

Score result: score 84, validation_penalty 0, classification `nearly_ready`; validation_analysis.has_issues=false.

Breakdown:

- quality 50/50
- coverage 2/15
- quantity 7/10
- UI 25/25

Metrics:

- requirements: 54/54 passing
- targets: 54/54 passing
- tests: 3/3 passing
- test_coverage_ratio: 0.0556
- last_test_run: 2026-04-04T01:12:30Z

Readiness gap: only 3 tests cover 54 requirements/targets. GCT reports quantity.tests below threshold and recommends adding more tests to reach the good threshold and optimal 2:1 test-to-requirement ratio.

Success check: rerun calculate/get/recommend and verify tests.count increases beyond 3 with focused automated coverage, coverage improves from 2/15, quantity.tests is no longer below threshold, and no validation issues are introduced.
