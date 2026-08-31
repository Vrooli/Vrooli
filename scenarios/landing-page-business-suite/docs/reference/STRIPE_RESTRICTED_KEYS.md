---
title: "Stripe Restricted Keys"
description: "How to configure a restricted Stripe key for this template"
category: "operational"
order: 12
audience: ["developers", "operators"]
---

# Stripe Restricted Keys

Use Stripe's restricted keys to shrink blast radius while keeping all billing features working (checkout, portal, subscription cancel, webhook reconciliation). Provision the restricted key (`rk_...`) through Vrooli's credential authority; LPBS resolves it in process and does not read the secret from the environment.

## Required permissions

Grant only the scopes the scenario actually needs:

- Stripe’s UI treats **Write** as a superset that includes **Read**; you cannot toggle both, and you do not need to.
- Checkout Sessions: **write** (create subscription / one-time sessions)
- Billing Portal Sessions: **write** (customer portal links). If your UI does not show this permission, skip it; the current template returns a dashboard URL placeholder and will use portal sessions only when enabled later.
- Subscriptions: **read** and **write** (cancel endpoint + webhook handling)
- Prices: **read** and Products: **read** (plan metadata validation)
- Customers: **read** (portal/session creation and webhook data)

Everything else should stay **No access**. If you do not cancel subscriptions from the API, you may drop `Subscriptions: write`, but the above set matches current and near-term flows.

## Create the restricted key

1. Stripe Dashboard → Developers → API keys → **Create restricted key**.
2. Name it for this app + environment (e.g., `lp-business-suite-prod`).
3. Apply the permissions listed above and save. Copy the `rk_test_...` or `rk_live_...` value.

## Wire it into this app

Provision these through the admin portal (Billing → Stripe) or the credential-authority provisioning flow:

- `stripe-publishable-key`: `pk_...` (the descriptor retains `STRIPE_PUBLISHABLE_KEY` only for the browser projection)
- `stripe-secret-key`: the **restricted** key `rk_...` (admin UI labels this as “Restricted Key”)
- `stripe-webhook-secret`: `whsec_...` from your webhook endpoint configuration

Webhook setup: in Stripe → Developers → Webhooks, add `https://<your-domain>/api/v1/webhooks/stripe` and select:
- `checkout.session.completed`
- `customer.subscription.created`
- `customer.subscription.updated`
- `customer.subscription.deleted`
- `invoice.paid`
- `invoice.payment_failed`

## Validate the setup

- Forward webhooks locally: `stripe listen --forward-to http://localhost:${API_PORT}/api/v1/webhooks/stripe` then `stripe trigger checkout.session.completed`.
- Create sessions: call `POST /api/v1/billing/create-checkout-session` and `GET /api/v1/billing/portal-url`; ensure no permission errors.
- If you use subscription cancel: `POST /api/v1/subscription/cancel` should succeed; otherwise add `Subscriptions: write`.
- Check the admin portal “Stripe Configuration” card: publishable/secret/webhook badges should show as set.

## Rotation tips

- Keep separate test and live restricted keys; never reuse the old `sk_...` key here.
- Rotate keys per environment; update the webhook secret when regenerating endpoints.
- Remove unused restricted keys from the Stripe Dashboard after a rotation window.
