---
name: "unit-health-improve"
description: "Calibrate Unit Health test-quality observations with independent holdouts and bounded mutation evidence."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["unit-health", "calibration", "holdout", "mutation"]
  status: "active"
  revision: 1
  requires:
    scenarios: ["unit-health"]
    commands: ["test-quality-calibrate"]
---
## Tools focus: Unit Health calibration

Use this skill to improve review evidence while preserving advisory status. Keep
the rule version, cohort policy, prompt version, and label vocabulary fixed
before comparing observations.

### Workflow

1. Run `unit-health.test-quality-sample` with a fixed seed and source identity.
2. Select a reviewed holdout after the policy and prompt are frozen.
3. Label each holdout independently before inspecting static or AI observations.
4. Run `test-quality-calibrate --holdout <file>` and inspect denominators, unknowns, false positives, false negatives, and disagreements.
5. Run mutation experiments only in a disposable workspace. Classify killed, survived, invalid, equivalent, out-of-contract, infrastructure-failure, and unknown outcomes.
6. Record an owner promotion decision. Comparison and mutation evidence never promote a rule automatically.

### Authority boundary

Write calibration receipts under the plan artifact directory. Do not edit source
tests, mutate the shared worktree, send secrets to a provider, or relabel a
development fixture as a reviewed holdout.

### Troubleshooting & Edge Cases

Missing labels, source drift, malformed model labels, and provider outages stay
unknown. A surviving valid mutant is a review finding, not an automatic defect.
