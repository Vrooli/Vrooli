---
title: "Core Concepts"
description: "Understanding the Landing Manager architecture and how components work together"
category: "concepts"
order: 3
audience: ["users", "developers", "agents"]
---

# Core Concepts

This document explains the key concepts and architecture of Landing Manager.

## Table of Contents

1. [Factory vs Template vs Generated](#factory-vs-template-vs-generated)
2. [A/B Testing System](#ab-testing-system)
3. [Variant Selection Flow](#variant-selection-flow)
4. [Section Architecture](#section-architecture)
5. [Stripe Integration Flow](#stripe-integration-flow)
6. [Data Flow Architecture](#data-flow-architecture)
7. [Lifecycle Management](#lifecycle-management)

---

## Factory vs Template vs Generated

Landing Manager uses a **meta-scenario pattern** with three distinct layers:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         LANDING MANAGER ECOSYSTEM                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────┐    copies     ┌─────────────────────┐         │
│  │      FACTORY        │ ──────────────▶│     GENERATED       │         │
│  │  (landing-manager)  │               │    (your-landing)    │         │
│  │                     │               │                      │         │
│  │  • Lists templates  │               │  • Runs independently│         │
│  │  • Generates new    │               │  • Has own ports     │         │
│  │  • Manages staging  │               │  • Own database      │         │
│  │  • Promotes to prod │               │  • Full stack app    │         │
│  └─────────────────────┘               └─────────────────────┘         │
│            │                                     ▲                      │
│            │ reads                               │                      │
│            ▼                                     │ based on             │
│  ┌─────────────────────┐                         │                      │
│  │      TEMPLATE       │ ─────────────────────────                      │
│  │ (landing-page-      │                                                │
│  │  react-vite)        │                                                │
│  │                     │                                                │
│  │  • PRD spec         │                                                │
│  │  • Source code      │                                                │
│  │  • Default config   │                                                │
│  │  • Section schemas  │                                                │
│  └─────────────────────┘                                                │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### The Three Layers Explained

| Layer | Location | Purpose | Contains |
|-------|----------|---------|----------|
| **Factory** | `scenarios/landing-manager/` | Management UI/API | Template list, generation, lifecycle control |
| **Template** | `scripts/scenarios/templates/landing-page-react-vite/` | Blueprint | Source code, PRD, schemas, defaults |
| **Generated** | `scenarios/landing-manager/generated/<slug>/` (staging) or `scenarios/<slug>/` (production) | Your app | Complete runnable landing page |

### Key Insight

The Factory **never runs** landing page code itself. It only:
1. Lists available templates
2. Copies template files to create new scenarios
3. Manages the lifecycle of generated scenarios

All landing page runtime (A/B testing, payments, admin) runs in the **Generated** scenario.

---

## A/B Testing System

Landing pages support whole-page A/B testing where different visitors see different variants.

### Variant Dimensions (Axes)

Each variant is defined along three axes:

```
┌───────────────────────────────────────────────────────────────────────┐
│                          VARIANT SPACE                                 │
├───────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  PERSONA (Who is this for?)                                           │
│  ├── ops_leader        → Enterprise operations director               │
│  ├── automation_freelancer → Solo builders / agencies                 │
│  └── product_marketer  → GTM / marketing leads                        │
│                                                                       │
│  JTBD (What job are they hiring us for?)                              │
│  ├── launch_bundle     → First production deployment                  │
│  ├── scale_services    → Turn automations into products               │
│  └── improve_conversions → Optimize existing flows                    │
│                                                                       │
│  CONVERSION STYLE (How do they want to buy?)                          │
│  ├── self_serve        → Direct checkout                              │
│  ├── demo_led          → Book a demo first                            │
│  └── founder_led       → Talk to the founder                          │
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
```

### Creating Effective Variants

1. **Start with Control**: The default variant based on your best hypothesis
2. **Change one axis**: Create variants that differ in one dimension
3. **Measure impact**: Compare conversion rates after sufficient traffic

Example test plan:
```
Control:     ops_leader × launch_bundle × self_serve
Variant A:   ops_leader × launch_bundle × demo_led     (test conversion style)
Variant B:   automation_freelancer × launch_bundle × self_serve (test persona)
```

---

## Variant Selection Flow

When a visitor arrives, here's how their variant is determined:

```
                    ┌─────────────────────────┐
                    │    Visitor Arrives      │
                    │    at Landing Page      │
                    └───────────┬─────────────┘
                                │
                                ▼
                    ┌─────────────────────────┐
                    │  URL has ?variant=slug  │
                    │        parameter?       │
                    └───────────┬─────────────┘
                                │
                    ┌───────────┴───────────┐
                    │                       │
                   YES                      NO
                    │                       │
                    ▼                       ▼
        ┌───────────────────┐   ┌─────────────────────────┐
        │ Force that variant │   │  localStorage has       │
        │ (for testing/      │   │  variant_id stored?     │
        │  preview)          │   └───────────┬─────────────┘
        └───────────────────┘               │
                                ┌───────────┴───────────┐
                                │                       │
                               YES                      NO
                                │                       │
                                ▼                       ▼
                    ┌───────────────────┐   ┌─────────────────────────┐
                    │ Return same       │   │  API selects variant    │
                    │ variant (sticky   │   │  based on weights       │
                    │ experience)       │   │                         │
                    └───────────────────┘   │  weights: [50, 30, 20]  │
                                            │  ─────────────────────  │
                                            │  roll random 0-100      │
                                            │  0-50   → Control       │
                                            │  50-80  → Variant A     │
                                            │  80-100 → Variant B     │
                                            └───────────┬─────────────┘
                                                        │
                                                        ▼
                                            ┌───────────────────────┐
                                            │ Store variant_id in   │
                                            │ localStorage for      │
                                            │ future visits         │
                                            └───────────────────────┘
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
┌─────────────────────────────────────────────────────────────────────┐
│                        LANDING PAGE                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  HERO SECTION                              order: 1         │   │
│  │  headline, subheadline, CTA, background image               │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  FEATURES SECTION                          order: 2         │   │
│  │  title, feature cards (icon, title, description)            │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  PRICING SECTION                           order: 3         │   │
│  │  tiers, prices, features per tier, CTAs                     │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  TESTIMONIALS SECTION                      order: 4         │   │
│  │  quotes, names, titles, photos                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  FAQ SECTION                               order: 5         │   │
│  │  question/answer pairs, accordion                           │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  CTA SECTION                               order: 6         │   │
│  │  headline, button, urgency messaging                        │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  FOOTER SECTION                            order: 7         │   │
│  │  links, social icons, copyright                             │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
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

### Adding Sections for Agents

See [AGENT.md](../api/templates/AGENT.md) for the 3-file process to add new section types.

---

## Stripe Integration Flow

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         STRIPE PAYMENT FLOW                              │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   VISITOR                   LANDING PAGE               STRIPE            │
│      │                           │                        │              │
│      │  1. Clicks "Buy Now"      │                        │              │
│      │ ─────────────────────────▶│                        │              │
│      │                           │                        │              │
│      │                           │  2. POST /checkout     │              │
│      │                           │     /create            │              │
│      │                           │ ──────────────────────▶│              │
│      │                           │                        │              │
│      │                           │  3. Returns            │              │
│      │                           │     checkout_url       │              │
│      │                           │ ◀──────────────────────│              │
│      │                           │                        │              │
│      │  4. Redirect to Stripe    │                        │              │
│      │ ◀─────────────────────────│                        │              │
│      │                           │                        │              │
│      │  5. Complete payment      │                        │              │
│      │ ─────────────────────────────────────────────────▶│              │
│      │                           │                        │              │
│      │  6. Redirect to success   │                        │              │
│      │ ◀─────────────────────────────────────────────────│              │
│      │                           │                        │              │
│      │                           │  7. Webhook:           │              │
│      │                           │     checkout.completed │              │
│      │                           │ ◀──────────────────────│              │
│      │                           │                        │              │
│      │                           │  8. Create             │              │
│      │                           │     subscription       │              │
│      │                           │     record             │              │
│      │                           │                        │              │
│      │                           │  9. Track              │              │
│      │                           │     conversion         │              │
│      │                           │     (variant_id)       │              │
│      │                           │                        │              │
└──────────────────────────────────────────────────────────────────────────┘
```

### Subscription Verification

Bundled apps (like Vrooli Ascension) can verify subscriptions:

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
┌─────────────────────────────────────────────────────────────────────────┐
│                         DATA FLOW OVERVIEW                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐               │
│  │   BROWSER   │     │   GO API    │     │  POSTGRES   │               │
│  │  (React UI) │     │   (Gin)     │     │             │               │
│  └──────┬──────┘     └──────┬──────┘     └──────┬──────┘               │
│         │                   │                   │                       │
│         │                   │                   │                       │
│         │  GET /landing-    │                   │                       │
│         │  config?variant=  │                   │                       │
│         │ ─────────────────▶│                   │                       │
│         │                   │  SELECT sections, │                       │
│         │                   │  pricing, variant │                       │
│         │                   │ ─────────────────▶│                       │
│         │                   │                   │                       │
│         │                   │ ◀─────────────────│                       │
│         │ ◀─────────────────│                   │                       │
│         │                   │                   │                       │
│         │  POST /metrics/   │                   │                       │
│         │  track            │                   │                       │
│         │ ─────────────────▶│                   │                       │
│         │                   │  INSERT event     │                       │
│         │                   │ ─────────────────▶│                       │
│         │                   │                   │                       │
│         │                   │                   │                       │
│         │  ADMIN: PATCH     │                   │                       │
│         │  /sections/{id}   │                   │                       │
│         │ ─────────────────▶│                   │                       │
│         │                   │  UPDATE sections  │                       │
│         │                   │ ─────────────────▶│                       │
│         │                   │                   │                       │
│         │ ◀─────────────────│                   │                       │
│         │  (Live preview)   │                   │                       │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
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
┌─────────────────────────────────────────────────────────────────────────┐
│                      GENERATED SCENARIO LIFECYCLE                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│                              ┌─────────┐                                │
│                              │ CREATED │                                │
│                              └────┬────┘                                │
│                                   │                                     │
│                                   │ make start                          │
│                                   ▼                                     │
│        ┌─────────┐          ┌─────────┐          ┌─────────┐           │
│        │ STOPPED │◀─────────│ RUNNING │─────────▶│  ERROR  │           │
│        └────┬────┘  stop    └────┬────┘  crash   └────┬────┘           │
│             │                    │                    │                 │
│             │                    │ restart            │ restart         │
│             │                    ▼                    │                 │
│             │              ┌─────────┐                │                 │
│             └─────────────▶│ RUNNING │◀───────────────┘                 │
│                   start    └────┬────┘                                  │
│                                 │                                       │
│                                 │ promote                               │
│                                 ▼                                       │
│                           ┌──────────┐                                  │
│                           │ PROMOTED │                                  │
│                           │ (moved to│                                  │
│                           │ /scenarios/)                                │
│                           └──────────┘                                  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Staging vs Production

| Location | Path | Purpose |
|----------|------|---------|
| **Staging** | `scenarios/landing-manager/generated/<slug>/` | Safe testing area |
| **Production** | `scenarios/<slug>/` | Live deployment |

The **Promote** action moves a scenario from staging to production.

### Lifecycle Commands

```bash
# Start a staging scenario
vrooli scenario start <slug> --path ./generated/<slug>

# Start a production scenario
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
| **Factory** | Landing Manager - the management tool |
| **Template** | Blueprint for generating landing pages |
| **Generated Scenario** | A complete landing page application |
| **Variant** | A version of the landing page for A/B testing |
| **Section** | A building block of the landing page (hero, pricing, etc.) |
| **Axes** | Dimensions along which variants differ (persona, JTBD, conversion style) |
| **Promote** | Move from staging to production |
| **Staging** | Test area inside `generated/` folder |

---

**Next**: [Admin Guide](ADMIN_GUIDE.md) | [API Reference](api/README.md) | [Troubleshooting](TROUBLESHOOTING.md)
