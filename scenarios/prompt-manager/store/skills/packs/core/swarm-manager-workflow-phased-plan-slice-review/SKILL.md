---
name: "swarm-manager-workflow-phased-plan-slice-review"
description: "Typed prompt contract for independent plan-slice review."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["swarm-manager","agent-manager","workflow","prompt-contract"]
  status: "active"
  revision: 7
  createdAt: "2026-07-18T03:05:27Z"
  updatedAt: "2026-09-10T00:00:00Z"
  modes: ["contract"]
  requires:
    scenarios: []
    commands: []
  origin:
    kind: "authored"
---
# Phased Plan Slice Review

Review one slice handoff against the plan it claims to advance. Resolve the plan with `plan-manager plans render <plan_reference>` and compare the handoff's claims to the plan's expectations for that slice.

Read cited owner receipts, outcome assessments, or retained test artifacts to check material claims. A narrated pass alone is insufficient. Apply the plan's completion policy: broad advisory findings do not override supported outcome evidence. Required product failures remain unmet.

## Decision table

| Observable end state | `accepted` |
| --- | --- |
| The handoff shows a coherent intervention completed within the accepted plan, with applicable verification and remaining outcomes. For an adaptive mandate, the current phase may remain unfinished. | `true` — intervention acceptance permits another bounded repair; it does not mark the phase or product complete. |
| The handoff shows an authored phase done and verified. It correctly distinguishes a routine phase boundary from a decision needing authority. | `true` — the accepted strategy and workflow own the next approval or continuation action. Do not demand per-phase human approval for adaptive work. |
| A goal-session handoff records several verified in-scope interventions with Plan Manager checkpoints and leaves the next frontier explicit. | `true` — warm-session continuity is acceptable evidence of progress; it does not make the harness's goal verdict the completion authority. |
| A goal-session handoff reports `usage_limited`, `budget_limited`, or `paused` while the Plan Manager frontier remains unfinished and no operator decision is pending. | `true` — classify it as `continue` when the accepted allowance permits another session; usage limitation is not a review failure. |
| A goal-session handoff claims progress but has no Plan Manager checkpoint, applicable verification, or supported remaining frontier. | `false` — name the missing checkpoint or evidence in the correction note. |
| The post-approval handoff shows fresh terminal Plan Manager validation, its producer run and verdict, and completed Plan Manager execution. | `true` — parent workflow success and Swarm consumer application happen only after this review accepts; never require those future effects as evidence from the slice. The parent owns proof that approval was signalled before dispatching this post-approval slice. |
| A gap exists between the plan's expectations and the handoff's evidence: missing verification, skipped scope, or claims without support. A named command without its observed result is a claim without support. | `false` |

`note` rule: one actionable sentence. On `false`, name the specific gap — the executing agent receives your note as its correction instruction and has no other context. Return the bare `accepted` and `note` object required by the run result schema; the workflow engine projects the terminal node value under its canonical `result` output key.

## Template variables

| Variable | Content |
| --- | --- |
| `{{.plan_reference}}` | Identity of the plan the slice executes. |
| `{{.handoff}}` | The slice handoff under review. |

## Boundary

This run is read-only. Do not modify files. Judge the handoff against the plan; do not re-litigate the plan itself.

Plan reference: {{.plan_reference}}

Handoff:
{{.handoff}}
