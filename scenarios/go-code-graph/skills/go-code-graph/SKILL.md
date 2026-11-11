---
name: "go-code-graph"
description: "Use Code Facts as the shared Go facts and graph surface, then use Go Code Graph for provider diagnostics and deterministic Go refactors."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["go-code-graph", "go", "golang", "imports", "callers", "dependency-graph", "refactor", "symbols"]
  requires:
    scenarios: ["code-facts", "go-code-graph", "tidiness-manager"]
    commands: ["code-facts facts describe", "tidiness-manager recommend-refactors", "go-code-graph rewrite plan", "go-code-graph rewrite apply"]
  origin:
    kind: "authored"
---

## go-code-graph

Code Facts brokers language facts and normalized graphs through one shared surface. Use it first when you need graph or fact evidence that another scenario can consume. Go Code Graph remains the Go parser provider and owns Go-specific graph diagnostics and mechanical rewrites; rewrites are planned first, then applied.

| You want | Command |
|---|---|
| Shared facts and graph for a scenario | `code-facts facts describe scenario:<name> --graphs --include parse_units --json` |
| Rank measured reduction opportunities across shared graph evidence | `tidiness-manager recommend-refactors <name> --limit 10` |
| Provider-specific Go graph diagnostics | `go-code-graph graph extract` |
| Plan a refactor | `go-code-graph rewrite plan` |
| Apply it | `go-code-graph rewrite apply` |
| Check determinism fixtures | `go-code-graph graph validate-fixture` |

Run `go-code-graph <group> <command> --help` for flags.
