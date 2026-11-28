---
name: "typescript-code-graph"
description: "Use Code Facts as the shared TypeScript and JavaScript facts and graph surface, then use TypeScript Code Graph for provider diagnostics and deterministic refactors."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["typescript-code-graph", "typescript", "ts", "imports", "modules", "dependency-graph", "refactor", "symbols"]
  requires:
    scenarios: ["code-facts", "typescript-code-graph", "tidiness-manager"]
    commands: ["code-facts facts describe", "tidiness-manager recommend-refactors", "typescript-code-graph rewrite plan", "typescript-code-graph rewrite apply"]
  origin:
    kind: "authored"
---

## typescript-code-graph

Code Facts brokers language facts and normalized graphs through one shared surface. Use it first when you need graph or fact evidence that another scenario can consume. TypeScript Code Graph remains the TypeScript/JavaScript parser provider and owns provider-specific graph diagnostics and mechanical rewrites; rewrites are planned first, then applied.

| You want | Command |
|---|---|
| Shared facts and graph for a scenario or JavaScript package | `code-facts facts describe scenario:<name> --graphs --include parse_units --json` |
| Rank measured reduction opportunities across shared graph evidence | `tidiness-manager recommend-refactors <name> --limit 10` |
| Provider-specific TypeScript graph diagnostics | `typescript-code-graph graph extract` |
| Plan a refactor | `typescript-code-graph rewrite plan` |
| Apply it | `typescript-code-graph rewrite apply` |
| Check determinism fixtures | `typescript-code-graph graph validate-fixture` |

Run `typescript-code-graph <group> <command> --help` for flags.
