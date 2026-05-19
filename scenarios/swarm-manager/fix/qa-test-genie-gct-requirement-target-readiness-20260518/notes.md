# QA evidence

Fresh GCT completeness run: `vrooli scenario completeness score calculate test-genie --json` persisted a snapshot, then `vrooli scenario completeness score get test-genie --json` calculated at 2026-05-18T10:02:07Z.

Result: functional_incomplete, final score 44, base_score 59, validation_penalty 15.

Spec-readiness evidence:
- Requirements: 1/3 passing, rate 0.3333, 7/20 quality points.
- Operational targets: 1/3 passing, rate 0.3333, 5/15 quality points.
- Requirement quantity: count 3, threshold below, 1/4 quantity points.
- Target quantity: count 3, threshold below, 1/3 quantity points.
- Validation issue: ungrouped_operational_targets, high severity, penalty 10, count 3, ratio 1.0, message: 100% of operational targets have 1:1 requirement mapping (max 15% recommended).
- Recommendations: increase requirement pass rate to 90%+ (impact 5); increase operational target pass rate to 90%+ (impact 4).

Acceptance check: rerun `vrooli scenario completeness score calculate test-genie`, `vrooli scenario completeness score get test-genie --json`, and `vrooli scenario completeness score validation test-genie --json`. The ungrouped_operational_targets issue should clear and requirement/target pass rates should be 90%+.