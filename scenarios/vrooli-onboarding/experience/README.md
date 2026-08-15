# experience/ — UX contract

This folder is the design-first sibling of [`requirements/`](../requirements/).
Requirements say what the scenario **does**; these specs say what the UI must
**communicate**, and in which states.

Validate after any edit:

```bash
experience-manager spec validate vrooli-onboarding --json
```

## What is declared here

| Kind | Count | Covers |
|---|---|---|
| Pages | 10 | The eight setup steps, plus health and glossary |
| Journeys | 6 | First run, re-entry, headless credential provisioning, safeguard consent, degraded continue, surface parity |

Each page declares its identity and route, the communication priorities it must
serve, its **states** (including the failure states), its **elements** with ARIA
roles, its **claims**, and the **bindings** that let a claim be checked against a
captured accessibility tree.

## Why the failure states are first-class

Most of the value in these specs is in the states that only appear when
something is wrong: `catalog-unavailable`, `no-backend`, `config-invalid`,
`partially-applied`, `probe-unavailable`, `blocked`. Those are the moments where
an operator either recovers or gives up, and they are exactly the states an
undeclared contract never checks.

Several claims are deliberately negative — `value-never-rendered`,
`no-binding-controls`, `degraded-continue-only-when-required-are-ready`. A
negative claim is written as an observable absence in a named state, never as a
bare "X does not exist", because only the former is checkable.

## Status and tiers

Every page is currently `draft`: the intent is authored ahead of the target
build, so reconciliation runs **advisory**. A page moves to `active` when its
build conforms, and from then on its machine-tier claims gate CI.

Claim tiers mean what they say:

- `machine` — deterministically checkable against the captured accessibility
  tree. These gate once the page is active.
- `manual` — human-attested against a recorded journey run, with expiry. Used
  where the claim is about the run as a whole ("the operator never edits a
  file"), which no DOM assertion can express.
- `aspirational` — stated intent with no check. Advisory only, never a failure.

## What is not here yet

**Component specs.** A component spec must reference a catalog story or examples
entry, and onboarding's primitives are still scenario-local copies rather than
shared-library components. The component specs land together with the
shared-library adoption that gives them specimens to reference
(`ONB-UX-DESIGN-SYSTEM`).

**BAS cases.** Each declared journey carries the spec entry id in its labels
(`ONB-UX-JOURNEY-EVIDENCE`). The six recorded runs are exported as
`bas/evidence/experience-journeys.json`; each receipt includes the BAS run id,
video artifact path, byte count, and SHA-256. Those recordings are also the
launch demo and the desktop evidence handed to deployment-manager. The
accessibility matrix remains a separate gate until its all-steps run is
recorded.

## Order of work

A UX change lands in this order:

1. Declare the state and the claim here.
2. Validate the contract.
3. Build until the claim holds.
4. Flip the page to `active` so the claim gates.

Adding a state to the build without declaring it here means it is never audited
— which is how error states end up unstyled, unannounced, and unreachable by
keyboard.
