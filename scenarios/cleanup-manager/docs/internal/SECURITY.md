# Security — Cleanup Manager

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

## Data Sensitivity

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Cleanup provider previews | medium | cleanup | May include host paths, resource names, image ids, or command output. |
| Cleanup audit messages | medium | cleanup | Redacted before storage when providers return paths or command output. |
| Policy/profile state | low | cleanup | Operator intent, approval mode, and provider enablement. |

## Auth And Authorization

The generated template does not include an auth provider. Add auth only
when product requirements identify protected data or user-specific
behavior. UI and CLI must not enforce business authorization locally;
authorization belongs at the API/service layer.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| None by default | n/a | no | Add entries when resources or third-party APIs require secrets. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Unsafe cleanup mutation | Provider bypass could delete or mutate host state without policy gates. | Providers use typed seams only; orchestrator requires preview, version checks, approval, and idempotency. | cleanup |
| Missing auth for product data | User/customer data could be exposed if added without access control. | Add API-layer auth before storing protected data. | deferred |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| Durable audit redaction coverage | medium | Expand when additional providers return richer host metadata. |
| No auth model | conditional | Required before protected or multi-user data. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
