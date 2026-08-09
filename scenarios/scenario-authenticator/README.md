# Scenario Authenticator

Scenario Authenticator is Vrooli’s local identity provider. It owns account
credentials, Argon2id password verification, RS256 access-token issuance,
rotating refresh-token families, JWKS publication, session revocation,
rate-limiting, and security audit events. Other scenarios resolve it by slug
and verify its tokens locally against the published JWKS; they do not send a
password or perform a per-request authorization callback.

## What You Get

The scenario provides a local, API-to-API identity boundary with short-lived
signed access tokens, invisible renewal on linked machines, and durable
session/audit controls. It deliberately does not claim true multi-realm,
MFA, federation, or API-key client-credential support yet.

## Runtime Surfaces

- Connect APIs under `api/handlers/` for registration, login, refresh, logout,
  validation, and session management.
- `/.well-known/jwks.json` for relying-party signature verification.
- The `authenticator` CLI for operator and owner account operations.
- SQLite persistence through the scenario storage seam and Redis-backed hot
  state for sessions and rate limiting.

Start and test the scenario through the Vrooli lifecycle:

```bash
make start
vrooli scenario test scenario-authenticator
```

The default realm is intentionally the only realm currently enabled. Realm
audiences and the wire claims (`user_id`, mirrored `sub`, `iss`, `aud`, and
RS256) are compatibility contracts for relying parties. Fine-grained scope
enforcement remains with each relying party; the authenticator stores opaque
scope grants and emits them explicitly in tokens.

## Documentation Map

See `docs/concepts/DOMAINS.md`, `docs/concepts/AUTHORIZATION.md`,
`docs/operations/RUNBOOK.md`, and `docs/internal/SECURITY.md` for ownership,
operation, and security contracts.

## Customize Safely

Keep lifecycle ownership, proto/manifest parity, RS256 verification, explicit
scope claims, and password secrecy intact when extending the scenario. Add
new authority through the governed scope catalog and authorization RPCs;
never add an ad-hoc CLI allowlist or pass a secret in argv.
