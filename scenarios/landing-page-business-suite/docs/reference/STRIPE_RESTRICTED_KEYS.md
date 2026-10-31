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

Set `STRIPE_MODE` to `test` or `live`, then provision the matching namespace.
When it is absent, LPBS defaults to live-mode migration behavior:

- Test: `stripe-test-publishable-key`, `stripe-test-secret-key`, and `stripe-test-webhook-secret`
- Live: `stripe-live-publishable-key`, `stripe-live-secret-key`, and `stripe-live-webhook-secret`

The legacy `stripe-publishable-key`, `stripe-secret-key`, and
`stripe-webhook-secret` fields remain a live-mode migration fallback. Test mode
never reads them. Secret values stay in the credential authority and are never
put in browser code.

The Stripe catalog must use the same mode as the credentials. In test mode,
set `BUNDLE_ENVIRONMENT=test` and `STRIPE_PLANS_PATH` to a separately
provisioned test catalog with bundle environment `test`; do not point test mode
at the production `.vrooli/plans.json` file. LPBS rejects an explicit mode and
catalog environment mismatch. Live mode defaults to the production catalog.

Keep the publishable and restricted server keys in the same Stripe mode. LPBS
rejects a recognizable `pk_test_`/`pk_live_` versus `sk_`/`rk_` mismatch before
persistence. Webhook secrets are validated separately by Stripe's signature
checks.

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

- Keep separate test and live restricted keys; never reuse a key across modes.
- Rotate keys per environment; update the webhook secret when regenerating endpoints.
- Remove unused restricted keys from the Stripe Dashboard after a rotation window.
