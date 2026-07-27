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
  ├─ construct services (PlanService, DownloadService, …, AIGatewayService)
  └─ setupRoutes
server.Run
  └─ blocks; on shutdown calls srv.Cleanup → db.Close
```

- `api/internal/*/schema.sql` is the sole schema authority. `applyRuntimeSchema` applies those files at runtime and test setup.
- `ConfigStore.LoadAll` is also called inside `resetDemoData`, so an admin "reset" reloads JSON config without restarting.

## Magic-link login

```
Client                       API                              Email provider
   │  POST /auth/magic-link    │                                    │
   ├──────────────────────────►│                                    │
   │                           │ rate-limit (5/15 min per email)    │
   │                           │ insert auth_tokens (token_type=magic_link, expires_at=+15m)
   │                           │ send email ─────────────────────► │
   │  202 Accepted             │                                    │
   ◄───────────────────────────│                                    │
   │                                                                │
   │  GET /auth/verify?token=… (from email link)                    │
   ├──────────────────────────►│                                    │
   │                           │ hash + lookup auth_tokens          │
   │                           │ mark used_at, mint JWT pair        │
   │                           │ insert user_sessions               │
   │  Set-Cookie/JSON tokens   │                                    │
   ◄───────────────────────────│                                    │
```

- Magic-link tokens are single-use (`used_at` is set on first verify; subsequent verifies return 401).
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
  ├─ aiGatewayService.OpenStream(provider)
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
