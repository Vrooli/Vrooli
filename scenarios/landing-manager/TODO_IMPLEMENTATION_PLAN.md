This plan was generated in ChatGPT while I was out of credits for my coding agent. This plan gives detailed instructions on how I want the landing-manager and corresponding landing-page-react-vite template to be set up, so that they're fully ready to be used to advertise and handle subscriptions for my first app deployment (the browser-automation-studio scenario, which will be part of a bundle).

========================================================================================================

# `IMPLEMENT_AND_VALIDATE.md`

# **IMPLEMENT & VALIDATE: Landing Manager + Landing Page Generator**

This document provides authoritative instructions for autonomous agents working within the **landing-manager** scenario and **landing-page-react-vite** template.

It defines:

* What the system must do
* How Stripe subscription metadata must be interpreted
* How landing pages must be generated, validated, and customized
* Required API support
* Required admin functionality
* Required tests
* Required PRD + Requirements updates
* Required implementation & validation workflows

Agents should treat this document as the *primary source of truth* when implementing or auditing the landing-page system.

---

# **1. PRIMARY OBJECTIVES**

The combined **landing-manager + landing-page-react-vite** system must support:

### **1.1 A/B-testable, API-driven landing pages**

* Landing pages are composed of ordered **sections**, where:

  * Their **existence**, **ordering**, and **properties/settings** are determined by the API.
  * The landing page must always display a **safe fallback** configuration if the API fails.

### **1.2 Admin-configurable landing page variants**

* Admins must be able to:

  * Create landing page variants
  * Edit section configurations
  * Enable/disable variants
  * Assign variant weights for probabilistic A/B testing
  * Assign variant slugs for deterministic marketing funnels
  * Preview variants before publishing
* Changes must be stored in the backend
* The API must expose variant definitions for frontends

### **1.3 Analytics per variant**

* Each landing page must:

  * Report variant impressions
  * Report primary CTA events
  * Send analytics to the backend
  * Allow admin to rank/compare variant performance

### **1.4 Subscription-aware landing experiences**

The landing system must support:

* Displaying accurate pricing pulled from Stripe metadata
* Supporting tiers: Solo, Pro, Studio, Business
* Displaying monthly + yearly versions
* Displaying "$1 intro month" only for monthly plans
* Displaying included credits and bonus credits
* Showing upgrade actions for logged-in users

### **1.5 Download pages for bundled apps**

For apps like **Browser Automation Studio**, landing pages must support:

* Hosting downloadable artifacts (S3 recommended)
* Supporting all platforms (Windows, Mac, Linux)
* Showing release notes
* Validating user subscriptions before enabling download
* Linking download analytics to variant & plan

### **1.6 Subscription validation in bundled apps**

Bundled apps must:

* Ask the Vrooli API for entitlements
* Receive feature flags based on plan tier
* Gracefully degrade when offline
* Store short-lived cached entitlements
* Provide a Settings → Account page for login/logout & plan viewing

### **1.7 Credit system alignment**

Landing-manager and landing pages must display credits in a user-friendly way, using Stripe metadata:

* Internal credits scale: `credits_per_usd`
* Display multiplier: `display_credits_multiplier`
* Display label: `display_credits_label`
* Per-plan included credits should be calculated properly

### **1.8 Full implementation of subscription + intro pricing logic**

The backend (landing-manager API) must:

* Interpret Stripe metadata
* Support “intro price → normal price” via subscription schedules
* Support credit top-ups
* Support donations (entitlements unchanged)
* Store all plan/tier/credit data internally

---

# **2. STRIPE METADATA SPECIFICATION**

Agents must ensure that the landing-manager decodes the following metadata correctly.

## **2.1 Subscription Product Metadata**

```
bundle_key: business_suite
bundle_name: Vrooli Business Suite
credits_per_usd: 1000000
display_credits_multiplier: 0.001
display_credits_label: credits
environment: production
```

* `credits_per_usd` = internal credit scaling (integer-based)
* `display_credits_multiplier` = UI scaling
* Displayed credits = internal * multiplier
* Internal credits = (usd_amount * credits_per_usd)

## **2.2 Subscription Price Metadata**

Example structure:

```
billing_interval: month | year
bonus_type: none | yearly_bonus | ...
bundle_key: business_suite
intro_enabled: true | false
intro_type: flat_amount
intro_amount_usd: 1
intro_periods: 1
intro_price_lookup_key: solo_monthly_intro
kind: subscription
monthly_included_credits: 5
one_time_bonus_credits: 0
plan_rank: 1
plan_tier: solo | pro | studio | business
```

Agents must ensure:

* Monthly plans may have intro pricing
* Yearly plans **must not** have intro pricing
* Intro pricing uses a separate `subscription_intro` price object
* Backend must create **subscription schedules** from this data

### **2.3 Intro Price Metadata**

```
kind: subscription_intro
bundle_key: business_suite
```

### **2.4 Credits Product Metadata**

```
bundle_compatible: business_suite
credit_mode: prepaid
credits_expiry_policy: no_expiry_24mo_inactivity
kind: credits
```

### **2.5 Credits Top-up Price Metadata**

```
is_variable_amount: true
kind: credits_topup
```

### **2.6 Donation Product Metadata**

```
grants_credits: false
grants_entitlements: false
kind: donation
```

### **2.7 Donation Price Metadata**

```
is_variable_amount: true
kind: supporter_contribution
```

---

# **3. BACKEND API REQUIREMENTS**

Agents must ensure the landing-manager API provides:

## **3.1 Variant + Section Endpoints**

```
GET /landing-config?variant=slug
GET /landing-variants
POST /admin/landing-variant
PATCH /admin/landing-variant/:id
DELETE /admin/landing-variant/:id
```

## **3.2 Subscription + Billing Endpoints**

```
POST /billing/create-checkout-session
POST /billing/create-credits-checkout-session
GET /billing/portal-url
GET /plans
GET /me/subscription
GET /me/credits
GET /entitlements (used by bundled apps)
```

## **3.3 Healing / Sync Endpoints**

```
POST /internal/subscriptions/reconcile
POST /webhooks/stripe
```

---

# **4. LANDING-PAGE TEMPLATE REQUIREMENTS**

The **landing-page-react-vite** template must:

## **4.1 Render a landing page entirely from API config**

* Section array:

  ```
  [
    { type: "hero", props: {...}, enabled: true },
    { type: "feature_grid", props: {...} },
    { type: "pricing", props: {...} },
    ...
  ]
  ```
* New section types should require minimal code
* Components must have strongly typed props

## **4.2 Always show a safe fallback**

In case of:

* API timeout
* API error
* CORS issues
* Admin misconfiguration

The landing page must load a baked-in fallback variant.

## **4.3 Include analytics hooks**

* Variant impression event
* CTA clicked
* Pricing option clicked
* Scroll-depth (optional)

## **4.4 Include account/login + upgrade buttons**

* Integrate with OAuth or email auth (per project config)
* If logged in, display plan & credit info
* If not logged in, show CTA to sign up

## **4.5 If applicable: download section**

* Read S3 URLs from API return
* Display Mac/Windows/Linux installers
* Validates user’s subscription before enabling download

---

# **5. IMPLEMENTATION WORKFLOW FOR AGENTS**

Agents must follow this sequence:

## **Step 1 — Investigate**

* Read existing landing-manager code
* Read existing landing-page-react-vite template
* Identify missing features
* Compare actual behavior to this document
* Identify mismatches with Stripe metadata expectations
* Verify plan decoding logic exists and is correct

## **Step 2 — Update the PRD.md**

* Add any missing **operational targets**
* Ensure coverage for:

  * A/B testing
  * fallback behavior
  * APIs
  * subscription awareness
  * download handling
  * entitlements
  * credit handling
  * Stripe schedule creation

## **Step 3 — Update the `/requirements` folder**

For each requirement:

* Add testable acceptance criteria
* Add or update specs for:

  * API endpoints
  * front-end behaviors
  * admin panel
  * subscription schedule logic
  * credit calculations
  * download gating

## **Step 4 — Implement features**

* Build missing endpoints
* Add section types
* Add variant management logic
* Add subscription schedule logic for intro pricing
* Implement entitlement-fetching in the landing page
* Make sure credit display uses Stripe metadata scaling
* Implement fallback landing layout

## **Step 5 — Add tests**

* Unit tests for plan-parsing
* API tests for landing-variant CRUD
* Stripe webhook tests
* Entitlement endpoint tests
* Intro pricing schedule tests
* Landing-page snapshot tests (variant & fallback)

## **Step 6 — Validate**

* Run landing pages with mock API down → ensure fallback works
* Confirm Stripe → DB → entitlements flow meets spec
* Confirm credits are displayed properly
* Confirm correct pricing is shown
* Confirm bundled app downloads validate subscription
* Confirm yearly plans do not get intro pricing
* Confirm top-ups adjust credit wallet correctly
* Confirm donation payments do nothing to entitlements

---

# **6. SUCCESS CRITERIA**

The agent’s work is considered complete when:

### ✔ All API endpoints exist and pass tests

### ✔ Landing pages load correctly from dynamic config

### ✔ Fallback variant works under all failure modes

### ✔ Admin panel supports variant editing + A/B test weights

### ✔ Variant analytics are collected

### ✔ Subscription pricing/tiers match Stripe metadata

### ✔ Intro pricing schedules are created correctly via API

### ✔ Credit balance is updated, displayed, and rolled over correctly

### ✔ Bundled app downloads validate subscription

### ✔ Landing-page-react-vite template is clean, extensible, and tested

### ✔ PRD.md and requirements/ folder reflect full system spec
