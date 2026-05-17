# GCT evidence

Fresh command sequence on 2026-05-17T04:02:02Z:

- `vrooli scenario completeness score calculate scenario-to-cloud`
- `vrooli scenario completeness score get scenario-to-cloud --json`
- `vrooli scenario completeness score validation scenario-to-cloud --json`
- `vrooli scenario completeness score recommend scenario-to-cloud --json`

Score result: score 41, base_score 49, validation_penalty 8, classification `functional_incomplete`.

Test depth evidence:

- tests: 3/3 passing, but only 3 tests for 13 requirements
- test_coverage_ratio: 0.231
- avg_depth: 1.0
- GCT recommendations: add more tests and reach optimal 2:1 test-to-requirement ratio

Validation issues:

- `monolithic_test_files`: severity medium, penalty 2. Worst offender `cli/main_test.go` validates STC-P0-001, STC-P0-002, STC-P0-003, and STC-P0-006 in one file.
- `superficial_test_implementation`: severity medium, penalty 2, count 2. Flagged `requirements/08-p1-targets/module.json` for STC-P1-001 and STC-P1-002.

Success check: rerun calculate/get/validation and verify monolithic_test_files and superficial_test_implementation disappear or drop below warning thresholds, tests.count increases with focused automated validations, coverage/depth improve from 0.231/1.0, and score improves from 41.
