---
title: "Security Posture"
description: "Authentication, authorization, secrets, and abuse-resistance posture"
category: "internal"
order: 104
audience: ["developers", "operators"]
internal: true
---

# Security Posture

The user-facing security surface is also covered by `docs/reference/SECURITY.md`. This document is the **internal** view: where the trust boundaries live, what we deliberately accept as a risk, and what would constitute a regression.

## Trust boundaries

```
                  ┌── Cloudflare/nginx ─── TLS termination
                  │
   public ───────►│
                  ├── LandingConfigService.GetLandingConfig, /api/v1/plans, /api/v1/branding,
                  │   /api/v1/metrics/track, /api/v1/waitlist,
                  │   FeedbackService.CreateFeedback
                  │   (NO auth — rate-limited only at infra)
                  │
                  ├── /api/v1/auth/*           (public; durable per-email, per-IP and per-code-guess throttles)
                  │
                  ├── /api/v1/me/*, /api/v1/ai/*, /api/v1/downloads,
                  │   /api/v1/billing/portal-url, /api/v1/usage/{summary,check}
                  │   (requireUserAuth — JWT)
                  │
                  ├── /api/v1/admin/*          (requireAdmin — cookie session)
                  │
                  ├── /api/v1/admin/remote-profiles{,/{id}/test,/{id}/proxy},
                  │   /api/v1/admin/download-artifacts/{presign-upload,commit},
                  │   /api/v1/admin/download-assets/apply,
                  │   /api/v1/deploy-readiness
                  │   (requireAdminOrService — admin cookie OR service bearer)
                  │
                  ├── /api/v1/usage/report     (service bearer only)
                  │
                  └── /api/v1/webhooks/stripe  (Stripe signature verification)
```

## Authentication mechanisms

| Mechanism | Used for | Storage | Rotation |
|-----------|----------|---------|----------|
| **bcrypt password + optional TOTP + cookie session** | Admin (operator) | `admin_sessions` row + `Set-Cookie` HttpOnly; TOTP secret sealed with the `admin-mfa-encryption-key` ring; recovery codes bcrypt-hashed | Manual via admin profile page; `admin-mfa-reset` CLI (service credential) for a lost authenticator |
| **Emailed 6-digit code or one-use link → JWT (access + refresh)** | End users | `auth_tokens` (hashed token, hashed code, browser-binding hash, stored app context, delivery status), `user_sessions`; refresh token rotates on use | Refresh-on-use; revocation flips `user_sessions.revoked` |
| **Service bearer token** | s2s (CLI, sister scenarios) | Authority-backed shared secret resolved in process | Manual; rotate through the credential authority |
| **Stripe webhook signature** | Stripe → us | n/a (header-based) | Per Stripe key rotation |

Customer sign-in rules:

- A sign-in request creates no `users` row; the account is created only when the link or code is proven.
- The verify page previews a link (`POST /auth/magic-link/preview`) and consumes it only after the person confirms (`POST /auth/verify`), so mail scanners that open or render links cannot use them. There is no consuming `GET`.
- A code is accepted only from the browser that requested it (`browser_binding`), and app callback context is returned only to that browser.
- Completing sign-in retires every other outstanding link and code for the address.
- A request whose email could not be delivered is retired and reported as `503 delivery_unavailable`; every well-formed address may sign up, so there is no account-existence oracle to protect. `GET /api/v1/admin/auth/delivery` reports 24-hour outcomes. Production sign-in requests enter the durable outbox and the registry router selects the enabled provider; the current deployment enables Mailgun only.
- `sign_in_email` in `/health` degrades on two independent conditions: the most recent send attempt in the last 24 hours failed, **or** a production deployment has no configured provider at all. The second condition exists because the first is attempt-driven: a deployment with a dead provider read healthy whenever nobody happened to be signing in, so the signal disappeared exactly when no customer was there to trip it.
- Stored provider credentials are re-verified hourly and the verdicts are published as provider-specific cached results. Presence is not usability: a key can be present and revoked, a password present and rotated, or a credential can lack the required send scope. The probes authenticate and hang up, so no mail is sent by a readiness check. Only an outright rejection degrades health and alerts; "not configured" is a declaration and an unreachable provider is not evidence of a bad credential. A persistent failure alerts once per six-hour window through the operator webhook, recorded in `auth_alert_log`; `GET /api/v1/admin/provider-credentials` serves the cached verdicts.
- Successful durable sends persist the selected provider and provider message ID in the outbox/attempt records and, for sign-in, update `auth_tokens`. Provider-specific metadata such as SendGrid tracking suppression applies only when that provider is explicitly enabled and selected; it is not a claim about the current Mailgun route.
- The hourly auth janitor purges sign-in requests older than one day, ended customer sessions after 30 days of inactivity, refresh-token history after 100 days, and administrator sessions one day after expiry. Active and recent rows remain.

The password/cookie and code/link/JWT rows describe the current LPBS
compatibility implementation. The target platform boundary moves person
identity, MFA, sessions, and coarse capabilities to
`scenario-authenticator`; LPBS retains product and commercial authority.
During that migration, LPBS identity-management requests must use a
server-side, actor-preserving API-to-API call through `api-core/discovery`.
They must not update authenticator storage directly or treat a product-admin
cookie as a universal identity-admin credential.

## Secrets handling

- Secrets are resolved in process by the Vrooli credential authority. The API does not read `~/.vrooli/secrets.json` or any tracked file, and generated credentials are witness-gated so a lost value cannot be silently re-minted over persisted data.
- Stripe restricted keys are preferred (`docs/reference/STRIPE_RESTRICTED_KEYS.md`).
- Remote-profile sessions are encrypted-at-rest in `remote_profiles.encrypted_session`. The versioned key ring lives in the credential authority, not in the DB or environment.
- Stripe and delivery credentials are write-through authority values; `payment_settings` and `download_storage_settings` retain only non-secret configuration.
- Session-secret rotation retains the previous codec key for overlap, so active users are not signed out by a rotation.
- The admin reset procedure (`AdminResetService/ResetDemoData`) does **not** wipe `admin_users` — credentials persist across resets.
- The declared `admin-default-password` authority value is the bootstrap administrator credential. `POST /admin/admin-credentials/reset` (service principal only) updates both the stored hash and the authority value, so restart reconciliation cannot revert an operator reset. The deployment may also rotate the value through the `scenario-to-cloud` credential lifecycle.
- The remote-profile proxy allowlist is exact and server-enforced. Its only cross-session mutating entries are download promotion, API-key/Stripe settings, and the service-principal recovery endpoints (`/admin/mfa/reset`, `/admin/admin-credentials/reset`); it is not a general admin tunnel.

## Authorization model

- **Admin:** flat. Anyone with the admin cookie can do anything under `/api/v1/admin/*`. No per-row scoping yet.
- **End user:** scoped to `users.id`. `/api/v1/me/*` always operates on the JWT subject; cross-user reads return `404` (not `403`) to avoid existence oracles.
- **Service bearer:** narrow allowlist of routes (see `requireAdminOrService` and `requireServiceAuth` call sites). Not a general-purpose admin substitute.

After the identity migration, the service-bearer path must be replaced or
constrained to the generated authenticator management client for identity
operations. Product-admin authorization remains in LPBS; identity-admin
authorization remains in the authenticator.

## Abuse resistance

- Sign-in throttles live in `auth_rate_events` (Postgres), so they hold across restarts and replicas and degrade to a process-local window if the database is unavailable: 5 requests / 15 min per email, 20 requests / hour per client IP, 5 wrong codes / 15 min per email.
- Admin sign-in: 5 failures / 15 min per email and 20 / 15 min per client IP lock further attempts; unknown emails spend the same bcrypt work as real ones; a TOTP time step is never accepted twice.
- Administrator second factors are required when `ADMIN_REQUIRE_MFA` says so, and otherwise derived from the environment. The derivation reads `LPBS_ENVIRONMENT` **and** `VROOLI_ENVIRONMENT`: cloud execution exports only the latter, so reading just the scenario-specific variable meant a deployed production process derived "not required" and accepted administrators with no second factor. The `admin_mfa_policy` health check asserts the effective policy through the same function the login path calls, rather than re-reading the variable, so enforcement and its watchdog cannot disagree.
- Client IP: `X-Forwarded-For` is walked right-to-left through trusted proxies (loopback by default, because browsers reach the API through the scenario's own UI server; `TRUSTED_PROXY_CIDRS` overrides). Client-supplied entries left of the first untrusted hop are ignored.
- Cross-site request forgery: state-changing requests that carry `admin_session`, `access_token`, or `refresh_token` cookies must be same-origin (`Sec-Fetch-Site`, falling back to `Origin`/`Referer`); see `api/csrf_guard.go`.
- Stripe webhook signature is verified before any body parsing.
- Idempotent webhook + idempotent credit reservation prevent replay-based credit inflation.
- Anomaly dispatcher emits an alert + audit row when intro-coupon usage, refund cadence, or other heuristics breach threshold (`payment_anomaly_log`).

## Known gaps (acknowledged, not "broken")

- Cross-site protection relies on fetch metadata and `SameSite=Lax` rather than a synchronizer token. If an admin portal is ever served from a different origin, allow that origin explicitly in `csrf_guard.go`.
- Admin two-factor authentication supports TOTP, recovery codes, and WebAuthn
  passkeys. Production enforcement is operator-controlled through
  `ADMIN_REQUIRE_MFA=true`; a production rollout must enroll the first admin
  factor before exposing the portal.
- Native-app authorization uses durable, hashed, one-use grants in `native_auth_grants`; the grant is burned before redirect or PKCE validation and is linked to the session created by a successful exchange.
- Social sign-in (Google, Apple) remains a platform roadmap item owned by
  `scenario-authenticator`; LPBS documents the relying-party hand-off and does
  not store provider credentials. Customer and administrator WebAuthn
  ceremonies, browser UX, durable credentials, and passkey step-up are now
  delivered; production rollout still depends on operator MFA and RP-origin
  configuration.
- Service bearer is HMAC of a static secret, not a JWT — fine for a small s2s mesh, would not scale to many callers.
- The UI uses `BrowserRouter` and does not use React Router's unstable RSC APIs. GHSA-qwww-vcr4-c8h2 is therefore tracked as a dependency warning rather than a shipped attack path; introducing an RSC router, RSC package, or unstable RSC API requires upgrading React Router to a patched release first.
- Security Health currently reports residual lockfile advisories from transitive build and test tooling, plus the `x/crypto/openpgp` advisory. OpenTelemetry was upgraded through Scenario Dependency Analyzer to `v1.42.0`, clearing GO-2026-5158. The UI directly pins `picomatch` 4.0.5 through Scenario Dependency Analyzer, which removed the vulnerable 4.x resolver path without weakening coverage policy. The remaining old `minimatch`, `brace-expansion`, `flatted`, and `picomatch` 2.x paths are held by ESLint, Tailwind, and test-tooling dependency graphs; `monaco-editor@0.56.0` similarly owns the residual `dompurify@3.4.8` path. Governed requests for the current Monaco packages resolve to those already-installed versions, so these paths cannot be safely overridden by hand. `golang.org/x/crypto@0.54.0` is also already current; GO-2026-5932 concerns its intentionally unmaintained `openpgp` package and has no named patched upstream release. Treat every new production dependency path as a trigger to re-run Security Health, and re-evaluate these residuals when their upstream owners publish a compatible release.

- `config/variants/agency-marketing-visionary.json` is a public content fixture. Its stable section identifiers trigger gitleaks' generic-key heuristic even though the file contains no credential fields. The scenario-local `.gitleaks.toml` suppresses only that exact fixture path while retaining every default rule for all other source and configuration files. Any future secret-bearing configuration must use the vault/env path and must never be added to this fixture allowlist.
# Native grant posture

Native-app authorization no longer stores token pairs or raw authorization codes
in process memory. Postgres owns the grant, its PKCE challenge, exact loopback
redirect, expiry, and eventual session linkage.

# Sign-in delivery posture

SendGrid event ingestion is ECDSA-verified against raw request bytes, freshness
bounded, idempotent by provider event ID, and normalized before customer-facing
status is returned. The delivery readiness report observes DNS and webhook
health without changing operator state; alert dispatch is bounded by shared
retry, timeout, backoff, and rate-limit transport.
