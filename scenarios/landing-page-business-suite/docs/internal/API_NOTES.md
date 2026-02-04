# Landing Page Business Suite API Notes

Last Updated: 2026-02-04

## Current Module Map
Route registration is organized by domain in `api/routes.go`:
- `registerHealthRoutes`: `/health`, `/api/v1/health`
- `registerLandingRoutes`: `/api/v1/landing-config`, `/api/v1/plans`, `/api/v1/variant-space`, `/api/v1/customize`
- `registerAuthRoutes`: `/api/v1/auth/*`
- `registerAccountRoutes`: `/api/v1/me/*`, `/api/v1/entitlements`, `/api/v1/downloads`
- `registerBillingRoutes`: `/api/v1/billing/*`, `/api/v1/checkout/create`, `/api/v1/webhooks/stripe`, `/api/v1/subscription/*`
- `registerAdminCoreRoutes`: `/api/v1/admin/login`, `/api/v1/admin/session`, `/api/v1/admin/profile`, `/api/v1/admin/settings/stripe`, `/api/v1/admin/reset-demo-data`
- `registerCommerceAdminRoutes`: `/api/v1/admin/download-*`, `/api/v1/admin/bundles*`, `/api/v1/admin/coupons*`, `/api/v1/admin/stripe/import*`
- `registerVariantRoutes`: `/api/v1/variants*`, `/api/v1/public/variants*`, `/api/v1/admin/variants/*`
- `registerContentRoutes`: `/api/v1/branding`, `/api/v1/admin/branding`, `/api/v1/admin/assets*`, `/api/v1/uploads/*`, `/api/v1/seo/*`, `/sitemap.xml`, `/robots.txt`
- `registerMetricsRoutes`: `/api/v1/metrics/*`
- `registerFeedbackRoutes`: `/api/feedback`, `/api/v1/admin/feedback*`
- `registerWaitlistRoutes`: `/api/v1/waitlist`, `/api/v1/admin/waitlist*`
- `registerCreditsRoutes`: `/api/v1/admin/api-keys*`, `/api/v1/admin/tiers*`, `/api/v1/admin/limits*`, `/api/v1/admin/apps/*/limits`, `/api/v1/usage/*`, `/api/v1/admin/usage`
- `registerAIRoutes`: `/api/v1/ai/*`
- `registerDocsRoutes`: `/api/v1/admin/docs/*`
- `registerAdminUserRoutes`: `/api/v1/admin/users*`

## Notes
- Root-level endpoints are intentionally reserved for infra/SEO (`/health`, `/sitemap.xml`, `/robots.txt`, `/api/feedback`).
- Auth boundaries are enforced at registration time via `requireUserAuth`, `requireAdmin`, and service auth wrappers.

## Recent Changes
- 2026-02-04: Consolidated route registration into domain-focused modules in `api/routes.go` to make capability boundaries explicit.
