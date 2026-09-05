# Error Handling

scenario-authenticator uses one error path for proto-typed operations and
one documented exception path for the non-RPC web standards (JWKS, OAuth
callbacks, SAML ACS). Because this is an auth boundary, error handling is
**security-relevant**: the wrong error message leaks account existence; the
wrong code tells a caller "retry" when it should say "stop." This document
defines the typed-error contract and the rules that keep it safe.

> **Status: implemented contract with planned extensions.** The typed error
> mapping below describes the live Connect/API boundary. Rows or domains
> explicitly marked future apply only to deferred MFA, federation, recovery,
> or multi-realm work; no `notes` template domain ships.

## Proto-Typed Operations

Proto-typed UI, CLI, and inter-scenario calls use Connect-RPC. Errors move
through three layers:

1. Domain/service code returns **typed sentinels** such as
   `identity.ErrInvalidCredentials`, `identity.ErrEmailTaken`, or
   `tokens.ErrTokenInvalid`.
2. The API transport edge maps those sentinels to `connect.Error` values
   in the domain's `service_error_mapping.go`.
3. The UI receives `ConnectError`, maps `ConnectError.code` to an
   `errors.<code>` i18n key with `ui/src/lib/errorMessage.ts`, and renders
   localized copy.

The CLI uses the same `connect.Error` values through cli-core. Human output
is English for now; future CLI i18n should use the same code names as the
UI catalog instead of string-matching messages.

## Sentinel Mapping

## Connect-code contract (auth domain errors)

| Domain error (sentinel) | Connect code | When | Notes |
|---|---|---|---|
| Bad credentials / unknown account on login | `unauthenticated` | Login with wrong password **or** no such account | **Same code, same message** for both — enumeration-safe (see below). |
| Invalid / expired / malformed token | `unauthenticated` | `Validate`, `Refresh`, `Logout` with a bad or expired token | Includes `alg=none`/HS-confusion rejection and signature failure — all surface as `unauthenticated`, never `internal`. |
| Reused (rotated) refresh token | `unauthenticated` | Refresh reuse detected | The reuse also revokes the token family and is audited ([`SECURITY.md`](SECURITY.md)); the caller just sees "unauthenticated." |
| Cross-realm token (`aud` mismatch) | `unauthenticated` | A token minted for realm A presented to realm B | The rejection is the security property; the code does not reveal which realm it belonged to. |
| Email already registered | `already_exists` | Register with a taken email | See the enumeration trade-off note below. |
| Weak / malformed input (password too short, bad email format, missing field) | `invalid_argument` | Register / password-change / realm-config validation | The validation **reason is relayed faithfully** (e.g. "password must be at least N characters") — this is not enumeration-sensitive and helps the user. |
| Role / scope / realm-admin failure | `permission_denied` | Management endpoint called by an under-privileged principal | RBAC enforced at the service layer; the UI/CLI never decide locally. |
| Dependency down (Redis unreachable, signing key unavailable) | `unavailable` | Session/revocation/rate-limit op when Redis is down; token issuance when the key is missing | Signals "retry later," distinct from a client error. Redis is required, not optional ([`PERFORMANCE.md`](PERFORMANCE.md)). |
| Unknown service/repository error | `internal` | Unexpected failure | The underlying error reaches operator logs; the **client body carries no internal detail**. |

When you add a domain, keep the mapping file next to that domain's service
layer. The handler should call the mapper instead of switching on domain
error types inline.

## Enumeration-safe messaging (the load-bearing rule)

An auth service must not let error responses reveal which accounts exist.
The discipline:

- **Login** returns the **same code and the same generic message** whether
  the account doesn't exist or the password is wrong — `unauthenticated`,
  "invalid credentials." Never "no such user" vs "wrong password."
- **Password reset** always responds success-shaped ("if the email exists,
  a reset link has been sent") regardless of whether the email is
  registered. (The old scenario already does this; the port keeps it.)
- **Registration** is the unavoidable tension: "email already registered"
  (`already_exists`) is genuinely useful to a legitimate user but does leak
  existence. Resolve it the way the old scenario does — a clear conflict
  message at the registration boundary — and lean on **rate limiting +
  account lockout** ([`SECURITY.md`](SECURITY.md)) to blunt enumeration-by-
  registration. Do not add the same existence signal to the *login* path.
- **MFA / token errors** never reveal whether a second factor is enrolled
  or which factor failed beyond what the flow strictly needs.

The rule of thumb: `invalid_argument` may relay the *input* validation
reason faithfully (that's about the request, not the account); credential/
identity errors stay generic.

## Faithful relay without leaking internals

Two kinds of "relay" pull in opposite directions; keep them straight:

- **Relay the validation reason** for `invalid_argument` — the user needs
  to know "password too short" to fix their input. This is faithful relay
  of a *client-fixable* condition.
- **Relay a federated provider's failure** (P1 OAuth/OIDC) at the right
  altitude: surface "social sign-in failed / was cancelled" to the user,
  log the provider's detailed error for operators, but **do not** forward
  raw provider error bodies, tokens, or internal URLs to the client.
- **Never relay internals.** A `internal` error returns a generic body; the
  real error, stack, SQL, or secret material goes to operator logs only.
  Tokens, password hashes, and the private key never appear in any error
  body or log line.

## Multipart REST Exceptions

The proto-typed Connect surface is the default. The only non-RPC HTTP
surfaces are the **web standards** that cannot be RPCs:
`/.well-known/jwks.json`, OAuth/OIDC redirect callbacks, and (P2) SAML ACS.
These (and any template multipart upload) use a stable REST error envelope
through `internal/httpx.WriteError`; the UI maps `ApiError.code` through the
same `errorMessage(...)` utility as Connect errors.

For these web-standard endpoints, error responses still respect the
enumeration and no-internal-leak rules above. A JWKS request when the
signing key is unavailable returns a service-unavailable status (the old
scenario returns `503` with "signing key not available"), not a `200` with
an empty set.

Use this split:

- **Connect-RPC** for messages that can be described by proto (Login,
  Register, Refresh, Logout, Validate, session/realm/role management).
- **REST** only for the non-RPC web standards (JWKS, callbacks, ACS) and
  template multipart.
- **Proto metadata responses** for REST results where a structured body
  applies.

Do not introduce a second general JSON transport for internal scenario
calls. If the payload is structured and Vrooli-owned, add a proto service
method.

## Cross-References

- [`SECURITY.md`](SECURITY.md) — enumeration resistance, no-secrets-in-logs, the threat model
- [`SEAMS.md`](SEAMS.md) — where the mapping file lives relative to each domain's service
- [`TESTING.md`](TESTING.md) — enumeration-safety and error-mapping tests
- [`../../PRD.md`](../../PRD.md) — Appendix A (the RP contract), Appendix C (carried-over invariants)
