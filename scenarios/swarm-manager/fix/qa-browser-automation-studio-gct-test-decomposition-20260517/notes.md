# GCT evidence

Fresh command sequence on 2026-05-17T04:02:10Z:

- `vrooli scenario completeness score calculate browser-automation-studio`
- `vrooli scenario completeness score get browser-automation-studio --json`
- `vrooli scenario completeness score validation browser-automation-studio --json`
- `vrooli scenario completeness score recommend browser-automation-studio --json`

Score result: score 38, base_score 46, validation_penalty 8, classification `foundation_laid`.

Test decomposition evidence:

- tests: 9 for 63 requirements
- test_coverage_ratio: 0.143
- avg_depth: 1.106
- GCT recommendations: add more tests and reach optimal 2:1 test-to-requirement ratio

Validation issue:

- `monolithic_test_files`: severity medium, penalty 8, 4 violations. Worst offender `.vrooli/lighthouse.json` validates BAS-PERF-DASHBOARD-LOAD, BAS-A11Y-DASHBOARD, BAS-PERF-WORKFLOW-INTERACTIVE, BAS-PERF-PROJECTS-PAGE, and BAS-A11Y-MOBILE in one artifact.

Success check: rerun validation/get and verify monolithic_test_files disappears or drops below warning threshold, broad lighthouse-only evidence is replaced or supplemented with focused automated assertions, coverage/depth improve, and score improves from 38.
