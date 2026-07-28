# Security — Content Desk

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

## Status — deferred, with one standing rule

A full posture review waits on implementation, but one rule is already binding
and must not be relaxed: **this scenario stores no credentials in any form.**
Account identity and platform tokens belong to the scheduler and the `vault`
resource. A schema change that introduces a token, cookie, or session column is
a defect. Claim evidence stores commands that will be re-run, so command
provenance and execution boundaries are the first real review topic.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

## Data Sensitivity

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| _(your product data)_ | classify per PRD | owning domain | Replace with real scenario data classification. |

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
| Unsafe file upload handling | Malicious or oversized upload could affect storage. | Multipart handler validates metadata and BlobStore seam isolates bytes. | template-reference |
| Missing auth for product data | User/customer data could be exposed if added without access control. | Add API-layer auth before storing protected data. | deferred |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No product-specific data classification | medium | Fill after PRD/domain map defines real data. |
| No auth model | conditional | Required before protected or multi-user data. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
