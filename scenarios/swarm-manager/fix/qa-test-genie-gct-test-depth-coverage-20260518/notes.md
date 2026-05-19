# QA evidence

Fresh GCT completeness run: `vrooli scenario completeness score calculate test-genie --json` persisted a snapshot, then `vrooli scenario completeness score get test-genie --json` calculated at 2026-05-18T10:02:07Z.

Result: functional_incomplete, final score 44, base_score 59, validation_penalty 15.

Test-depth evidence:
- Raw test pass rate is green: 3/3 passing, rate 1.0, 15/15 quality points.
- Test quantity is still below threshold: count 3, 0/3 quantity points.
- Test coverage ratio is 1.0, 4/8 coverage points.
- Depth score avg_depth is 1.0, 2/7 coverage points.
- Validation issue: insufficient_test_coverage, medium severity, penalty 5, ratio 1.0, message: Suspicious 1:1 test-to-requirement ratio (3 tests for 3 requirements). Expected: 1.5-2.0x ratio with diverse test sources.
- Recommendations: add more tests to reach good threshold (impact 2); add tests to reach optimal 2:1 test-to-requirement ratio (impact 3).

Acceptance check: rerun `vrooli scenario completeness score calculate test-genie`, `vrooli scenario completeness score get test-genie --json`, and `vrooli scenario completeness score validation test-genie --json`. The insufficient_test_coverage issue should clear, the test-to-requirement ratio should move toward 2:1, and avg_depth should improve from 1.0 toward 3.0+.