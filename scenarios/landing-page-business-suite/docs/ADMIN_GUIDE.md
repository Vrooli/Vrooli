---
title: "Admin Portal Guide"
description: "How to use the admin portal to manage your landing page"
category: "getting-started"
order: 2
audience: ["users", "marketers"]
---

# Admin Portal Guide

This guide covers how to use the admin portal in your landing page to manage content, run A/B tests, view analytics, and configure payments.

## Accessing the Admin Portal

Navigate to your landing page's admin URL:
```
http://localhost:<port>/admin
```

The admin portal is **not linked** from the public landing page for security.

### Default Credentials

```
Email: admin@localhost
Password: changeme123
```

-> **Important**: Change these credentials immediately in production via the **Profile** page (`/admin/profile`) to remove seeded defaults.

---

## Dashboard Overview

After logging in, you'll see the admin dashboard:

```
+---------------------------------------------------------------------------+
|  Admin Portal                              [Analytics]  [Customization]  [*]|
+---------------------------------------------------------------------------+
|                                                                             |
|  OVERVIEW                                      Last 7 days v               |
|  +--------------+  +--------------+  +--------------+  +--------------+    |
|  |    1,247     |  |    3.2%      |  |     40       |  |   $1,160     |    |
|  |   Visitors   |  |  Conv. Rate  |  | Conversions  |  |   Revenue    |    |
|  |   ^ +12%     |  |   ^ +0.4%    |  |   ^ +8       |  |   ^ +$290    |    |
|  +--------------+  +--------------+  +--------------+  +--------------+    |
|                                                                             |
|  VARIANT PERFORMANCE                                                        |
|  +--------------------------------------------------------------------------+
|  |  Variant        | Visitors | Conversions | Rate  | Status             | |
|  |-----------------+----------+-------------+-------+--------------------|  |
|  |  Control        |    623   |     18      | 2.9%  | * Active (50%)     | |
|  |  Holiday Promo  |    624   |     22      | 3.5%  | * Active (50%)     | |
|  +--------------------------------------------------------------------------+
|                                                                             |
|  RECENT EVENTS                                                              |
|  * page_view on Control - 2 min ago                                         |
|  * cta_click on Holiday Promo - 5 min ago                                   |
|  * checkout_started on Holiday Promo - 12 min ago                           |
|                                                                             |
+---------------------------------------------------------------------------+
```

The dashboard has two main modes:

| Mode | Purpose |
|------|---------|
| **Analytics / Metrics** | View visitor data, conversion rates, A/B test results |
| **Customization** | Edit content, manage variants, configure settings |

---

## Analytics Dashboard

### Key Metrics

The analytics view shows:

| Metric | Description |
|--------|-------------|
| **Total Visitors** | Unique visitors across all variants |
| **Conversion Rate** | Percentage of visitors who converted (paid) |
| **Top CTAs** | Which call-to-action buttons get the most clicks |
| **Variant Performance** | Side-by-side comparison of A/B test variants |

### Filtering Data

- **Date Range**: Select start and end dates to filter metrics
- **Variant Filter**: View stats for a specific variant only

### Understanding Variant Stats

Each variant shows:
- **Views**: How many times this variant was shown
- **Clicks**: CTA button clicks
- **Conversions**: Completed checkouts
- **Conversion Rate**: Conversions / Views x 100%
- **Trend**: Daily performance chart

---

## A/B Testing

### How Variants Work

When a visitor arrives at your landing page:

1. **URL Override**: If `?variant=<slug>` in URL, show that variant
2. **Returning Visitor**: If they've seen a variant before (localStorage), show the same one
3. **New Visitor**: Randomly select based on variant weights

### Managing Variants

#### Viewing Variants

Go to **Customization -> Variants** to see all variants.

| Status | Meaning |
|--------|---------|
| **Active** | Included in random selection for new visitors |
| **Archived** | Visible in analytics but not shown to new visitors |
| **Deleted** | Permanently removed |

#### Creating a New Variant

1. Go to **Customization -> Variants**
2. Click **Create New Variant**
3. Fill in:
   - **Name**: Display name (e.g., "Holiday Special")
   - **Slug**: URL identifier (e.g., "holiday-special")
   - **Weight**: Traffic percentage (0-100)
   - **Axes**: Persona, JTBD, Conversion Style

New variants start as a copy of the Control variant's sections.

#### Adjusting Traffic Weights

Weights determine how traffic is split. Example:
- Control: 50
- Variant A: 30
- Variant B: 20

Total doesn't need to equal 100 - weights are normalized.

#### Archiving a Variant

Archiving removes a variant from rotation while preserving its data:
1. Go to **Variants**
2. Click **Archive** on the variant
3. Confirm

Archived variants still appear in historical analytics.

---

## Content Customization

### Editing Sections

Each landing page is composed of sections:

| Section Type | Purpose |
|--------------|---------|
| **Hero** | Main headline, subheadline, primary CTA |
| **Features** | Product feature grid |
| **Pricing** | Pricing tiers and comparison |
| **Testimonials** | Customer quotes |
| **FAQ** | Frequently asked questions |
| **CTA** | Secondary call-to-action block |
| **Video** | Embedded video content |
| **Footer** | Links, copyright, social |

### Edit Workflow

1. Go to **Customization**
2. Select a **Variant** to edit
3. Click on a **Section** card
4. Edit fields in the form (left side)
5. See changes in **Live Preview** (right side)
6. Click **Save**

```
+---------------------------------------------------------------------------+
|  Customization > Control > Hero Section                         [<- Back]  |
+---------------------------------------------------------------------------+
|                                                                             |
|  EDIT FORM                          |  LIVE PREVIEW                         |
|  ---------------------------------  |  ---------------------------------    |
|                                     |                                       |
|  Headline *                         |  +---------------------------------+  |
|  +-----------------------------+   |  |                                 |  |
|  | Ship Products 10x Faster    |   |  |   Ship Products 10x Faster     |  |
|  +-----------------------------+   |  |                                 |  |
|  Max 80 characters                  |  |   The fastest way to launch    |  |
|                                     |  |   your SaaS landing page.      |  |
|  Subheadline                        |  |                                 |  |
|  +-----------------------------+   |  |      [Start Free Trial]        |  |
|  | The fastest way to launch   |   |  |                                 |  |
|  | your SaaS landing page.     |   |  +---------------------------------+  |
|  +-----------------------------+   |                                       |
|                                     |                                       |
|  CTA Button Text                    |                                       |
|  +-----------------------------+   |                                       |
|  | Start Free Trial            |   |                                       |
|  +-----------------------------+   |                                       |
|                                     |                                       |
|  CTA Button URL                     |                                       |
|  +-----------------------------+   |                                       |
|  | /signup                     |   |                                       |
|  +-----------------------------+   |                                       |
|                                     |                                       |
|           [Cancel]  [Save]          |                                       |
|                                     |                                       |
+---------------------------------------------------------------------------+
```

Changes take effect immediately - no deployment needed.

### Live Preview

The preview updates within 300ms of typing (debounced). This lets you see exactly how your changes will look without saving first.

---

## Branding Settings

### Site Branding

Go to **Customization -> Branding** to configure:

| Field | Purpose |
|-------|---------|
| **Site Name** | Appears in browser tab, header |
| **Tagline** | Short description |
| **Logo** | Upload your logo image |
| **Favicon** | Browser tab icon |
| **Primary Color** | Main accent color (hex) |

### SEO Settings

Per-variant SEO configuration:

| Field | Purpose |
|-------|---------|
| **Meta Title** | Browser tab / search results title |
| **Meta Description** | Search results description |
| **OG Image** | Social media preview image |
| **Canonical URL** | Preferred URL for search engines |

---

## Stripe Setup

### Configuration Steps

1. Go to **Customization -> Stripe Settings**
2. Enter your Stripe keys:
   - **Publishable Key**: `pk_test_...` or `pk_live_...`
   - **Restricted Key**: `rk_test_...` or `rk_live_...` (created in Stripe as a restricted key)
   - **Webhook Secret**: `whsec_...`
3. Save

### Setting Up Webhooks

In your Stripe Dashboard:

1. Go to **Developers -> Webhooks**
2. Add endpoint: `https://your-domain.com/api/v1/webhooks/stripe`
3. Select events:
   - `checkout.session.completed`
   - `customer.subscription.created`
   - `customer.subscription.updated`
   - `customer.subscription.deleted`
   - `invoice.paid`
   - `invoice.payment_failed`

### Testing Payments

Use Stripe test mode keys and test card numbers:
- Success: `4242 4242 4242 4242`
- Decline: `4000 0000 0000 0002`

---

## Download Management

For landing pages with downloadable apps:

### Configuring Downloads

Go to **Customization -> Downloads** to manage:

| Field | Purpose |
|-------|---------|
| **App Key** | Unique identifier |
| **App Name** | Display name |
| **Description** | What the app does |
| **Platforms** | Windows, Mac, Linux |
| **Installers** | Download URLs per platform |
| **Store Links** | App Store, Google Play URLs |
| **Release Notes** | What's new in this version |

### Entitlement Requirements

Downloads are gated by subscription status. Visitors must have an active subscription to download.

---

## Agent Customization

Landing pages can be customized by AI agents.

### Triggering Agent Customization

1. Go to **Customization -> Agent**
2. Write a **Brief** describing what you want
3. Optionally select a **Persona** (optimization style)
4. Click **Trigger Customization**

The agent will create a GitHub issue and begin investigating your landing page.

### Example Briefs

```
Make the hero section more compelling for enterprise buyers.
Focus on security and compliance messaging.
```

```
Add a comparison section showing us vs competitors.
Emphasize our AI-native approach and cost savings.
```

---

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Cmd/Ctrl + S` | Save current section |
| `Cmd/Ctrl + K` | Quick search / command palette |
| `Escape` | Close dialog / cancel |
| `Tab` | Navigate form fields |

---

## Best Practices

### A/B Testing

1. **One variable at a time**: Only change one thing between variants
2. **Sufficient sample size**: Wait for statistical significance (usually 100+ conversions per variant)
3. **Run long enough**: At least 2 weeks to account for weekly patterns
4. **Document hypotheses**: Note what you expected before seeing results

### Content

1. **Clear headlines**: State the benefit in 8 words or fewer
2. **Specific CTAs**: "Start Free Trial" beats "Get Started"
3. **Social proof**: Include real testimonials with names/photos
4. **Mobile first**: Check preview at mobile widths

### Performance

1. **Compress images**: Use WebP format, under 200KB
2. **Limit video**: One video section maximum
3. **Test regularly**: Run Lighthouse audits monthly

---

## Troubleshooting Admin Portal

### "Session expired"

Your login timed out. Log in again at `/admin/login`.

### Changes not saving

1. Check browser console for errors
2. Ensure the API is running (`/health` returns `healthy`)
3. Try refreshing the page

### Analytics not updating

Metrics are near-real-time with up to 5-minute delay. If data is more than 10 minutes stale:
1. Check API health
2. Check database connection
3. Review API logs

### Stripe webhooks failing

1. Verify webhook secret is correct
2. Check webhook endpoint is accessible
3. Review Stripe Dashboard -> Webhooks for error details

---

**Next**: [Core Concepts](CONCEPTS.md) | [API Reference](api/README.md)
