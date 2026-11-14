# PostgreSQL Resource Requirements

## Purpose

Provide a locally managed PostgreSQL service for Vrooli scenarios that need a
relational database. The resource is a native managed service. Linux artifacts
are extracted from a digest-pinned OCI source; Docker is not a runtime
requirement. macOS and Windows artifacts follow the platform declarations.

## Authority

`resource.json` is the active contract for acquisition, lifecycle, ports,
credentials, storage, and health. Use the resource README and
[`docs/OPERATIONS.md`](docs/OPERATIONS.md) for operator context. The shared
control plane owns resource lifecycle and host operations; do not recreate
those operations in a scenario-local shell workflow.

## Current Contract

| Area | Contract |
|---|---|
| Lifecycle | Managed through `vrooli resource install|start|stop|restart|status postgres` and the standard `resource-postgres` CLI. |
| Runtime | Native PostgreSQL executable from the staged artifact; no runtime Docker dependency. |
| Port | The manifest declares PostgreSQL on host port 5433. |
| Credentials | Managed resource credentials provide `POSTGRES_USER`, `POSTGRES_PASSWORD`, and database connection exports. Keep secret values out of logs and health output. |
| Storage | Resource state and database files use control-plane-resolved resource storage paths. |
| Readiness and liveness | Readiness performs a PostgreSQL startup handshake on `127.0.0.1:5433` and succeeds once the server requests authentication or reports ready. Liveness separately checks TCP reachability. Neither check proves application credentials or a query work. |

## Requirements

- Acquire and verify platform artifacts through the declared managed-service contract.
- Keep database state outside the source tree and preserve it across supported restarts.
- Keep credentials in the governed credential path and redact them from output.
- Report readiness honestly. The startup handshake proves PostgreSQL accepts protocol clients, not that application credentials or queries succeed.
- Keep lifecycle, installation, and host remediation in the shared control plane.

## Readiness Boundary

The shared control-plane probe does not authenticate or issue a query. Consumers
must still verify their own credentials, selected database, and required SQL
operations. The probe uses configured user/database names when present, but
does not send or log a password.

## Historical Context

This file previously described a retired container and shell implementation, a
universal v2 shell CLI, and a hard-coded-user health-check defect. Those details belong to
a retired implementation and are not active requirements. The 2025-09-11
health-check fix is historical; the active readiness contract is the PostgreSQL
startup handshake in `resource.json`. Old completion percentages and test-pass claims have been
removed because they do not establish current runtime maturity.
