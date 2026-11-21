---
name: "meta-optimization-manager"
description: "Find which scenarios lack Answer, Validate or Guide coverage and how ready the fleet is for local coding agents: the readiness scoreboard, coverage gaps per scenario, the ranked next-best gaps to close, observed condition findings, and template convergence."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["meta-optimization-manager", "agent-readiness", "readiness", "coverage", "gaps", "next-best", "fleet", "local-coding-agents", "convergence"]
  requires:
    scenarios: ["meta-optimization-manager"]
    commands: ["meta-optimization-manager coverage status"]
  origin:
    kind: "authored"
---

## meta-optimization-manager

Aggregates coverage across the fleet's declared denominators.

| You want | Command |
|---|---|
| Readiness scoreboard | `meta-optimization-manager coverage status` |
| Which cells are uncovered | `meta-optimization-manager coverage list-cells`, `meta-optimization-manager coverage explain-cell` |
| What to work on next | `meta-optimization-manager focus next`, `meta-optimization-manager gaps list`, `meta-optimization-manager gaps show` |
| Observed conditions | `meta-optimization-manager condition status` |
| Template convergence | `meta-optimization-manager convergence status` |

Run `meta-optimization-manager <group> <command> --help` for flags.
