# Current validation problems

This is the maintained problems register for deployment-manager; it records only
environmental or implementation findings that still reproduce.

| Finding | Cause | Scope |
|---|---|---|
| Go aggregate coverage | The final aggregate behavior-coverage runs meet the declared policy: 75.5% API (3,692/4,889 statements) and 75.1% CLI (1,597/2,127 statements). Per-package coverage and compatibility-orchestration branches remain advisory follow-up debt. | Remaining implementation |
| Test architecture debt | Unit-health still reports one injectable-seam warning for package-level process construction and advisory traceability/coverage debt. The declared thresholds remain unchanged. | Remaining implementation |

No suppression, waiver, or lowered threshold is used for these findings.

## Work ladder

- Rung: W3
- Evidence: The registered goal now has P0 contract coverage in `OT-P0-038` and `OT-P0-039`; `business-health validate scenario deployment-manager` is L3 with zero findings, the business phase run `20260805-045327-80af2739` is clean, requirements sync reports 42/104 complete with fresh evidence, and Test Genie run `20260805-043919-d4ca82e1` reports 19/19 phases passed.
- Blocker: The scenario ladder is closed for the current contract. Provider maturity reports retain advisory debt, but the active storage validation provider is runnable and is not a release blocker.
- Measured: 2026-08-05
