---
title: "Seams & Architecture"
description: "Testability boundaries, responsibility zones, and substitution points"
category: "technical"
order: 6
audience: ["developers"]
---

# Seams & Architecture

> **Last Updated**: 2026-01-07
> **Purpose**: Document deliberate boundaries (seams) where behavior can vary or be substituted without invasive changes

## Overview

A seam is a deliberate boundary where behavior can vary or be substituted without invasive changes. In this scenario, seams keep Stripe integration and landing-page logic testable without calling real services.

This document reflects the current code; claims here have been verified against the implementation.

---

## Responsibility Zones

**Transport (HTTP handlers)**  
`api/*_handlers.go` validate input, enforce auth, and delegate to services. Examples: `stripe_handlers.go`, `payment_settings_handlers.go`, `metrics_handlers.go`, `variant_handlers.go`, `seo_handlers.go` (now delegating merging to `SEOService`).

**Domain/services**  
`PlanService`, `StripeService`, `PaymentSettingsService`, `ContentService`, `VariantService`, `SEOService`, and `MetricsService` contain business rules and orchestration. Presentation layers must not reach into SQL or Stripe directly. SEO persistence now flows through `VariantService.UpdateSEOConfigBySlug`, which owns slug lookup, JSON encoding, and writes so the admin handler stays transport-only.

**Infrastructure/data**  
SQL access lives inside services (`plan_service.go`, `stripe_service.go`, `payment_settings_service.go`, etc.). Environment parsing and router wiring live in `main.go`.

**UI contracts**  
Typed clients under `ui/src/shared/api/*.ts` are the sole boundary for React surfaces (public landing + admin portal). Controllers such as `seoController.ts` adapt those clients for components so React views no longer make raw `fetch` calls.

---

## Landing Config & Fallback Seams

- **Fallback payload provider** (`LandingConfigService.UseFallbackProvider`)  
  - Baked fallback content loads once from `.vrooli/variants/fallback.json` (or `defaultFallbackLandingJSON`). Each response clones the payload, so mutations in one request cannot leak into the next.  
  - Tests can inject a minimal fallback provider to avoid disk reads and to assert fallback behavior when variant selection, section retrieval/renderability, pricing, or download listing fails. The response is marked `fallback` while still attempting to include branding.

---

## Stripe Seams (priority)

- **Config loading seam** (`StripeService.RefreshConfig`)  
  - Source of truth is `loadStripeConfig`: pulls env vars (`STRIPE_PUBLISHABLE_KEY`, `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`, optional `STRIPE_API_BASE`) then overlays admin overrides from `PaymentSettingsService.GetStripeSettings`.  
  - Tests can bypass env + DB by injecting a loader via `StripeService.UseConfigLoader(...)` and calling `RefreshConfig` to apply.  
  - Placeholders keep `hasSecret/hasWebhook` false when keys are absent, preventing accidental live calls.

- **HTTP seam for Stripe calls** (`StripeService.doStripeRequest`)  
  - All Stripe network traffic flows through `stripeAPIURL` + injected client.  
  - Swap the client with `StripeService.UseHTTPClient(...)` or point to a mock server with `STRIPE_API_BASE`.  
  - This keeps checkout/portal/subscription calls testable without real Stripe access.

- **Admin settings seam** (`payment_settings_handlers.go`, `payment_settings_service.go`)  
  - Admin endpoints handle normalization, redaction, and persistence of Stripe keys.  
  - After writes, `StripeService.RefreshConfig` is invoked so runtime state follows storage.  
  - `ConfigSnapshot` redacts secrets while exposing `*_set` flags and source to the UI.

- **Webhook verification seam** (`StripeService.VerifyWebhookSignature`, `handleStripeWebhook`)  
  - Incoming payloads must pass signature verification using the active webhook secret before any Stripe state is persisted.  
  - Tests can sign payloads using the helper in `stripe_handlers_test.go` or override config via the loader seam.

- **Subscription/cache seam** (`StripeService.VerifySubscription`)  
  - Subscription state is cached in Postgres and refreshed from Stripe when stale.  
  - All cache invalidation and reconciliation stay inside `StripeService`; handlers only translate errors and params.

---

## Other Seams

- **Plan/pricing lookup** (`plan_service.go`, `landing_config_service.go`)  
  Centralizes bundle/price metadata. Handlers and UI pricing components must resolve plans via the service to avoid duplicated Stripe IDs or plan tiers.

- **Metrics ingestion** (`metrics_service.go`, `metrics_handlers.go`)  
  Validation and storage happen in the service; handlers only marshal/unmarshal requests.

- **Content & variants** (`content_service.go`, `variant_service.go`)  
  UI content and A/B variants are isolated behind services so new presentations do not touch SQL.
- **SEO composition** (`seo_service.go`)  
  Combines site branding defaults with per-variant SEO config and drives sitemap/robots responses. Handlers call the service; admin UI uses the `seoController` + shared API client instead of direct `fetch` calls to keep transport concerns separate from editing logic.

---

## Testing Guidance

- For Stripe tests, prefer the new seams:  
  - Inject mock config with `UseConfigLoader` + `RefreshConfig`.  
  - Inject an `httptest.Server` client with `UseHTTPClient` (or set `STRIPE_API_BASE`).  
  - Sign webhooks with the helpers in `stripe_handlers_test.go` to exercise signature enforcement.
- Landing config fallback tests should inject a provider via `UseFallbackProvider` to avoid depending on baked JSON and to confirm payloads are copied per request.
- Use `resetStripeTestData`, `upsertTestBundleProduct`, and `insertBundlePrice` helpers to seed pricing without touching production fixtures.
- Admin settings tests should go through `handleUpdateStripeSettings`/`handleGetStripeSettings` to ensure redaction and refresh paths are covered.

---

## Anti-Patterns

- Talking to Stripe directly from handlers or UI; always go through `StripeService`.
- Bypassing `PlanService`/`PaymentSettingsService` when reading or mutating pricing or Stripe keys.
- Skipping `RefreshConfig` after overriding Stripe config in tests.

---

## See Also

- `docs/STRIPE_RESTRICTED_KEYS.md`
- `docs/STRIPE_WEBHOOKS.md`
- `docs/api/payments.md`
