---
title: "Bundled App Entitlements"
description: "Integration guide for bundled apps with subscription verification"
category: "integration"
order: 1
audience: ["developers"]
---

# Bundled App Entitlements & Offline Guidance

This reference helps bundled apps such as **Browser Automation Studio** integrate with the landing-page stack so they can gate downloads, feature flags, and credit balances without sacrificing offline resilience.

## 1. Runtime APIs & payloads

| Endpoint | Purpose | Key fields |
| --- | --- | --- |
| `GET /api/v1/entitlements` | Primary contract for bundled apps. Returns subscription status, plan tier, credits, and feature flags extracted from the Stripe metadata. | `status`, `plan_tier`, `price_id`, `credits`, `features`, `subscription`. |
| `GET /api/v1/me/subscription` | Mirrors the Stripe subscription cache so the app can show plan details without parsing the heavier entitlements payload. | `subscription_id`, `plan_tier`, `bundle_key`, `updated_at`. |
| `GET /api/v1/me/credits` | Returns the wallet balances in internal credits plus display metadata (`display_credits_label`, `display_credits_multiplier`). | `balance_credits`, `bonus_credits`, display helpers. |
| `GET /api/v1/downloads?platform=<platform>` | Authorizes download artifacts (Windows/mac/Linux). Responds with metadata only if the caller has an active subscription. | `artifact_url`, `release_notes`, `requires_entitlement`. |

All responses respect the default bundle (`BUNDLE_KEY=business_suite`) and use the Stripe metadata already seeded via `bundle_products` / `bundle_prices`.

## 2. Cache, TTL, and offline fallback

1. **Cache TTL** – The landing-page API caches subscription lookups for `SUBSCRIPTION_CACHE_TTL_SECONDS` (default `60`). Store the timestamp returned under `cached_at` and refuse to use stale data more than this many seconds old unless the app is offline.
2. **Offline mode** – If the API cannot be reached, fall back to the last successful entitlements payload:
   * Persist it locally (disk, SQLite, config file, etc.).
   * Use the cached `status`, `plan_tier`, and `credits` to maintain gating decisions.
   * Show a light warning to the user that the data is cached and may be outdated.
3. **Re-validation** – When connectivity is restored, refresh `/entitlements` and replace the cached model. The cache TTL is short by design to keep entitlements fresh without hammering Stripe webhooks.

## 3. Feature flags & plan metadata

* Plans expose a `features` array in `bundle_prices.metadata`. The `AccountService` copies this list into `EntitlementPayload.Features` so bundled apps can enable UI/UX variations per tier.
* Use `plan_tier` + `plan_rank` to gate capability tiers (solo → pro → studio → business).
* Intro pricing is surfaced via `subscription_schedules` and the `intro_*` fields stored by the API; rely on these fields to show “$1 intro” messaging only for monthly plans.

## 4. Download gating + analytics

* Always call `GET /api/v1/entitlements` before showing download buttons.
* If the response `status` is not `active`/`trialing`, keep the download button disabled and present guidance that the plan needs to be upgraded.
* When the download is allowed, hit `/api/v1/downloads?platform=<platform>` instead of streaming directly from S3 so the server can log the variant + plan metadata and perform entitlement checks.

## 5. Credits & top-ups

* Credits are stored as internal units (`credits_per_usd` × USD); the API multiplies by `display_credits_multiplier` for UI-friendly numbers (`display_credits_label` holds the unit name).
* Top-up purchases (Stripe price kind `credits_topup`) add `credits_per_usd` × `amount_total` to the wallet and log a `credit_transactions` row for auditing.
* The bundled app should show both `balance_credits` and `bonus_credits` so users understand their spending capacity.

## 6. Subscription schedules & intro pricing

* When a monthly plan has `intro_enabled`, the API creates a row in `subscription_schedules` with `intro_amount_cents`, `intro_periods`, and `next_billing_at`.
* Browser Automation Studio can surface this schedule to explain the upcoming renewal and approximate billing date.
* Schedules are stored alongside `subscription_id`, which matches the Stripe subscription identifier returned in the webhook.

## 7. Debugging tips

* Use `SUBSCRIPTION_CACHE_TTL_SECONDS` to increase the cache window during integrations or to rapidly invalidate stale entitlements.
* Check `/api/v1/entitlements?user=<email>` (the landing template accepts `X-User-Email` headers or `user` query params) when replicating logged-in behavior.
* Download gating logs appear in the landing-page metrics dashboard; correlate `download` events with `variant_id` + `plan_tier` for analytics.

Following this guidance keeps Browser Automation Studio aligned with the landing page subscriptions, ensures downloads remain protected, and lets the app gracefully degrade when network connectivity wavers.
