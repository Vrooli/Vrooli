# Flows

## Purpose Of This Document

Document the main user and system workflows.

## Flow Inventory

- Build actual graph from upstream facts.
- Compare actual graph to declared scenario dependencies.
- Scan deployment readiness and metadata gaps.
- Export DAG or graph JSON for downstream tools.

## Flow Details

Graph and drift flows are read-only. Scan/apply flows may update scenario manifests when explicitly requested.

## State Machines

Long-running scenario test and lifecycle states are owned by Vrooli lifecycle and test-genie.

## Maturity Ladder

Current flows are implemented and validated through API, CLI, and scenario tests.

## Production Shape

Operators run the scenario through lifecycle-managed API/UI processes.

## Deferred / Unmodeled Flows

Formal workflow modeling is deferred until SDA owns a mutable multi-step planning workflow.

## Cross-References

- `ARCHITECTURE.md`
- `DOMAINS.md`
- `../internal/TESTING.md`
