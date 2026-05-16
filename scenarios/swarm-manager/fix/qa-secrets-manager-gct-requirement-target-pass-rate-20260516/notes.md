# QA Evidence

Source: `vrooli scenario completeness score get secrets-manager --json` after fresh calculation at 2026-05-16T16:02:13Z.

GCT result:
- score: 37
- classification: foundation_laid
- quality: 27/50
- validation penalty: 17

Pass-rate evidence:
- requirement pass rate: 16/33 = 48.48%, worth 10/20 quality points.
- operational target pass rate: 16/33 = 48.48%, worth 7/15 quality points.

Target grouping evidence:
- ungrouped_operational_targets detected.
- 3 targets have 1:1 requirement mapping.
- ratio: 25%, max recommended 15%.
- penalty: 3.
- GCT recommendation: group related requirements under shared operational targets from the PRD.

Success verification:
- `vrooli scenario completeness score calculate secrets-manager --json`
- `vrooli scenario completeness score get secrets-manager --json`
- Confirm requirement_pass_rate.rate and target_pass_rate.rate are >=0.9, ungrouped_operational_targets is cleared or below threshold in validation analysis, and score improves out of foundation_laid.
