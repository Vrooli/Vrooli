---
name: "browser-automation-studio-rehabilitation-worker"
description: "Implement one owner-selected Browser Automation Studio rehabilitation boundary in a protected, evidence-backed worker run."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["contract"]
  tags: ["browser-automation-studio", "rehabilitation", "worker", "protected-workflow"]
  status: "active"
  revision: 2
  requires:
    scenarios: ["browser-automation-studio", "prompt-manager"]
    commands: ["prompt-manager skill read", "go test"]
  origin:
    kind: "authored"
---
## Browser Automation Studio rehabilitation worker contract

Implement exactly one owner-selected Browser Automation Studio boundary. The
owner's boundary, instruction, and qualification rows are authoritative; do
not select a different boundary or claim work that was not performed.

### Variable legend

`{{.boundary}}` is the single boundary selected by the owner.

`{{.worker_instruction}}` is the bounded operator instruction for that boundary.

`{{.required_rows}}` is immutable owner input describing required qualification
rows; preserve it and do not weaken or reinterpret it.

### Outcome work table

| Observable state | Outcome |
| --- | --- |
| The selected boundary is implemented within the declared writable paths and focused checks pass | Report the actual changed files, checks, remaining work, and concrete simplification. |
| The boundary is only partly implemented or a check is unavailable/fails | Preserve the partial state and report the exact remaining or unverified condition; do not claim completion. |
| The requested change requires another boundary, an unauthorized path, or control-plane/evidence mutation | Do not expand scope; report the precise deferral and continue only with permitted work. |

### Authority boundary

Write only the paths supplied by the protected sandbox. Preserve behavior and
data, remove replaced paths when the boundary permits it, and run focused
checks. Do not commit, restart managed services, edit campaign/control records,
run qualification or setpoint programs, or choose the next boundary.

The protected workflow node's `sandboxConfig.writePolicy` is the controller's
authoritative grant. The launch prompt may not repeat those literal paths; do
not report the grant as unavailable merely because it is absent from prose.
Use explicit paths in the owner instruction when present, otherwise inspect
only the node's granted owners. Never widen the grant or edit an unlisted file.

### Efficiency and failure discipline

The controller has already admitted the boundary and supplied the write
allowlist. Do not repeat repository-wide discovery, `git status`, full-document
`cat`, or prompt-manager discovery. Read the active packet and only the
allowlisted owners plus directly relevant tests, using bounded excerpts. Do not
inspect or plan edits in a tempting but non-allowlisted file.

Use focused checks that cover the changed owners and one bounded
formatting/diff check; do not rerun unchanged checks. Never retry an identical
failed command more than once. If a write, service, network, or validation
operation is unavailable, record it in `remaining` and move on; do not spend
the turn diagnosing infrastructure or polling.

Use the full admitted worker allowance when it produces useful progress. Work
through the semantic exit gate for this boundary, batching related edits and
their focused checks across turns; a single defect, test, receipt, context
compaction, or session turn does not close the boundary. Stop only when the
allowlisted boundary is implemented and checked, no further in-scope progress
is possible, or the owner allowance is actually exhausted. If several tool
calls repeat the same unchanged inspection or failed operation, stop repeating,
record the exact evidence gap, and hand off. If no permitted change is
possible after the initial inspection, return the structured handoff
immediately. Always report changed, verified, remaining, and simplification.

### Method

Use the BAS `docs/internal/TESTING.md` contract and the controller-supplied
boundary. The controller, not the worker, owns campaign skills, evidence
rotation, qualification, setpoint reads, rebuilds, and service lifecycle.
