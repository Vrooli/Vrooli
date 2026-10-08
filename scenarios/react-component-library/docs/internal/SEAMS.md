# Seams — React Component Library

Behavioral claims are current only within the specific checks in the register. Other guidance and unverified descriptions below are **design intent**, not claims of current implementation. See the [behavior claim register](TESTING.md#behavior-claim-register).

This document records the boundaries that keep the component library
reusable. A seam is an explicit adapter or fixture boundary; it is not a
shortcut around a consuming scenario's API or authorization policy.

| Seam | Owner | Rule | Test shape |
|---|---|---|---|
| Catalog and release | RCL catalog service | Released assets are immutable; drafts and promotion are governed by catalog contracts | Catalog fixtures and gate calibration |
| Preview composition | RCL preview runner | Subject, frame, harness, and fixture remain separate roles | Story contract and rendered evidence |
| Scenario API adapter | Consuming scenario | Components receive typed data/actions; business rules remain in the scenario API | Provider-backed component tests |
| Identity presentation | Consuming scenario plus shared auth components | RCL renders signed-out, sign-in, MFA, session, account-link, and denied states; it never verifies tokens or grants access | Deterministic auth-state fixtures and accessibility tests |
| Authorization | Consuming scenario API | Domain and object authorization is enforced at the service boundary, not by component visibility | API authorization tests plus UI state tests |
| External integrations | Consuming scenario | Credentials and provider clients stay behind the scenario integration seam | Fake clients and failure-state tests |

## Identity-specific rule

Reusable account controls may standardize layout, copy, loading/error states,
keyboard behavior, and typed callbacks. They must not create a local user
database, store bearer tokens, infer business-account ownership from email
equality, or treat an LPBS website session as local Vrooli authority.

## Target account/security asset contract

The reusable asset set should be organized around state and intent rather than
around a provider-specific client. The initial contract is:

| Asset family | Renders | Receives from the consuming scenario | Must not decide |
|---|---|---|---|
| Sign-in and registration | Credentials, provider choice, loading, generic failure, MFA challenge | Typed submit callbacks and provider metadata | Whether a principal may access a domain object |
| Session and account security | Active sessions, sign-out, sign-out-everywhere, password/MFA actions | Session rows, capability labels, typed action callbacks | Session validity or identity-provider state |
| User administration | Search, lock/unlock, reset, role/capability and audit states | Paginated records, available operations, confirmation policy | Whether an operation is allowed; the API decides |
| Account linking | Consent, requested scope, success, expiry, revoke | One-time flow state and business-account summary | Linking by email equality or token copying |
| Entitlement presentation | Plan/feature/limit state and upgrade/unavailable explanations | Signed or server-projected entitlement state | Granting a commercial entitlement |

Every security-sensitive asset must support explicit `loading`, `ready`,
`unauthenticated`, `forbidden`, `expired`, `error`, and `unavailable` states.
Destructive actions require a typed confirmation callback and an accessible
focus/announcement path. The consuming scenario supplies the API adapter,
provider configuration, policy text, and audit semantics.

The authenticator's eventual admin console and LPBS's product-admin console
may share these presentation assets, but they must not share an implicit
authority. A component must be usable with a local provider, a hosted
provider, or an external provider without changing its security meaning.

The project identity and deployment contract is
[Identity and Authentication](../../../../docs/concepts/IDENTITY-AND-AUTHENTICATION.md);
the component-library security boundary is
[`SECURITY.md`](SECURITY.md).
