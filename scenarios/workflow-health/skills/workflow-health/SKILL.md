---
name: "workflow-health"
description: "Validate and find browser workflows of a scenario: statically validate its BAS workflow cases, run a durable execution validation, find an existing workflow that already tests something, and preview or apply deterministic fixes."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["workflow-health", "workflows", "bas", "browser-tests", "validation", "find-workflow", "fix"]
  requires:
    scenarios: ["workflow-health"]
    commands: ["workflow-health validate scenario"]
  origin:
    kind: "authored"
---

## workflow-health

Static validation is fast; execution validation is a durable run you wait on once.

| You want | Command |
|---|---|
| Quick static check | `workflow-health validate scenario` |
| Run workflows for real | `workflow-health validate start`, `workflow-health validate wait`, `workflow-health validate get` |
| Find a workflow for an intent | `workflow-health workflows search` |
| Fix | `workflow-health fix preview`, `workflow-health fix apply` |

Run `workflow-health <group> <command> --help` for flags.
