---
title: "Payment Endpoints"
description: "Stripe integration and billing APIs"
category: "reference"
order: 6
audience: ["developers"]
---

# Payment Endpoints

Endpoints for Stripe integration, subscriptions, credits, and entitlements.

## Checkout

### POST /checkout/create

Creates a Stripe checkout session for one-time payments.

**Authentication:** None

**Request:**
```json
{
  "price_id": "price_xxx",
  "customer_email": "user@example.com",
  "success_url": "/success",
  "cancel_url": "/cancel"
}
```

**Response:**
```json
{
  "session_id": "cs_xxx",
  "url": "https://checkout.stripe.com/..."
}
```

**Usage:**
```javascript
const { url } = await fetch('/api/v1/checkout/create', {
  method: 'POST',
  body: JSON.stringify({ price_id: 'price_xxx', customer_email: email })
}).then(r => r.json());

window.location.href = url;
```

---

### POST /billing/create-checkout-session

Creates a checkout session for subscription plans.

**Authentication:** None

**Request:**
```json
{
  "price_id": "price_xxx",
  "customer_email": "user@example.com"
}
```

**Response:** Same as `/checkout/create`

---

### POST /billing/create-credits-checkout-session

Creates a checkout session for credit top-ups.

**Authentication:** None

**Request:**
```json
{
  "credits_amount": 1000,
  "customer_email": "user@example.com"
}
```

**Response:** Same as `/checkout/create`

---

## Billing Portal

### GET /billing/portal-url

Returns Stripe customer portal URL for managing subscriptions.

**Authentication:** User identity required (header or query)

**Headers:** `X-User-Email: user@example.com`

**Response:**
```json
{
  "url": "https://billing.stripe.com/session/..."
}
```

The portal allows customers to:
- View invoices
- Update payment method
- Cancel subscription
- Change plan

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
| `checkout.session.completed` | Create/update subscription |
| `customer.subscription.created` | Record new subscription |
| `customer.subscription.updated` | Update subscription status |
| `customer.subscription.deleted` | Mark subscription canceled |
| `invoice.paid` | Add credits for credit purchases |
| `invoice.payment_failed` | Mark subscription past_due |

**Response:** `200 OK` on success

**Testing locally:**
```bash
# Use Stripe CLI
stripe listen --forward-to localhost:3000/api/v1/webhooks/stripe

# In another terminal
stripe trigger checkout.session.completed
```

---

## Subscription Verification

### GET /subscription/verify

Verifies subscription status for a user.

**Authentication:** None (but requires user identity)

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `user` | string | User email or identity |

**Response:**
```json
{
  "status": "active",
  "plan_tier": "pro",
  "subscription_id": "sub_xxx",
  "current_period_end": "2024-02-15T00:00:00Z"
}
```

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

### POST /subscription/cancel

Cancels a subscription.

**Authentication:** Admin session required

**Request:**
```json
{
  "user_identity": "user@example.com"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Subscription canceled",
  "ends_at": "2024-02-15T00:00:00Z"
}
```

**Note:** Cancellation takes effect at the end of the current billing period.

---

## Account & Entitlements

### GET /me/subscription

Returns current user's subscription details.

**Authentication:** User identity via header

**Headers:** `X-User-Email: user@example.com`

**Response:**
```json
{
  "status": "active",
  "plan_tier": "pro",
  "plan_name": "Pro Monthly",
  "current_period_start": "2024-01-15T00:00:00Z",
  "current_period_end": "2024-02-15T00:00:00Z"
}
```

---

### GET /me/credits

Returns current user's credit balance.

**Authentication:** User identity via header

**Response:**
```json
{
  "balance_credits": 5000,
  "bonus_credits": 1000,
  "display_multiplier": 1.0,
  "display_label": "credits"
}
```

---

### GET /entitlements

Returns feature entitlements for the user.

**Authentication:** User identity via header

**Response:**
```json
{
  "has_active_subscription": true,
  "plan_tier": "pro",
  "features": {
    "downloads_enabled": true,
    "api_access": true,
    "priority_support": false
  },
  "bundled_apps": ["app_1", "app_2"]
}
```

---

## Downloads

### GET /downloads

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

### GET /admin/settings/stripe

Returns Stripe configuration status.

**Authentication:** Admin session required

**Response:**
```json
{
  "configured": true,
  "publishable_key": "pk_test_xxx...",
  "has_secret_key": true,
  "has_webhook_secret": true,
  "dashboard_url": "https://dashboard.stripe.com/..."
}
```

---

### PUT /admin/settings/stripe

Updates Stripe configuration.

**Authentication:** Admin session required

**Request:**
```json
{
  "publishable_key": "pk_test_xxx",
  "secret_key": "sk_test_xxx",
  "webhook_secret": "whsec_xxx",
  "dashboard_url": "https://dashboard.stripe.com/..."
}
```

---

## See Also

- [API Overview](README.md)
- [Admin Guide - Stripe Setup](../ADMIN_GUIDE.md#stripe-setup)
