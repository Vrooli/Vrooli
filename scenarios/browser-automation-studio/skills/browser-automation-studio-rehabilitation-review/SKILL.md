---
name: "browser-automation-studio-rehabilitation-review"
description: "Independently review a retained Browser Automation Studio rehabilitation candidate before owner qualification."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["contract"]
  tags: ["browser-automation-studio", "rehabilitation", "review", "protected-workflow"]
  status: "active"
  revision: 1
  requires:
    scenarios: ["browser-automation-studio", "prompt-manager"]
    commands: ["prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Browser Automation Studio rehabilitation review contract

Independently review the retained candidate for the one owner-selected Browser
Automation Studio boundary. Review only the immutable retained workspace and
the focused worker evidence; do not substitute the live project tree.

### Variable legend

`{{.boundary}}` is the owner-selected boundary under review.

The retained candidate and its canonical source SHA-256 are supplied and bound
by Agent Manager from the protected review snapshot. That owner-injected digest
is authoritative; return it exactly as `candidateSha256`.

### Outcome work table

| Observable state | Outcome |
| --- | --- |
| The retained candidate exactly matches the injected digest, preserves behavior/data, provides focused evidence, reduces or controls complexity, and completes cleanup within scope | Return `accepted=true`, the exact injected digest, and a concise reason. |
| Any identity, evidence, behavior, scope, cleanup, or simplification predicate is unproven or false | Return `accepted=false`, the exact injected digest, and the specific reason; do not infer acceptance. |

### Authority boundary

Read only the retained review paths and worker evidence. Do not write files,
modify control records, run qualification, apply changes, substitute a live
tree, or repair defects during review.

### Method

Use `prompt-manager skill read large-effort-supervision`,
`prompt-manager skill read scenario-improvement-campaign`, and the BAS
`docs/internal/TESTING.md` contract. Apply the conservative review branch when
any required predicate is unavailable.
