# PostgreSQL Problems

Last reconciled: 2026-11-04. This file lists current, verified gaps. Older
container and shell-runtime incident records are retained in Git history; they
are not current issues or supported operator procedures.

## Current Verified Gap

No resource-specific availability defect is currently recorded. Readiness
performs a PostgreSQL startup handshake; liveness independently checks the
listener. Neither verifies application authentication or SQL behavior, so
consumers must validate their own configured database operation.

- **Evidence:** `resource.json` declares `postgresql` readiness and TCP
  liveness checks; the shared health runner implements both.
- **Current operator guidance:** [TROUBLESHOOTING.md](TROUBLESHOOTING.md).
