---
title: "Stripe Webhooks & Signing Secret"
description: "Set up the webhook endpoint and signing secret used by this template"
category: "operational"
order: 13
audience: ["developers", "operators"]
---

# Stripe Webhooks & Signing Secret

The Stripe webhook signing secret (`whsec_...`) is required to verify events in `POST /api/v1/webhooks/stripe`. Without it, the app rejects webhook calls and subscription/credits data will not sync.

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
5. Stripe shows a **Signing secret** (`whsec_...`). Copy it and set it in the app.

## Wire it into the app

Set via admin portal (**Customization → Stripe Settings → Webhook Secret**) or environment:
- `STRIPE_WEBHOOK_SECRET=whsec_...`

### How it’s used

- Loaded by `StripeService` (`stripe_service.go`) from DB/env and cached at startup/refresh.  
- Verified in `VerifyWebhookSignature` before any webhook event is processed. Missing or invalid secrets cause the endpoint to return an error and drop the event.

## Validation checklist

- With `stripe listen` forwarding locally, run `stripe trigger checkout.session.completed`; the webhook should return 200 (not signature error).  
- Admin “Stripe Configuration” card shows Webhook Secret as set.  
- Subscription status updates after webhook events (check `/api/v1/subscription/verify` if needed).

## Rotation

- Regenerate the signing secret on the webhook endpoint page, then update the app (admin form or env).  
- Keep separate webhook endpoints and secrets for test vs. live mode.  
- Remove old endpoints/secrets once the new one is live.
