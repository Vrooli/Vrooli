# Current validation problems

This is the maintained problems register for deployment-manager; it records only
environmental or implementation findings that still reproduce.

| Finding | Cause | Scope |
|---|---|---|
| Go aggregate coverage | The final aggregate behavior-coverage runs meet the declared policy: 75.5% API (3,692/4,889 statements) and 75.1% CLI (1,597/2,127 statements). Per-package coverage and compatibility-orchestration branches remain advisory follow-up debt. | Remaining implementation |
| Test architecture debt | Unit-health still reports one injectable-seam warning for package-level process construction and advisory traceability/coverage debt. The declared thresholds remain unchanged. | Remaining implementation |

No suppression, waiver, or lowered threshold is used for these findings.

## Work ladder

- Rung: W3 (learning capture and measurement implementation).
- Evidence: the named readiness goal requires an attributable learning loop; Deployment Manager OT-P0-044 supplies that contract. The archived cross-ramp goal and desktop OT-P0-004 cover agent tooling for the desktop ramp. Business and requirements gates passed on 2026-09-04. This task implements the operator-approved learning recommendations without changing release promises.
- Blocker: none for the learning setup; live outcome baselines remain unearned. Shared Memory UI/attestation findings are recorded in the learning progress entry.
- Measured: 2026-09-04
