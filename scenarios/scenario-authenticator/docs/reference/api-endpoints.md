# API Endpoints — Scenario Authenticator

> **Target/planned surface — generated from proto + `cli/manifest.json`
> during implementation (Gate 6); not yet shipped.** Nothing below the
> `health` domain is implemented. This document describes the *target*
> contract authored during the documentation-first orientation pass so
> the proto/handler/CLI work has a contract to build against. The
> concrete surface is generated from
> `packages/proto/schemas/scenario-authenticator/v1/<domain>/` and bound
> in [`cli/manifest.json`](../../cli/manifest.json) during implementation;
> RPC method names, request/response field names, and error codes here
> are the *planned* shapes and may be refined when the proto is authored.

Human-readable reference for the API. Once implemented, the
machine-readable source of truth is
[`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) — doc
generators, Postman collection builders, and SDK stubs read it directly,
and the CI gate fails if the JSON drifts from the registered handlers or
from the CLI commands it claims to mirror.

Wire shapes for every endpoint will live in
`packages/proto/schemas/scenario-authenticator/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients. Tests,
handlers, UI clients, and CLI handlers all consume generated types — no
hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
The thin REST edge (below) uses the template error envelope
(`packages/proto/schemas/scenario-authenticator/v1/errors/errors.proto`):

```json
{ "code": "<canonical_code>", "message": "<human readable>", "details": [...] }
```

Canonical REST codes: `invalid_request` (400), `unauthorized` (401),
`not_found` (404), `internal` (500). Add to the proto enum when a new
REST-exception failure mode appears.

## Architecture invariants (read before adding an endpoint)

scenario-authenticator is an **Identity Provider (IdP)**; adopting
scenarios are **Relying Parties (RPs)**. The contract these endpoints
realize is bound by hard invariants (PRD Appendix A–C):

- **Consumed API-to-API only.** There are no cross-origin browser calls
  anywhere in the model. A browser talks only to its own scenario's API;
  that API forwards same-origin if it must reach the authenticator (the
  pattern device-sync-hub uses today). RPs resolve scenario-authenticator
  by slug through `api-core/discovery`.
- **RPs verify tokens locally.** RPs fetch the JWKS once and verify
  RS256-signed tokens offline — they never call back per request. Token
  *validation* RPCs exist for diagnostics and opaque-token edge cases,
  not for the hot path.
- **Realm = tenant boundary.** Tokens are `aud`-scoped to a realm; a
  token minted for realm A is rejected by realm B's verifier. The default
  realm exists from day one (OT-P0-008).
- **Claims contract (carried over, do not break):** `user_id`/`sub`,
  `email`, `roles`, `iss: scenario-authenticator`, `aud` (realm).
  Algorithm locked to exactly **RS256** (reject `none`/HS confusion).
  Passwords hashed with **Argon2id**; only hashes and signed material at
  rest, never plaintext.

### Transport split — Connect everywhere except the REST edge

Everything is a Connect RPC **except** the narrow set of non-RPC web
standards that cannot be Connect calls. The complete REST edge is:

| REST endpoint | Why it must be REST | Tier |
|---|---|---|
| `GET /.well-known/jwks.json` | Public key set that RPs and standard JWT libraries fetch with a plain `GET`; a web standard, not an RPC. | P0 |
| `GET /api/v1/auth/oauth/{provider}/callback` | OAuth2/OIDC redirect target the upstream provider calls with a browser redirect; the shape is dictated by the provider. | P1 |
| `DELETE /api/v1/sessions/{id}` | Carried-over live cross-scenario revoke contract device-sync-hub calls today; preserved verbatim (or delivered as the Connect equivalent) in lockstep so the live consumer needs zero changes. | P0 |
| `POST /api/v1/saml/{realm}/acs` | SAML Assertion Consumer Service the IdP POSTs to; a SAML web standard. | P2 |
| `GET /health`, `GET /api/v1/health` | Operational probe lifecycle systems and load balancers read without a Connect client. | shipped |

Every other endpoint in this document is a Connect RPC at
`POST /vrooli.scenario_authenticator.v1.<domain>.<Service>/<Method>`.

---

## System

### `GET /health` (shipped)

Service health check. Returns API readiness plus dependency status.
Also mounted at `/api/v1/health` for client callers. This is an
operational REST exception by design: lifecycle systems, load balancers,
and curl probes must be able to read it without a Connect client.

| | |
|---|---|
| **Auth** | None |
| **Response** | `Response { status: string, readiness: bool, service: string, timestamp: string, version: string, uptime_seconds: int64, dependencies: map<string, DependencyStatus> }` |
| **Errors** | None — always returns 200 with `status: "unhealthy"` if a dependency fails |
| **CLI** | `scenario-authenticator status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at
`packages/proto/schemas/scenario-authenticator/v1/health/health.proto`
and mirrors `api-core/health.Response` field-for-field. Dependency
status will include SQLite (storage seam) and Redis once they are wired.

---

## identity (P0)

Accounts and credentials. Realm-scoped principals, Argon2id password
hashes, email-verification state, and password reset/recovery. Owns no
token minting (that is `tokens`) and no plaintext — only hashes at rest.
Proto: `…/v1/identity/identity.proto`.

> Error relay is faithful but **must not leak account existence** where
> that aids enumeration: registration of an existing email and a login
> with a bad password both surface a generic message; password-reset
> always returns success regardless of whether the email exists.

### `IdentityService/Register` — P0

Create a realm-scoped account and return the initial token pair.

| | |
|---|---|
| **Auth** | None (public; rate-limited + lockout-guarded) |
| **Request** | `RegisterRequest { realm: string (default realm if empty), email: string, password: string, username: string (optional) }` |
| **Response** | `RegisterResponse { user: User, access_token: string, refresh_token: string }` |
| **Errors** | `invalid_argument` — bad email/weak password (faithful validation reason)<br>`already_exists` — email taken in realm (relayed without confirming existence where enumeration matters)<br>`internal` |
| **CLI** | `scenario-authenticator auth register --email <e> --password <p> [--realm <r>] [--username <u>]` |

### `IdentityService/Login` — P0

Verify credentials and issue a token pair plus a tracked session.

| | |
|---|---|
| **Auth** | None (public; rate-limited + lockout-guarded) |
| **Request** | `LoginRequest { realm: string, email: string, password: string }` |
| **Response** | `LoginResponse { user: User, access_token: string, refresh_token: string, mfa_required: bool, mfa_challenge_id: string }` |
| **Errors** | `unauthenticated` — invalid credentials (generic; no account-existence leak)<br>`failed_precondition` — account locked out<br>`internal` |
| **CLI** | `scenario-authenticator auth login --email <e> --password <p> [--realm <r>]` |

When `mfa_required` is true the response carries no tokens; the client
completes `MfaService/VerifyChallenge` (see [mfa](#mfa-p1)) to obtain them.

### `IdentityService/GetCurrentUser` (whoami) — P0

Resolve the principal behind the presented access token.

| | |
|---|---|
| **Auth** | Bearer access token |
| **Request** | `GetCurrentUserRequest {}` |
| **Response** | `GetCurrentUserResponse { user: User }` |
| **Errors** | `unauthenticated` — missing/expired/invalid token |
| **CLI** | `scenario-authenticator auth whoami` |

### `IdentityService/RequestPasswordReset` — P1

Begin password recovery. Always returns success (no enumeration); a
single-use, expiring reset token is minted only if the email exists.

| | |
|---|---|
| **Auth** | None |
| **Request** | `RequestPasswordResetRequest { realm: string, email: string }` |
| **Response** | `RequestPasswordResetResponse { accepted: bool (always true) }` |
| **Errors** | `internal` |
| **CLI** | `scenario-authenticator auth reset-request --email <e> [--realm <r>]` |

### `IdentityService/CompletePasswordReset` — P1

Complete recovery with a valid, unexpired, single-use reset token; all
sessions for the principal are revoked on success.

| | |
|---|---|
| **Auth** | Reset token (in request body) |
| **Request** | `CompletePasswordResetRequest { token: string, new_password: string }` |
| **Response** | `CompletePasswordResetResponse { success: bool }` |
| **Errors** | `invalid_argument` — weak password<br>`failed_precondition` — invalid/expired/used token<br>`internal` |
| **CLI** | `scenario-authenticator auth reset-complete --token <t> --password <p>` |

### `IdentityService/VerifyEmail` — P1

Confirm an email address with a single-use verification token.

| | |
|---|---|
| **Auth** | Verification token (in request body) |
| **Request** | `VerifyEmailRequest { token: string }` |
| **Response** | `VerifyEmailResponse { success: bool }` |
| **Errors** | `failed_precondition` — invalid/expired token |
| **CLI** | `scenario-authenticator auth verify-email --token <t>` |

#### `User` shape (planned)

| Field | Type | Notes |
|---|---|---|
| `id` | string (UUID) | `sub`/`user_id` in claims |
| `realm` | string | Owning realm (tenant boundary) |
| `email` | string | Realm-scoped; same email in another realm is a distinct principal |
| `username` | string | Optional |
| `roles` | `string[]` | Emitted as a token claim |
| `email_verified` | bool | |
| `created_at` | `google.protobuf.Timestamp` | |
| `last_login` | `google.protobuf.Timestamp` | |

---

## tokens (P0)

Token issuance + RS256 signing + JWKS publication + rotating
refresh-token families with reuse detection. Owns the load-or-generate
signing keypair persisted to the storage root. Proto: `…/v1/tokens/tokens.proto`.

### `GET /.well-known/jwks.json` (REST edge) — P0

The published JSON Web Key Set. RPs and standard JWT libraries fetch this
once and verify RS256 tokens **locally and offline** — this is the
keystone of the local-verify contract. Exposes only the public key(s).
Also mounted at `/api/v1/auth/jwks` for the carried-over consumer path.

| | |
|---|---|
| **Auth** | None (public by design) |
| **Response** | Standard JWKS document: `{ keys: [ { kty, kid, alg: "RS256", use: "sig", n, e } ] }` |
| **Errors** | `500 internal` — key load failure |
| **CLI** | `scenario-authenticator token jwks` |

```bash
curl "http://localhost:${API_PORT}/.well-known/jwks.json"
```

### `TokensService/Refresh` — P0

Exchange a refresh token for a new access/refresh pair (rotation).
Presenting an already-rotated (reused) refresh token revokes the **entire
token family** and writes an audit event (reuse detection, OT-P0-003).

| | |
|---|---|
| **Auth** | Refresh token (in request body) |
| **Request** | `RefreshRequest { refresh_token: string }` |
| **Response** | `RefreshResponse { access_token: string, refresh_token: string }` |
| **Errors** | `unauthenticated` — invalid/unknown refresh token<br>`failed_precondition` — reused token (family revoked)<br>`internal` |
| **CLI** | `scenario-authenticator auth refresh --refresh-token <t>` |

### `TokensService/Validate` — P0

Server-side validation of an access token (diagnostics / opaque-token
edge cases). **Not the RP hot path** — RPs verify locally via JWKS. Honors
the token blacklist and realm `aud` scoping; rejects `none`/HS confusion.

| | |
|---|---|
| **Auth** | None (token supplied in request) |
| **Request** | `ValidateRequest { token: string, expected_aud: string (optional realm check) }` |
| **Response** | `ValidateResponse { valid: bool, claims: Claims }` |
| **Errors** | Never errors on an invalid token — returns `valid: false`; `internal` only on infra failure |
| **CLI** | `scenario-authenticator token validate --token <t>` |

#### `Claims` shape (carried over — do not break)

| Claim | Type | Notes |
|---|---|---|
| `sub` / `user_id` | string | Principal id |
| `email` | string | |
| `roles` | `string[]` | RP trusts these for coarse realm roles |
| `scopes` | `string[]` | P1 — per-realm scope claims |
| `iss` | string | Always `scenario-authenticator` |
| `aud` | string | Realm audience; verifier rejects a mismatched `aud` |
| `exp` / `iat` | int64 | Standard JWT timestamps |

---

## sessions (P0)

Server-tracked sessions backed by Redis hot state; list, per-session
revoke, and "log out everywhere". Proto: `…/v1/sessions/sessions.proto`.

### `SessionsService/ListSessions` — P0

List active sessions for the caller (or, for an admin, a target user /
all sessions in scope).

| | |
|---|---|
| **Auth** | Bearer access token |
| **Request** | `ListSessionsRequest { user_id: string (optional; admin only for others), scope: string (e.g. "all"; admin only), limit: int32 }` |
| **Response** | `ListSessionsResponse { sessions: Session[], total: int32 }` |
| **Errors** | `permission_denied` — non-admin requesting another user / all<br>`internal` |
| **CLI** | `scenario-authenticator session list [--user-id <id>] [--scope all] [--limit <n>]` |

### `SessionsService/RevokeSession` — P0

Revoke one session (blacklist its access token, revoke its refresh
token, drop the Redis record). Mirrors the REST `DELETE` below.

| | |
|---|---|
| **Auth** | Bearer access token (owner or admin) |
| **Request** | `RevokeSessionRequest { session_id: string }` |
| **Response** | `RevokeSessionResponse { success: bool }` |
| **Errors** | `not_found` — unknown session<br>`permission_denied` — not owner/admin<br>`internal` |
| **CLI** | `scenario-authenticator session revoke <session-id>` |

### `DELETE /api/v1/sessions/{id}` (REST edge — carried over) — P0

The **live cross-scenario revoke contract** device-sync-hub calls today.
Preserved verbatim (or delivered as the Connect `RevokeSession` above in
lockstep) so the live consumer needs zero changes. Same semantics and
auth as `RevokeSession`.

| | |
|---|---|
| **Auth** | Bearer access token (owner or admin) |
| **Path params** | `id` — session identifier |
| **Response** | `{ success: true, message: string }` |
| **Errors** | `404 not_found`, `403` (insufficient permissions), `500 internal` |
| **CLI** | `scenario-authenticator session revoke <session-id>` |

### `SessionsService/RevokeAllSessions` (log out everywhere) — P0

Revoke every session for the principal (e.g. after a password change).

| | |
|---|---|
| **Auth** | Bearer access token (self), or admin for a target |
| **Request** | `RevokeAllSessionsRequest { user_id: string (optional; admin only) }` |
| **Response** | `RevokeAllSessionsResponse { revoked: int32 }` |
| **Errors** | `permission_denied`, `internal` |
| **CLI** | `scenario-authenticator auth logout --all` |

### `SessionsService/Logout` — P0

End the current session and blacklist its access token.

| | |
|---|---|
| **Auth** | Bearer access token |
| **Request** | `LogoutRequest {}` |
| **Response** | `LogoutResponse { success: bool }` |
| **Errors** | `unauthenticated`, `internal` |
| **CLI** | `scenario-authenticator auth logout` |

#### `Session` shape (planned)

| Field | Type | Notes |
|---|---|---|
| `session_id` | string | |
| `user_id` | string | |
| `ip_address` | string | |
| `user_agent` | string | |
| `created_at` | `google.protobuf.Timestamp` | |
| `expires_at` | `google.protobuf.Timestamp` | |

---

## realms (P0 default realm → P1 multi-realm)

The tenant boundary. A realm is an isolated identity namespace with its
own user pool, branding, password policy, token TTLs, enabled methods,
and allowed redirect URIs. At P0 a single default realm exists and issues
`aud`-scoped tokens; full CRUD lands at P1. Proto: `…/v1/realms/realms.proto`.

### `RealmsService/GetRealm` — P0

Fetch a realm's public configuration (default realm at P0).

| | |
|---|---|
| **Auth** | Bearer access token (admin for full config; public subset for hosted login branding) |
| **Request** | `GetRealmRequest { id: string }` |
| **Response** | `GetRealmResponse { realm: Realm }` |
| **Errors** | `not_found`, `internal` |
| **CLI** | `scenario-authenticator realm get <id>` |

### `RealmsService/ListRealms` — P1

List realms (system admin).

| | |
|---|---|
| **Auth** | Bearer access token (system admin) |
| **Response** | `ListRealmsResponse { realms: Realm[] }` |
| **Errors** | `permission_denied`, `internal` |
| **CLI** | `scenario-authenticator realm list` |

### `RealmsService/CreateRealm` — P1

Provision a new tenant realm (B2B per-customer or B2C per-product).

| | |
|---|---|
| **Auth** | Bearer access token (system admin) |
| **Request** | `CreateRealmRequest { slug, display_name, policy: RealmPolicy, branding: RealmBranding }` |
| **Response** | `CreateRealmResponse { realm: Realm }` |
| **Errors** | `invalid_argument`, `already_exists`, `internal` |
| **CLI** | `scenario-authenticator realm create --slug <s> --name <n> [...]` |

### `RealmsService/UpdateRealm` — P1

Update per-realm policy, branding, token TTLs, enabled methods, redirect
URIs.

| | |
|---|---|
| **Auth** | Bearer access token (realm or system admin) |
| **Request** | `UpdateRealmRequest { id, policy?, branding?, token_ttls?, enabled_methods?, redirect_uris? }` |
| **Response** | `UpdateRealmResponse { realm: Realm }` |
| **Errors** | `not_found`, `invalid_argument`, `internal` |
| **CLI** | `scenario-authenticator realm update <id> [...]` |

### `RealmsService/DeleteRealm` — P1

Delete a realm and its user pool (destructive; confirmation-gated in UI).

| | |
|---|---|
| **Auth** | Bearer access token (system admin) |
| **Request** | `DeleteRealmRequest { id: string }` |
| **Response** | `DeleteRealmResponse { success: bool }` |
| **Errors** | `not_found`, `failed_precondition` (default realm cannot be deleted), `internal` |
| **CLI** | `scenario-authenticator realm delete <id>` |

#### `Realm` shape (planned)

| Field | Type | Notes |
|---|---|---|
| `id` | string | |
| `slug` | string | `aud` audience value |
| `display_name` | string | |
| `is_default` | bool | The single default realm |
| `policy` | `RealmPolicy` | Password rules, lockout thresholds, enabled methods |
| `branding` | `RealmBranding` | Logo, colors (rendered on hosted login at P1) |
| `token_ttls` | `TokenTtls` | Access + refresh TTLs |
| `redirect_uris` | `string[]` | Allowed OAuth/OIDC redirect targets |

---

## authorization (P0 admin/user → P1 scopes)

Role and scope *definitions* per realm and their assignment; emitted as
token claims. Fine-grained "can-they" enforcement is delegated to RPs
(a hosted policy engine is P2, OT-P2-004). Proto: `…/v1/authorization/authorization.proto`.

| RPC | Tier | Purpose | CLI |
|---|---|---|---|
| `ListRoles` | P0 | List realm roles (admin/user baseline). | `scenario-authenticator role list` |
| `CreateRole` | P1 | Define a realm role beyond admin/user. | `scenario-authenticator role create --name <n>` |
| `DeleteRole` | P1 | Remove a realm role. | `scenario-authenticator role delete <name>` |
| `AssignRole` | P0 | Assign a role to a user. | `scenario-authenticator role assign --user <id> --role <r>` |
| `RevokeRole` | P0 | Remove a role assignment. | `scenario-authenticator role revoke --user <id> --role <r>` |
| `ListScopes` | P1 | List realm scope definitions. | `scenario-authenticator scope list` |
| `CreateScope` | P1 | Define a realm scope. | `scenario-authenticator scope create --name <n>` |
| `AssignScope` | P1 | Attach a scope to a role/user. | `scenario-authenticator scope assign [...]` |

Common errors: `permission_denied` (non-admin), `not_found`,
`invalid_argument`, `internal`. Roles/scopes are surfaced as the `roles`
/ `scopes` token claims for RP consumption.

---

## audit (P0)

Append-only, queryable log of security-relevant events (sign-in,
sign-out, token-family revoke, MFA changes, admin actions). Proto: `…/v1/audit/audit.proto`.

### `AuditService/QueryEvents` — P0

Query audit events with filters, newest-first, per realm.

| | |
|---|---|
| **Auth** | Bearer access token (admin) |
| **Request** | `QueryEventsRequest { realm: string, user_id: string (optional), action: string (optional), since: Timestamp (optional), limit: int32 }` |
| **Response** | `QueryEventsResponse { events: AuditEvent[], total: int32 }` |
| **Errors** | `permission_denied`, `internal` |
| **CLI** | `scenario-authenticator audit query [--user-id <id>] [--action <a>] [--since <ts>] [--limit <n>]` |

#### `AuditEvent` shape (planned)

| Field | Type | Notes |
|---|---|---|
| `id` | string | |
| `user_id` | string | May be empty for pre-auth events |
| `action` | string | e.g. `user.logged_in`, `token.family.revoked` |
| `ip_address` | string | |
| `user_agent` | string | |
| `success` | bool | |
| `metadata` | `map<string,string>` | Event-specific detail |
| `created_at` | `google.protobuf.Timestamp` | |

---

## mfa (P1)

Second factors: TOTP enrollment/challenge/recovery codes and WebAuthn
passkeys. Proto: `…/v1/mfa/mfa.proto`.

| RPC | Tier | Purpose | CLI |
|---|---|---|---|
| `EnrollTotp` | P1 | Begin TOTP enrollment; returns secret + provisioning URI/QR. | `scenario-authenticator mfa enroll-totp` |
| `ActivateTotp` | P1 | Confirm enrollment with a code; returns recovery codes (shown once). | `scenario-authenticator mfa activate-totp --code <c>` |
| `VerifyChallenge` | P1 | Complete a login MFA challenge (TOTP code or recovery code); returns the token pair. | `scenario-authenticator mfa verify --challenge <id> --code <c>` |
| `DisableTotp` | P1 | Remove TOTP from the account. | `scenario-authenticator mfa disable-totp` |
| `RegisterPasskey` | P1 | Begin WebAuthn passkey registration (returns creation options). | `scenario-authenticator mfa passkey register` |
| `AuthenticatePasskey` | P1 | Complete a passkey assertion challenge. | (browser/self-service flow; CLI assists) |
| `ListPasskeys` | P1 | List registered passkeys. | `scenario-authenticator mfa passkey list` |
| `RemovePasskey` | P1 | Remove a passkey credential. | `scenario-authenticator mfa passkey remove <id>` |

Per-realm enforcement policy (whether MFA is required) lives on the realm
(`RealmPolicy.enabled_methods`). Recovery codes are single-use and stored
hashed. Common errors: `unauthenticated`, `failed_precondition` (invalid
code / wrong enrollment state), `not_found`, `internal`.

---

## federation (P1 social → P2 SAML/OIDC-provider)

Inbound external identity. OAuth2/OIDC social providers (Google, GitHub,
Microsoft) with account linking at P1; SAML and OIDC-provider mode at P2.
OAuth CSRF state lives in Redis. Proto: `…/v1/federation/federation.proto`.

### `FederationService/ListProviders` — P1

List the realm's configured/enabled social providers.

| | |
|---|---|
| **Auth** | None (public; drives hosted-login provider buttons) |
| **Request** | `ListProvidersRequest { realm: string }` |
| **Response** | `ListProvidersResponse { providers: Provider[] }` |
| **CLI** | `scenario-authenticator oauth providers [--realm <r>]` |

### `FederationService/StartOAuth` — P1

Begin a social sign-in: mints CSRF state (stored in Redis with a short
TTL) and returns the upstream authorization URL the caller redirects to.

| | |
|---|---|
| **Auth** | None |
| **Request** | `StartOAuthRequest { realm: string, provider: string, redirect_uri: string (must be allow-listed on the realm) }` |
| **Response** | `StartOAuthResponse { authorization_url: string, state: string }` |
| **Errors** | `invalid_argument` — unknown/disabled provider or non-allow-listed redirect<br>`internal` |
| **CLI** | `scenario-authenticator oauth start --provider <p> [--realm <r>]` |

### `GET /api/v1/auth/oauth/{provider}/callback` (REST edge) — P1

The redirect target the upstream provider calls (browser redirect). The
shape — `?code=…&state=…` query params — is dictated by the provider, so
this is a REST web-standard exception, not an RPC. Validates CSRF
`state`, exchanges the code, finds-or-links the account, and completes
the sign-in.

| | |
|---|---|
| **Auth** | None (CSRF-protected by `state`) |
| **Path params** | `provider` — `google` / `github` / `microsoft` |
| **Query** | `code`, `state` |
| **Errors** | `400 invalid_request` — invalid/expired `state` or missing `code`<br>`500 internal` — token exchange / userinfo failure |

### `POST /api/v1/saml/{realm}/acs` (REST edge) — P2

SAML 2.0 Assertion Consumer Service. The IdP POSTs the SAML response
here; a SAML web standard, so REST. (P2; deferred.)

---

## apikeys (P1)

Non-human principals: hashed API keys and the client-credentials grant
so machine/service callers authenticate without a human session. Proto:
`…/v1/apikeys/apikeys.proto`.

### `ApiKeysService/CreateApiKey` — P1

Create an API key. The plaintext key is returned **once** on creation;
only its hash is stored.

| | |
|---|---|
| **Auth** | Bearer access token |
| **Request** | `CreateApiKeyRequest { name: string, scopes: string[], expires_in_days: int32 (0 = no expiry) }` |
| **Response** | `CreateApiKeyResponse { id: string, name: string, key: string (shown once), scopes: string[], expires_at: Timestamp }` |
| **Errors** | `invalid_argument` — missing name<br>`internal` |
| **CLI** | `scenario-authenticator apikey create --name <n> [--scope <s>...] [--expires-in <days>]` |

### `ApiKeysService/ListApiKeys` — P1

List the caller's non-revoked API keys (never returns the secret).

| | |
|---|---|
| **Auth** | Bearer access token |
| **Response** | `ListApiKeysResponse { keys: ApiKey[] }` |
| **CLI** | `scenario-authenticator apikey list` |

### `ApiKeysService/RevokeApiKey` — P1

Revoke an API key by id (ownership-checked).

| | |
|---|---|
| **Auth** | Bearer access token (owner) |
| **Request** | `RevokeApiKeyRequest { id: string }` |
| **Response** | `RevokeApiKeyResponse { success: bool }` |
| **Errors** | `not_found` — unknown/already-revoked key<br>`internal` |
| **CLI** | `scenario-authenticator apikey revoke <id>` |

### `ApiKeysService/IssueClientToken` (client-credentials grant) — P1

Exchange an API key (client credentials) for a short-lived access token
carrying the key's scopes — no human session involved.

| | |
|---|---|
| **Auth** | API key (in request) |
| **Request** | `IssueClientTokenRequest { api_key: string, realm: string }` |
| **Response** | `IssueClientTokenResponse { access_token: string, expires_at: Timestamp }` |
| **Errors** | `unauthenticated` — invalid/expired/revoked key<br>`internal` |
| **CLI** | `scenario-authenticator apikey token --api-key <k> [--realm <r>]` |

---

<!-- EXAMPLE-DOMAIN:notes START -->
## Example domain — `notes` (removed by `vrooli scenario detemplate`)

The scaffold ships one fully worked CRUD vertical slice as a copyable
reference. Copy its layering when implementing the first real auth domain
(`identity` or `tokens`); `vrooli scenario detemplate scenario-authenticator`
removes this section once a real domain is green.

### `POST /vrooli.scenario_authenticator.v1.notes.NotesService/ListNotes`

List notes through the generated Connect-RPC service, newest-first.

| | |
|---|---|
| **Auth** | None (template default; scenarios add auth as needed) |
| **Response** | `ListNotesResponse { notes: Note[] }` (capped at 100 by `notes.Service`) |
| **Errors** | `500 internal` — repository read failure |
| **CLI** | `scenario-authenticator notes list` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.scenario_authenticator.v1.notes.NotesService/ListNotes" \
  -H 'Content-Type: application/json' \
  -d '{}'
```

### `POST /vrooli.scenario_authenticator.v1.notes.NotesService/CreateNote`

Create a note through the generated Connect-RPC service.

| | |
|---|---|
| **Auth** | None (template default) |
| **Request** | `CreateNoteRequest { title: string (required), body: string (optional) }` |
| **Response** | `CreateNoteResponse { note: Note }` |
| **Errors** | `invalid_argument` — missing/whitespace-only title<br>`internal` — repository write failure |
| **CLI** | `scenario-authenticator notes create --title <title> [--body <body>]` |

Title validation (non-empty after whitespace trim) lives in
`internal/notes/service.go`, **not** the handler. The Connect handler only
translates `notes.ErrInvalidNote` into `invalid_argument`.

### `POST /api/v1/notes/{id}/attachments`

Upload opaque file bytes through the documented REST multipart exception.
The response is still proto-typed metadata.

| | |
|---|---|
| **Auth** | None (template default) |
| **Path params** | `id` — note identifier |
| **Request** | `multipart/form-data` with `file` part |
| **Response** | `UploadAttachmentResponse { attachment: Attachment }` |
| **Errors** | `400 invalid_request` — malformed multipart or missing file<br>`404 not_found` — no note with that id<br>`500 internal` — blob or metadata persistence failure |
| **CLI** | `scenario-authenticator notes attach <id> --file <path>` |

Defined in `packages/proto/schemas/scenario-authenticator/v1/notes/notes.proto`.
<!-- EXAMPLE-DOMAIN:notes END -->

---

## Adding a new endpoint

Once the first real domain is green, adding an endpoint follows the
standard proto-first flow:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/scenario-authenticator/v1/<domain>/`. Prefer a
   Connect RPC; only use a REST path when it is one of the documented web
   standards (JWKS, OAuth/OIDC callback, SAML ACS, the carried-over
   session revoke). Run `make generate`.
2. Implement the generated handler method in
   `api/handlers/<domain>/connect_handler.go`; keep it thin (business
   logic lives in the service, not the handler — see the carried-over
   crypto invariants in [`../internal/SECURITY.md`](../internal/SECURITY.md)).
3. Update endpoint metadata in `api/handlers/<domain>/module.go`.
4. Bind the CLI mirror (or list it in `omitted[]` with a reason) in
   [`cli/manifest.json`](../../cli/manifest.json) — the single source of
   truth for the CLI surface.
5. Run `make endpoints`; do not edit
   [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) by hand.
6. Update this document and add tests for the touched layers.
7. Add a row to [`../internal/SEAMS.md`](../internal/SEAMS.md) if you
   introduced a new test-substitutable interface.

The CI gate enforces endpoint-manifest freshness and the API↔CLI mapping
contract (every Connect endpoint is bound or omitted in `cli/manifest.json`).

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md) — env vars, signing-key path, realm/policy knobs
- [`ui-manifest.md`](ui-manifest.md) — the UI surface that composes these domains
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — domain ownership map
- [`../../PRD.md`](../../PRD.md) — operational targets + Appendix (IdP/RP, realms, invariants)
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — handler/service/repository seams
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — RS256/JWKS/Argon2id posture
