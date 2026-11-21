# Redis Problems

Last reconciled: 2026-11-04. This file lists current, verified gaps; it does not
assert a current Redis outage.

## Current Verified Gap

No resource-specific availability defect is currently recorded. Readiness
issues Redis `PING` and requires `PONG`; liveness independently checks the
listener. The current resource contract does not configure Redis authentication.
If authentication is added, readiness must use the governed credential source
and redact credentials from all health output.

- **Evidence:** `resource.json` declares Redis protocol readiness and TCP
  liveness; the shared health runner implements both.
- **Current operator guidance:** [README.md](README.md) and
  [docs/OPERATIONS.md](docs/OPERATIONS.md).
