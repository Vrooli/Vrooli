# Flows - Tech Tree Designer

## Purpose Of This Document

Record user/system workflows and lifecycle state machines. Phase 1 has no product workflow beyond health checks.

## Flow Inventory

| Flow | Domain | Status | Notes |
|---|---|---|---|
| Health check | health | implemented | Lifecycle and UI can read API readiness. |
| Live graph describe/query/export | graph | planned | Added in Phase 2/3. |
| Planned proto validate/materialize | planning | planned | Added in Phase 4. |
| Roadmap progress rollup | roadmap | planned | Added in Phase 5. |

## Flow Details

### Health check

1. Lifecycle, CLI, or UI requests `/health`.
2. API probes database reachability.
3. API returns proto-shaped JSON health response.

## State Machines

No product state machine is implemented in Phase 1.

## Maturity Ladder

Future planning/materialization flows should use formal workflow coverage if they include retries, cancellation, stale completion, or cleanup invariants.

## Production Shape

Health is a REST ops probe. Product flows should be Connect-RPC unless they meet an explicit REST exception reason.

## Deferred / Unmodeled Flows

- Graph neighborhood/path/ancestry.
- Planning file add/edit/remove/validate/materialize.
- Roadmap milestone progress.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`DOMAINS.md`](DOMAINS.md)
- [`../internal/SEAMS.md`](../internal/SEAMS.md)
