---
title: "Core Concepts"
description: "Understanding the landing page architecture and how components work together"
category: "concepts"
order: 3
audience: ["users", "developers", "agents"]
---

# Core Concepts

This document explains the key concepts and architecture of your landing page.

## Table of Contents

1. [A/B Testing System](#ab-testing-system)
2. [Variant Selection Flow](#variant-selection-flow)
3. [Section Architecture](#section-architecture)
4. [Stripe Integration Flow](#stripe-integration-flow)
5. [Data Flow Architecture](#data-flow-architecture)
6. [Lifecycle Management](#lifecycle-management)

---

## A/B Testing System

Landing pages support whole-page A/B testing where different visitors see different variants.

### Variant Dimensions (Axes)

Each variant is defined along three axes:

```
+-----------------------------------------------------------------------+
|                          VARIANT SPACE                                 |
+-----------------------------------------------------------------------+
|                                                                       |
|  PERSONA (Who is this for?)                                           |
|  +-- ops_leader        -> Enterprise operations director              |
|  +-- automation_freelancer -> Solo builders / agencies                |
|  +-- product_marketer  -> GTM / marketing leads                       |
|                                                                       |
|  JTBD (What job are they hiring us for?)                              |
|  +-- launch_bundle     -> First production deployment                 |
|  +-- scale_services    -> Turn automations into products              |
|  +-- improve_conversions -> Optimize existing flows                   |
|                                                                       |
|  CONVERSION STYLE (How do they want to buy?)                          |
|  +-- self_serve        -> Direct checkout                             |
|  +-- demo_led          -> Book a demo first                           |
|  +-- founder_led       -> Talk to the founder                         |
|                                                                       |
+-----------------------------------------------------------------------+
```

### Creating Effective Variants

1. **Start with Control**: The default variant based on your best hypothesis
2. **Change one axis**: Create variants that differ in one dimension
3. **Measure impact**: Compare conversion rates after sufficient traffic

Example test plan:
```
Control:     ops_leader x launch_bundle x self_serve
Variant A:   ops_leader x launch_bundle x demo_led     (test conversion style)
Variant B:   automation_freelancer x launch_bundle x self_serve (test persona)
```

---

## Variant Selection Flow

When a visitor arrives, here's how their variant is determined:

```
                    +-------------------------+
                    |    Visitor Arrives      |
                    |    at Landing Page      |
                    +-----------+-------------+
                                |
                                v
                    +-------------------------+
                    |  URL has ?variant=slug  |
                    |        parameter?       |
                    +-----------+-------------+
                                |
                    +-----------+-----------+
                    |                       |
                   YES                      NO
                    |                       |
                    v                       v
        +-------------------+   +-------------------------+
        | Force that variant|   |  localStorage has       |
        | (for testing/     |   |  variant_id stored?     |
        |  preview)         |   +-----------+-------------+
        +-------------------+               |
                                +-----------+-----------+
                                |                       |
                               YES                      NO
                                |                       |
                                v                       v
                    +-------------------+   +-------------------------+
                    | Return same       |   |  API selects variant    |
                    | variant (sticky   |   |  based on weights       |
                    | experience)       |   |                         |
                    +-------------------+   |  weights: [50, 30, 20]  |
                                            |  ---------------------  |
                                            |  roll random 0-100      |
                                            |  0-50   -> Control      |
                                            |  50-80  -> Variant A    |
                                            |  80-100 -> Variant B    |
                                            +-----------+-------------+
                                                        |
                                                        v
                                            +-----------------------+
                                            | Store variant_id in   |
                                            | localStorage for      |
                                            | future visits         |
                                            +-----------------------+
```

### Why localStorage?

- **Simple**: No server-side session management
- **Private**: Works without authentication
- **Persistent**: Survives page refreshes
- **Trade-off**: Doesn't sync across devices (acceptable for landing pages)

---

## Section Architecture

Landing pages are built from composable sections:

```
+---------------------------------------------------------------------+
|                        LANDING PAGE                                  |
+---------------------------------------------------------------------+
|                                                                     |
|  +-------------------------------------------------------------+   |
|  |  HERO SECTION                              order: 1         |   |
|  |  headline, subheadline, CTA, background image               |   |
|  +-------------------------------------------------------------+   |
|                              |                                      |
|  +-------------------------------------------------------------+   |
|  |  FEATURES SECTION                          order: 2         |   |
|  |  title, feature cards (icon, title, description)            |   |
|  +-------------------------------------------------------------+   |
|                              |                                      |
|  +-------------------------------------------------------------+   |
|  |  PRICING SECTION                           order: 3         |   |
|  |  tiers, prices, features per tier, CTAs                     |   |
|  +-------------------------------------------------------------+   |
|                              |                                      |
|  +-------------------------------------------------------------+   |
|  |  TESTIMONIALS SECTION                      order: 4         |   |
|  |  quotes, names, titles, photos                              |   |
|  +-------------------------------------------------------------+   |
|                              |                                      |
|  +-------------------------------------------------------------+   |
|  |  FAQ SECTION                               order: 5         |   |
|  |  question/answer pairs, accordion                           |   |
|  +-------------------------------------------------------------+   |
|                              |                                      |
|  +-------------------------------------------------------------+   |
|  |  CTA SECTION                               order: 6         |   |
|  |  headline, button, urgency messaging                        |   |
|  +-------------------------------------------------------------+   |
|                              |                                      |
|  +-------------------------------------------------------------+   |
|  |  FOOTER SECTION                            order: 7         |   |
|  |  links, social icons, copyright                             |   |
|  +-------------------------------------------------------------+   |
|                                                                     |
+---------------------------------------------------------------------+
```

### Section Storage

Sections are stored in PostgreSQL as JSONB:

```sql
CREATE TABLE sections (
    id SERIAL PRIMARY KEY,
    variant_id INTEGER REFERENCES variants(id),
    section_type VARCHAR(50) NOT NULL,  -- 'hero', 'features', etc.
    content JSONB NOT NULL,             -- Flexible content structure
    "order" INTEGER NOT NULL,           -- Display order
    enabled BOOLEAN DEFAULT true
);
```

---

## Stripe Integration Flow

```
+--------------------------------------------------------------------------+
|                         STRIPE PAYMENT FLOW                              |
+--------------------------------------------------------------------------+
|                                                                          |
|   VISITOR                   LANDING PAGE               STRIPE            |
|      |                           |                        |              |
|      |  1. Clicks "Buy Now"      |                        |              |
|      | ------------------------->|                        |              |
|      |                           |                        |              |
|      |                           |  2. POST /checkout     |              |
|      |                           |     /create            |              |
|      |                           | ---------------------->|              |
|      |                           |                        |              |
|      |                           |  3. Returns            |              |
|      |                           |     checkout_url       |              |
|      |                           | <----------------------|              |
|      |                           |                        |              |
|      |  4. Redirect to Stripe    |                        |              |
|      | <-------------------------|                        |              |
|      |                           |                        |              |
|      |  5. Complete payment      |                        |              |
|      | -------------------------------------------->       |              |
|      |                           |                        |              |
|      |  6. Redirect to success   |                        |              |
|      | <--------------------------------------------       |              |
|      |                           |                        |              |
|      |                           |  7. Webhook:           |              |
|      |                           |     checkout.completed |              |
|      |                           | <----------------------|              |
|      |                           |                        |              |
|      |                           |  8. Create             |              |
|      |                           |     subscription       |              |
|      |                           |     record             |              |
|      |                           |                        |              |
|      |                           |  9. Track              |              |
|      |                           |     conversion         |              |
|      |                           |     (variant_id)       |              |
|      |                           |                        |              |
+--------------------------------------------------------------------------+
```

### Subscription Verification

Bundled apps can verify subscriptions:

```
GET /api/v1/subscription/verify?user=user@example.com

Response:
{
  "status": "active",          // active, trialing, canceled, none
  "plan_tier": "pro",
  "current_period_end": "2024-02-15T00:00:00Z"
}
```

Responses are cached for 60 seconds maximum to balance performance with freshness.

---

## Data Flow Architecture

```
+-------------------------------------------------------------------------+
|                         DATA FLOW OVERVIEW                              |
+-------------------------------------------------------------------------+
|                                                                         |
|  +-------------+     +-------------+     +-------------+               |
|  |   BROWSER   |     |   GO API    |     |  POSTGRES   |               |
|  |  (React UI) |     |   (Gin)     |     |             |               |
|  +------+------+     +------+------+     +------+------+               |
|         |                   |                   |                       |
|         |                   |                   |                       |
|         |  GET /landing-    |                   |                       |
|         |  config?variant=  |                   |                       |
|         | ----------------->|                   |                       |
|         |                   |  SELECT sections, |                       |
|         |                   |  pricing, variant |                       |
|         |                   | ----------------->|                       |
|         |                   |                   |                       |
|         |                   | <-----------------|                       |
|         | <-----------------|                   |                       |
|         |                   |                   |                       |
|         |  POST /metrics/   |                   |                       |
|         |  track            |                   |                       |
|         | ----------------->|                   |                       |
|         |                   |  INSERT event     |                       |
|         |                   | ----------------->|                       |
|         |                   |                   |                       |
|         |                   |                   |                       |
|         |  ADMIN: PATCH     |                   |                       |
|         |  /sections/{id}   |                   |                       |
|         | ----------------->|                   |                       |
|         |                   |  UPDATE sections  |                       |
|         |                   | ----------------->|                       |
|         |                   |                   |                       |
|         | <-----------------|                   |                       |
|         |  (Live preview)   |                   |                       |
|                                                                         |
+-------------------------------------------------------------------------+
```

### Fallback Strategy

If the API is unavailable, the UI loads a baked-in fallback configuration:

```
.vrooli/variants/fallback.json
```

This ensures visitors see something (the fallback variant) even during API outages.

---

## Lifecycle Management

### Scenario States

```
+-------------------------------------------------------------------------+
|                      SCENARIO LIFECYCLE                                  |
+-------------------------------------------------------------------------+
|                                                                         |
|                              +---------+                                |
|                              | CREATED |                                |
|                              +----+----+                                |
|                                   |                                     |
|                                   | make start                          |
|                                   v                                     |
|        +---------+          +---------+          +---------+           |
|        | STOPPED |<---------| RUNNING |--------->|  ERROR  |           |
|        +----+----+  stop    +----+----+  crash   +----+----+           |
|             |                    |                    |                 |
|             |                    | restart            | restart         |
|             |                    v                    |                 |
|             |              +---------+                |                 |
|             +------------->| RUNNING |<---------------+                 |
|                   start    +---------+                                  |
|                                                                         |
+-------------------------------------------------------------------------+
```

### Lifecycle Commands

```bash
# Start the scenario
cd scenarios/<slug>
make start

# Or using Vrooli CLI
vrooli scenario start <slug>

# Stop
vrooli scenario stop <slug>

# View logs
vrooli scenario logs <slug> --tail 100

# Check status
vrooli scenario status <slug>
```

---

## Glossary

| Term | Definition |
|------|------------|
| **Variant** | A version of the landing page for A/B testing |
| **Section** | A building block of the landing page (hero, pricing, etc.) |
| **Axes** | Dimensions along which variants differ (persona, JTBD, conversion style) |
| **Control** | The baseline variant that other variants are compared against |
| **Fallback** | Default content shown when API is unavailable |

---

**Next**: [Admin Guide](ADMIN_GUIDE.md) | [API Reference](api/README.md) | [Troubleshooting](TROUBLESHOOTING.md)
