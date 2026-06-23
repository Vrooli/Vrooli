# Security — Network Manager

## Purpose Of This Document

Define the security and privacy posture for a scenario that can observe and change local network behavior.

## Data Sensitivity

High-sensitivity data includes device identity, DNS query visibility, filtering policy, router/resolver credentials, rollback records, and household profile membership.

## Auth And Authorization

P0 must distinguish ordinary viewing from privileged operations. Persistent changes, query-log visibility, export, and rollback should require an authorized operator. Future household profiles should not expose one household member's activity to another by default.

## Secrets

Expected secrets:

- AdGuard Home API credentials or token.
- Future resolver/router adapter credentials.

Secrets must be stored through Vrooli secret handling and never written into docs, logs, reports, or exported evidence packs.

## Threat Model

| Threat | Mitigation |
|---|---|
| DNS misconfiguration breaks internet access | Preview, approval, rollback, manual recovery instructions. |
| Query logs become surveillance | Minimal retention, explicit visibility, household defaults. |
| Unauthorized policy changes | Operator authorization and audit records. |
| Router credentials leak | Secret storage and redacted logs. |
| Unsupported automation lies to user | Capability reports and unsupported reasons. |
| TLS interception temptation | Explicit non-goal for home use. |

## Security Gaps

- Auth model is not yet implemented.
- Resolver resource secret handling is not yet implemented.
- Router adapter credential policy is deferred.
- Query-log retention defaults require implementation before deployment.

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md)
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md)
- [`DECISIONS.md`](DECISIONS.md)
