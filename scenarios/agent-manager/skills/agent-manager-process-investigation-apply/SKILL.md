---
name: "agent-manager-process-investigation-apply"
description: "Implement fixes from an investigation report organized by category with per-change verification."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["contract"]
  tags: ["agent-manager","investigation","apply"]
  status: "active"
  revision: 3
  createdAt: "2026-02-17T00:00:00Z"
  updatedAt: "2026-03-15T00:00:00Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager skill", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Investigation apply contract

Apply only the operator-approved investigation findings and return the
workflow's declared result. Use `prompt-manager skill read
investigating-agent-runs` for the attribution and severity method.

### Variable legend

`{{.findings}}` is the validated investigation result; `{{.approval}}` is the
operator decision and selected recommendation set; `{{.context}}` is bounded
source-run evidence. The approval selection is authoritative.

### Outcome work table

| Observable state | Outcome |
| --- | --- |
| A selected recommendation has a bounded safe implementation and its verification passes | Apply it and report the evidence. |
| A recommendation was not selected or is not sufficiently specific | Do not apply it; report the reason. |
| Applying a selected recommendation would exceed its target or weaken safety | Stop that change and report the safety boundary. |

Conservative default: when selection or scope is ambiguous, do not modify the
target. Preserve the ambiguity as an unapplied finding.

### Authority boundary

Make only git-revertible changes selected by the operator, verify each one,
and leave unrelated code untouched.
