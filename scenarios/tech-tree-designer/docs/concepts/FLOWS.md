# Flows - Tech Tree Designer

## Purpose Of This Document

Record user/system workflows and lifecycle state machines for graph, planning, ontology, and health surfaces.

## Flow Inventory

| Flow | Domain | Status | Notes |
|---|---|---|---|
| Health check | health | implemented | Lifecycle and UI can read API readiness. |
| Live graph describe/query/export | graph | implemented | Connect API, CLI, and UI graph routes are available. |
| Planned proto validate/materialize | planning | implemented | Planned scenario CRUD, file storage, validation, and materialization are available. |
| Ontology coverage rollup | ontology | implemented | Capability hierarchy, fulfillment links, graph-derived coverage, and focus ranking are available. |

## Flow Details

### Health check

1. Lifecycle, CLI, or UI requests `/health`.
2. API probes database reachability.
3. API returns proto-shaped JSON health response.

### Live graph describe/query/export

1. API calls the `GraphSource` seam.
2. The current source reads `proto-health` `DescribeScenariosProtos` and maps surfaces to scenario nodes and proto-import edges.
3. The graph service overlays planned scenarios from the planning store.
4. Connect, CLI, or UI consumers request describe, neighborhood, path, ancestors, or export results.

### Planned proto validate/materialize

1. An operator or agent creates a planned scenario and stores planned `.proto` files.
2. Validation compiles planned files against live `packages/proto/schemas` plus planned overlays.
3. Findings are returned without mutating the shared proto tree.
4. Materialization writes validated files under `packages/proto/schemas/<slug>/` and runs proto generation.

### Ontology coverage rollup

1. Ontology stores capability hierarchy, progression edges, and fulfillment links.
2. Coverage reads the current graph through the scenario-source seam rather than owning implementation topology.
3. Results classify capabilities as built, in-flight, or gaps, and report unmapped scenarios separately.

## State Machines

Planning materialization is the only write flow with a validation gate: invalid findings block writes to `packages/proto/schemas/<slug>/`.

## Maturity Ladder

Future long-running planning/materialization flows should use formal workflow coverage if they add retries, cancellation, stale completion, or cleanup invariants.

## Production Shape

Health is a REST ops probe. Product flows should be Connect-RPC unless they meet an explicit REST exception reason.

## Deferred / Unmodeled Flows

- AI strategic analysis.
- SDA-backed graph source.
- Scenario scaffold generation from a planned node.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`DOMAINS.md`](DOMAINS.md)
- [`../internal/SEAMS.md`](../internal/SEAMS.md)
