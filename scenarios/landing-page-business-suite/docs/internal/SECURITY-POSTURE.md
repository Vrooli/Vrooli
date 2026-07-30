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
                  ├── /api/v1/auth/*           (public; internal rate limiter, 5 / 15 min)
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
| **bcrypt password + cookie session** | Admin (operator) | `admin_sessions` row + `Set-Cookie` HttpOnly | Manual via admin profile page |
| **Magic-link → JWT (access + refresh)** | End users | `auth_tokens`, `user_sessions`; refresh token rotates on use | Refresh-on-use; revocation flips `user_sessions.revoked` |
| **Service bearer token** | s2s (CLI, sister scenarios) | Env var on the caller; verified against an HMAC of a shared secret | Manual; rotate via lifecycle env update |
| **Stripe webhook signature** | Stripe → us | n/a (header-based) | Per Stripe key rotation |

## Secrets handling

- Secrets are read from env vars first, then from `~/.vrooli/secrets.json` (lifecycle "Secrets Tab"). They are **never** read from a tracked file.
- Stripe restricted keys are preferred (`docs/reference/STRIPE_RESTRICTED_KEYS.md`).
- Remote-profile sessions are encrypted-at-rest in `remote_profiles.encrypted_session`. The key lives in env, not in the DB.
- The admin reset procedure (`AdminResetService/ResetDemoData`) does **not** wipe `admin_users` — credentials persist across resets.

## Authorization model

- **Admin:** flat. Anyone with the admin cookie can do anything under `/api/v1/admin/*`. No per-row scoping yet.
- **End user:** scoped to `users.id`. `/api/v1/me/*` always operates on the JWT subject; cross-user reads return `404` (not `403`) to avoid existence oracles.
- **Service bearer:** narrow allowlist of routes (see `requireAdminOrService` and `requireServiceAuth` call sites). Not a general-purpose admin substitute.

## Abuse resistance

- Magic-link request rate limiter: 5 / 15 min per normalized email (`magicLinkLimiter`).
- Stripe webhook signature is verified before any body parsing.
- Idempotent webhook + idempotent credit reservation prevent replay-based credit inflation.
- Anomaly dispatcher emits an alert + audit row when intro-coupon usage, refund cadence, or other heuristics breach threshold (`payment_anomaly_log`).

## Known gaps (acknowledged, not "broken")

- No CSRF token on cookie-authenticated endpoints — admin portal is same-origin and uses a `SameSite=Lax` cookie. If an admin-portal subdomain is ever served separately, this needs to change.
- No 2FA on admin login.
- Service bearer is HMAC of a static secret, not a JWT — fine for a small s2s mesh, would not scale to many callers.
- The UI uses `BrowserRouter` and does not use React Router's unstable RSC APIs. GHSA-qwww-vcr4-c8h2 is therefore tracked as a dependency warning rather than a shipped attack path; introducing an RSC router, RSC package, or unstable RSC API requires upgrading React Router to a patched release first.
- Security Health currently reports residual lockfile advisories from transitive build and test tooling, plus the `x/crypto/openpgp` advisory. OpenTelemetry was upgraded through Scenario Dependency Analyzer to `v1.42.0`, clearing GO-2026-5158. The UI directly pins `picomatch` 4.0.5 through Scenario Dependency Analyzer, which removed the vulnerable 4.x resolver path without weakening coverage policy. The remaining old `minimatch`, `brace-expansion`, `flatted`, and `picomatch` 2.x paths are held by ESLint, Tailwind, and test-tooling dependency graphs; `monaco-editor@0.56.0` similarly owns the residual `dompurify@3.4.8` path. Governed requests for the current Monaco packages resolve to those already-installed versions, so these paths cannot be safely overridden by hand. `golang.org/x/crypto@0.54.0` is also already current; GO-2026-5932 concerns its intentionally unmaintained `openpgp` package and has no named patched upstream release. Treat every new production dependency path as a trigger to re-run Security Health, and re-evaluate these residuals when their upstream owners publish a compatible release.

- `config/variants/agency-marketing-visionary.json` is a public content fixture. Its stable section identifiers trigger gitleaks' generic-key heuristic even though the file contains no credential fields. The scenario-local `.gitleaks.toml` suppresses only that exact fixture path while retaining every default rule for all other source and configuration files. Any future secret-bearing configuration must use the vault/env path and must never be added to this fixture allowlist.
