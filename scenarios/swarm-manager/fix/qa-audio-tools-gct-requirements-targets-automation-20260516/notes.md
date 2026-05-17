# GCT evidence

Fresh command sequence on 2026-05-16T22:01:23Z:

- `vrooli scenario completeness score calculate audio-tools --json`
- `vrooli scenario completeness score get audio-tools --json`
- `vrooli scenario completeness score validation audio-tools --json`
- `vrooli scenario completeness score recommend audio-tools --json`

Score result: score 15, base_score 25, validation_penalty 10, classification `early_stage`.

Breakdown:

- quality 0/50
- coverage 2/15
- quantity 0/10
- UI 23/25

Metrics:

- requirements: 0/1 passing
- targets: 0/1 passing
- tests: 0/0 passing
- API beyond health: 3

Validation issue:

- `missing_test_automation`: 100% manual validations, penalty 10, ratio 1.0. Recommendation: replace manual validations with automated tests.

GCT recommendations:

1. Increase test pass rate to 90%+.
2. Increase requirement pass rate to 90%+.
3. Increase operational target pass rate to 90%+.
4. Add more tests to reach good threshold.
5. Add tests to reach optimal 2:1 test-to-requirement ratio.

Success check: rerun the calculate/get/validation commands and verify classification improves out of `early_stage`, requirement/target/test pass rates reach at least 90%, and the missing_test_automation validation issue is gone or non-blocking.
