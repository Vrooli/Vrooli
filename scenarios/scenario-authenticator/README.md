# Scenario Authenticator

Scenario Authenticator is Vrooli’s identity provider. It owns account
credentials, Argon2id password verification, RS256 access-token issuance,
rotating refresh-token families, JWKS publication, session revocation,
rate-limiting, MFA primitives, machine bindings, coarse capability grants,
and security audit events. Other scenarios resolve it by slug and verify its
tokens locally against the published JWKS; they do not send a password or
perform a per-request authorization callback.

## What You Get

The scenario provides a local, API-to-API identity boundary with short-lived
signed access tokens, rotating refresh sessions, MFA enrollment/challenge
primitives, machine exchange, and durable session/audit controls. The shipped
deployment is a single default realm. True multi-realm administration,
external federation, API-key/client-credential management, and a complete
admin/self-service UI remain staged capabilities.

## Runtime Surfaces

- Connect APIs under `api/handlers/` for registration, login, refresh, logout,
  validation, session management, and MFA.
- Hosted login at `/auth/login`, including an explicit `View read-only version`
  path for applications whose mutations require a separate relying-party
  authorization step. The hosted page calls the authenticator's own typed
  AccountsService and never places bearer tokens in a URL.
- `/.well-known/jwks.json` for relying-party signature verification.
- The `authenticator` CLI for operator and owner account operations.
- SQLite persistence through the scenario storage seam and a tier-dependent
  hot-state store for sessions, revocation, and rate limiting. Durable local
  hot state is valid for one desktop/local replica; shared Redis is required
  when correctness spans multiple replicas.

Start and test the scenario through the Vrooli lifecycle:

```bash
make start
vrooli scenario test scenario-authenticator
```

The default realm is intentionally the only realm currently enabled. Realm
audiences and the wire claims (`user_id`, mirrored `sub`, `iss`, `aud`, and
RS256) are compatibility contracts for relying parties while the platform
migrates toward resource-specific audiences. Fine-grained scope enforcement
remains with each relying party; the authenticator stores coarse capability
grants and emits them explicitly in tokens.

Scenario-authenticator is not automatically required in every Tier 2 bundle.
A bundled application may use `personal_local` mode with no human sign-in,
while `local_multi_user` mode explicitly provisions a private local realm.
Remote desktop clients use the configured server authority. LPBS remains the
authority for customer accounts, subscriptions, downloads, and commercial
entitlements; account linking is explicit and revocable.

## Documentation Map

See the project-level [`../../docs/concepts/IDENTITY-AND-AUTHENTICATION.md`](../../docs/concepts/IDENTITY-AND-AUTHENTICATION.md),
[`../../docs/concepts/AUTHORIZATION.md`](../../docs/concepts/AUTHORIZATION.md),
`docs/concepts/DOMAINS.md`, `docs/concepts/AUTHORIZATION.md`,
`docs/operations/RUNBOOK.md`, and `docs/internal/SECURITY.md` for ownership,
deployment, operation, and security contracts.

## Customize Safely

Keep lifecycle ownership, proto/manifest parity, RS256 verification, explicit
scope claims, and password secrecy intact when extending the scenario. Add
new authority through the governed scope catalog and authorization RPCs;
never add an ad-hoc CLI allowlist or pass a secret in argv.
