---
name: "go-code-graph"
description: "Understand and refactor Go code mechanically: extract the file, package, symbol and import graph of a Go module (who imports this package, who calls it), then plan and apply deterministic refactors such as moving a symbol."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["go-code-graph", "go", "golang", "imports", "callers", "dependency-graph", "refactor", "symbols"]
  requires:
    scenarios: ["go-code-graph"]
    commands: ["go-code-graph graph extract"]
  origin:
    kind: "authored"
---

## go-code-graph

Deterministic extraction; rewrites are planned first, then applied.

| You want | Command |
|---|---|
| Graph of a Go module | `go-code-graph graph extract` |
| Plan a refactor | `go-code-graph rewrite plan` |
| Apply it | `go-code-graph rewrite apply` |
| Check determinism fixtures | `go-code-graph graph validate-fixture` |

Run `go-code-graph <group> <command> --help` for flags.
