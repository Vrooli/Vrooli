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
  ├─ ConfigStore.LoadAll               (reads config/variants/*.json; imports branding.json once into site_settings)
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
T3  API    → upsert subscriptions; attributed account grant + funding lot + ledger link in one transaction
T4  Stripe → may re-deliver the same event (network/retry) — idempotent
```

- The user's *entitlements* are gated by the local `subscriptions` row state, not by the Stripe API. We only re-query Stripe lazily (e.g. on `/subscription/verify`).

## AI credit admission and settlement

The required order is: authenticate the actor, verify membership in the selected business account, derive the account subscription tier and verified app/global scope, validate both applicable caps and the hard spend ceiling, then reserve funding before provider dispatch. Caller app metadata is not identity. Public device-key issuance creates global account credentials and does not promote a requested app label into billing authority. Such credentials can spend under the global meter when no app caps require a verified binding; configured app caps fail closed without that binding. Trusted app-bound credentials enforce global and exact-app usage plus pending reservations under the account lock.

After dispatch, settlement atomically allocates eligible lots and records the measured receipt under the immutable account, actor, app, and operation binding. Charge is capped at the hold; excess is an overrun. Exact settlement replay is a no-op and conflicting replay fails. A proven pre-dispatch failure may release the hold; its released key operation is terminal and a retry needs a new operation ID. A timeout, disconnect, or ambiguous provider outcome retains the hold for durable Gateway-receipt reconciliation by `internal/aidoor/recovery.go`. Automatic expiry is not evidence that an already dispatched call did not spend. See [funding flows](../concepts/FLOWS.md) for SM1/SM2 and the numeric boundaries.

After total public-response loss, `GET /api/v1/ai/receipt` uses the original
issued bearer key and original `Idempotency-Key` to derive the same operation.
`internal/aidoor/receipt.go` first checks the key claim and account/actor-bound
reservation, then may reconcile a pending hold from the private Gateway receipt.
It cannot dispatch. Only public accounting status and exact terminal charged
credits leave this boundary; unknown/pending never imply zero or successful
content delivery. Wrong-key/account and nonexistent reads are indistinguishable.
See the [public accounting recovery contract](../reference/api/OVERVIEW.md#hosted-ai-accounting-recovery).

SM3 request ordering is owned by `api/internal/aidoor/request.go` and its consumed `request.spec.json` (level 4). The contract covers all 80 state/event pairs and eight traces; tests enumerate the production vocabulary and check reachable composition invariants. Existing J2/J16 journeys replay observed transitions from the real mounted Door, including receipt recovery after a process crash. The request model is ephemeral: it never replaces wallet reservations or Gateway receipts. The wallet owner admits device/session caps and its reservation in the same account transaction. A definitive wallet refusal commits a terminal zero-cost cap marker; a process crash before commit leaves neither cap consumption nor a wallet hold. Provider dispatch follows that commit. A reconstructed request starts in reconciliation and cannot dispatch. Duplicate private receipts cannot perform another commit; a receipt for another operation or with unknown cost cannot settle. Formal generation was attempted but the Quint Rust evaluator exceeded its recursion limit on the full matrix; no level-5 proof is claimed.

Count-based reservations follow a separate non-credit path: authenticate the principal, derive its subscription tier, reserve current usage plus pending quota for the default ten minutes or bounded custom window, then finalize count usage or release pending capacity. The count owner rejects cost-based meters before state mutation and never allocates a funding lot.

## Device and session key lifecycle (SM5)

SM5's level-3 evidence belongs to `api/internal/aidoor/keys.spec.json` and
`api/journeytest/key_lifecycle_test.go`. The contract derives validity from the
existing key rows and membership: device/session is a kind, revocation and expiry
are independent facts, and a session also depends on its parent. It introduces no
persisted lifecycle status. The exhaustive matrix includes missing keys and every
combination of those facts, including overlapping invalidations, denied operations
and repeated events: 50 fact compositions × 14 events = 700 pairs, eight lifecycle
traces, eight paid invalidation traces, and issuance bounds. Missing rows retain
issuer membership/parent prerequisites but have no own revocation/expiry facts.
Replay uses real Postgres and the mounted Door; expiry inputs
are deterministic past timestamps, not a second clock or lifecycle owner.

Issuance creates a fresh identity and never revives an old key. Device issuance
requires consumer identity and account membership. Session issuance, self/child
revocation and scoped requests enter through Door authentication. An active device
may revoke its own child repeatedly; a revoked caller cannot authenticate again.
Sessions cannot mint children or revoke parents/siblings. Parent expiry bounds
session expiry; parent invalidation or membership removal denies subsequent use.
Cap consumption remains an orthogonal wallet admission concern under `caps.go`;
the lifecycle matrix must leave monetary rows unchanged, and realistic paid traces
must check both device and session charges and zero new spend after invalidation.

## Funding reversal lifecycle

The commerce owner verifies signed `charge.dispute.*`, `charge.refunded`, and `radar.early_fraud_warning.created/updated` events. Payment reversal state is separate from the funding lot's spend/exhaustion state. SM4's consumed level-4 contract is path:scenarios/landing-page-business-suite/api/internal/commerce/funding_reversal.spec.json. It declares four orthogonal projections instead of combining independent payment, dispute, refund and warning facts into one status:

| Projection | Explicit pairs | Production owner and conformance |
|---|---:|---|
| Checkout settlement | 36 | Real `CheckoutService` creation, `StripeWebhookService` and `CreditTopupService` over Postgres; `TestJ17_PaymentCheckoutSpecMatrixAndTraces` in path:scenarios/landing-page-business-suite/api/journeytest/funding_test.go consumes the matrix and three traces. |
| Dispute ordering | 110 | `nextDisputeStatus`; path:scenarios/landing-page-business-suite/api/internal/commerce/funding_reversal_test.go checks production vocabulary, unique/exhaustive pairs, terminal behavior and four traces. `TestJ17_FundingDisputeSpecRealTraces` replays those traces through signed webhooks and Postgres. |
| Cumulative refund | 15 | `applyFundingReversal`; `TestJ17_FundingRefundSpecMatrixAndTrace` checks every pair and the partial/full/stale/retry trace against funding, the loss rollup and outbox amounts. |
| Sticky fraud warning | 4 | `applyFundingReversal`; `TestJ17_FundingFraudWarningSpecMatrixAndTrace` checks every pair and a created/updated/replayed trace against the stored warning and freeze. |

The reversal journey consumers live in path:scenarios/landing-page-business-suite/api/journeytest/fraud_test.go. These tests reuse the real J1/J16 plant, with only Stripe and the upstream router faked. All existing J1/J16 names remain unchanged. The webhook event vocabulary is checked against the production dispatch cases, so a new handled payment event requires updating the contract. The checkout projection uses actual persisted `open`, legacy `pending` and `complete` states plus an absent-session precondition; unpaid/async-pending remains an unfunded open/pending session. Failed/expired checkout notifications are currently ignored and never imply payment. Creation is the real service prelude, not another workflow/store. The spec makes no level-5 claim; SM1/SM2 retain their generated flow owners.

Dispute freeze and reservation finalize share the canonical account transaction lock. A pre-freeze hold can finalize against restricted funding, records its source allocation, and updates the one loss rollup. New reservations cannot consume that funding. Refunds apply only the increase in Stripe's cumulative refunded amount. Delayed inquiry snapshots cannot demote a formal chargeback. A lost dispute may receive a genuine late-win financial correction; another active dispute still keeps its funding frozen. Expanded Stripe balance transactions are stored by immutable ID; their signed gross amounts and separate fees drive partial-return corrections even if the dispute remains lost. Source snapshots merge these facts instead of replacing a cumulative counter. Account-age hold/cap policy is data; wallet timestamps are normalized to UTC before SQL comparisons.

Restricted settlement keeps the original hold's funding eligibility: the lot must be settled and have existed, become spendable, and remained unexpired at admission. The real Postgres regression in `business_account_funding_postgres_test.go` checks that a later fraud freeze cannot make an unsettled lot fund a prior hold. `composed_money_model_postgres_test.go` checks the finite status/settlement/source/mode/time/value/account product against repository reads, account/app/global holds and terminal settlements against production SQL, and refund/dispute before, after, and concurrent with finalization through the webhook owner. `journeytest/composed_money_door_test.go` drives the mounted Door and Gateway against unavailable-lot cases and a settled replay receipt. `moneycomposition/flow/flow.json` now adds a generated finite tuple model whose refusal and settlement/replay traces are asserted against complete SQL-derived observations in that same real Postgres-backed Door/Gateway journey. This binds those selected outcomes to production; the full SM1/SM2 interleaving target remains held.

The same transaction writes immutable Money Ledger deltas. Production main starts the existing reservation-cleanup worker at a one-minute cadence and Server.Cleanup cancels and joins it before closing the database. It drains a bounded batch each tick using mode-specific, operator-configured sink records; no sink or trust values are seeded. A failed or uncertain delivery retains the row and external identity. Money Ledger owns postings and idempotency, including append-only corrections. Tests use isolated real Money Ledger modules and SQLite plus the real Postgres wallet and Gateway, with only Stripe/upstream routers faked.

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

The server uses bounded in-process maintenance tickers. Time-based work runs:

- **At request time** — rate-limit windows, JWT expiry checks.
- **At the next polling tick of the dispatcher** — anomaly alerts.
- **Every minute in the reservation-cleanup worker** — committed funding Ledger outbox delivery, expired non-dispatched credit holds and quota cleanup; the shutdown owner joins this worker before closing the database.
- **Out of band via the operator CLI** — bulk imports, remote-profile rotations.

If a true scheduler is added, document its tick cadence here and in `assumptions.md`.
# Native-app authorization grant lifecycle

Native authorization codes are stored as SHA-256 hashes in `native_auth_grants` for
60 seconds. Authorization consumes the browser sign-in proof without creating a
session; the token exchange burns the grant, validates the exact loopback redirect
and PKCE verifier, then creates the native session. A failed verifier cannot be
retried. Reuse of a consumed grant revokes its attached session.
