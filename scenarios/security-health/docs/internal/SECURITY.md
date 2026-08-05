# Security — Security Health

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
| Template notes data | low | notes reference | Local development data only; replace with real scenario data classification. |
| Attachment bytes | unknown | notes reference | Treat as potentially sensitive if retained in product scope. |

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
| Unsafe file upload handling | Malicious or oversized upload could affect storage. | Multipart handler validates metadata and BlobStore seam isolates bytes. | template-reference |
| Missing auth for product data | User/customer data could be exposed if added without access control. | Add API-layer auth before storing protected data. | deferred |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No product-specific data classification | medium | Fill after PRD/domain map defines real data. |
| No auth model | conditional | Required before protected or multi-user data. |

## Supply-Chain Enforcement Contract

Security Health owns the normalized finding and policy contract. Construction
facts are discovered through typed ecosystem adapters for Go, pnpm/npm, Yarn,
Bun, Python, Rust, and evidence-only C/C++. Missing or failed scanners are
explicit findings rather than a clean result when guarded or enforcing policy
is selected. Findings carry stable class, evidence state, owner, confidence,
and fix-class metadata.

Dependency mutations are routed through Scenario Dependency Analyzer. Managed
reproduction commands use frozen lockfile semantics and disable lifecycle
scripts where the package manager exposes that control. Build exceptions must
identify an owner, reason, command, and non-advisory policy mode.

The local agent-policy runner uses integrity-checked, expiring provider
snapshots. Advisory and guided profiles remain usable during provider outage;
guarded and enforcing profiles deny high-risk mutations when no healthy,
fresh provider or snapshot is available. Native resource adapters report their
actual enforcement posture and do not infer hook firing from file presence.

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
