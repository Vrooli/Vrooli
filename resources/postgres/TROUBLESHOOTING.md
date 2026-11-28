# PostgreSQL Troubleshooting

This guide covers the current native managed-service resource. Readiness uses a
PostgreSQL startup handshake; `healthy` means the server accepts protocol
clients, not that application authentication or queries succeed. See
[PRD.md](PRD.md) for the readiness boundary.

## Start with owner status and logs

```sh
vrooli resource status postgres --json
vrooli resource logs postgres --tail 100
```

Use `vrooli resource restart postgres` only when a restart is appropriate for
the affected scenarios. The control plane owns the PostgreSQL process and its
lifecycle; do not start `postgres` directly or use retired container and shell
procedures.

## Application cannot connect

1. Check the resource status and recent resource logs.
2. Confirm the consumer resolves the intended `POSTGRES_HOST`, `POSTGRES_PORT`,
   `POSTGRES_DB`, and `POSTGRES_USER` from its declared dependency.
3. Use the consumer's own health check or query path to verify database access.
   Resource protocol readiness does not authenticate or execute a query.
4. Keep `POSTGRES_PASSWORD` and `POSTGRES_URL` out of terminal transcripts,
   logs, screenshots, and bug reports. Use the governed credential path to
   refresh credentials.

## Resource reports unhealthy

Read the probe reason and resource logs before restarting. Keep database files
in the control-plane-managed storage paths; do not delete data directories,
WAL files, or resource state as a troubleshooting shortcut. If logs indicate a
port conflict or damaged data, preserve the evidence and repair it through the
resource owner workflow.
