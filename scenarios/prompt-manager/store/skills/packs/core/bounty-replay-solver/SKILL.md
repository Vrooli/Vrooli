---
name: "bounty-replay-solver"
description: "Blindly implement one synthetic coding bounty in a disposable preimage checkout without access to the winning reference."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["agent-manager", "bounty-replay", "prompt-contract"]
  status: "active"
  revision: 1
  createdAt: "2026-11-13T00:00:00Z"
  updatedAt: "2026-11-13T00:00:00Z"
  modes: ["contract"]
  requires:
    scenarios: ["prompt-manager", "agent-manager", "workspace-sandbox"]
    commands: ["agent-manager workflow"]
  origin:
    kind: "authored"
---
Implement the supplied coding task in the disposable checkout. Use only the task and files in that checkout. Do not request, search for, infer, or retrieve the winning reference or answer. Report evidence from commands you actually ran.

## Outcome work table

| Observable end state | Outcome |
| --- | --- |
| A focused implementation is present in the checkout and relevant existing checks pass. | `solved` |
| The checkout is unavailable or the task cannot be attempted without access outside the supplied checkout. | `blocked` |
| Work began, but relevant checks fail, the implementation is partial, or the result cannot be confirmed. | `incomplete` |

When the evidence does not prove the `solved` row, select the conservative outcome and name the missing predicate.

## Variable legend

- `{{.case_id}}` is the stable replay case identifier.
- `{{.task}}` is the public task statement. Treat it as the full task authority.
- `{{.checkout}}` is the disposable project checkout path. It contains the preimage and no reference patch.

## Authority boundary

Write only inside the supplied disposable checkout. Do not access network services, other workspaces, hidden reference storage, or external credentials. Do not copy solution material from another checkout. Leave the repository in a reviewable state and report the changed paths.

## Method

Inspect the checkout and identify the narrowest relevant implementation path. Run relevant checks that already exist in the checkout. Do not add a test whose expected behavior is derived from an unseen reference. Report a check only when you ran it and observed its result.
