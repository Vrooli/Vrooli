---
title: "Landing Endpoints"
description: "Public landing page configuration APIs"
category: "reference"
order: 2
audience: ["developers"]
---

# Landing Endpoints

Public APIs for retrieving landing page configuration. No authentication required.

## LandingConfigService.GetLandingConfig

Returns one immutable, resolved product presentation with public commerce and
delivery observations. This operation is read-only; fetching a page is not an
analytics exposure.

**Authentication:** None

**Transport:** Connect-RPC `POST /landing_page_business_suite.v1.LandingConfigService/GetLandingConfig`

**Request fields:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `variant_slug` | string | Optional variant slug to force a specific variant |
| `visitor_id` | string | Anonymous correlation ID for deterministic weighted assignment when no variant is forced |
| `route` | string | Bundle root `/` (also the empty default) or public `/apps/:slug` |
| `locale` | string | Configured locale; empty selects the bundle default |

**Example request:**

```json
{
  "route": "/apps/aquila",
  "locale": "en",
  "visitor_id": "visitor_example"
}
```

`presentation.page` contains typed blocks, all marketing/display copy, and their
configured ordering. `presentation.mode` is `empty`, `single_app`, `bundle` or
`app_detail`. `presentation.diagnostics` identifies the requested/resolved route,
variant, revision, locale, content digest, commerce snapshot and asset references.
Assignment source and weight fingerprint are present only when applicable.
Pricing and downloads remain owner-supplied facts; they do not determine which
apps belong to the bundle. Do not infer an app from the first download row.

The typed path does not populate the old `sections`, `header`, `branding` or
`variant` marketing fields. Consumers use `presentation` and its diagnostics.
Unknown/private app routes return NotFound. Missing root publication returns
Unavailable, with no mutable legacy narrative fallback. Responses are no-store.
An owner outage can leave the page readable with explicitly unavailable actions.

**Example:**

```bash
# Get a specific variant through the scenario CLI
landing-page-business-suite landing-config --variant holiday-special --json
```

## LandingConfigService.RecordPresentationExposure

Connect-RPC `POST /landing_page_business_suite.v1.LandingConfigService/RecordPresentationExposure`
records a displayed public weighted assignment. No authentication is required;
the supplied identity is not an authentication or purchase credential.

Send `visitor_id`, `variant_slug`, `revision`, `route`, `locale`, `block_digest`,
`weight_fingerprint`, and `source` set to
`PRESENTATION_ASSIGNMENT_SOURCE_WEIGHTED_VISITOR`. The service rechecks the exact
public resolution and deterministic assignment before the existing metrics owner
deduplicates it. `{ "recorded": false }` means the valid assignment was already
recorded. An invalid/stale proof returns an error, not a successful exposure.

Do not call this operation for an explicit variant URL, draft preview, admin
screen, download chooser or a page that failed to become visible and ready.
The browser shares its anonymous identity with the server-rendered bootstrap;
it does not fetch another random variant during hydration. A metrics error does
not invalidate an otherwise successful page load.

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

- [API Overview](OVERVIEW.md)
- [Variants](variants.md) - A/B testing endpoints
- [Payments](payments.md) - Stripe integration
