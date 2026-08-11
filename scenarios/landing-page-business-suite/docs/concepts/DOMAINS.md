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
| administration | Admin authentication, operator accounts, remote profiles, and administrator settings. | Securely operate the landing business suite without exposing administration on public surfaces. | Admin users, sessions, remote-profile records, operator settings. | CRUD / entity | integration / client | admin, administrator, profile, remote profile | `api/auth.go`, `api/auth_test.go`, `api/user_management_handlers.go`, `api/admin_profile_handlers_test.go`, `api/remote_profile_sessions_handlers.go`, `api/handlers/admin/`, `api/handlers/administration/`, `api/internal/administration/`, `api/internal/administration/remote_profile_service.go`, `api/internal/administration/remote_profile_repository.go`, `api/internal/administration/remote_profile_client.go`, `api/apikeys_service.go`, `api/internal/administration/remote_profile_session_metadata.go`, `ui/src/App.tsx`, `ui/src/surfaces/admin-portal/`, `bas/cases/01-foundation/admin-login-loads.json` |
| commerce | Pricing, Stripe billing, subscriptions, credits, entitlements, coupons, and payment anomaly policy. | Sell subscriptions and credits while preserving correct entitlement decisions. | Plans, prices, subscriptions, credit balances, coupons, payment settings, anomaly-dispatch records. | Temporal workflow | integration / client, policy / rules | stripe, billing, subscription, credits, entitlement, coupon, payment | `api/account_service.go`, `api/account_service_entitlements_test.go`, `api/billing_handlers.go`, `api/coupon_connect.go`, `api/handlers/account/`, `api/handlers/bundles/`, `api/handlers/commerce/`, `api/handlers/coupons/`, `api/plan_catalog.go`, `api/stripe_service.go`, `api/stripe_service_test.go`, `api/payment_anomaly_service.go`, `api/internal/commerce/plan_store.go`, `api/internal/commerce/plan_service.go`, `api/internal/commerce/plan_pricing.go`, `api/internal/commerce/payment_anomaly_service.go`, `api/internal/commerce/anomaly_dispatcher.go`, `api/internal/commerce/catalog.go`, `api/internal/commerce/stripe_price.go`, `api/internal/commerce/stripe_repository.go`, `api/internal/commerce/` |
| content | Public content, branding, assets, and waitlist capture. | Publish and improve the visitor-facing landing experience. | Branding, assets, waitlist entries. | CRUD / entity | configuration / settings | branding, asset, waitlist, content | `api/assets_handlers.go`, `api/assets_service.go`, `api/waitlist_handlers.go`, `api/waitlist_service.go`, `api/templates/saas-landing-page.json`, `api/internal/content/`, `ui/src/styles.css`, `ui/src/surfaces/public-landing/` |
| delivery | Download catalog, hosted artifacts, release manifests, and entitlement-gated delivery. | Deliver bundled applications only to entitled customers. | Download apps, artifacts, release metadata, storage settings. | Binary / blob | policy / rules, integration / client | download, artifact, release, installer, delivery | `api/handlers/delivery/` (HTTP transport), `api/download_authorizer_test.go`, `api/download_entitlement_integration_test.go`, `api/download_service.go` (composition adapter), `api/internal/delivery/authorizer.go`, `api/internal/delivery/authorizer_test.go`, `api/internal/delivery/catalog.go`, `api/internal/delivery/service.go`, `api/internal/delivery/artifact.go`, `api/internal/delivery/requests.go`, `api/internal/delivery/storage.go`, `api/internal/delivery/s3.go` |
| deployment | Deployment-readiness aggregation across managed artifacts, catalog configuration, remote profiles, and plans. | Give operators a safe pre-deploy verdict without embedding product policy in lifecycle wiring. | No independent persisted data; reads owned delivery, administration, and commerce state. | aggregation | operational probe | readiness, deployment, preflight | `api/handlers/deployment/`, `api/deploy_readiness_handler_test.go` |
| experimentation | Variant selection and A/B lifecycle management. | Select and manage measurable public landing variants safely. | Variant definitions, section ordering, selection configuration. | Configuration / settings | policy / rules | variant, experiment, ab | `api/handlers/experimentation/` (HTTP and Connect transport), `api/handlers/variant_space/` (selection-space transport), `api/variant_handlers.go`, `api/variant_handlers_test.go`, `api/config_store.go` (composition adapter), `api/internal/experimentation/{variant_space,config_store,header_config,types}.go`, `ui/src/app/providers/LandingVariantProvider.tsx`, `ui/src/app/providers/LandingVariantProvider.test.tsx`, `bas/cases/01-foundation/public-landing-loads.json` |
| intelligence | Credit-accounted metered inference requests and upstream provider access. | Provide authenticated AI access with metered usage and provider isolation. | Provider credentials, model usage, AI request records. | Integration / client | temporal workflow | ai, metered inference, model, openrouter, usage | `api/handlers/intelligence/` (HTTP transport), `api/internal/intelligence/metered_inference_service.go`, `api/internal/intelligence/metered_inference_client.go`, `api/internal/intelligence/openrouter_client.go`, `api/internal/intelligence/` |
| landing | Public landing configuration aggregation and resilient fallback projection. | Assemble a renderable public landing response from configuration, pricing, catalog, and display-safe offers without assigning those source domains to the transport. | No independently persisted data; reads configuration, pricing, and delivery state. | aggregation | configuration / settings | landing, landing config, fallback, section, branding | `api/internal/landing/`, `api/handlers/config/`, `api/landing_config_composition.go`, `api/landing_config_connect.go`, `api/landing_config_test_compat_test.go`, `api/landing_config_service_test.go` |
| metrics | Event ingestion, feedback capture, analytics summaries, and conversion reporting. | Measure variant performance and customer engagement without double counting. | Events, feedback requests, analytics aggregates, anomaly alerts. | Reporting / query | policy / rules | metric, analytics, event, conversion, attribution, feedback | `api/handlers/metrics/`, `api/metrics_service.go`, `api/metrics_service_test.go`, `api/handlers/feedback/`, `api/internal/metrics/`, `ui/src/shared/api/metrics.ts`, `ui/src/shared/hooks/useMetricsHook.ts`, `.vrooli/endpoints.json` |

## Non-Domains

`api/handlers/config/` is the generated Connect transport for the landing
domain; its historical folder name is retained to avoid a needless public
package-path migration.

- `api/internal/schema/` is the schema registry and composition substrate; it contains no product rules.
- `api/internal/testutil/` is test-only shared support; production packages must not import it.
- `api/internal/envx/` is generic runtime environment substrate; it contains no product-domain rules.
- `api/internal/logx/` is generic runtime logging substrate; it contains no product-domain rules.
- `api/internal/clock/` is generic runtime time substrate; it contains no product-domain rules.
- `api/internal/securevalue/` is generic authenticated-encryption substrate; it contains no product-domain rules.
- `api/internal/contracts/` contains transport-neutral values shared by product domains; it contains no persistence or product behavior.
- `api/main.go` and `api/routes.go` are composition roots; they wire domains but do not own product behavior.
