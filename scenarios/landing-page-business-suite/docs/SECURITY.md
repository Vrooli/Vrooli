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

---

## Authentication Architecture

### Admin Authentication Flow

The admin portal uses session-based authentication with bcrypt password hashing:

```
┌──────────────────────────────────────────────────────────────────┐
│                    Authentication Flow                           │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. POST /api/v1/admin/login                                     │
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
│     └─────────────────────────────────────────────────┘         │
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

### Default Admin Account

On first run, a default admin account is seeded:

| Field | Value |
|-------|-------|
| Email | `admin@localhost` |
| Password | `admin123` |

**⚠️ CRITICAL: Change the default password immediately in production!**

```sql
-- Update admin password (hash generated with bcrypt)
UPDATE admin_users
SET password_hash = '$2a$10$<your-new-hash>'
WHERE email = 'admin@localhost';
```

Generate a new hash:
```bash
# Using htpasswd
htpasswd -bnBC 10 "" 'YourNewSecurePassword' | tr -d ':\n'

# Or via Go
go run -e 'fmt.Println(bcrypt.GenerateFromPassword([]byte("YourNewPassword"), 10))'
```

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

The session encryption key is loaded from `SESSION_SECRET`:

```bash
# Generate a secure session secret
openssl rand -base64 32
```

**⚠️ CRITICAL:** Always set `SESSION_SECRET` in production. If not set, a development placeholder is used with a warning logged.

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
| **Public** | `/api/v1/landing-config`, `/api/v1/plans`, `/api/v1/branding` | None |
| **Public** | `/api/v1/metrics/track`, `/api/v1/variants/select` | None |
| **Public** | `/api/v1/public/variants/*` | None |
| **Admin** | `/api/v1/admin/*` | `requireAdmin` middleware |
| **Admin** | `/api/v1/variants` (POST/PATCH/DELETE) | `requireAdmin` middleware |
| **Admin** | `/api/v1/sections/*` | `requireAdmin` middleware |
| **Admin** | `/api/v1/metrics/summary`, `/api/v1/metrics/variants` | `requireAdmin` middleware |

### Protected Admin Endpoints

All admin operations require authentication:

```
POST   /api/v1/admin/logout
POST   /api/v1/admin/reset-demo-data
GET    /api/v1/admin/settings/stripe
PUT    /api/v1/admin/settings/stripe
GET    /api/v1/admin/download-apps
POST   /api/v1/admin/download-apps
PUT    /api/v1/admin/download-apps/{app_key}
GET    /api/v1/admin/bundles
PATCH  /api/v1/admin/bundles/{bundle_key}/prices/{price_id}
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
| Stripe Restricted Key (`STRIPE_SECRET_KEY`) | Encrypted in `payment_settings` |
| Webhook Secret | Encrypted in `payment_settings` |
| Session Secret | Environment variable only |

---

## Environment Variables

### Security-Critical Variables

| Variable | Purpose | Required |
|----------|---------|----------|
| `SESSION_SECRET` | Encrypts session cookies | **Yes** in production |
| `STRIPE_WEBHOOK_SECRET` | Verifies Stripe webhooks | Yes if using Stripe |
| `DATABASE_URL` | Database connection (use SSL) | Yes |

### Recommended Settings

```bash
# Generate secure values
SESSION_SECRET=$(openssl rand -base64 32)

# Stripe keys from dashboard
STRIPE_PUBLISHABLE_KEY=pk_live_xxx
STRIPE_SECRET_KEY=sk_live_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx

# Database with SSL
DATABASE_URL=postgres://user:pass@host:5432/db?sslmode=require
```

---

## Production Security Checklist

### Pre-Deployment

- [ ] **Change default admin password** - `admin@localhost` / `admin123` must be changed
- [ ] **Set SESSION_SECRET** - Generate cryptographically random 32+ byte key
- [ ] **Enable HTTPS** - All traffic must be encrypted
- [ ] **Set Secure cookie flag** - Modify `auth.go` line 99: `session.Options.Secure = true`
- [ ] **Configure CORS** - Restrict to your production domain only
- [ ] **Use SSL for database** - `sslmode=require` in connection string
- [ ] **Set Stripe webhook secret** - Configure in admin portal or environment

### Deployment

- [ ] **Disable demo reset** - Do NOT set `ENABLE_ADMIN_RESET=true` in production
- [ ] **Review environment variables** - No secrets in code or version control
- [ ] **Enable logging** - Structured logs for security events
- [ ] **Configure rate limiting** - Add middleware for login attempts
- [ ] **Set up monitoring** - Alert on failed login attempts

### Post-Deployment

- [ ] **Rotate secrets periodically** - SESSION_SECRET, database password
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
| **Weak Secrets** | Generate strong `SESSION_SECRET` |
| **Replay Attacks** | Stripe webhook timestamps verified |
| **Man-in-the-Middle** | Deploy behind HTTPS |

### Rate Limiting (Not Implemented)

Consider adding rate limiting for:
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

- [Configuration Guide](CONFIGURATION_GUIDE.md) - Environment variable reference
- [Deployment Guide](DEPLOYMENT.md) - Production deployment steps
- [API Reference](api/README.md) - Endpoint documentation
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues
