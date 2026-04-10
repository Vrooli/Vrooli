---
title: "Glossary"
description: "Domain vocabulary and terminology for landing page development"
category: "concepts"
order: 4
audience: ["users", "developers", "agents"]
---

# Glossary

A comprehensive reference for terminology used throughout the Landing Page Business Suite.

---

## A/B Testing Terms

| Term | Definition |
|------|------------|
| **A/B Test** | An experiment comparing two or more page variants to determine which performs better |
| **Axes** | Dimensions along which variants differ: persona, JTBD (job to be done), and conversion style |
| **Control** | The baseline variant that other variants are compared against; typically the original or best-performing version |
| **Conversion** | A completed goal action (purchase, signup, download) tracked for variant comparison |
| **Conversion Rate** | Percentage of visitors who complete a conversion (conversions / visitors * 100) |
| **Conversion Style** | How the variant appeals to visitors: emotional, technical, or visionary |
| **JTBD** | Job To Be Done - the task or problem the visitor is trying to solve (automation, testing, entrepreneurship, marketing) |
| **Persona** | Target audience segment the variant is optimized for (silentFounder, soloDev, qaEngineer, etc.) |
| **Statistical Significance** | Confidence level that observed differences aren't due to random chance (typically 95%) |
| **Sticky Experience** | Visitors see the same variant across sessions via localStorage persistence |
| **Variant** | A version of the landing page with specific content, styling, or messaging for A/B testing |
| **Variant Space** | The full matrix of possible combinations across all axes (persona x JTBD x conversion style) |
| **Weight** | Percentage of traffic allocated to a variant (e.g., 50% control, 30% variant A, 20% variant B) |

---

## Page Structure Terms

| Term | Definition |
|------|------------|
| **CTA** | Call To Action - a button or link prompting user action (e.g., "Start Free Trial") |
| **CTR** | Click-Through Rate - percentage of visitors who click a specific element |
| **Fallback** | Default content shown when the API is unavailable; baked into `.vrooli/fallback/fallback.json` |
| **Hero Section** | The prominent top section with headline, subheadline, and primary CTA |
| **Landing Config** | The API response containing all sections, pricing, and variant data for rendering |
| **Section** | A building block of the landing page (hero, features, pricing, testimonials, FAQ, CTA, footer) |
| **Section Order** | Numeric value determining the display sequence of sections |
| **Section Type** | Category of section content: `hero`, `features`, `pricing`, `testimonials`, `faq`, `cta`, `footer` |

---

## Payment & Subscription Terms

| Term | Definition |
|------|------------|
| **Checkout Session** | Stripe-hosted payment page created via `POST /billing/create-checkout-session` |
| **Credits** | Virtual currency that users can spend on features; calculated from `credits_per_usd` and purchases |
| **Credits Per USD** | Multiplier for converting payment amount to credits (set in Stripe product metadata) |
| **Display Credits Multiplier** | Factor for showing inflated credit numbers to users (cosmetic, not actual value) |
| **Entitlements** | Feature flags granted by subscription tier (determines what features user can access) |
| **Intro Pricing** | Discounted initial period (e.g., $1 for first month) configured in Stripe |
| **Plan Tier** | Subscription level (free, starter, pro, enterprise) with different features |
| **Portal URL** | Stripe-hosted page where customers manage billing (`GET /billing/portal-url`) |
| **Subscription Status** | Current state: `active`, `trialing`, `canceled`, `past_due`, `none` |
| **Top-up** | Additional credit purchase added to existing balance |
| **Webhook** | HTTP callback from Stripe notifying of events (checkout.completed, subscription.updated, etc.) |

---

## Technical Terms

| Term | Definition |
|------|------------|
| **Admin Portal** | Protected `/admin` interface for managing content, variants, and analytics |
| **API Gateway** | Entry point for all API requests; handles routing, auth, and rate limiting |
| **Breadcrumb** | Navigation trail showing current location (e.g., Admin / Customization / Hero) |
| **Design Token** | Named CSS variable for consistent styling (colors, spacing, typography) |
| **JSONB** | PostgreSQL binary JSON type used for flexible section content storage |
| **Lifecycle** | Scenario state management: created → running → stopped/error |
| **Live Preview** | Real-time preview updates as admin edits content (debounced 300ms) |
| **Seam** | Testability boundary in code where components can be isolated for testing |
| **Styling.json** | Configuration file (`.vrooli/styling.json`) containing design tokens |
| **Variant Space JSON** | Configuration file (`.vrooli/variant_space.json`) defining available axes |

---

## Metrics & Analytics Terms

| Term | Definition |
|------|------------|
| **Click Event** | Tracked interaction with a clickable element (includes element ID and variant) |
| **Event** | Any tracked user action: page_view, scroll_depth, click, form_submit, conversion |
| **Form Submit Event** | Tracked form submission (contact, signup, etc.) |
| **Idempotent** | Property ensuring duplicate event submissions don't cause double-counting |
| **Page View** | Tracked when landing page loads (includes variant_id for attribution) |
| **Scroll Depth** | Tracked percentage bands of page scrolled (25%, 50%, 75%, 100%) |
| **Time Range Filter** | Admin dashboard filter for viewing metrics by date range |
| **Variant Filtering** | Admin dashboard filter for viewing metrics by specific variant |

---

## File & Configuration Terms

| Term | Definition |
|------|------------|
| **branding.json** | Site branding configuration (`.vrooli/branding.json`) |
| **endpoints.json** | API endpoint specifications (`.vrooli/endpoints.json`) |
| **fallback.json** | Offline-safe landing payload (`.vrooli/fallback/fallback.json`) |
| **lighthouse.json** | Performance targets configuration (`.vrooli/lighthouse.json`) |
| **service.json** | Scenario lifecycle configuration (`.vrooli/service.json`) |
| **style-packs/** | Design variation presets (`.vrooli/style-packs/`) |
| **styling.json** | Design tokens and CSS theming (`.vrooli/styling.json`) |
| **variant_space.json** | A/B testing axes configuration (`.vrooli/variant_space.json`) |
| **variants/*.json** | Variant content snapshots (`.vrooli/variants/`) |

---

## See Also

- [Core Concepts](CONCEPTS.md) - Detailed explanations with diagrams
- [Architecture](ARCHITECTURE.md) - System design and component relationships
- [Configuration Guide](../guides/CONFIGURATION_GUIDE.md) - All config file formats
- [API Reference](../reference/api/README.md) - Endpoint documentation
