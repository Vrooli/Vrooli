---
name: "structure-health"
description: "Check a scenario, resource or repository has the right folder structure and lifecycle wiring: validate one target or all, list the structural rules, roll up conformance across the fleet, and preview or apply deterministic config fixes."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["structure-health", "structure", "folder-structure", "lifecycle", "conformance", "scaffold-check", "rules", "validation"]
  requires:
    scenarios: ["structure-health"]
    commands: ["structure-health validate scenario"]
  origin:
    kind: "authored"
---

## structure-health

Profile-aware: each target kind is checked against its own rules.

| You want | Command |
|---|---|
| Check one scenario | `structure-health validate scenario` |
| Check everything | `structure-health validate all`, `structure-health fleet scan` |
| What are the rules | `structure-health rules list`, `structure-health rules coverage` |
| Fix config | `structure-health fix-config run` |

Run `structure-health <group> <command> --help` for flags.
