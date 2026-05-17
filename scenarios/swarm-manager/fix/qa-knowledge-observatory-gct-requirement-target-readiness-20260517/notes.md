# GCT evidence

Fresh command sequence on 2026-05-17T04:01:54Z:

- `vrooli scenario completeness score calculate knowledge-observatory`
- `vrooli scenario completeness score get knowledge-observatory --json`
- `vrooli scenario completeness score validation knowledge-observatory --json`
- `vrooli scenario completeness score recommend knowledge-observatory --json`

Score result: score 52, base_score 69, validation_penalty 17, classification `functional_incomplete`.

Requirement/target readiness evidence:

- requirements: 25/44 passing, rate 0.568, 11/25 quality points
- targets: 25/44 passing, rate 0.568, 9/25 quality points
- GCT recommendation priority 1: increase requirement pass rate to 90%+
- GCT recommendation priority 2: increase operational target pass rate to 90%+

Validation issue tied to target decomposition:

- `ungrouped_operational_targets`: severity high, penalty 6, count 9, ratio 0.6. Message: 60% of operational targets have 1:1 requirement mapping; max recommended is 15%.

Success check: rerun calculate/get/validation and verify requirement and target pass rates reach at least 90%, ungrouped_operational_targets disappears or drops below warning threshold, and score improves from 52.
