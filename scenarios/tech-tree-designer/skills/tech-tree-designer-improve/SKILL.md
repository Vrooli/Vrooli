---
name: tech-tree-designer-improve
description: "Develop Tech Tree Designer against its selected ecosystem-design outcomes: preserve measurement gaps, choose the owning repair layer, and iterate within one authorized mandate."
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: [practice]
  tags: [ecosystem, development, evidence, proposals, improvement]
  status: active
  revision: 2
  requires:
    scenarios: [tech-tree-designer, program-runtime]
    commands: [program-runtime library run, prompt-manager skill read]
  origin:
    kind: authored
---
## Practice focus: Tech Tree Designer Development

### 1. Focus and scope

Develop the selected ecosystem-design outcomes, not a higher diagnostic score.
This role is explicitly requested for scenario development; fleet applicability
counts do not limit that request. It does not authorize product work by itself.
Read `path:scenarios/tech-tree-designer/docs/START-HERE.md` and
`path:docs/agent-system/SCENARIO_DEVELOPMENT.md`. Use
`prompt-manager skill read scenario-improvement-campaign` for execution under an
approved mandate. Observation-only callers report the next repair without acting.

### 2. Setpoint

The PRD owns outcomes. The selected mandate owns which are required.
`path:scenarios/tech-tree-designer/docs/internal/TESTING.md#outcome-evidence-inventory`
maps every outcome to its measurement gap, protocol and repair owner.
Run `tech-tree-designer.setpoint-read` through `program-runtime library run`.
Its contract owns row IDs, output semantics and bounds; its 18 outcome IDs match
the PRD's `OT-*` IDs, without folding optional tiers into P0 acceptance.
As of 2026-09-11 `tech-tree-designer.setpoint-read` exceeds its 60-second
wall-clock ceiling and returns no board. The first intervention is to make it
read; do not estimate rows while the board is absent.
Numeric acceptance for OT-P0-008 is undecided, separately from sensor absence.
Registry and ontology readings are diagnostic-only and never satisfy OT-P0-009.

### 3. Sensors

The board consumes existing proto-plan inventory, ontology coverage and the
Prompt Manager skill-set program. Its diagnostic bindings do not run product
qualification. Read the child envelope and unresolved readings, not its process
exit status. Source errors invalidate coverage; a count without a target stays
unbanded. Neither a skill declaration nor registered presence proves skill quality.

Use `business-health validate scenario tech-tree-designer` for contract/evidence
diagnostics, `vrooli scenario requirements validate tech-tree-designer` for registry
structure, and scoped Test Genie phases under `path:docs/TESTING.md` for fresh
validation. Accept product evidence only against the selected target revision,
applicable cohort, complete case set, freshness and owner validity contract.
Historical foundation receipts do not cover expanded proposal semantics.

### 4. Golden corpora

No approved product corpus floor exists yet. The protocol matrix in TESTING.md
and proposed cohorts in PERFORMANCE.md define what qualification must cover,
not a passing baseline. Program contract fixtures and local board regressions
test evidence handling only. Preserve adversarial and held-out cases; do not
drop a difficult target kind, machine cohort or interruption point to pass.

### 5. Routes

Read PROBLEMS.md each cycle as well as the board. Choose the first applicable
row below; within that row choose the lowest selected PRD ID. Implement successive
in-scope repairs under the same mandate. Do not create per-repair approval items.

| First applicable condition | Owner / method | Expected evidence change |
| --- | --- | --- |
| Cancellation, exhausted budget or missing authority | Active engagement owner | Durable checkpoint and explicit unmet boundary; no further effects |
| Proposed change alters selected targets, floors or granted effects | Mandate amendment through Swarm | Reviewed amendment, never a self-approved new finish line |
| Applicable directive conflicts with the PRD | `scenario-work-ladder` W0 | Resolved contract applicability before dependent work |
| Requirement mapping is wrong | `scenario-work-ladder` W1 | Obligation links match the governing target |
| Existing completion claim lacks evidence | `requirements-traceability-steer` W2 | Located or freshly executed qualifying evidence; unsupported claims remain unresolved |
| Selected outcome has `pending_telemetry` | `measures-adoption`, then owning domain implementation | Owner-defined executable measurement with validity and cohort checks |
| Existing sensor lacks a governed binding | Binding owner through `scenario-work-ladder` | Executable governed read; no private HTTP substitute |
| A diagnostic read failed | Owning scenario; `scientific-debugging` if cause unknown | Reproducible failure and restored comparable reading |
| Valid outcome evidence is out of band | Owning graph/planning/ontology domain; choose W3 via `scenario-work-ladder` | Behavior and its protected regression pass, then remeasure |
| Board has no actionable selected row but ledger has in-scope capability debt | Owning method for the first dependency-blocking ledger entry | Capability/evidence gap closed without inventing a band |

Cross-owner repair requires the actual grant. When it is absent, return the
prerequisite to the existing engagement; do not silently expand scope.

### 6. Anti-gaming

Apply `prompt-manager skill read improvement-do-and-dont`: D1, D2, D3 and its
skeptic test. Do not edit requirements to match unsupported completion claims,
count proto-only materialization as repository-wide apply, count authored links
as verified fulfillment, band current measurements as targets, or treat missing
rows as satisfied. Preserve pinned revisions, failures and known-issue history.

### 7. Evidence

Retain target identity, selected outcomes, before/after receipts, method changes,
unmet obligations and the next authorized action in the active engagement's log.
Reuse automatic capture. For direct sessions use the shared Memory work-record
protocol via `prompt-manager skill read vrooli-memory`; do not create a parallel
ledger. Preserve Program Runtime IDs and Test Genie run IDs, not just summaries.

### 8. Stop rules

Completion needs every selected required outcome and its owner validity rules.
An `ok` board, readable documentation or passing setup fixture is insufficient.
Missing targets or measurements remain unmet. Implement permanent prerequisites
within authority; do not retry them as transient outages. Use owner waits for
pending work. Honor aggregate budgets and cancellation, and checkpoint when a
material amendment is needed. No universal number of improvement cycles proves
completion. The setup review proposal in DECISIONS.md is not an approved mandate.

### Troubleshooting & Edge Cases

| Symptom | Response |
| --- | --- |
| All outcome rows unknown | Expected until owner sensors ship; take the selected telemetry route, not a success exit. |
| Setup fixture passes but product test fails | Keep the product failure; setup evidence covers only the reader. |
| Swarm development launch is not qualified | Finish authorized setup and retain its evidence; do not launch or substitute an ungoverned loop. |
| No approved scale floor | Retain `target: null`; review the proposal in PERFORMANCE.md before acceptance. |
| Child program is partial or malformed | Preserve failure and unaffected diagnostics; do not accept child presence flags as complete evidence. |
