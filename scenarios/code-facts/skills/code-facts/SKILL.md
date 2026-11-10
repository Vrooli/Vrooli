---
name: "code-facts"
description: "Find evidence in the code: search symbols, file domains and contracts, see where a function, contract or endpoint is defined and implemented, check proto adoption, and read or rebuild the code index."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["code-facts", "code-search", "symbols", "contracts", "endpoints", "definition", "where-is", "proto-adoption", "code-index"]
  requires:
    scenarios: ["code-facts"]
    commands: ["code-facts facts search"]
  origin:
    kind: "authored"
---

## code-facts

Target-aware code evidence for scenarios and generic code roots.

| You want | Command |
|---|---|
| Where is this defined | `code-facts facts search` |
| Describe a target's facts | `code-facts facts describe`, `code-facts facts surfaces` |
| Is an endpoint really implemented | `code-facts facts endpoint-proof` |
| Proto adoption | `code-facts facts proto-adoption` |
| Index state | `code-facts index status`, `code-facts index reconcile` |

Run `code-facts <group> <command> --help` for flags.
