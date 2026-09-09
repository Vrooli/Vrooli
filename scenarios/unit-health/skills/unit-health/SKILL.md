---
name: "unit-health"
description: "Operate Unit Health validation and bounded test-quality review with explicit static, runtime, and advisory evidence."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["unit-health", "testing", "quality", "evidence"]
  status: "active"
  revision: 1
  requires:
    scenarios: ["unit-health"]
    commands: ["unit-health validate scenario"]
    skills: ["test", "unit-testing-architecture-steer"]
---
## Tools focus: Unit Health validation

Use Unit Health as the owner of static test-quality assessment and its evidence
projection. Read the canonical authoring guide before changing test structure:
`path:docs/testing/UNIT-TEST-AUTHORING.md`.

### Scope

In scope: `unit-health validate scenario <name>`, static quality findings, native
assertion activity, requirement links, and bounded sampled-review evidence. Out
of scope: editing tests through Unit Health, treating AI labels as proof, or
using missing execution as a clean result.

### Decision table

| State | Action |
|---|---|
| Need structural findings | Run `unit-health validate scenario <name>` and inspect rule, status, reason, and support profile. |
| Need runtime evidence | Request the relevant Test Genie phase; keep its receipt separate from static results. |
| Need semantic prioritization | Use `unit-health.test-quality-sample` with a fixed source identity and seed. |
| Sample is partial, stale, or unavailable | Report the status and limitation; do not infer a clean cohort. |
| Need a rule promotion | Use the improve skill and independent holdout evidence. |

### Output expectations

Keep source and test identities with every observation. Report denominator,
selection policy, seed, controls, model/profile evidence, and limitations. Keep
sampled review advisory. Use human-readable CLI output unless machine output is
needed for a bounded receipt.

### Troubleshooting & Edge Cases

- `NOT_EXECUTED` means runtime evidence is absent. Run Test Genie through its owner lifecycle.
- A changed source digest makes an old cohort non-comparable. Select a new cohort.
- Unsupported syntax or unavailable dependencies remain `unknown`.
- Provider errors remain `unavailable` or `partial`; never convert them to clean.
