# Security — React Component Library

Behavioral claims are current only within the specific checks in the register. Other guidance and unverified descriptions below are **design intent**, not claims of current implementation. See the [behavior claim register](TESTING.md#behavior-claim-register).

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.


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

The D · Nocturne redesign adds visible write actions: assign/triage a finding, accept a composition, request review, create/extract/adopt, publish and reset preferences. A visible button is not a grant. The owning API must authorize the operation and target; the UI must render permission-denied distinctly and must not infer authority from a rendered control. The workbench chrome and preview must not store credentials or expose them to the iframe. See [WORKBENCH-REDESIGN.md](../concepts/WORKBENCH-REDESIGN.md) and [../reference/api-endpoints.md](../reference/api-endpoints.md).

When a consuming scenario needs accounts, use the shared project contract and
provider rather than inventing a local identity model. The reusable UI layer
may provide presentation components for sign-in, account switching, MFA,
sessions, invitations, and explicit business-account linking, but it must not
decide whether an API operation is allowed. API/service handlers verify the
canonical principal and enforce capability, object, mode, and commercial
entitlement policy.

The desktop default is `personal_local`, so a bundled app should not display a
mandatory sign-in unless its operator enables multi-user, remote, or shared
provider mode. An LPBS website session and a desktop supervisor token are
different credentials. Account linking is an explicit short-lived flow, never
email matching or browser-token copying. The project contract is [Identity and
Authentication](../../../../docs/concepts/IDENTITY-AND-AUTHENTICATION.md).

Any future account controls added to this library should be documented as
presentation and interaction contracts, with keyboard accessibility, loading,
error, expired-session, and offline states. They must not create a second
source of truth for identity or authorization.

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

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
