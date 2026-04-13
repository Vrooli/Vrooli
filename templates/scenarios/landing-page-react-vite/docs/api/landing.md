---
title: "Landing Endpoints"
description: "Public landing page configuration APIs"
category: "reference"
order: 2
audience: ["developers"]
---

# Landing Endpoints

Public endpoints for retrieving landing page configuration. No authentication required.

## GET /landing-config

Returns the complete landing page configuration including sections, pricing, and downloads.

**Authentication:** None

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `variant` | string | Optional variant slug to force a specific variant |

**Response:**
```json
{
  "variant": {
    "id": 1,
    "slug": "control",
    "name": "Control (Original)",
    "status": "active"
  },
  "sections": [
    {
      "id": 1,
      "section_type": "hero",
      "content": {
        "headline": "Build Landing Pages in Minutes",
        "subheadline": "...",
        "cta_text": "Get Started",
        "cta_url": "/signup"
      },
      "order": 1,
      "enabled": true
    }
  ],
  "pricing": {
    "bundle": { ... },
    "monthly": [ ... ],
    "yearly": [ ... ]
  },
  "downloads": [ ... ],
  "branding": {
    "site_name": "My Landing",
    "logo_url": "/logo.png"
  }
}
```

**Example:**
```bash
# Get default variant
curl http://localhost:3000/api/v1/landing-config

# Force specific variant
curl http://localhost:3000/api/v1/landing-config?variant=holiday-special
```

---

## GET /plans

Returns pricing plans and bundle information.

**Authentication:** None

**Response:**
```json
{
  "bundle": {
    "bundle_key": "business_suite",
    "name": "Business Suite",
    "stripe_product_id": "prod_xxx",
    "credits_per_usd": 100,
    "display_credits_multiplier": 1.0,
    "display_credits_label": "credits"
  },
  "monthly": [
    {
      "stripe_price_id": "price_xxx",
      "plan_name": "Pro Monthly",
      "plan_tier": "pro",
      "billing_interval": "month",
      "amount_cents": 2900,
      "currency": "usd"
    }
  ],
  "yearly": [
    {
      "stripe_price_id": "price_yyy",
      "plan_name": "Pro Yearly",
      "plan_tier": "pro",
      "billing_interval": "year",
      "amount_cents": 29000,
      "currency": "usd"
    }
  ]
}
```

---

## GET /variant-space

Returns the variant space definition with persona, JTBD, and conversion style axes.

**Authentication:** None

**Response:**
```json
{
  "_name": "landingPageVariantSpace",
  "axes": {
    "persona": {
      "variants": [
        {
          "id": "ops_leader",
          "label": "Operations Leader",
          "description": "Director/VP of Operations running multi-scenario deployments.",
          "defaultWeight": 0.4
        }
      ]
    },
    "jtbd": {
      "variants": [
        {
          "id": "launch_bundle",
          "label": "Launch Bundle",
          "description": "First production deployment"
        }
      ]
    },
    "conversionStyle": {
      "variants": [
        {
          "id": "self_serve",
          "label": "Self-Serve",
          "description": "Direct checkout"
        }
      ]
    }
  },
  "constraints": {
    "disallowedCombinations": [
      {
        "persona": "automation_freelancer",
        "conversionStyle": "demo_led",
        "jtbd": "improve_conversions"
      }
    ]
  }
}
```

---

## GET /branding

Returns public branding information.

**Authentication:** None

**Response:**
```json
{
  "site_name": "My Landing",
  "tagline": "Your tagline here",
  "logo_url": "/uploads/logo.png",
  "favicon_url": "/uploads/favicon.ico",
  "theme_primary_color": "#F97316",
  "canonical_base_url": "https://example.com"
}
```

---

## SEO Endpoints

### GET /seo/{slug}

Returns SEO metadata for a variant.

**Authentication:** None

**Response:**
```json
{
  "title": "My Landing - Home",
  "description": "Build amazing products with our platform",
  "og_image_url": "/uploads/og-image.png",
  "canonical_url": "https://example.com"
}
```

### GET /sitemap.xml

Returns XML sitemap for search engines.

**Authentication:** None

**Response:** XML sitemap document

### GET /robots.txt

Returns robots.txt content.

**Authentication:** None

**Response:** Plain text robots.txt

---

## See Also

- [API Overview](README.md)
- [Variants](variants.md) - A/B testing endpoints
- [Payments](payments.md) - Stripe integration
