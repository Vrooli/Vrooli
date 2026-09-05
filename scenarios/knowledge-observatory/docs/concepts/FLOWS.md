# Flows

## Flow Inventory

| Flow | Trigger | Completion signal |
|---|---|---|
| Documentation health check | API/CLI request | Contract findings and health score returned |
| Documentation read | API/CLI request with doc identifier | Manifest alias resolves to a concrete document |
| Append-log entry | API/CLI request | Entry is written under the manifest-declared log heading |
| Documentation healing | Operator request | Agent diff is reviewed, approved, or rejected |
| Deep search | Operator query | Agent-backed search job reaches terminal status |

## State Machines

Documentation healing and deep search both follow `pending -> running ->
needs_review/complete/failed`, with approval transitions for doc healing.

## Deferred / Unmodeled Flows

Bulk knowledge pruning and semantic diffing remain future PRD targets.

## Cross-References

- [DOMAINS.md](DOMAINS.md)
- [INTEGRATIONS.md](INTEGRATIONS.md)
- [../internal/ERROR-HANDLING.md](../internal/ERROR-HANDLING.md)
