---
name: "swarm-manager-workflow-phased-plan-slice-review"
description: "Typed prompt contract for independent plan-slice review."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["swarm-manager","agent-manager","workflow","prompt-contract"]
  status: "active"
  revision: 6
  createdAt: "2026-07-18T03:05:27Z"
  updatedAt: "2026-08-29T00:00:00Z"
  modes: ["contract"]
  requires:
    scenarios: []
    commands: []
  origin:
    kind: "authored"
---
# Phased Plan Slice Review

Review one slice handoff against the plan it claims to advance. Resolve the plan with `plan-manager plans render <plan_reference>` and compare the handoff's claims to the plan's expectations for that slice.

## Decision table

| Observable end state | `accepted` |
| --- | --- |
| The handoff shows the slice done per the plan, with verification quoted: the handoff states the commands run and their observed pass results. A terminal Plan Manager validation operation with its producer run id and verdict is valid evidence for the validation it governs; do not require a second literal command. | `true` |
| The handoff shows an authored phase done and verified, then explicitly stops for operator approval before terminal DoD. | `true` — approval and terminal DoD are future workflow obligations, not evidence this pre-approval slice can already possess. |
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
