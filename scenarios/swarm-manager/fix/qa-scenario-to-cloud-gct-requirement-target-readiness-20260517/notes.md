# GCT evidence

Fresh command sequence on 2026-05-17T04:02:02Z:

- `vrooli scenario completeness score calculate scenario-to-cloud`
- `vrooli scenario completeness score get scenario-to-cloud --json`
- `vrooli scenario completeness score validation scenario-to-cloud --json`
- `vrooli scenario completeness score recommend scenario-to-cloud --json`

Score result: score 41, base_score 49, validation_penalty 8, classification `functional_incomplete`.

Requirement/target readiness evidence:

- requirements: 1/13 passing, rate 0.077, 2/25 quality points
- targets: 0/8 passing, rate 0, 0/25 quality points
- GCT recommendation priority 1: increase requirement pass rate to 90%+
- GCT recommendation priority 2: increase operational target pass rate to 90%+

Validation issue tied to target decomposition:

- `ungrouped_operational_targets`: severity medium, penalty 4, count 3, ratio 0.375. Message: 37% of operational targets have 1:1 requirement mapping; max recommended is 15%.

Success check: rerun calculate/get/validation and verify requirement and target pass rates reach at least 90%, ungrouped_operational_targets disappears or drops below warning threshold, and score improves from 41.
