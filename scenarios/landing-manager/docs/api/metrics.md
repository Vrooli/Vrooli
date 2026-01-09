---
title: "Metrics Endpoints"
description: "Analytics and event tracking APIs"
category: "reference"
order: 5
audience: ["developers"]
---

# Metrics Endpoints

Endpoints for tracking and retrieving analytics data.

## Event Tracking

### POST /metrics/track

Tracks an analytics event. Idempotent (safe to retry).

**Authentication:** None

**Request:**
```json
{
  "variant_id": 1,
  "event_type": "page_view",
  "event_data": {
    "path": "/",
    "referrer": "https://google.com"
  },
  "session_id": "sess_abc123",
  "visitor_id": "vis_xyz789"
}
```

**Event Types:**

| Type | Description | Required Data |
|------|-------------|---------------|
| `page_view` | Page was viewed | `path` |
| `scroll_depth` | User scrolled | `depth` (0-100) |
| `click` | Element clicked | `element_id` |
| `form_submit` | Form submitted | `form_id` |
| `conversion` | Payment completed | `plan_tier`, `amount_cents` |
| `download` | File downloaded | `app_key`, `platform` |

**Response:** `201 Created`
```json
{
  "success": true,
  "message": "Event tracked successfully"
}
```

**Errors:**
- `400 Bad Request` - Invalid event type or missing required fields

**Notes:**
- `session_id` groups events from the same browser session
- `visitor_id` identifies unique visitors (stored in localStorage)
- Events are deduplicated to prevent double-counting

---

## Analytics (Admin)

### GET /metrics/summary

Returns analytics dashboard summary.

**Authentication:** Admin session required

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `start_date` | string | Start date (YYYY-MM-DD), default: 7 days ago |
| `end_date` | string | End date (YYYY-MM-DD), default: today |

**Response:**
```json
{
  "total_visitors": 1234,
  "total_conversions": 45,
  "overall_conversion_rate": 3.65,
  "variants": [
    {
      "variant_slug": "control",
      "visitors": 600,
      "conversions": 25,
      "conversion_rate": 4.17
    },
    {
      "variant_slug": "variant-a",
      "visitors": 634,
      "conversions": 20,
      "conversion_rate": 3.15
    }
  ],
  "top_ctas": [
    {
      "element_id": "hero-cta",
      "clicks": 234,
      "ctr": 18.97
    },
    {
      "element_id": "pricing-pro-cta",
      "clicks": 156,
      "ctr": 12.65
    }
  ]
}
```

---

### GET /metrics/variants

Returns detailed stats per variant with trend data.

**Authentication:** Admin session required

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `start_date` | string | Start date (YYYY-MM-DD) |
| `end_date` | string | End date (YYYY-MM-DD) |
| `variant` | string | Optional: filter to specific variant slug |

**Response:**
```json
{
  "start_date": "2024-01-08",
  "end_date": "2024-01-15",
  "stats": [
    {
      "variant_slug": "control",
      "views": 600,
      "clicks": 120,
      "conversions": 25,
      "conversion_rate": 4.17,
      "trend": [
        { "date": "2024-01-08", "views": 80, "conversions": 3 },
        { "date": "2024-01-09", "views": 85, "conversions": 4 },
        { "date": "2024-01-10", "views": 90, "conversions": 5 }
      ]
    }
  ]
}
```

---

## Scroll Depth Tracking

Scroll depth is tracked in bands:

| Band | Depth Range |
|------|-------------|
| 1 | 0-25% |
| 2 | 25-50% |
| 3 | 50-75% |
| 4 | 75-100% |

Example event:
```json
{
  "variant_id": 1,
  "event_type": "scroll_depth",
  "event_data": {
    "depth": 75
  },
  "session_id": "sess_abc123",
  "visitor_id": "vis_xyz789"
}
```

---

## Click Tracking

Clicks are tracked by element ID:

```json
{
  "variant_id": 1,
  "event_type": "click",
  "event_data": {
    "element_id": "hero-cta",
    "element_text": "Get Started"
  },
  "session_id": "sess_abc123",
  "visitor_id": "vis_xyz789"
}
```

Element IDs should be consistent across variants for comparison.

---

## Conversion Tracking

Conversions are tracked when Stripe checkout completes:

```json
{
  "variant_id": 1,
  "event_type": "conversion",
  "event_data": {
    "plan_tier": "pro",
    "billing_interval": "month",
    "amount_cents": 2900,
    "stripe_session_id": "cs_xxx"
  },
  "session_id": "sess_abc123",
  "visitor_id": "vis_xyz789"
}
```

---

## Client-Side Implementation

Example JavaScript for tracking:

```javascript
const API_BASE = '/api/v1';

// Generate or retrieve visitor ID
function getVisitorId() {
  let id = localStorage.getItem('visitor_id');
  if (!id) {
    id = 'vis_' + Math.random().toString(36).substr(2, 9);
    localStorage.setItem('visitor_id', id);
  }
  return id;
}

// Generate session ID (per tab)
const sessionId = 'sess_' + Math.random().toString(36).substr(2, 9);

// Track event
async function track(eventType, eventData = {}) {
  const variantId = parseInt(localStorage.getItem('variant_id'), 10);

  await fetch(`${API_BASE}/metrics/track`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      variant_id: variantId,
      event_type: eventType,
      event_data: eventData,
      session_id: sessionId,
      visitor_id: getVisitorId()
    })
  });
}

// Usage
track('page_view', { path: window.location.pathname });
track('click', { element_id: 'hero-cta' });
```

---

## See Also

- [API Overview](README.md)
- [Variants](variants.md) - A/B testing with metrics
- [Admin Guide](../ADMIN_GUIDE.md#analytics-dashboard) - Using the analytics dashboard
