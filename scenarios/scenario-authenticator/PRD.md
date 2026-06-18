# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
Purpose: scenario-authenticator is the Vrooli fleet's foundational Identity Provider (IdP). It owns *who you are* — accounts, credentials, token issuance and signing, sessions, MFA, federation, and API keys — so every other scenario stops reinventing auth. It is consumed API-to-API (never via cross-origin browser calls): adopting scenarios resolve it by slug through `api-core/discovery` and verify its tokens locally and statelessly against its published JWKS. The scenario is being rewritten from the latest `react-vite` template to move off shared Postgres (which created a fleet-wide blast radius) onto per-scenario SQLite via the `api-core/storage` seam, to adopt proto/Connect-RPC as its primary contract, and to re-center the entire design on a multi-tenant **realm** primitive.

Target users: (1) Other Vrooli scenarios acting as Relying Parties (RPs) — the primary API consumers; device-sync-hub is the first live consumer, with future landing pages, landing-page-business-suite, and hosted SaaS products to follow. (2) End users who register, sign in, enroll MFA, and manage sessions through a self-service UI. (3) Realm and system administrators who manage realms, users, roles, and audit logs through an admin console. (4) Operators self-hosting the Vrooli stack on a local same-network server.

Deployment surfaces: Go API server with Connect-RPC as the primary typed contract; a thin REST edge only for web standards that are not RPCs (`/.well-known/jwks.json`, OAuth2/OIDC redirect callbacks, and later SAML ACS); Go CLI with full surface parity; React+Vite+TypeScript+Tailwind UI providing an admin console, end-user self-service, and hosted login/consent screens. Two deployment shapes share one binary: local same-network (single default realm, SQLite) and hosted/cloud-as-a-dependency (an adopting scenario embeds scenario-authenticator and provisions a realm per customer or product; the `api-core/storage` seam can back onto a managed DB at scale).

Value proposition: One permanent, secure identity capability the entire fleet builds on. Stateless local JWT verification means consumers scale without calling back to the authenticator on the hot path. The realm primitive makes the same code serve a single household server and a multi-tenant SaaS. Moving to SQLite removes the shared-database failure class while the storage seam keeps a clean path to a managed DB for cloud scale. It is the highest-leverage compound-value seam in the fleet: every future product that needs users reuses this instead of rolling its own auth.

## 🎯 Operational Targets

Operational targets are measurable outcomes; checkboxes may auto-update based on validation. They are intentionally granular so each maps to at least one requirement module and test. The architectural keystone (IdP ↔ Relying Party split), the realm tenant model, and the carried-over crypto invariants are detailed in the Appendix; the targets below realize them.

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Account Authentication | Email/password registration and sign-in work end-to-end through the Connect API, with input validation and faithful error relay that does not leak account existence.
- [ ] OT-P0-002 | RS256 Token Issuance + JWKS + Local-Verify Contract | Access tokens are RS256-signed with a persisted load-or-generate keypair in the storage root; the public key is published at `/.well-known/jwks.json`; the JWT claims/issuer (`user_id`/`sub`, `email`, `roles`, `iss: scenario-authenticator`, `aud`) match the carried-over contract so device-sync-hub's local verifier needs zero changes.
- [ ] OT-P0-003 | Refresh-Token Rotation with Reuse Detection | Short-lived access tokens pair with rotating refresh tokens; presenting a rotated (reused) refresh token revokes the entire token family and is audited.
- [ ] OT-P0-004 | Argon2id Password Hashing | Passwords are hashed with Argon2id at a documented cost; only hashes and signed material are stored at rest, never plaintext secrets.
- [ ] OT-P0-005 | Sessions + Revocation | Server-tracked sessions support list and per-session revoke (and "log out everywhere"), backed by Redis hot state; the `/api/v1/sessions/{id}` revoke contract device-sync-hub calls is preserved or delivered as a Connect equivalent in lockstep.
- [ ] OT-P0-006 | Rate Limiting + Account Lockout | Auth endpoints enforce rate limiting and account lockout (brute-force defense) with in-memory primary state and Redis for cross-replica coordination.
- [ ] OT-P0-007 | Auth-Event Audit Log | Security-relevant events (sign-in, sign-out, token-family revoke, MFA changes, admin actions) are recorded to a queryable audit log.
- [ ] OT-P0-008 | Default Realm + aud-Scoped Tokens | The realm primitive exists from day one: a single default realm issues `aud`-scoped tokens, and verification rejects a token whose `aud` does not match the verifying realm (no cross-tenant acceptance even with one realm).
- [ ] OT-P0-009 | Baseline RBAC | Admin and user roles are carried over and enforced on management endpoints; role claims are emitted in the token for RP consumption.
- [ ] OT-P0-010 | Connect-RPC Surface + CLI Parity | Login, Register, Refresh, Logout, Validate, and session management are proto-owned Connect RPCs with full Go CLI parity; the only REST endpoints are the non-RPC web standards (JWKS, callbacks).
- [ ] OT-P0-011 | SQLite Persistence via Storage Seam + Redis Hot State | All persistence is SQLite through the `api-core/storage` seam (no shared Postgres); Redis backs sessions, revocation, OAuth CSRF state, and rate limiting; schema changes are additive migrations, never recreation.
- [ ] OT-P0-012 | Live Consumer Migration | device-sync-hub continues to verify tokens unchanged, and its identity forwarder migrates from REST to the typed Connect client; the live first-run owner-bootstrap flow is proven end-to-end before P1 begins.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Realms as a True Tenant Boundary | Realm CRUD with per-realm branding, password policy, token TTLs, enabled methods, and allowed redirect URIs; realm-scoped user pools (same email is distinct principals across realms); realm administrators.
- [ ] OT-P1-002 | TOTP MFA + Recovery Codes | TOTP enrollment, challenge, and recovery codes are ported and hardened, with per-realm enforcement policy.
- [ ] OT-P1-003 | OAuth2/OIDC Social Federation | Inbound social sign-in (Google, GitHub, Microsoft) with account linking and CSRF-protected callbacks, ported from the existing implementation.
- [ ] OT-P1-004 | API Keys + Client-Credentials Grant | Hashed API keys and a client-credentials grant let machine/service principals authenticate without a human session.
- [ ] OT-P1-005 | Scopes/Permissions RBAC per Realm | Roles and scopes are defined per realm (beyond admin/user); definitions are owned here and emitted as claims, with enforcement delegated to RPs.
- [ ] OT-P1-006 | Passkeys / WebAuthn | WebAuthn passkeys are a first-class credential and second factor.
- [ ] OT-P1-007 | Admin Console UI | A production-polished console manages realms, users, roles/scopes, sessions, and audit, with destructive-action confirmation gates.
- [ ] OT-P1-008 | End-User Self-Service UI | Users manage profile, MFA enrollment, active sessions (review/revoke), connected accounts, and password change.
- [ ] OT-P1-009 | Account Recovery | Password-reset and email-verification flows are delivered securely (single-use, expiring tokens; no enumeration).

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | SAML 2.0 + Enterprise SSO | SAML service-provider support and enterprise SSO integration for realms that require it.
- [ ] OT-P2-002 | OIDC-Provider Mode + Token Introspection | scenario-authenticator becomes an OIDC provider itself ("Login with Vrooli") and offers token introspection for opaque-token RPs.
- [ ] OT-P2-003 | Groups / Teams / Org Hierarchy | A grouping layer above the user (groups, teams, organizations) with inherited roles/scopes.
- [ ] OT-P2-004 | Delegated Authorization/Policy Engine | An optional fine-grained authorization engine RPs can call to delegate domain "can-they" decisions.
- [ ] OT-P2-005 | Per-Realm Key Isolation + Automated Rotation | Per-realm signing keys and automated rotation with overlapping `kid`s during rollover.
- [ ] OT-P2-006 | Managed-DB Backing + Multi-Instance HA | The storage seam is exercised against a managed server DB with multi-instance HA behind a Redis-shared session/rate-limit layer.
- [ ] OT-P2-007 | SCIM Provisioning | SCIM user provisioning/deprovisioning for enterprise directory integration.

## 🧱 Tech Direction Snapshot
Preferred stacks: Go API server with Connect-RPC (proto-owned contracts) as the primary surface; a minimal REST edge strictly for non-RPC web standards (JWKS well-known endpoint, OAuth/OIDC callbacks, SAML ACS); React+Vite+TypeScript+Tailwind UI using the vrooli-default operational-console kit for the admin console, self-service UI, and hosted login/consent screens; Go CLI with full surface parity to the Connect API.

Preferred storage: SQLite is the local and default substrate via the `api-core/storage` seam — this is the mechanism that removes the shared-Postgres blast radius. The seam keeps the database swappable to a managed server DB for cloud scale, so SQLite is a default, not a lock-in. Redis is retained and earns its keep as the shared hot-state store for sessions, revocation, OAuth CSRF state, and distributed rate limiting across replicas. Only token and credential hashes plus signed material are stored at rest — never plaintext secrets. SQLite schema changes are always additive migrations, never database recreation. Rate limiting uses in-memory state as the primary layer with Redis for cross-replica coordination.

Integration strategy: API-to-API only. Consumers resolve scenario-authenticator by slug via `api-core/discovery` and verify tokens locally against JWKS. There are no cross-origin browser calls anywhere in the model — a browser talks only to its own scenario's API, which forwards same-origin if it must reach the authenticator (the pattern device-sync-hub uses today). Shared workflows are preferred over resource CLI over direct API where a choice exists. device-sync-hub's forwarder migrates from the existing REST calls to the typed Connect client in lockstep with the P0 launch so the live first-run flow is proven end-to-end before P1 work begins.

Non-goals: Not a domain authorization engine — adopting scenarios own fine-grained "can-they" decisions (a delegated policy engine is deferred to P2). No plaintext credentials at rest under any circumstance. Never gate behind payment any capability a self-hoster can run with their own keys; monetization is realized by enabling metered or gated user tiers in adopting products via LPBS entitlements, not by metering the authenticator itself. No compatibility shims or legacy dead code — when a contract changes, both sides change in lockstep. This is a scaffold-regen plus verbatim crypto-port, not a clean-room reimplementation of the auth core; RS256/JWKS/claims/hashing are ported exactly, not re-derived.

## 🤝 Dependencies & Launch Plan
Required resources: Redis (sessions, revocation, OAuth CSRF state, distributed rate limiting across replicas). SQLite via the `api-core/storage` seam (no shared Postgres). The signing keypair persisted to the storage root (private.pem / public.pem, load-or-generate pattern carried over verbatim).

Scenario dependencies: None upstream — scenario-authenticator is a foundational, bottom-of-the-stack capability with no scenario-level dependencies of its own. Downstream consumers (Relying Parties) that depend on this scenario: device-sync-hub (live today, first migrated consumer), and future adopters including landing pages, landing-page-business-suite, and hosted SaaS products.

Operational risks: Auth is a security boundary — the single biggest risk is re-deriving crypto incorrectly during the rewrite; mitigated by porting RS256/JWKS/claims/Argon2id hashing verbatim, locking the algorithm to exactly RS256, and explicitly rejecting `none` and HS-family confusion at the verification layer. Signing-key persistence and rotation must be deliberate or every restart invalidates live tokens; the load-or-generate pattern with file-backed keypairs directly addresses this. SQLite is a single-writer store — that is the authenticator's own write-throughput ceiling; mitigated because the hot path (token verification) is stateless and never touches SQLite, and the storage seam allows a managed DB at scale when throughput demands it. Redis availability gates session-revocation correctness and distributed rate-limit accuracy; Redis should be treated as a required dependency, not optional. Realm isolation correctness (aud-scoping ensuring a token minted for realm A is rejected by realm B) must be enforced at issuance and verification time and covered by integration tests — a misconfiguration here constitutes a cross-tenant token leak.

Launch sequencing: (1) **P0 core** — accounts + Argon2id + RS256/JWKS/keypair persistence + refresh-token rotation with reuse detection + Redis-backed sessions + rate limiting + account lockout + auth-event audit log + single default realm with aud-scoped tokens + Connect-RPC surface (Login, Register, Refresh, Logout, Validate, session management) + CLI parity + SQLite via storage seam. (2) **Live consumer migration** — migrate device-sync-hub's forwarder from REST to the typed Connect client in lockstep and prove the live first-run flow end-to-end; no P1 work begins until this is green. (3) **P1** — realms as a real tenant boundary, TOTP MFA, OAuth social federation with account linking, API keys and client-credentials grant, scopes RBAC, WebAuthn/passkeys, admin console, end-user self-service UI, account recovery flows. (4) **P2** — SAML/enterprise SSO, OIDC-provider mode ("Login with Vrooli"), groups/orgs hierarchy, delegated policy engine, per-realm key isolation and automated rotation, managed-DB/HA via storage seam, SCIM provisioning.

## 🎨 UX & Branding
User experience: Three distinct UI audiences share one deployment. (a) **Admin console** — realm, user, role/scope, session, and audit management for realm and system administrators; surfaces are dense and data-forward with clear destructive-action confirmation gates (e.g., realm deletion, session revocation). (b) **End-user self-service** — profile editing, MFA enrollment, active-session review and revoke, connected accounts, and password change; surfaces are minimal and task-focused. (c) **Hosted login and consent screens** — the screens adopting scenarios redirect to or embed same-origin for sign-in, registration, MFA challenge, and OAuth consent; these must be clean, trustworthy, and fast, with per-realm branding (logo, colors) rendered at P1. Error messages relay validation reasons faithfully (e.g., "password is too short") without leaking account-existence information where that would aid enumeration attacks. Security prompts — MFA challenge, session revocation confirmation, "new sign-in from unrecognized device" — communicate state plainly without alarmism.

Visual design: vrooli-default operational-console design language — clean, trustworthy, secure, and unobtrusive. Light and dark themes delivered via the kit. Per-realm branding (logo and color overrides) on hosted login/consent screens is a P1 deliverable so hosted products present their own identity. PWA install surface is maintained: the seeded `ui/public/site.webmanifest`, `apple-icon-180.png`, `favicon-196.png`, and maskable manifest icons are kept valid; generic icons are replaced when final product branding is confirmed.

Accessibility: WCAG AA across all surfaces. Full keyboard navigation for every auth and management control — including MFA enrollment, session revocation, and OAuth consent flows. All template accessibility primitives are preserved: `role`, `aria-*` attributes, and `data-testid` selectors for automated testing. Interactive security prompts must meet focus-management and announcement requirements so screen-reader users receive the same timely security information as sighted users.

## 📎 Appendix

**A. Architectural keystone — Identity Provider (IdP) ↔ Relying Party (RP).** This is an OIDC/Keycloak-style split. scenario-authenticator is the IdP; every adopting scenario is an RP.

| The authenticator (IdP) owns — *"who you are"* | The adopting scenario (RP) owns — *"what you may do here"* |
|---|---|
| Identities, credentials, password hashing | Verifies the token locally via JWKS (never calls back per request) |
| Token issuance + RS256 signing + JWKS + rotation | Trusts claims (`sub`/`user_id`, `aud`/realm, `roles`, `scopes`) |
| MFA, sessions, revocation, recovery | Domain authorization: can *this* principal do *this* action on *this* resource |
| Federation (OAuth/social/SAML), realm config | Maps the verified identity to its own domain objects |
| The *definitions* of realm-level roles/scopes | Never sees a password |

Boundary rule of thumb: the authenticator answers "valid principal + coarse realm roles/scopes"; the app answers fine-grained "can-they". Hosting a delegated fine-grained policy engine is a P2 ambition (OT-P2-004), not the default.

**B. Multi-tenancy — the Realm primitive.** The tenant unit is a **Realm** (promoting the old `applications` registry into a real tenant boundary). A realm is an isolated identity namespace with its own user pool (the same email is distinct principals in two realms), branding, password policy, token TTLs, enabled methods, allowed redirect URIs, and an `aud`-scoped token audience so a token minted for realm A is rejected by realm B's verifier. Local same-network deployments use one default realm (feels single-tenant, generalizes for free); hosted/SaaS deployments provision a realm per customer (B2B) or product (B2C).

**C. Carried-over invariants (port verbatim — do NOT re-derive).** These are correct in the old scenario and MUST be preserved so the live consumer keeps verifying without changes. The rewrite is a scaffold-regen + crypto-port, not a clean-room reimplementation.
- **RS256 + persisted keypair**: load-or-generate `private.pem`/`public.pem` in the storage root (old scenario already does this); algorithm locked to exactly RS256, rejecting `none`/HS confusion.
- **JWKS** at `/.well-known/jwks.json` so RPs verify locally.
- **JWT claims/issuer**: `user_id`/`sub`, `email`, `roles`, `iss: scenario-authenticator`, plus `aud` (realm) — identical to today so device-sync-hub's verifier is unchanged.
- **Sessions revoke** surface (`/api/v1/sessions/{id}`) is a live cross-scenario contract — preserved, or delivered as the Connect equivalent in lockstep.

**D. Ecosystem-fit classification.** Role: foundational **interface-enabler** — its product *is* being consumed by other scenarios. Interfaces: **Programmatic** (Connect/CLI, the primary product; clean typed surface + JWKS local-verify contract) + **Direct UI** (admin console + self-service, production-polished) + Conversational later (realm/user management as agent tools). Highest-leverage compound-value seam in the fleet: the realm + local-JWKS-verify contract lets every future product reuse auth instead of rolling its own. Monetization: free/BYOK for self-hosters; it *enables* metered/gated tiers in adopting products via LPBS entitlements rather than metering itself.
