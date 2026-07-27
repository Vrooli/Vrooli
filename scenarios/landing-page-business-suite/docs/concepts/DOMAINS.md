---
title: "Domains"
description: "Product capability ownership and source-path boundaries"
category: "concepts"
order: 5
audience: ["developers"]
---

# Domains

This inventory is the authoritative product-domain map for Landing Page
Business Suite. It records ownership for the requirement-to-code join and
keeps product logic distinct from shared API infrastructure.

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| administration | Admin authentication, operator accounts, remote profiles, and administrator settings. | Securely operate the landing business suite without exposing administration on public surfaces. | Admin users, sessions, remote-profile records, operator settings. | CRUD / entity | integration / client | admin, administrator, profile, remote profile | `api/auth.go`, `api/auth_test.go`, `api/user_management_handlers.go`, `api/admin_profile_handlers_test.go`, `api/remote_profile_sessions_handlers.go`, `api/remote_profiles_service.go`, `api/apikeys_service.go`, `api/internal/admin/`, `ui/src/App.tsx`, `ui/src/surfaces/admin-portal/`, `bas/cases/01-foundation/admin-login-loads.json` |
| commerce | Pricing, Stripe billing, subscriptions, credits, entitlements, coupons, and payment anomaly policy. | Sell subscriptions and credits while preserving correct entitlement decisions. | Plans, prices, subscriptions, credit balances, coupons, payment settings. | Temporal workflow | integration / client, policy / rules | stripe, billing, subscription, credits, entitlement, coupon, payment | `api/account_service.go`, `api/account_service_entitlements_test.go`, `api/billing_handlers.go`, `api/coupon_handlers.go`, `api/plan_service.go`, `api/plan_service_test.go`, `api/stripe_handlers.go`, `api/stripe_service.go`, `api/stripe_service_test.go`, `api/internal/financial/`, `api/internal/operations/` |
| content | Public content, branding, assets, feedback, and waitlist capture. | Publish and improve the visitor-facing landing experience. | Branding, assets, feedback, waitlist entries. | CRUD / entity | configuration / settings | branding, asset, feedback, waitlist, content | `api/assets_handlers.go`, `api/assets_service.go`, `api/feedback_handlers.go`, `api/feedback_service.go`, `api/waitlist_handlers.go`, `api/waitlist_service.go`, `api/templates/saas-landing-page.json`, `api/internal/content/`, `ui/src/styles.css`, `ui/src/surfaces/public-landing/` |
| delivery | Download catalog, hosted artifacts, release manifests, and entitlement-gated delivery. | Deliver bundled applications only to entitled customers. | Download apps, artifacts, release metadata, storage settings. | Binary / blob | policy / rules, integration / client | download, artifact, release, installer, delivery | `api/download_authorizer_test.go`, `api/download_entitlement_integration_test.go`, `api/download_hosting.go`, `api/download_service.go`, `api/internal/download/` |
| experimentation | Variant selection, landing configuration fallback, and A/B lifecycle management. | Select and manage measurable public landing variants safely. | Variant definitions, section ordering, selection configuration. | Configuration / settings | policy / rules | variant, experiment, ab, fallback, landing config | `api/landing_config_service.go`, `api/landing_config_service_test.go`, `api/variant_handlers.go`, `api/variant_handlers_test.go`, `api/variant_space.go`, `ui/src/app/providers/LandingVariantProvider.tsx`, `ui/src/app/providers/LandingVariantProvider.test.tsx`, `bas/cases/01-foundation/public-landing-loads.json` |
| intelligence | Credit-accounted AI gateway requests and upstream provider access. | Provide authenticated AI access with metered usage and provider isolation. | Provider credentials, model usage, AI request records. | Integration / client | temporal workflow | ai, gateway, model, openrouter, usage | `api/ai_gateway_handlers.go`, `api/ai_gateway_service.go`, `api/ai_gateway_service_test.go`, `api/openrouter_client.go` |
| metrics | Event ingestion, analytics summaries, and conversion reporting. | Measure variant performance and customer engagement without double counting. | Events, analytics aggregates, anomaly alerts. | Reporting / query | policy / rules | metric, analytics, event, conversion, attribution | `api/metrics_handlers.go`, `api/metrics_handlers_test.go`, `api/metrics_service.go`, `api/metrics_service_test.go`, `api/internal/metrics/`, `ui/src/shared/hooks/useMetricsHook.ts`, `.vrooli/endpoints.json` |

## Non-Domains

- `api/internal/schema/` is the schema registry and composition substrate; it contains no product rules.
- `api/internal/testutil/` is test-only shared support; production packages must not import it.
- `api/internal/envx/`, `api/internal/logx/`, and `api/internal/clock/` are generic runtime substrate.
- `api/main.go` and `api/routes.go` are composition roots; they wire domains but do not own product behavior.
