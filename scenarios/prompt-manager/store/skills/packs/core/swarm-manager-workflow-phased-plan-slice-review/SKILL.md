---
name: "swarm-manager-workflow-phased-plan-slice-review"
description: "Typed prompt contract for independent plan-slice review."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["swarm-manager","agent-manager","workflow","prompt-contract"]
  status: "active"
  revision: 1
  createdAt: "2026-07-18T03:05:27Z"
  updatedAt: "2026-07-18T03:05:27Z"
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
| The handoff shows the slice done per the plan, with verification quoted: the handoff states the commands run and their observed pass results. | `true` |
| A gap exists between the plan's expectations and the handoff's evidence: missing verification, skipped scope, or claims without support. A named command without its observed result is a claim without support. | `false` |

`note` rule: one actionable sentence. On `false`, name the specific gap — the executing agent receives your note as its correction instruction and has no other context.

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
