# Active transition catalog

This is the operator-facing projection of
[`../../.vrooli/swarm-transitions/registry.json`](../../.vrooli/swarm-transitions/registry.json),
the machine-readable source of truth. The registry decides the active kind,
input contract, terminal outcomes, prerequisites, and apply action; this page
exists to make that contract discoverable without reverse-engineering code.

## How to classify an interaction

```text
person writes and interprets conversation -> Agent Session backed by a Run
code needs an agent result                -> declared Agent Manager workflow
no agent judgment                         -> deterministic Swarm domain action
```

For workflow transitions, Swarm builds the immutable input snapshot and
performs the authorized, evidence-aware, exact-once apply. Agent Manager owns
the prompt, Runs, validation, routing, waits, retries, and provenance.

## Active transitions

| Transition | Kind | Active contract / owner | Swarm terminal apply |
| --- | --- | --- | --- |
| `session.meta_orchestration` | Session / Run | `session-input/v1`; human-led planning | `apply_session_proposal` |
| `session.swarm_operations` | Session / Run | `session-input/v1`; human-led operations | `apply_session_proposal` |
| `session.workflow_authoring` | Session / Run | `session-input/v1`; human-led workflow design and proposal | `apply_session_proposal` |
| `proposal.apply` | Deterministic | `proposal-apply-input/v1` | `apply_proposal` |
| `capture.classify` | Workflow | `swarm-manager/capture-classify` | `apply_capture_classification` |
| `backlog.refine` | Workflow | `swarm-manager/backlog-workshop-round` | `apply_backlog_refinement` |
| `backlog.clarify` | Workflow | `swarm-manager/backlog-clarify` | `apply_backlog_clarification` |
| `plan.author` | Workflow | `swarm-manager/plan-author`; Plan Manager validates | `bind_validated_plan_ref` |
| `plan.repair` | Workflow | `swarm-manager/plan-repair`; Plan Manager validates | `bind_validated_plan_ref` |
| `research.refine` | Workflow | `swarm-manager/backlog-workshop-round` | `apply_research_refinement` |
| `research.conclude` | Workflow | `swarm-manager/research-conclude` | `apply_research_conclusion` |
| `plan.execute` | Workflow | `swarm-manager/phased-plan-drain`; evidence-gated | `apply_plan_execution` |
| `work.review` | Workflow | `swarm-manager/independent-review`; evidence-gated | `apply_review_outcome` |
| `review.evidence_request` | Workflow | `swarm-manager/evidence-request` | `apply_review_evidence_request` |
| `work.correct` | Workflow | `swarm-manager/work-correct`; evidence-gated | `apply_correction_outcome` |
| `work.fix_and_revalidate` | Workflow | `swarm-manager/fix-and-revalidate`; evidence-gated | `apply_validation_outcome` |
| `work.retry` | Workflow | `swarm-manager/phased-plan-drain` | `apply_plan_execution` |
| `work.follow_up` | Workflow | `swarm-manager/work-follow-up` | `apply_follow_up` |
| `work.control` | Deterministic | `workflow-control-input/v1` | `control_workflow_execution` |
| `initiative.discover` | Workflow | `swarm-manager/initiative-discover` | `apply_initiative_proposal` |
| `initiative.plan` | Workflow | `swarm-manager/initiative-plan`; Plan Manager validates | `bind_initiative_plan_ref` |
| `initiative.execute` | Workflow | `swarm-manager/initiative-execute`; evidence-gated | `apply_initiative_execution` |
| `initiative.review` | Workflow | `swarm-manager/initiative-review`; evidence-gated | `apply_initiative_review` |
| `scenario.spec_sync` | Workflow | `swarm-manager/scenario-spec-sync` | `apply_scenario_spec_sync` |

The table deliberately does not manufacture a one-to-one replacement for
retired operating modes. Workflows are reusable capabilities selected by the
transition's authority and typed contract. The current prompt owner and
declaration status are maintained with the workflow source; a workflow is not
ready to start until Agent Manager has reconciled its declaration and resolved
its pinned prompt provenance.

## Historical status

Operating-mode and agent-operation records are read-only historical provenance.
They must not be selected for new work, used as active session advice, or
treated as an alternate orchestration runtime. See the
[cutover inventory](../operations/migration/WORKFLOW-CUTOVER-INVENTORY.md) for
retirement evidence and the [Target Operating Model](../concepts/TARGET-OPERATING-MODEL.md)
for the normative ownership boundary.
