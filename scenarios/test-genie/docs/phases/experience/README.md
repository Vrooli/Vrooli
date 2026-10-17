# Experience Phase

The `experience` phase delegates to `experience-manager` through the shared
`scenario-validation/v1` provider contract.

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton.

## North Star

The experience spec is a trustworthy, machine-checkable contract whose active
machine-tier claims are grounded in the live built UI: *the built UI structure
satisfies the machine-tier experience spec*. Manual-tier claims carry honest,
current attestations, and deferred perception debt (visual saliency, glance
judgment) is visible without failing suites. At maximum maturity every capability
ladder is at its top rung — `spec_contract` L3 (contract clean),
`structure_reconciliation` L3 (structure clean), `manual_evidence` L2
(attestations fresh), and `perception_advisory` surfacing advisory perception
evidence — so the scenario's declared experience intent is provably realized in
the shipped UI.

## The rungs and their gates

Each capability declares a monotone ladder (each rung implies the one below). The
phase standing surfaces the focus capability's rung and its single next unlock.

**`spec_contract`** — is the experience contract parseable and internally consistent?
- L0 Contract unavailable → *Add an `experience/index.json` contract.*
- L1 Contract parseable → *Resolve schema and index parity findings.*
- L2 References linked → *Ground active pages in structure evidence.*
- L3 Contract clean → maximum; experience intent is a trustworthy machine-checkable contract.

**`structure_reconciliation`** — are machine-tier claims grounded in captured UI structure?
- L0 Capture unavailable → *Capture the target page accessibility tree.*
- L1 Bindings joinable → *Resolve binding and checker coverage gaps.*
- L2 Claims checked → *Resolve failed or unproven claims.*
- L3 Structure clean → maximum; the built UI satisfies the machine-tier spec.

**`manual_evidence`** — do manual-tier claims carry fresh attestations?
- L0 No attestation evidence → *Record an attestation with author, rationale, and expiry.*
- L1 Attestations present → *Refresh expired attestations.*
- L2 Attestations fresh → maximum; manual claims are honestly current.

**`perception_advisory`** (deferred, non-gating)
- L0 Perception unevaluated → *Calibrate deterministic perception checks in the P2 tier.*
- L1 Advisory perception visible → perception debt is shown without failing suites.

## What each finding means

Each finding caps the named capability at a rung; only ERROR/BLOCKER severities
fail the phase, so warnings and advisories are honest, non-failing debt.

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `experience.schema_invalid` | spec_contract | L1 | ERROR | Yes |
| `experience.index_parity` | spec_contract | L1 | ERROR | Yes |
| `experience.ref_unresolved` / `experience.prd_ref_unmatched` | spec_contract | L2 | ERROR | Yes |
| `experience.binding_orphan` / `experience.tier_violation` | spec_contract | L2 | ERROR | Yes |
| `experience.route_unspecced` / `experience.state_missing` | spec_contract | L2 | WARNING | No |
| `experience.binding_unresolved` | structure_reconciliation | L1 | ERROR | Yes |
| `experience.claim_failed` / `experience.affordance_missing` | structure_reconciliation | L2 | ERROR | Yes |
| `experience.floor_*` (overflow, viewport_fill, chrome_pinned, tap_target_size, …) | structure_reconciliation | L2 | ERROR | Yes |
| `experience.claim_unverifiable` | structure_reconciliation | L1 | WARNING | No |
| `experience.claim_unproven` | structure_reconciliation | L2 | WARNING | No |
| `experience.capture_unavailable` | structure_reconciliation | L0 | INFO | No |
| `experience.attestation_expired` | manual_evidence | L1 | WARNING | No |
| `experience.importance_mismatch` / `experience.glance_judge_mismatch` | perception_advisory | L1 | WARNING (advisory) | No |

## The canonical fix

- **Spec-contract findings** (`schema_invalid`, `index_parity`, `ref_unresolved`,
  `prd_ref_unmatched`, `binding_orphan`, `tier_violation`, `route_unspecced`,
  `state_missing`) → source-level edits to `experience/index.json`,
  `experience/pages/*.json`, and `experience/journeys/*.json`: fix the schema
  shape, reconcile the index with the documents it lists, point references at the
  intended page/state/element, match PRD operational targets, repair the bound UI
  contract, and set the deliberate claim tier. Load the `experience-spec-authoring`
  skill; these are author-intent fixes, not mechanical ones.
- **Structure-reconciliation findings** (`binding_unresolved`, `claim_failed`,
  `affordance_missing`, `floor_*`) → reconcile the built UI against the machine-tier
  spec: fix the selector/markup a binding resolves against, change the
  product/design/implementation so the failed claim passes, add the missing
  accessible affordance, or correct the layout so the shell floor (no horizontal
  overflow, viewport fill, pinned chrome, tap-target size) holds. Add the `ux`
  skill for affordance/layout floors.
- **`claim_unverifiable` / `claim_unproven`** → choose a supported claim type (or
  extend the checker), then capture the evidence — or deliberately re-tier the
  claim rather than leaving it unproven.
- **`attestation_expired`** → record a fresh human attestation (author, rationale,
  expiry) for the manual-tier claim.
- **`capture_unavailable`** → an operational condition, not a source fix: make a
  BAS accessibility capture reachable so machine-tier claims can be checked.
- **`importance_mismatch` / `glance_judge_mismatch`** → perception-tier debt,
  deferred to the P2 tier; visible but non-gating, no fix required for v1.

## How to verify

```bash
# See the current rung, gaps, and next move for every capability:
experience-manager validate scenario <scenario>

# Or drive it through Test Genie and read the per-phase scorecard + findings:
test-genie execute <scenario> --phases experience
test-genie runs findings --scenario <scenario>
```

The `experience` line in the scorecard shows the focus capability's current rung,
the single highest-unlock next move, and a runnable doc-search topic that resolves
back to the sections above.

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

Experience Manager always reports ERROR-severity findings as failed validation
truth. Test Genie may independently mark that failure advisory or non-gating
for a rollout, but it must retain the failed/degraded provider state and its
remediation; no environment variable may rewrite an ERROR into `PASSED`.

## Degradation

Parser-era checks run without Browser Automation Studio. Later structure
reconciliation may require BAS captures; BAS unavailability should produce
skipped or informational findings, never a failed phase caused only by missing
capture infrastructure.
