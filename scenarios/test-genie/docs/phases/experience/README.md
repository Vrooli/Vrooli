# Experience Phase

The `experience` phase delegates to `experience-manager` through the shared
`scenario-validation/v1` provider contract.

## What It Checks

- Parses `experience/index.json`, `experience/pages/*.json`, and
  `experience/journeys/*.json` as `scenario-experience-spec/v1`.
- Enforces parser-era contract rules: schema shape, index parity,
  cross-document references, PRD operational-target references, binding
  integrity, and claim tier semantics.
- Reports maturity across the spec-contract, structure-reconciliation,
  manual-evidence, and deferred perception-advisory capability ladders.

## Applicability

The phase is presence-keyed. It applies only when the target scenario has an
`experience/` folder. Scenarios that have not adopted the experience axis are
not failed by this phase; their absence is fleet-sweep debt for
`experience-manager`, not a suite regression.

## Gating

`EXPERIENCE_ALIGNMENT_GATE=strict` makes error-severity experience findings
fail the shared provider response. The default rollout mode keeps findings
visible and maturity-scored without failing suites while the experience axis is
adopted fleet-wide.

## Degradation

Parser-era checks run without Browser Automation Studio. Later structure
reconciliation may require BAS captures; BAS unavailability should produce
skipped or informational findings, never a failed phase caused only by missing
capture infrastructure.
