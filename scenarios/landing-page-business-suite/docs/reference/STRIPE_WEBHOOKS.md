---
title: "Stripe Webhooks & Signing Secret"
description: "Set up the webhook endpoint and signing secret used by this template"
category: "operational"
order: 13
audience: ["developers", "operators"]
---

# Stripe Webhooks & Signing Secret

The Stripe webhook signing secret (`whsec_...`) is required to verify events in `POST /api/v1/webhooks/stripe`. Without it, the app rejects webhook calls and subscription/credits data will not sync.

## LPBS commerce environment policy

LPBS uses `https://vrooli.com` as its canonical origin. Configure that value
through the scenario's `PUBLIC_BASE_URL` setting and the credential-authority
entry `vrooli/landing-page-business-suite/public-base-url`; do not put a
customer email, Stripe secret, or webhook secret in source or browser code.

Stripe test and live mode are separate environments. Set `STRIPE_MODE=test` or
`STRIPE_MODE=live` at runtime. The publishable key, server key, webhook
endpoint/secret, customer, product, price, and coupon must all come from the
same mode. Test mode reads the strict authority fields
`stripe-test-publishable-key`, `stripe-test-secret-key`, and
`stripe-test-webhook-secret`; it never falls back to live credentials. Live
mode reads the corresponding `stripe-live-*` fields and temporarily falls back
to the legacy fields during migration. When `STRIPE_MODE` is absent, LPBS uses
the live-mode migration behavior. Test acceptance uses:

```text
https://vrooli.com/api/v1/webhooks/stripe
```

with test-mode credentials and test-mode catalog records. The live credentials
and live catalog remain readiness-only until an operator explicitly approves
live payment activity. Never use the live key to create test products or
prices.

The catalog is mode-bound as well as the credentials. For test acceptance,
point `STRIPE_PLANS_PATH` at a separately provisioned catalog whose bundle has
`"environment": "test"` (and set `BUNDLE_ENVIRONMENT=test`). The production
`.vrooli/plans.json` contains live price IDs and must not be used in test mode.
LPBS fails closed when an explicit `STRIPE_MODE` and the catalog environment do
not match. Live mode uses the production catalog by default.

Credit top-ups are one-time, non-expiring purchases. The production catalog
policy requires fixed `$10`, `$50`, and `$100` top-up prices, calculated with
the bundle's default `credits_per_usd` conversion of `1,000,000` internal
credits per USD (`1,000` displayed credits per USD). A top-up is accepted only
when its Stripe price is imported through the LPBS catalog path and its signed
checkout event is applied once to the account wallet.

Stripe refund events are not an automatic credit reversal path yet. LPBS does
not claim that `charge.refunded` or `refund.created` reverses a top-up. Until a
provider-backed refund workflow is enabled, an operator must review the Stripe
refund, the account's credit transaction history, and consumed balance before
applying an audited administrative adjustment. This limitation prevents a
duplicate or out-of-order refund event from silently driving a wallet below
its authoritative ledger state.

## Required events

Add these events to the endpoint:
- `checkout.session.completed` (activate subscriptions, credit top-ups)
- `customer.subscription.created`
- `customer.subscription.updated`
- `customer.subscription.deleted`
- `invoice.paid`
- `invoice.payment_failed`

## Create the signing secret

1. Stripe Dashboard → **Developers → Webhooks**.  
2. Click **+ Add endpoint** (or edit the existing one).  
3. Endpoint URL: `https://<your-domain>/api/v1/webhooks/stripe`  
   - Local: `stripe listen --forward-to http://localhost:${API_PORT}/api/v1/webhooks/stripe`  
4. Select the events above and save.  
5. Stripe shows a **Signing secret** (`whsec_...`). Copy it and provision it in the LPBS credential authority.

## Wire it into the app

Set via admin portal (**Billing → Stripe → Webhook Secret**) or the governed credential command:
- `vrooli credentials provision --identity vrooli/landing-page-business-suite --field stripe-webhook-secret`

For test mode, provision `stripe-test-webhook-secret`; for an explicit live
configuration, provision `stripe-live-webhook-secret`. The publishable and
server keys use the matching `stripe-test-*` or `stripe-live-*` fields.

### How it’s used

- Loaded by `StripeService` through the credential-authority-backed payment-settings seam and cached at startup/refresh.
- Verified in `VerifyWebhookSignature` before any webhook event is processed. Missing or invalid secrets cause the endpoint to return an error and drop the event.

## Validation checklist

- With `stripe listen` forwarding locally, run `stripe trigger checkout.session.completed`; the webhook should return 200 (not signature error).  
- Admin “Stripe Configuration” card shows Webhook Secret as set.  
- Subscription status updates after webhook events (check `/api/v1/subscription/verify` if needed).

## Rotation

- Regenerate the signing secret on the webhook endpoint page, then update the app (admin form or env).  
- Keep separate webhook endpoints and secrets for test vs. live mode.  
- Remove old endpoints/secrets once the new one is live.
