# GCT evidence

Fresh command sequence on 2026-05-16T22:01:39Z:

- `vrooli scenario completeness score calculate app-monitor --json`
- `vrooli scenario completeness score get app-monitor --json`
- `vrooli scenario completeness score validation app-monitor --json`
- `vrooli scenario completeness score recommend app-monitor --json`

Score result: score 33, base_score 56, validation_penalty 23, classification `foundation_laid`.

Breakdown:

- quality 21/50
- coverage 3/15
- quantity 8/10
- UI 24/25

Metrics:

- requirements: 7/19 passing
- targets: 7/19 passing
- tests: 3/6 passing
- last_test_run: 2026-05-04T03:15:51Z

Relevant validation issue:

- `ungrouped_operational_targets`: severity high, penalty 10, count 19, ratio 1.0. Message: 100% of operational targets have 1:1 requirement mapping; max recommended is 15%.

Why this item exists: the target/requirement model appears auto-expanded rather than grouped around cohesive business outcomes, and the low pass rates compound that readiness gap.

Success check: rerun calculate/get/validation and verify ungrouped_operational_targets disappears or drops below 15%, requirements and targets are grouped under PRD-level capabilities, requirement/target pass rates improve from 7/19, and score improves from 33.
