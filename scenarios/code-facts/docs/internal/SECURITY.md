# Security — Code Facts

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

| Data | Sensitivity | Owner | Notes |
|---|---|---|---|
| Source paths, symbols, and route metadata | internal | Code Facts maintainers | Derived from the local repository; do not expose outside the trusted project boundary. |
| Cache payloads | internal | Code Facts maintainers | May contain source-derived facts and paths; governed by local cache retention. |

## Auth And Authorization

The generated template does not include an auth provider. Add auth only
when product requirements identify protected data or user-specific
behavior. UI and CLI must not enforce business authorization locally;
authorization belongs at the API/service layer.

## Secrets

| Secret | Source | Required? | Notes |
|---|---|---|---|
| None by default | n/a | no | Add entries when resources or third-party APIs require secrets. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Untrusted target traversal | A caller could request files outside the repository. | Resolver normalizes and validates target roots; analyzers receive bounded roots only. | active |
| Source-derived metadata disclosure | Internal paths or symbols could leave the trusted project. | Search Hub is a local capability; registration and status are local-only. | active |
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
