---
title: "Temporal Flows"
description: "Time-ordered behaviors: startup, lifecycle, recurring jobs, async paths"
category: "internal"
order: 105
audience: ["developers"]
internal: true
---

# Temporal Flows

How time-ordered behaviors line up. Read this when reasoning about startup ordering, async dispatch, or "what happens when X arrives in flight."

## Server startup

```
preflight.Run                         (api/main.go)
  └─ may re-exec the process after rebuild → returns true; main returns
NewServer
  ├─ database.Connect                 (with retry/backoff)
  ├─ seedDefaultData
  │   ├─ applyRuntimeSchema            (domain-owned declarative DDL)
  │   ├─ upsert seeded admin (id=1)
  │   ├─ seed payment_settings (id=1)
  │   ├─ seedDownloadDefaults
  │   └─ seedTierLimitsDefaults
  ├─ ConfigStore.LoadAll               (reads config/variants/*.json + branding.json)
  ├─ construct services (PlanService, DownloadService, …, MeteredInferenceService)
  └─ setupRoutes
server.Run
  └─ blocks; on shutdown calls srv.Cleanup → db.Close
```

- `api/internal/*/schema.sql` is the sole schema authority. `applyRuntimeSchema` applies those files at runtime and test setup.
- `ConfigStore.LoadAll` is also called inside `resetDemoData`, so an admin "reset" reloads JSON config without restarting.

## Customer sign-in (code or link)

```
Browser                      API                              Email provider
   │  POST /auth/magic-link {email, browser_binding, context?}      │
   ├──────────────────────────►│                                    │
   │                           │ throttle (auth_rate_events)        │
   │                           │ insert auth_tokens: hashed token + hashed 6-digit code,
   │                           │   binding hash, app context; no users row yet
   │                           │ send code + link (SendGrid → SMTP) ►│
   │  200 {expires_at} | 503 delivery_unavailable                   │
   ◄───────────────────────────│                                    │
   │                                                                │
   │  A) POST /auth/verify-code {email, code, browser_binding}      │
   │  B) POST /auth/magic-link/preview {token}  → confirm screen    │
   │     POST /auth/verify {token, browser_binding}                 │
   │  C) POST /auth/authorize {token|code, PKCE} → loopback redirect│
   ├──────────────────────────►│                                    │
   │                           │ mark used_at; retire sibling requests
   │                           │ get-or-create user, mark verified  │
   │                           │ insert user_sessions, mint JWT pair│
   │  Set-Cookie + tokens (+ context for the same browser)          │
   ◄───────────────────────────│                                    │
```

- Links and codes are single-use and expire after 15 minutes; finishing sign-in by either retires all outstanding requests for the address.
- A resend keeps earlier codes valid until they expire, so a code the person is already typing still works.
- `GET /auth/authorize` without a credential redirects to `/auth/login` with the same PKCE parameters; this is how native clients start.
- An hourly authentication janitor deletes sign-in requests that expired more than a day ago, ended customer sessions after 30 days of inactivity, refresh-token history older than 100 days, and administrator sessions expired for more than one day; throttle events older than a day are pruned on use.
- Refresh-token rotation: each `/auth/refresh` issues a new refresh token and revokes the prior one.

## Stripe checkout & webhook

```
T0  Client → POST /billing/create-checkout-session
T0  API    → Stripe.CheckoutSession.Create
T0  API    → return session_url
T1  Client → redirect to Stripe-hosted page
T2  Stripe → user pays
T3  Stripe → POST /api/v1/webhooks/stripe (signature verified)
T3  API    → dedupe by stripe_event_id; if seen, return 200 no-op
T3  API    → upsert subscriptions row, increment credits, log credit_transactions
T4  Stripe → may re-deliver the same event (network/retry) — idempotent
```

- The user's *entitlements* are gated by the local `subscriptions` row state, not by the Stripe API. We only re-query Stripe lazily (e.g. on `/subscription/verify`).

## AI streaming with credit reservation

```
POST /api/v1/ai/stream  (requireUserAuth)
  ├─ usageService.Reserve(user, est_credits)        → credit_reservations row, status=pending
  ├─ meteredInferenceService.OpenStream(provider)
  ├─ stream chunks back to client (SSE)
  ├─ on stream end:
  │     usageService.Finalize(reservation, actual_credits)
  │       └─ status=finalized; usage_records += actual; wallet -= actual
  ├─ on client disconnect / error / timeout:
  │     usageService.Release(reservation)
  │       └─ status=released; no usage charged
  └─ background sweeper expires reservations older than TTL (status=expired)
```

The reservation row is the canonical "did this stream charge or not" record — never trust the wallet balance alone to answer that question.

## Anomaly dispatch (background)

```
Detection (inline)  →  payment_anomaly_log INSERT, dispatch_status='pending'
                                         │
                                         ▼
Dispatcher tick (anomaly_alert_dispatcher.go)
   ├─ SELECT … WHERE dispatch_status='pending' ORDER BY created_at LIMIT N
   ├─ POST configured webhook
   ├─ on success → dispatch_status='dispatched', dispatched_at=NOW()
   └─ on failure → increment dispatch_attempts, store dispatch_error, retry next tick (backoff)
```

The detecting request returns to the client *before* the dispatcher runs.

## Recurring / scheduled

This scenario currently has **no in-process scheduler**. Anything time-based runs:

- **At request time** — rate-limit windows, JWT expiry checks.
- **At the next polling tick of the dispatcher** — anomaly alerts.
- **Out of band via the operator CLI** — bulk imports, remote-profile rotations.

If a true scheduler is added, document its tick cadence here and in `assumptions.md`.
# Native-app authorization grant lifecycle

Native authorization codes are stored as SHA-256 hashes in `native_auth_grants` for
60 seconds. Authorization consumes the browser sign-in proof without creating a
session; the token exchange burns the grant, validates the exact loopback redirect
and PKCE verifier, then creates the native session. A failed verifier cannot be
retried. Reuse of a consumed grant revokes its attached session.
