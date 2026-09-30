---
title: "Security Guide"
description: "Authentication, session management, CORS, and production security checklist"
category: "operational"
order: 8
audience: ["developers", "operators"]
---

# Security Guide

This document covers security architecture, configuration, and best practices for deploying landing pages generated from this template.

## Table of Contents

1. [Authentication Architecture](#authentication-architecture)
2. [Session Management](#session-management)
3. [Authorization Model](#authorization-model)
4. [CORS Configuration](#cors-configuration)
5. [Stripe Webhook Security](#stripe-webhook-security)
6. [Database Security](#database-security)
7. [Environment Variables](#environment-variables)
8. [Production Security Checklist](#production-security-checklist)
9. [Common Vulnerabilities](#common-vulnerabilities)

Sign-in mail is sent with SendGrid tracking disabled (`lpbs-auth` category and
request correlation argument) and records the provider message ID. SMTP
fallback uses a 10-second dial limit and a 20-second protocol deadline; it
requires TLS before authenticating and supports implicit TLS on port 465.
The hourly authentication janitor retains recent investigation data while
purging expired sign-in requests, ended customer sessions, refresh history
older than 100 days, and administrator sessions expired for more than one day.

---

## SendGrid delivery event security

`POST /api/v1/webhooks/sendgrid` is a provider-owned REST exception. LPBS caps
the raw body at 1 MiB, verifies the recent ECDSA signature over the exact
`timestamp || body` bytes before JSON parsing, rejects invalid requests with
401, and stores event IDs idempotently. Browser delivery status is scoped to the
email plus browser-binding hash and exposes only normalized statuses and reason
classes; raw provider bounce text remains server-side.

The webhook public key is an optional operator credential (`sendgrid-webhook-public-key`)
resolved through the credential authority. DNS readiness is read-only and
degrades health rather than preventing the API from serving.

## Authentication Architecture

### Project identity and LPBS ownership

This guide describes the current LPBS website compatibility boundary. LPBS
currently authenticates its website users with its local emailed-code or
one-use-link/JWT compatibility flow, while the target platform contract assigns person
identity, MFA, machine bindings, and local authorization to
`scenario-authenticator`. LPBS remains the authority for business accounts,
subscriptions, commercial entitlements, usage, and signed entitlement leases.

Do not treat these as interchangeable:

| Credential or decision | Authority | Meaning |
|---|---|---|
| `scenario-authenticator` access token | Identity provider | Who the person or machine is and what local capabilities are allowed |
| LPBS entitlement lease | LPBS | Which commercial features and limits the linked business account has |
| Desktop supervisor token | Desktop runtime | Whether a local process may call the loopback supervisor; not a human identity |
| LPBS website session | LPBS compatibility surface | Access to the deployed LPBS website; not automatic local-bundle access |

For bundled apps, local use is `personal_local` and does not require LPBS
sign-in. Multi-user, remote, or paid-account linking is explicit setup.
Linking uses a short-lived, scoped browser/device flow. Matching email
addresses, copied website tokens, and request-body identity fields are never
enough to establish an account link. See the project-level [Identity and
Authentication contract](../../../../docs/concepts/IDENTITY-AND-AUTHENTICATION.md).

### Customer sign-in endpoints

All are public and throttled durably (see `docs/internal/SECURITY-POSTURE.md`).

| Method | Path | Body | Result |
|--------|------|------|--------|
| POST | `/api/v1/auth/magic-link` | `email`, `browser_binding`, optional app `context` | Emails a 6-digit code and a one-use link; `200 {expires_at}`, or `503` with `reason: delivery_unavailable` |
| POST | `/api/v1/auth/magic-link/preview` | `token`, `browser_binding` | Masked email, expiry, flow, and (same browser only) app context. Does not consume the link |
| POST | `/api/v1/auth/verify` | `token`, `browser_binding` | Consumes the link; sets HttpOnly cookies |
| POST | `/api/v1/auth/verify-code` | `email`, `code`, `browser_binding` | Consumes the newest matching request from the same browser; sets HttpOnly cookies |
| GET/POST | `/api/v1/auth/authorize` | PKCE parameters plus `token` or `email`+`code` | Native apps: one-use authorization code to a loopback redirect. `GET` with no credential redirects to `/auth/login` |

`browser_binding` is a random value the sign-in pages keep in the browser's
local storage. It never authenticates on its own; it limits code entry and app
context to the browser that started sign-in. Failures carry a stable `reason`
(`token_expired`, `token_used`, `token_invalid`, `code_invalid`,
`rate_limited`, `delivery_unavailable`).

### Google sign-in hand-off

Google sign-in is planned in `scenario-authenticator`, not implemented as an
LPBS-local provider. When enabled, `scenario-authenticator` is the relying
party for the OAuth/OIDC callback, CSRF state, verified provider identity, and
external-account link. LPBS receives only the verified principal through the
typed relying-party boundary and remains authoritative for business-account
membership, subscriptions, entitlements, and product sessions.

Linking is explicit and scoped: a matching email is only a candidate for user
review, never proof of ownership; the user must approve the link while
authenticated; provider subject plus issuer, realm, and audience are stored as
the external identity; and unlinking or revocation must not delete the LPBS
business account. LPBS must not store Google refresh tokens, provider
passwords, or raw OAuth callbacks.

### Provider credential verification

Stored provider credentials are re-verified on an hourly loop, because a
credential breaks between deploys: a key is revoked, a password rotated, a
Stripe key swapped for the other mode. Each probe authenticates and stops
there.

| Probe | What it asks | Health dependency |
|---|---|---|
| SendGrid | `GET /v3/scopes`; the key must authenticate **and** carry `mail.send` | `sign_in_email_credentials` |
| Site SMTP relay | connect, STARTTLS, `AUTH`, `QUIT` — no message is sent | `sign_in_smtp_credentials` |
| Stripe | key mode versus the declared `STRIPE_MODE`, then read-only `GET /v1/account` | `payments_credentials` |

Only an outright rejection degrades health and raises an alert. A credential
that is not configured is a declaration rather than a defect, and an
unreachable provider is not evidence of a bad credential, so both are silent:
every warning this produces is meant to be actionable. A persistent failure
alerts once per six-hour window through the operator webhook and is recorded
in `auth_alert_log`. `GET /api/v1/admin/provider-credentials` serves the
cached verdicts; probe details never contain a credential value.

Health reads the cache and never probes inline, so a health request cannot be
delayed by a third party and a busy poller cannot turn into a probe flood.

### Passkeys and administrator second factors

LPBS is the WebAuthn relying party for its website RP ID (`vrooli.com` in
production). Customer passkeys support passwordless browser sign-in and
recent-authentication step-up. Administrator passkeys are an additional
second factor alongside TOTP and recovery codes; enrollment, login, and
reauthentication use one-use, browser-bound ceremonies and durable credential
records.

The requirement itself comes from `ADMIN_REQUIRE_MFA` when set, and is
otherwise derived from the environment — read from `LPBS_ENVIRONMENT` or
`VROOLI_ENVIRONMENT`, so a cloud-deployed process that only receives the
control-plane variable still derives "required". The `admin_mfa_policy` health
check asserts that effective policy rather than re-reading the variable, so a
production deployment that is not enforcing second factors says so instead of
passing silently. A production rollout still needs a first administrator
second-factor enrollment before the portal is exposed.

### Desktop account-link endpoints

The desktop link protocol is separate from the compatibility
`/api/v1/auth/authorize` flow, which returns LPBS website tokens for existing
clients. The desktop protocol is:

1. The desktop client opens the LPBS browser login with the requested
   installation, resource, audience, scopes, S256 challenge, and loopback
   redirect. The login page displays the requested capability set before the
   user continues.
2. After the emailed code or link establishes the browser's HttpOnly same-origin session,
   the browser lists the user's LPBS business accounts. If more than one is
   available, the user must select one; LPBS verifies membership server-side.
   The browser then `POST`s `/api/v1/desktop/links` with the selected account,
   installation, resource audience, requested scopes, S256 PKCE challenge, and
   loopback redirect. LPBS returns only a short-lived, one-use code and stores
   its hash.
3. The local desktop side `POST`s `/api/v1/desktop/links/redeem` with the code,
   verifier, installation, and resource. Its bearer credential must verify to a
   human `scenario-authenticator` principal; LPBS derives the local subject
   from that verified token rather than trusting the request body.
4. LPBS persists the scoped relationship and returns a signed entitlement
   lease whose subject, business-account projection, installation, audience,
   and scopes are explicit. The desktop stores that lease through the native
   credential authority and verifies it locally until `not_after`. Lease
   consumers must bind verification to the expected business account,
   installation, resource, audience, link ID, and exact granted scope set; a
   valid signature alone is not permission to replay the lease in another
   desktop context.

Both `/api/v1/desktop/links` (LPBS actor) and
`/api/v1/desktop/links/local` (local actor) support revocation. A failed
provider proof, wrong installation/resource, wrong PKCE verifier, expired
code, reused code, or revoked link fails closed. Neither endpoint accepts an
LPBS website access token as a local identity proof.

Revocation prevents new lease issuance and makes the durable link unavailable
through the LPBS status/redeem surfaces. An already-issued offline lease is
bounded by its signed `not_after` time; desktop clients must discard it when
the local link is revoked or when an online status refresh reports revocation.

The scenario-to-desktop template exposes this flow as `auth.connectDesktop`.
The caller must provide a proof obtained from the declared local identity
provider through the template's `onResolveLocalIdentityProof` seam. The
supervisor bearer, LPBS access token, and request-body principal are rejected
as substitutes for that proof.

### Identity administration after migration

LPBS must keep product administration and identity administration separate.
The LPBS administrator may also be an authenticator administrator, but that
relationship is established during explicit bootstrap or account linking and
is recorded as a scoped mapping.

For a self-hosted LPBS mini-Vrooli, the server-side LPBS API resolves
`scenario-authenticator` with `api-core/discovery` and uses its generated
Connect client for identity operations. The request carries the verified
LPBS actor, target principal, requested operation, and product context. The
authenticator performs its own capability check and audit write. LPBS records
the product-side audit event and never writes authenticator tables.

The management operations are user search, lock/unlock, credential-reset
initiation, MFA reset/recovery, session revocation, capability assignment,
and identity-audit lookup. They are not all present in the current API; the
LPBS UI must not present an unimplemented action as available or fall back to
the legacy `admin_users` table for a local authenticator operation.

For public hosted LPBS, use a dedicated/shared hosted identity boundary or an
explicit external provider. Do not create one local authenticator instance
per product request. Before replicas are enabled, the hosted deployment must
have durable shared identity storage, shared revocation/rate-limit state,
signing-key rotation, backup/restore evidence, and audit retention.

### Admin Authentication Flow

The admin portal uses session-based authentication with bcrypt password hashing:

```
┌──────────────────────────────────────────────────────────────────┐
│                    Authentication Flow                           │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. POST /landing_page_business_suite.v1.AdminAuthService/Login │
│     ┌─────────┐                                                  │
│     │ Client  │ ──── email + password ────►  ┌─────────┐        │
│     └─────────┘                               │   API   │        │
│                                               └────┬────┘        │
│                                                    │             │
│  2. Verify credentials                             ▼             │
│     ┌─────────────────────────────────────────────────┐         │
│     │ SELECT password_hash FROM admin_users           │         │
│     │ WHERE email = $1                                │         │
│     │                                                 │         │
│     │ bcrypt.CompareHashAndPassword(hash, password)   │         │
│     │ (unknown email: same bcrypt work; failures are  │         │
│     │  throttled per email and per client IP)         │         │
│     └─────────────────────────────────────────────────┘         │
│                                                    │             │
│  2b. If two-factor is on and totp_code is empty:   ▼             │
│      failed_precondition → client asks for the code; │           │
│      verify TOTP (no step reuse) or a recovery code  │           │
│                                                    │             │
│  3. Create session                                 ▼             │
│     ┌─────────────────────────────────────────────────┐         │
│     │ Set-Cookie: admin_session=<encrypted-value>     │         │
│     │ HttpOnly=true; SameSite=Lax; Path=/; MaxAge=7d  │         │
│     └─────────────────────────────────────────────────┘         │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Password Hashing

Passwords are hashed using bcrypt with a cost factor of 10:

```go
// Hash new passwords
hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

// Verify passwords
err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(inputPassword))
```

**Security properties:**
- Salted hashes (salt is embedded in the hash)
- Adaptive cost factor (increases with hardware improvements)
- Timing-safe comparison

### Initial Admin Account

Development may seed an admin account with an ephemeral password when no
credential is configured. The generated password is intentionally not stable
across restarts. Production does not have a default account credential: startup
requires `ADMIN_DEFAULT_PASSWORD` from the deployment secret store. The value
must contain at least 12 characters; this rejects the former short demo
credential before the API opens a database connection.

#### Option 1: Environment Variable Override (Recommended for Deployments)

Override the default credentials using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `ADMIN_DEFAULT_EMAIL` | Admin account email | `admin@localhost` |
| `ADMIN_DEFAULT_PASSWORD` | Admin account password (plaintext, hashed on startup) | Required in production; 12+ characters; ephemeral development value otherwise |

**For scenario-to-cloud deployments:**
1. Navigate to the Secrets Tab in scenario-to-cloud
2. Add `ADMIN_DEFAULT_EMAIL` with your desired admin email
3. Add `ADMIN_DEFAULT_PASSWORD` with a strong password (12+ chars, letters and numbers)
4. Check "Restart scenario after changes"
5. The scenario will restart with the new credentials

On startup, the API reads these environment variables, hashes the password, and seeds/updates the admin user. This approach is ideal for:
- Cloud deployments where you want consistent credentials across restarts
- Automated provisioning where credentials come from a secrets manager
- Scenarios where manual login to change credentials isn't feasible initially

#### Option 2: Admin Portal (Post-Deployment)

Use the admin portal ( `/admin/profile` ) to rotate credentials:
- Enter your current password, provide a new email, and save.
- Enter your current password, choose a strong new password (12+ chars, letters and numbers), and save.

The change is applied to the live session immediately; the default hash is removed from the database.

#### Option 3: Operator Recovery (Locked Out)

When the current password is unknown, `POST /api/v1/admin/admin-credentials/reset`
is the service-principal-only recovery path. It never accepts a browser admin
session, updates the stored bcrypt hash, mirrors the new password into the
`admin-default-password` authority value so a restart on the bootstrap account
cannot revert it, revokes the account's other sessions, and records an
`admin_credential_reset` security event. It is reachable:

- locally: `landing-page-business-suite admin-credential-reset --new-password-stdin`
  (the CLI presents the LPBS service credential from the credential authority);
- through a stored remote profile: the same command with `--profile-tag <tag>`;
- through the deployment credential lifecycle: rotate the declared
  `admin-default-password` binding in `scenario-to-cloud`, which redistributes
  the value and restarts the consumer.

Both recovery endpoints (`/admin/mfa/reset` and
`/admin/admin-credentials/reset`) are on the exact remote-profile proxy
allowlist and require the service principal on the target.

---

## Session Management

### Cookie Configuration

Sessions are managed via the `gorilla/sessions` package with encrypted cookies:

| Property | Development | Production |
|----------|-------------|------------|
| Cookie Name | `admin_session` | `admin_session` |
| HttpOnly | `true` | `true` |
| Secure | `false` | **`true`** |
| SameSite | `Lax` | `Lax` or `Strict` |
| MaxAge | 7 days | 7 days (adjust as needed) |
| Path | `/` | `/` |

### Session Secret

The session encryption key is resolved from the credential authority at
`vrooli/landing-page-business-suite:session-secret`:

```bash
# LPBS mints this generated credential on first start and records a witness.
# Never mint a replacement by hand after a witness exists.
```

**⚠️ CRITICAL:** The generated session credential must exist in the credential
authority in production. The API refuses to
start without it. Development generates a cryptographically random ephemeral
key and logs a warning; sessions therefore end after an API restart. No shared
or committed fallback signing key exists.

Production startup also requires a 12+-character `ADMIN_DEFAULT_PASSWORD` from
the deployment secret store. Development uses an ephemeral password hash when
it is absent, so a known built-in admin credential can never authenticate a
request.

Production startup also requires the canonical `PUBLIC_BASE_URL`. It must be an
absolute HTTPS URL. `AUTH_MAGIC_LINK_BASE_URL` remains a legacy callback-URL
fallback for migrations and is ignored when `PUBLIC_BASE_URL` is present. This
prevents a deployment from sending customers a localhost or HTTP verification
link.

### Dependency advisory posture

The API's dependency graph is reviewed with Security Health and a reachability
scan (`govulncheck ./...`) after governed upgrades. As of 2026-07-27,
`govulncheck` reports zero invoked vulnerabilities.

num[decision]:two module-level notices remain intentionally documented rather than hidden:

- `golang.org/x/crypto/openpgp` is marked unmaintained by upstream even at the
  current `x/crypto` release. This API does not import or invoke `openpgp`; the
  notice is non-reachable and has no patched version to upgrade to.
- `github.com/docker/docker` is brought in only by the test-only
  `testcontainers-go` dependency. The newest public Go module release is
  `v28.5.2+incompatible`; several advisory records request an unavailable
  `v29.3.1` line. The API uses the latest published compatible version, and the
  testcontainers dependency must be reevaluated when Docker publishes a Go
  module release that resolves those advisories.

Do not add `replace` directives, edit `go.sum`, or suppress these records by
hand. All dependency changes must go through Scenario Dependency Analyzer.

### React Router RSC advisory posture

Security Health currently reports `GHSA-qwww-vcr4-c8h2` for
`react-router-dom 7.18.1`. The advisory describes an RSC-mode CSRF bypass and
names `8.3.0` as the first patched version, but the npm registry's published
`react-router-dom` latest is still `7.18.1`; there is no governed published
upgrade to install.

The LPBS UI is a Vite client application using `BrowserRouter` and does not
ship React Server Components, a server router, RSC action endpoints, or an RSC
build plugin. That means the advisory's described RSC action path is absent
from this deployment. This is a documented applicability assessment, **not** a
scanner suppression: continue to check the registry and Security Health on
each dependency review, and upgrade through Scenario Dependency Analyzer as
soon as a compatible published remediation exists.

### Session Lifecycle

```
┌─────────────────────────────────────────────────────────────┐
│                    Session Lifecycle                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Login                                                      │
│  ├─► Verify credentials                                     │
│  ├─► Create encrypted session cookie                        │
│  └─► Update last_login timestamp                            │
│                                                             │
│  Authenticated Request                                      │
│  ├─► Extract session cookie                                 │
│  ├─► Decrypt and validate                                   │
│  └─► Proceed if email exists in session                     │
│                                                             │
│  Logout                                                     │
│  ├─► Set MaxAge = -1 (expires cookie)                       │
│  └─► Log logout event                                       │
│                                                             │
│  Session Expiry                                             │
│  └─► Cookie expires after MaxAge (default: 7 days)          │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Session Validation

The `requireAdmin` middleware protects admin endpoints:

```go
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        session, _ := sessionStore.Get(r, "admin_session")
        email, ok := session.Values["email"].(string)
        if !ok || email == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next(w, r)
    }
}
```

---

## Authorization Model

### Endpoint Protection

Endpoints are categorized by access level:

| Access Level | Endpoints | Protection |
|--------------|-----------|------------|
| **Public** | `LandingConfigService.GetLandingConfig`, `/api/v1/plans`, `/api/v1/branding` | None |
| **Public** | `VariantService.SelectVariant`, `VariantService.GetPublicVariant` | Finite experiment identity only; no marketing copy or snapshots |
| **Admin** | `ProductPresentationAdminService` (all methods) | Administrator authentication; generation guards for mutations |
| **Admin** | `/api/v1/admin/*` | `requireAdmin` middleware |
| **Admin** | Registered `/api/v1/variants*` reads/writes and snapshot recovery | `requireAdmin` middleware |
| **Admin** | `/api/v1/metrics/summary`, `/api/v1/metrics/variants` | `requireAdmin` middleware |

### Protected Admin Endpoints

The old `/api/v1/public/variants/*` routes are retired. Legacy section snapshots
remain private even when a section's enabled flag is true. Public presentation
reads enforce app membership, visibility, publication and route scope separately.

All admin operations require authentication:

```
POST   /landing_page_business_suite.v1.AdminAuthService/Logout
POST   /landing_page_business_suite.v1.AdminResetService/ResetDemoData
POST   /landing_page_business_suite.v1.StripeSettingsService/GetStripeSettings
POST   /landing_page_business_suite.v1.StripeSettingsService/UpdateStripeSettings
POST   /landing_page_business_suite.v1.StripeSettingsService/RevealStripeSecret
GET    /api/v1/admin/download-apps
POST   /api/v1/admin/download-apps
PUT    /api/v1/admin/download-apps/{app_key}
DELETE /api/v1/admin/download-apps/{app_key}
POST   /landing_page_business_suite.v1.BundleAdminService/ListBundleCatalog
POST   /landing_page_business_suite.v1.BundleAdminService/UpdateBundlePrice
POST   /landing_page_business_suite.v1.CouponAdminService/*
GET    /api/v1/admin/branding
PUT    /api/v1/admin/branding
POST   /api/v1/admin/branding/clear-field
GET    /api/v1/admin/assets
POST   /api/v1/admin/assets/upload
GET    /api/v1/admin/assets/{id}
DELETE /api/v1/admin/assets/{id}
PUT    /api/v1/admin/variants/{slug}/seo
```

---

## CORS Configuration

### Default Behavior

The template uses `gorilla/handlers` for recovery but does **not** enable CORS by default. For cross-origin requests:

### Enabling CORS

Add CORS middleware in `main.go`:

```go
import "github.com/gorilla/handlers"

// In setupRoutes or Server.Start:
corsOrigins := handlers.AllowedOrigins([]string{"https://your-domain.com"})
corsMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
corsHeaders := handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})
corsCredentials := handlers.AllowCredentials()

handler := handlers.CORS(corsOrigins, corsMethods, corsHeaders, corsCredentials)(s.router)
```

### CORS Configuration Guidelines

| Environment | Allowed Origins | Credentials |
|-------------|-----------------|-------------|
| Development | `http://localhost:*` | Yes |
| Staging | Specific staging domain | Yes |
| Production | Production domain only | Yes |

**⚠️ Never use `*` for allowed origins when credentials are enabled.**

---

## Stripe Webhook Security

### Signature Verification

All Stripe webhooks are verified using the webhook signing secret:

```go
// Handler extracts signature
signature := r.Header.Get("Stripe-Signature")

// Service verifies before processing
if !s.VerifyWebhookSignature(body, signature) {
    return errors.New("invalid webhook signature")
}
```

### Configuration

```bash
# Set webhook secret from Stripe Dashboard
export STRIPE_WEBHOOK_SECRET=whsec_xxx
```

Or configure via Admin Portal → Settings → Stripe.

### Webhook Security Properties

- **Timestamp validation**: Prevents replay attacks
- **HMAC-SHA256**: Cryptographic signature verification
- **Raw body preservation**: Signature computed on exact payload

---

## Database Security

### Connection Security

```bash
# Production: Always use SSL
DATABASE_URL=postgres://user:pass@host:5432/db?sslmode=require
```

### Query Safety

- All queries use parameterized statements (`$1`, `$2`, etc.)
- No string concatenation for SQL queries
- ORM/raw SQL uses parameter binding

Example:
```go
// Safe - parameterized
db.QueryRow("SELECT * FROM admin_users WHERE email = $1", email)

// Unsafe - never do this
db.QueryRow("SELECT * FROM admin_users WHERE email = '" + email + "'")
```

### Sensitive Data Storage

| Data Type | Storage Method |
|-----------|----------------|
| Passwords | bcrypt hash (never plaintext) |
| Stripe Restricted Key | Credential authority (`vrooli/landing-page-business-suite:stripe-secret-key`) |
| Webhook Secret | Credential authority (`vrooli/landing-page-business-suite:stripe-webhook-secret`) |
| Remote profile sessions (`admin_session`) | Encrypted in `remote_profiles` |
| Session Secret | Credential authority (`vrooli/landing-page-business-suite:session-secret`) |

---

## Environment Variables

### Security-Critical Variables

LPBS resolves its authored credentials from the credential authority. Do not set
the fields below as process environment variables; provision them with the
governed credential flow and verify them with `vrooli credentials doctor`.

| Variable | Purpose | Required |
|----------|---------|----------|
| `session-secret` | Signs session cookies | **Yes** in production |
| `remote-profile-encryption-key` | Encrypts stored remote admin sessions | **Yes** in production |
| `api-key-encryption-key` | Encrypts stored AI provider API keys | **Yes** in production |
| `stripe-webhook-secret` | Verifies Stripe webhooks | Yes if using Stripe |
| `DATABASE_URL` | Database connection (use SSL) | Yes |
| `ADMIN_DEFAULT_EMAIL` | Admin account email | No (default: `admin@localhost`) |
| `ADMIN_DEFAULT_PASSWORD` | Admin account password | **Yes** in production; ephemeral development value otherwise |

### Recommended Settings

```bash
# Generated LPBS credentials are minted by LPBS and recorded in the credential authority.
# Do not create replacement values by hand after a witness exists.

# Admin credentials (required for production)
ADMIN_DEFAULT_EMAIL=admin@yourdomain.com
ADMIN_DEFAULT_PASSWORD=$(openssl rand -base64 16)  # Strong random password

# Stripe keys from the dashboard are provisioned into the credential authority.

# Database with SSL
DATABASE_URL=postgres://user:pass@host:5432/db?sslmode=require
```

---

## Production Security Checklist

### Pre-Deployment

- [ ] **Set production admin credentials** - Set `ADMIN_DEFAULT_EMAIL` and `ADMIN_DEFAULT_PASSWORD` in the deployment secret store before startup
- [ ] **Verify generated credentials** - Confirm `session-secret`, `remote-profile-encryption-key`, and `api-key-encryption-key` are present with `vrooli credentials doctor`
- [ ] **Enable HTTPS** - All traffic must be encrypted
- [ ] **Set secure-cookie policy** - Keep `LPBS_SECURE_COOKIES` enabled (the production default)
- [ ] **Configure CORS** - Restrict to your production domain only
- [ ] **Use SSL for database** - `sslmode=require` in connection string
- [ ] **Set Stripe webhook secret** - Configure in the admin portal or credential authority
- [ ] **Use the pinned Go toolchain** - Build API and CLI with Go 1.25.12 or newer; both modules enforce this minimum to include patched standard-library security fixes

### Deployment

- [ ] **Disable demo reset** - Do NOT set `ENABLE_ADMIN_RESET=true` in production
- [ ] **Review environment variables** - No secrets in code or version control
- [ ] **Enable logging** - Structured logs for security events
- [ ] **Configure rate limiting** - Add middleware for login attempts
- [ ] **Set up monitoring** - Alert on failed login attempts

### Post-Deployment

- [ ] **Rotate credentials deliberately** - Use the documented overlap/ring rotation process; do not replace generated values by minting on a missing-store response
- [ ] **Audit admin access** - Review `last_login` timestamps
- [ ] **Keep dependencies updated** - Monitor for security advisories
- [ ] **Test webhook signatures** - Verify Stripe events are validated

### Security Headers (Recommended)

Add these headers via reverse proxy or middleware:

```
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

---

## Common Vulnerabilities

### Prevented by Design

| Vulnerability | Prevention |
|---------------|------------|
| **SQL Injection** | Parameterized queries throughout |
| **XSS** | React escapes by default; no `dangerouslySetInnerHTML` |
| **CSRF** | SameSite cookies; session-based auth |
| **Password Storage** | bcrypt hashing with salt |
| **Session Fixation** | New session created on login |

### Requires Configuration

| Vulnerability | Required Action |
|---------------|-----------------|
| **Insecure Cookies** | Set `Secure=true` for HTTPS |
| **Weak Secrets** | Use generated authority credentials and provision operator credentials through the governed flow |
| **Replay Attacks** | Stripe webhook timestamps verified |
| **Man-in-the-Middle** | Deploy behind HTTPS |

### Rate Limiting and abuse controls

The current compatibility login and public endpoints must retain bounded
rate-limiting and abuse controls at the API boundary. Do not treat the
example below as the complete production policy; verify the active middleware
and deployment configuration before making a security claim. The controls
should cover:
- Login attempts (prevent brute force)
- API endpoints (prevent abuse)
- Webhook endpoints (prevent flooding)

Example with middleware:
```go
import "golang.org/x/time/rate"

var limiter = rate.NewLimiter(rate.Every(time.Second), 10) // 10 req/sec

func rateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !limiter.Allow() {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

---

## See Also

- [Configuration Guide](../guides/CONFIGURATION_GUIDE.md) - Environment variable reference
- [Deployment Guide](../guides/DEPLOYMENT.md) - Production deployment steps
- [API Reference](api/OVERVIEW.md) - Endpoint documentation
- [Troubleshooting](../guides/TROUBLESHOOTING.md) - Common issues
# Consumer subscription identity

Consumer access tokens are signed by landing-page-business-suite with RSA-2048
or stronger `RS256` keys. The private key is authority-only and is supplied by
the `CONSUMER_AUTH_PRIVATE_KEY` credential. Relying scenarios receive only the
public key set at `/.well-known/jwks.json`; each token carries a `kid`, and the
authority may publish the current and previous key during rotation.

Production verification allows a bounded 30-second clock skew. Relying
scenarios may cache the public key set for at most five minutes and must deny
requests when it cannot establish a valid token. The former symmetric HS256
consumer-session path is deliberately invalidated: HS256 tokens are rejected
as an unsupported algorithm and are not upgraded or accepted during rotation.

Production must set `CONSUMER_AUTH_KEY_ID` to a deployment-specific key ID and
may set `CONSUMER_AUTH_PREVIOUS_JWKS` to a JSON JWKS containing public overlap
keys during rotation. LPBS signs only with the current private key; previous
keys are verification-only and can be removed after the longest access-token
TTL plus clock-skew window.

## Local fixture commands

The following commands operate only against a loopback API base and a
non-production local authority. They seed the normal users, subscriptions,
wallets, and tier-limit tables; they never call Stripe or another payment
provider. The seed operation is idempotent by normalized email.

```text
landing-page-business-suite fixture-seed --email user@example.test --tier solo --credits 1000 --bundle-key business_suite
landing-page-business-suite fixture-token --email user@example.test
landing-page-business-suite fixture-balance --email user@example.test
landing-page-business-suite fixture-zero --email user@example.test
```

The server also requires a loopback request host and refuses these routes in
production. The CLI rejects non-loopback API bases before making a request.
# Native authorization exchange

`POST /api/v1/auth/token` accepts the unchanged `{code, code_verifier,
redirect_uri}` shape. Codes are durable, hashed, one-use grants; exchanges are
limited to 8 KiB and throttled per client IP. The code is burned before redirect
or PKCE validation, so verifier mistakes cannot be guessed repeatedly.
## Administrator session hardening

Administrator cookies are only hints: every request must resolve a live
server-side `admin_sessions` row and pass its absolute expiry, idle timeout and
assurance policy. Production defaults `ADMIN_REQUIRE_MFA=true`; an unenrolled
administrator receives an `enrollment_only` session that can reach only the
explicit MFA enrollment allow-list. Sensitive credential, payment, API-key,
remote-profile, download, pricing and MFA-management operations require a
recent password plus TOTP or recovery-code proof through
`AdminAuthService.Reauthenticate` (default 10 minutes, bounded to 5–30).
Service-principal routes use their separate machine credential and do not use
browser step-up.

The `admin_security_events` table stores hashed session identifiers and is
retained for 400 days. Security notifications are tracking-free and failures
never block the security action.
