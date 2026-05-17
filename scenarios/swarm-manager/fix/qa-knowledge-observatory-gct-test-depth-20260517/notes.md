# GCT evidence

Fresh command sequence on 2026-05-17T04:01:54Z:

- `vrooli scenario completeness score calculate knowledge-observatory`
- `vrooli scenario completeness score get knowledge-observatory --json`
- `vrooli scenario completeness score validation knowledge-observatory --json`
- `vrooli scenario completeness score recommend knowledge-observatory --json`

Score result: score 52, base_score 69, validation_penalty 17, classification `functional_incomplete`.

Test depth evidence:

- tests: 3/3 passing, but only 3 tests for 44 requirements
- test_coverage_ratio: 0.068
- avg_depth: 1.0
- GCT recommendations: add more tests and reach optimal 2:1 test-to-requirement ratio

Validation issues:

- `monolithic_test_files`: severity high, penalty 6, 3 violations. Worst offender `api/metrics_test.go` validates KO-P0-002 and KO-QM-001 through KO-QM-005 in one file.
- `superficial_test_implementation`: severity medium, penalty 5, count 5. Flagged files: `api/search_performance_test.go`, `api/graph_api_test.go`, `api/graph_performance_test.go`, `api/api_contract_test.go`, `api/cors_test.go`.

Success check: rerun calculate/get/validation and verify monolithic_test_files and superficial_test_implementation disappear or drop below warning thresholds, tests.count increases with focused automated validations, coverage/depth improve from 0.068/1.0, and score improves from 52.
