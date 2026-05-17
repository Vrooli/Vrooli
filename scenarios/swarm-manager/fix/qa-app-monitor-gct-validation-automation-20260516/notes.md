# GCT evidence

Fresh command sequence on 2026-05-16T22:01:39Z:

- `vrooli scenario completeness score calculate app-monitor --json`
- `vrooli scenario completeness score get app-monitor --json`
- `vrooli scenario completeness score validation app-monitor --json`
- `vrooli scenario completeness score recommend app-monitor --json`

Score result: score 33, base_score 56, validation_penalty 23, classification `foundation_laid`.

Automation evidence:

- tests: 3/6 passing
- test_pass_rate points: 8/15
- GCT recommendation priority 1: increase test pass rate to 90%+
- GCT recommendation priority 4/5: add more tests and reach optimal 2:1 test-to-requirement ratio

Validation issues:

- `missing_test_automation`: severity medium, penalty 11, count 12, ratio 0.6, complete_with_manual 6. Message: 60% of validations are manual; max recommended is 10%.
- `monolithic_test_files`: severity medium, penalty 2. Worst offender `ui/src/components/views/HomeView.tsx` validates AM-P0-001, AM-P0-002, AM-P0-004, and AM-P1-005 in one file.

Success check: rerun calculate/get/validation and verify missing_test_automation and monolithic_test_files disappear or drop below warning thresholds, test pass rate improves from 3/6 to at least 90%, tests.count increases with focused automated API/UI/e2e validations, and score improves from 33.
