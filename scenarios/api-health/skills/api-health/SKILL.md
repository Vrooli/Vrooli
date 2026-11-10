---
name: "api-health"
description: "Check a scenario's API: validate its health endpoint, lifecycle, HTTP semantics and runtime hygiene, preview deterministic fixes and apply them."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["api-health", "api", "http", "health-check", "endpoints", "validation", "runtime-hygiene"]
  requires:
    scenarios: ["api-health"]
    commands: ["api-health validate scenario"]
  origin:
    kind: "authored"
---

## api-health

A delegated Test Genie validation target.

| You want | Command |
|---|---|
| Validate one scenario's API | `api-health validate scenario` |
| Preview mechanical fixes | `api-health validate fix-preview` |
| Apply them | `api-health validate fix-apply` |

Run `api-health <group> <command> --help` for flags.

Always preview fixes before applying.
