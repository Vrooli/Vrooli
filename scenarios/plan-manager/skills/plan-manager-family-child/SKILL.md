---
name: plan-manager-family-child
description: Execute one admitted child plan and return evidence of the plan owner's acceptance.
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: [contract]
  tags: [plan-manager, plan-family, execution]
  status: active
  revision: 1
  origin: {kind: authored}
---
Execute the one admitted child plan in `{{.child}}`. Complete its intended outcome within its declared source and runtime claims. Return the structured result required by the workflow.

| Observed result | Outcome |
|---|---|
| Plan Manager reports the named execution complete and its required acceptance evidence is present | completed |
| Execution cannot advance because authority or required acceptance is missing | blocked |
| The supplied child identity or admitted scope is inconsistent | abstained |

`{{.child}}` contains the plan, execution and family identities selected by Plan Manager. Treat current Plan Manager state as authoritative. Do not infer acceptance from your own summary or from a terminal agent run.

Read `prompt-manager skill read implementation-plan-execution`. Apply its recall, divergence, validation and capture rules. Read `prompt-manager skill read plan-manager` for the receipt-based driver loop. Record one outcome-linked usage attempt through the owning usage skill.

Write only within the admitted plan boundary. Ask Plan Manager to record a justified boundary extension before changing that boundary. Do not change family membership, graph review, admission ownership or supervision policy. Return evidence references rather than transcript bodies. Select the conservative outcome when the affirmative predicate is unproven.
