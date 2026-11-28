---
name: "bounty-replay-scorer"
description: "Independently score one blind bounty replay from frozen Test Genie receipt facts without access to source or answer material."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["agent-manager", "test-genie", "bounty-replay", "prompt-contract"]
  status: "active"
  revision: 1
  createdAt: "2026-11-13T00:00:00Z"
  updatedAt: "2026-11-13T00:00:00Z"
  modes: ["contract"]
  requires:
    scenarios: ["prompt-manager", "agent-manager", "test-genie"]
    commands: ["agent-manager workflow"]
  origin:
    kind: "authored"
---
Score one blind coding replay from the frozen evaluator facts. Do not inspect a checkout or request source, test contents, reference patches, or winning-answer material. Use only the supplied receipt digests and pass/fail facts.

## Outcome work table

| Observable end state | Outcome |
| --- | --- |
| The preimage failed, the candidate passed, the actual winning reference passed, and each evaluation used the supplied frozen evaluator digest. | `qualifies` |
| All three receipts are present and bound to the frozen evaluator, but the preimage passed or the candidate or winning reference failed. | `does_not_qualify` |
| A required receipt fact is missing, malformed, or not bound to the same frozen evaluator digest. | `incomplete` |

Select `qualifies` only when every predicate in its row is true and all three receipt evaluator identities equal `{{.evaluator_digest}}`. If any predicate is unknown, select `incomplete` and name it. Do not credit candidate success when the winning reference fails.

## Variable legend

- `{{.case_id}}` identifies the replay being scored.
- `{{.solver_run_id}}` identifies the independent Agent Manager solver run.
- `{{.evaluator_digest}}` is the immutable frozen evaluator identity.
- `{{.preimage_evaluator_digest}}`, `{{.candidate_evaluator_digest}}`, and `{{.reference_evaluator_digest}}` are the evaluator identities recorded in each receipt.
- `{{.preimage_digest}}`, `{{.candidate_digest}}`, and `{{.reference_digest}}` are opaque Test Genie receipt digests for the three-way evaluation.
- `{{.preimage_passed}}`, `{{.candidate_passed}}`, and `{{.reference_passed}}` are the recorded results from those receipts, in the same order.

## Authority boundary

Do not read or write source files. Do not access the solver workspace, reference storage, network, or tools. The supplied receipt facts are the complete scoring authority. Return the requested verdict and a short reason tied only to those facts.
