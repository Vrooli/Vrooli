# Security — Scenario Authenticator

This document records the scenario's security and privacy posture. It is
the highest-stakes security document in the fleet: scenario-authenticator
is the **Identity Provider (IdP)** every other scenario trusts. A defect
here is not a local bug — it is a fleet-wide authentication failure. Read
this before touching crypto, tokens, hashing, sessions, realms, or any
verification path.

> **Status: documentation-first.** Nothing in this document is built yet.
> The auth core is a **verbatim crypto-port** of the working old scenario
> (see [`../../PRD.md`](../../PRD.md) Appendix C), not a clean-room
> reimplementation. The single biggest risk in the rewrite is re-deriving
> proven crypto incorrectly; the discipline below is "port, do not
> re-derive."

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists, and what is allowed at rest?
- How is access controlled, and where is the tenant boundary enforced?
- Where do secrets (the signing keypair) come from?
- Which threats are known and how are they mitigated?
- Why is this the highest-leverage boundary to get right?

## The governing principle — port the proven crypto, do not re-derive it

This rewrite is a scaffold-regen onto react-vite + the `api-core/storage`
seam, **plus** a faithful port of the old scenario's crypto. The old
scenario's RS256/JWKS/claims/keypair-persistence is correct and is
verified in production today by device-sync-hub. Re-deriving any of it
risks breaking a live consumer or, worse, silently weakening it.

The carried-over invariants (port verbatim):

- **RS256, algorithm locked.** Tokens are signed with RS256 against a
  persisted RSA keypair. Verification accepts **only** RS256 and rejects
  `none` and the HS-family. The old code asserts the parsed token's method
  is `*jwt.SigningMethodRSA` before trusting the key; the port keeps that
  assertion (algorithm-confusion defense — never let the attacker pick the
  algorithm).
- **JWKS at `/.well-known/jwks.json`** so Relying Parties verify locally,
  offline, with no per-request callback. Only the public modulus/exponent
  is published; the private key never leaves this service.
- **JWT claims/issuer**: `user_id`/`sub`, `email`, `roles`,
  `iss: scenario-authenticator`, plus `aud` (the realm) — identical to the
  old contract so device-sync-hub's verifier needs zero changes.
- **Load-or-generate persisted keypair** (`private.pem` / `public.pem`) in
  the storage root, so the key is stable across restarts. Generating a new
  key on every boot would invalidate every live token.

The one **deliberate strengthening**: password hashing moves from bcrypt
(`bcrypt.DefaultCost`, `golang.org/x/crypto/bcrypt` in the old scenario)
to **Argon2id**. This is not a re-derivation of the token crypto; it is a
documented hardening of the credential store, applied at the password
boundary only. See "Argon2id" below.

## Data Sensitivity

scenario-authenticator owns the fleet's most sensitive data class:
credentials and signing material. The rule is absolute — **only hashes and
signed material are stored at rest; never a plaintext secret.**

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Password hashes | critical | `identity` | Argon2id hash + per-hash salt/parameters. The plaintext password is never stored, never logged, never echoed. |
| RSA signing private key | critical | `tokens` | `private.pem` in the storage root, file perms `0600`. Signs every access token. Never published, never returned by any endpoint, never logged. |
| RSA signing public key | low (public by design) | `tokens` | Published at JWKS. Public by intent — RPs need it to verify. |
| Refresh-token family records | high | `tokens` | Only the **hash** of each refresh token at rest (the old scenario SHA-256-hashes refresh tokens in Redis; the port keeps hash-at-rest). The raw token is returned once to the client and never re-readable. |
| Session records | high | `sessions` | Redis hot state: user id, session id, IP, user-agent, expiry. Revocable per-session and per-user ("log out everywhere"). |
| TOTP secrets / recovery codes (P1) | critical | `mfa` | Second-factor seeds. Recovery codes hashed at rest; TOTP shared secret stored encrypted/secret-managed, shown to the user only at enrollment. |
| API key material (P1) | high | `apikeys` | Only the **hash** of each API key at rest; the raw key is shown once at creation. |
| Linked external identities + OAuth CSRF state (P1) | medium-high | `federation` | Provider subject ids + tokens; CSRF `state` is one-time-use in Redis. |
| Audit events | medium | `audit` | Security-relevant events (who/what/when/result). Must not contain plaintext credentials or raw tokens. |
| Realm config (policy, branding, redirect URIs) | medium | `realms` | Per-tenant policy; misconfiguration is a tenant-isolation risk (see below). |

## Auth And Authorization — the IdP↔RP boundary

scenario-authenticator answers **"valid principal + coarse realm
roles/scopes."** It does **not** answer fine-grained "can this principal do
this action on this resource" — that stays with the Relying Party (PRD
Appendix A). A delegated policy engine is a P2 ambition (OT-P2-004), not
the default.

- **RPs verify locally, never call back.** A consuming scenario fetches
  JWKS once, caches the public key, and verifies every token in-process.
  There is no per-request `/validate` round-trip on the hot path. This is
  both the scale lever ([`PERFORMANCE.md`](PERFORMANCE.md)) and a security
  property: the authenticator is not a single point of latency or failure
  on every authenticated request.
- **API-to-API only — no cross-origin browser calls.** A browser talks
  only to its own scenario's API, which forwards same-origin to the
  authenticator if it must (the device-sync-hub forwarder pattern).
  Adopters resolve scenario-authenticator by slug via `api-core/discovery`,
  never by a hard-coded cross-origin URL. This keeps tokens and cookies off
  the cross-origin attack surface entirely.
- **Management endpoints enforce RBAC.** Admin/user roles are carried over
  and enforced at the API/service layer (OT-P0-009). The UI and CLI never
  enforce business authorization locally — they render and relay; the
  server decides.

## aud-scoping is cross-tenant isolation (a misconfig here is a token leak)

The realm is the tenant boundary (PRD Appendix B). Even the single default
realm issues `aud`-scoped tokens, and verification **rejects a token whose
`aud` does not match the verifying realm.** This is not a P1 nicety —
it ships in P0 (OT-P0-008) precisely because retrofitting tenant isolation
is a re-architecture, and a gap here is a **cross-tenant token leak**: a
token minted for realm A accepted by realm B.

- Enforced at **both** issuance (the `aud` claim is stamped from the
  issuing realm) and verification (the verifier checks `aud` against its
  realm). Neither side alone is sufficient.
- The same email in two realms is **two distinct principals**. Realm-scoped
  user pools are the data-layer half of this isolation.
- This invariant has a **mandatory regression test** — the cross-realm
  rejection test in [`TESTING.md`](TESTING.md) is a must-have, not optional
  coverage.

## Threat Model

This scenario is **pre-implementation**; the table below enumerates the
defenses that must be built (or ported), not deficiencies in shipped code.

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Algorithm confusion (`none` / HS256 forgery) | Attacker forges tokens by downgrading the verify algorithm. | Lock verification to RS256; reject `none` and HS-family by asserting the signing method is RSA before trusting the key. Unit-tested with crafted `alg=none`/`alg=HS256` tokens. | to-port |
| Cross-tenant token acceptance (`aud` not enforced) | A token for realm A is accepted by realm B — cross-tenant breach. | Stamp `aud` at issuance from the issuing realm; reject mismatched `aud` at verification. Mandatory cross-realm rejection test. | to-build |
| Refresh-token theft / replay | A stolen refresh token is reused to mint access tokens. | Rotating refresh tokens with **reuse detection**: presenting a rotated (already-redeemed) refresh token revokes the entire token family and is audited. | to-build |
| Password brute-force / credential stuffing | Online guessing of credentials. | Rate limiting on auth endpoints + account lockout after repeated failures (in-memory primary state, Redis for cross-replica coordination). | to-port |
| Credential-store compromise (DB exfiltration) | Stolen hashes are cracked offline. | Argon2id (memory-hard) at a documented cost; per-hash salt; only hashes at rest. No plaintext, ever. | to-build |
| Account enumeration | Differential responses reveal which emails are registered. | Enumeration-resistant messaging on login, register, and password-reset — generic "invalid credentials" / "if the email exists…" responses that don't distinguish "no such account" from "wrong password." (The old scenario already does this for reset; the port keeps and extends it.) | to-port |
| Signing-key loss / churn | Restart regenerates the key and invalidates every live token. | Load-or-generate persisted keypair; the key is stable across restarts. Private key file-perm `0600`, never logged. | to-port |
| Stale-token / post-logout access | A revoked session's token is still honored. | Server-tracked sessions with per-session and per-user revocation; revocation state in Redis; access tokens are short-lived so the revocation window is bounded. | to-port |
| Privilege escalation via forged roles | A user mints/edits their own role claims. | Roles are claims **signed** by the authenticator and verified by RPs against JWKS; a tampered token fails signature verification. Role assignment is an RBAC-gated admin operation. | to-build |
| OAuth CSRF (login-flow forgery) (P1) | Forged callback links a victim's session to an attacker account. | CSRF `state` is generated server-side, one-time-use, validated and deleted on callback (ported from the old OAuth flow). | to-port |
| Leakage via logs / errors | Secrets or tokens end up in logs or error bodies. | Never log plaintext passwords, raw tokens, or the private key. Error messages relay validation reasons faithfully without echoing secrets (see [`ERROR-HANDLING.md`](ERROR-HANDLING.md)). | to-build |
| MFA bypass / replay (P1) | TOTP code reuse or weak recovery-code handling. | TOTP challenge with replay-window handling; recovery codes single-use and hashed; per-realm enforcement policy. | to-port |

## Signing-key persistence and rotation

- **Persistence (P0):** load-or-generate `private.pem`/`public.pem` in the
  storage root; stable across restarts. This is ported verbatim.
- **`kid` discipline:** JWKS publishes a stable `kid` derived from the
  public key (the old scenario uses a truncated SHA-256 fingerprint of the
  DER-encoded public key). RPs match a token's `kid` against the JWKS set.
- **Rotation (P2, OT-P2-005):** rotation publishes **overlapping `kid`s**
  during rollover — the old key stays in the JWKS set until its
  longest-lived token expires, so in-flight tokens keep verifying while new
  tokens use the new key. Per-realm key isolation is the P2 extension. The
  `kid`-in-JWKS shape ported in P0 is what makes non-breaking rotation
  expressible later.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| RSA signing keypair (`private.pem`/`public.pem`) | persisted to the `api-core/storage` root, load-or-generate | yes | The one secret the service always holds. Private key never published; file perms `0600`. |
| OAuth provider client id/secret (P1) | api-core secrets / env per realm | no | Only for social federation (Google/GitHub/Microsoft). Per-realm; never logged. |
| SMTP / mail credentials (P1) | api-core secrets | no | Only for email-verification / password-reset delivery. |

## Security Gaps

Pre-implementation gaps — defenses that must be built/ported before the
relevant surface ships:

| Gap | Severity | Revisit Trigger |
|---|---|---|
| RS256 lock + `none`/HS rejection not yet ported | critical | Before any token is issued or verified. |
| `aud` issuance + cross-realm rejection not yet built | critical | Before more than one realm exists; ship in P0 regardless. |
| Refresh reuse-detection / family-revoke not yet built | high | Before refresh tokens are issued (P0). |
| Argon2id hashing not yet implemented (old scenario is bcrypt) | high | Before the first account is registered. |
| Rate limiting + account lockout not yet ported | high | Before auth endpoints accept external traffic. |
| Audit log not yet built | medium | Before P0 close — security events must be recorded from day one. |
| Enumeration-safe messaging not yet verified across login/register | medium | Before the auth endpoints ship. |

## Cross-References

- [`../../PRD.md`](../../PRD.md) — Appendix A (IdP↔RP), B (realms), C (carried-over invariants)
- [`SEAMS.md`](SEAMS.md) — signing-key provider, realm resolver, and other test-substitutable boundaries
- [`TESTING.md`](TESTING.md) — the crypto-invariant and cross-realm-rejection regression tests
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — enumeration-safe error mapping
- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and the RP contract
- [`PROBLEMS.md`](PROBLEMS.md) — why this rewrite exists (the shared-DB blast radius) and the security-boundary risk
