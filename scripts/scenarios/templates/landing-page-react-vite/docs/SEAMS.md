# Landing Page React Vite Template – Responsibility Seams

This template mirrors a full landing-page scenario, so we keep scenario-level clarity on who owns what.

## Responsibility Zones
- **Entry / presentation** – HTTP handlers in `api/*_handlers.go` (e.g., `account_handlers.go`, `variant_handlers.go`, `metrics_handlers.go`) only parse/validate transport concerns, enforce auth middleware, and serialize responses. Client utilities live in `ui/src/shared/api/*.ts` and exclusively call REST endpoints.
- **Coordination / domain** – Services such as `LandingConfigService`, `VariantService`, `ContentService`, `PlanService`, `MetricsService`, and the new `DownloadAuthorizer` sequence work while hiding HTTP/DB details. They encapsulate business rules (variant selection, pricing assembly, analytics validation, download gating) so presentation code does not need to duplicate logic.
- **Integrations / infrastructure** – Database access stays in services like `download_service.go`, `content_service.go`, `account_service.go`, etc. They are the only layer issuing SQL or touching storage/Stripe adapters. Environment/config parsing (`main.go`, `.env` files) and scenario lifecycle glue live alongside these services.
- **Cross-cutting concerns** – Logging helpers in `logging.go`, middleware (session/auth) in `auth.go`, and helper utilities (`writeJSON`, `resolveUserIdentity`) remain thin seams that presentation code reuses without embedding domain rules.

## Boundary Clarifications Applied
- `/api/v1/downloads` now routes through `DownloadAuthorizer` (see `download_service.go`). The handler no longer reasons about plan bundles or entitlement states; it simply validates request params and maps domain errors (`ErrDownloadNotFound`, `ErrDownloadRequiresActiveSubscription`) to HTTP status codes. The authorizer coordinates `DownloadService` (integration) and `AccountService` (entitlement provider) and owns the access policy.
- Variant HTTP handlers now live in `api/variant_handlers.go`, keeping transport validation and mux wiring separate from `VariantService`'s domain logic. Services can evolve (e.g., axes validation, weighting rules) without touching presentation code, and handlers only translate `VariantService` errors into HTTP responses.
- Download API calls used by the public landing page were moved out of `ui/src/shared/api/landing.ts` into `ui/src/shared/api/downloads.ts`. This keeps landing config clients focused on presentation data, while entitlement-aware download logic is centralized with the rest of the account/download integration helpers.
- Metrics ingestion validation moved into `MetricsService.TrackEvent`. The service enforces required fields and allowed event types via `MetricValidationError`, letting handlers focus purely on transport-level error reporting. This keeps A/B analytics logic testable without HTTP context and ensures future ingestion paths (jobs, batch imports) reuse identical validation.
- Admin portal React routes now hand off orchestration and validation to thin controllers under `ui/src/surfaces/admin-portal/controllers`. `variantEditorController.ts` owns variant data loading, axes selection defaults, slug sanitization, and persistence so `VariantEditor.tsx` only renders form state. `analyticsController.ts` builds reporting windows and performs metrics API calls, keeping `AdminAnalytics.tsx` focused on filters and cards rather than date math or transport handling.
- The section customization surface now consumes `sectionEditorController.ts`, which parses route IDs once, encapsulates the `getSection`/`updateSection` flow, and returns a normalized form snapshot. `SectionEditor.tsx` only manages debounced inputs and layout, while the controller enforces integration boundaries for both fetch and save paths.
- `useMetrics` now extends the shared `MetricEvent` API type instead of redeclaring its own interface, so both the tracking hook and the HTTP client point at the same analytics contract and future schema changes propagate through a single seam.
- Shared UX utilities such as the live-preview debounce now reside in `ui/src/shared/hooks`. Routes import `useDebounce` instead of embedding timer logic, preventing future surfaces from copy/pasting cross-cutting concerns into presentation components.

## How to Extend Safely
- **Presentation changes** (new endpoints or UI calls) should stick to parsing and delegating to services. If a handler needs to apply business rules, extract the rule into a service or helper first.
- **Domain updates** go into the relevant service (`MetricsService`, `DownloadAuthorizer`, etc.). Add dedicated tests so logic can evolve without starting the full stack.
- **Integration updates** (schema tweaks, Stripe wiring, etc.) belong in the service touching that system. Keep SQL/SDK usage localized so domain services can be reasoned about in isolation.
- **Cross-cutting enhancements** (logging, tracing, feature flags) should hook through middleware or helper seams so they do not leak into core domain logic.

Future contributors can read this file alongside the service implementations to quickly identify where to add logic (handlers vs. services vs. adapters) without re-entangling responsibilities.
