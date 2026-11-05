---
title: "Payment Endpoints"
description: "Stripe integration and billing APIs"
category: "reference"
order: 6
audience: ["developers"]
---

# Payment Endpoints

Endpoints for Stripe integration, subscriptions, credits, and entitlements.

## Checkout and billing management

Billing transport uses the generated Connect service
`landing_page_business_suite.v1.LandingPagePaymentsService`. With the LPBS API
base URL (normally `/api/v1`), the procedures are:

- `POST /landing_page_business_suite.v1.LandingPagePaymentsService/CreateCheckoutSession`
- `POST /landing_page_business_suite.v1.LandingPagePaymentsService/VerifySubscription`
- `POST /landing_page_business_suite.v1.LandingPagePaymentsService/CancelSubscription`
- `POST /landing_page_business_suite.v1.LandingPagePaymentsService/GetBillingPortal`

`CreateCheckoutSession` accepts a Stripe `price_id`, validated success and
cancel URLs, and `session_kind` (`SUBSCRIPTION` or `CREDITS_TOPUP`). Credit
top-ups require a customer email. When `business_account_id` is supplied,
the caller must be an authenticated account member; email alone cannot select
the billing account. The response contains the hosted Checkout URL.

`GetBillingPortal` requires the authenticated LPBS user and accepts an
optional absolute `return_url`. The Stripe portal handles invoices, payment
methods, cancellation, and plan changes.

---

## Webhooks

### POST /webhooks/stripe

Handles Stripe webhook events. Verifies signature before processing.

**Authentication:** Stripe signature verification

**Headers:**
```
Stripe-Signature: t=xxx,v1=xxx
```

**Handled Events:**

| Event | Action |
|-------|--------|
| `checkout.session.completed` | Create/update subscription or apply a validated credit top-up |
| `customer.subscription.created` | Record new subscription |
| `customer.subscription.updated` | Update subscription status |
| `customer.subscription.deleted` | Mark subscription canceled |
| `invoice.paid` | Refresh subscription payment state |
| `invoice.payment_failed` | Mark subscription past_due |

**Response:** `200 OK` on success

Stripe refund events are intentionally not treated as automatic credit
reversals. An operator must reconcile a refund against the Stripe record,
credit transaction history, and consumed balance before applying an audited
administrative adjustment. LPBS does not claim support for automatic
`charge.refunded` or `refund.created` top-up reversal.

**Testing locally:**
```bash
# Use Stripe CLI
stripe listen --forward-to localhost:3000/api/v1/webhooks/stripe

# In another terminal
stripe trigger checkout.session.completed
```

---

## Subscription verification

`VerifySubscription` accepts `user_identity` (a normalized email or Stripe
customer ID) and returns the cached subscription state. Webhook updates refresh
the cache; a stale cache may trigger a provider refresh within the documented
cache policy.

**Status Values:**

| Status | Description |
|--------|-------------|
| `active` | Paid and current |
| `trialing` | In trial period |
| `past_due` | Payment failed, grace period |
| `canceled` | User canceled, ends at period end |
| `unpaid` | Past grace period |
| `none` | No subscription found |

**Caching:** Responses cached for up to 60 seconds.

---

`CancelSubscription` accepts `user_identity` and is admin-protected. It
requests period-end cancellation and returns the resulting subscription state.

---

## Account and entitlements

Authenticated account reads use the generated Connect service
`landing_page_business_suite.v1.AccountService`:

- `POST /landing_page_business_suite.v1.AccountService/GetMySubscription`
- `POST /landing_page_business_suite.v1.AccountService/GetMyCredits`
- `POST /landing_page_business_suite.v1.AccountService/GetEntitlements`
- `POST /landing_page_business_suite.v1.AccountService/GetCommercialContext`

Identity is derived from the authenticated session. Caller-supplied email
headers or query parameters do not select another account. Credits include the
authoritative balance and the bundle's display label/multiplier. Entitlements
include subscription state, plan tier, feature flags, credit metrics, and
subscription metadata.

The compatibility HTTP endpoint `GET /api/v1/entitlements` is also
authenticated. It rejects a requested `user` that differs from the session
identity and returns a short-lived signed entitlement lease for downstream
gating.

---

## Downloads

### GET /api/v1/downloads

Returns download URL for an entitled asset.

**Authentication:** User identity required

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `app` | string | Required: app key |
| `platform` | string | Required: `windows`, `mac`, or `linux` |

**Response:**
```json
{
  "artifact_url": "https://...",
  "release_version": "1.2.3",
  "release_notes": "Bug fixes...",
  "checksum": "sha256:abc..."
}
```

**Errors:**
- `400 Bad Request` - Missing app or platform
- `403 Forbidden` - Active subscription required
- `404 Not Found` - App or platform not found

---

## Stripe Admin Settings

### StripeSettingsService (Connect)

The authenticated admin settings surface uses generated Connect procedures:

- `POST /landing_page_business_suite.v1.StripeSettingsService/GetStripeSettings`
- `POST /landing_page_business_suite.v1.StripeSettingsService/UpdateStripeSettings`
- `POST /landing_page_business_suite.v1.StripeSettingsService/RevealStripeSecret`

`GetStripeSettings` returns Stripe configuration status. All credential values,
including the publishable key and anomaly webhook URL, are redacted; use the
boolean snapshot and `anomaly_webhook_url_set` indicators for configuration
status.

**Authentication:** Admin session required

**Response:**
```json
{
  "configured": true,
  "publishable_key": "pk_test_xxx...",
  "has_secret_key": true,
  "has_webhook_secret": true,
  "dashboard_url": "https://dashboard.stripe.com/...",
  "anomaly_webhook_url_set": true,
  "anomaly_webhook_enabled": true,
  "anomaly_rate_limits": "{\"checkout_subscription_missing\":{\"burst\":3,\"refill_seconds\":300}}"
}
```

`RevealStripeSecret` is the only operation that returns one unredacted value.
Its request is `{ "field": "secret_key" }` (or `webhook_secret`,
`publishable_key`, or `anomaly_webhook_url`).

---

`UpdateStripeSettings` accepts the same fields in its typed request. The
`anomaly_rate_limits` field is a JSON-object string because it is persisted as
JSONB while preserving optional-field semantics.

Updates Stripe configuration.

**Authentication:** Admin session required

**Request:**
```json
{
  "publishable_key": "pk_test_xxx",
  "secret_key": "sk_test_xxx",
  "webhook_secret": "whsec_xxx",
  "dashboard_url": "https://dashboard.stripe.com/...",
  "anomaly_webhook_url": "https://hooks.slack.com/services/T0/B0/XYZ",
  "anomaly_webhook_enabled": true,
  "anomaly_rate_limits": {
    "checkout_subscription_missing": { "burst": 3, "refill_seconds": 300 }
  }
}
```

Anomaly fields:

- `anomaly_webhook_url` — HTTPS endpoint that receives POSTed anomaly payloads. Validated as a URL and rejected unless `https://`. Treated as a secret; redacted on GET.
- `anomaly_webhook_enabled` — master switch. When `true`, an `anomaly_webhook_url` must also be set (either in this request or already persisted), otherwise the request is rejected with `400`.
- `anomaly_rate_limits` — per-`anomaly_type` token-bucket overrides. Shape: `{ "<type>": { "burst": N, "refill_seconds": M } }`. Defaults when unset are `burst=5`, `refill_seconds=60`.

On successful save, the server refreshes its in-memory anomaly-dispatch config so subsequent dispatches use the new values without a restart.

---

## See Also

- [API Overview](OVERVIEW.md)
- [Admin Guide - Stripe Setup](../../guides/ADMIN_GUIDE.md#stripe-setup)
