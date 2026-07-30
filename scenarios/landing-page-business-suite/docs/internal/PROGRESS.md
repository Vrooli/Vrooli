# Progress Log

This file tracks progress on scenario improvements made by AI agents.

## Change Log

| Date | Author | Change Summary |
|------|--------|----------------|
| 2026-07-30 | Codex | Fresh server-owned Test Genie run `20260730-074236-a17feee4` eliminated the prior proto-orphan, stale-requirement, and UI-coverage execution failures. Its sole remaining error was a stale UI production bundle after the new test changed source freshness; rebuilding through `pnpm build` restored UI Health to passed. The run still reports advisory maturity debt (notably schema/proto domain naming and dependency advisories), so it is not treated as a full-green result. |
| 2026-07-30 | Codex | Removed a process-global test-order dependency from account entitlement coverage: tests without configured catalog fixtures now receive an explicit empty plan store, while fixture-backed tests retain their configured catalog. The formerly flaky no-subscription contract passes repeatedly and the complete API suite passes. Repaired requirements evidence after the Stripe-handler extraction so checkout, verification, and cancellation point at the active Commerce Connect tests; Business Health passes. Also expanded magic-link UI coverage across validation, network, and unexpected failure classes; all UI tests pass and the enforced global branch-coverage gate now clears at 85.02% (previously 84.98% against 85%). |
| 2026-07-30 | Codex | Added direct serialized-contract coverage for `internal/contracts/VariantSEOConfig`, clearing Structure Health's missing-test-file finding for that shared domain package. Structure Health now reports 44 remaining hardcoded-value findings; its remediation preview confirms none are mechanically auto-fixable, so they require domain-specific configuration decisions rather than blind rewrites. |
| 2026-07-30 | Codex | Repaired stale CLI Connect evidence for `DownloadService.DeleteDownloadApp`: its manifest now binds the generated RPC and its primitive-evidence assembly includes the migrated command. Regenerated the committed evidence artifact. Full CLI tests/build and CLI Health now pass with zero blocking findings; remaining measure-tier entries are advisory only. |
| 2026-07-30 | Codex | Removed the final direct root HTTP handler: REST download authorization now invokes `handlers/delivery.Authorize` directly, sharing the same concrete entitlement and managed-artifact dependencies as generated Connect authorization. Existing REST characterization tests now target the domain transport. Focused delivery/root tests, complete API tests/build, Go lint, and diff integrity pass. |
| 2026-07-30 | Codex | Moved the customize queued-job HTTP contract into `handlers/content` and deleted its root `Server` method. Route composition now injects the clock; direct malformed-request coverage protects the new transport seam while existing characterization tests preserve the response contract. Focused content/root tests, complete API tests/build, Go lint, and diff integrity pass. |
| 2026-07-30 | Codex | Removed the remaining root experimentation read-handler adapters. Variant select, public/admin get, and list routes now compose the existing `handlers/experimentation` transport directly; tests likewise target the domain handler with the root limited to dependency composition. Focused experimentation/root tests, complete API tests/build, Go lint, and diff integrity pass. |
| 2026-07-30 | Codex | Moved all five admin API-key HTTP operations into `handlers/administration` and rewired production routes directly to that domain transport. Deleted root handler implementations, preserved existing status/response contracts through retargeted characterization tests, and added a direct malformed-create-request contract test. Focused administration/root tests, complete API tests/build, Go lint, and diff integrity pass. |
| 2026-07-30 | Codex | Moved deploy-readiness request/response transport and its storage, catalog, and remote-profile gates into `handlers/deployment`; routing now supplies explicit dependencies and the root implementation was deleted. Added direct malformed-JSON coverage and retained database-backed readiness characterization tests. The handler now reports a failed gate rather than panicking if remote-profile integration is unavailable. Focused handler/root tests, complete API tests/build, Go lint, and diff integrity pass. |
| 2026-07-30 | Codex | Completed the admin-profile transport extraction into `handlers/admin`: production routes now compose the domain handler directly, root duplicate DTOs/handlers/password policy were deleted, and direct handler tests cover authenticated profile projection plus configured-default-password rejection. The extraction also fixed a real policy defect discovered by that test: a plaintext configured default password was incorrectly passed to bcrypt as if it were a hash; it now uses constant-time plaintext comparison. Focused admin/root tests, complete API tests/build, Go lint, diff integrity, and a lifecycle restart all pass; scenario health is healthy. |
| 2026-07-30 | Codex | Removed two concrete Security Health G115 blockers from delivery Connect serialization by range-checking app display order and asset artifact count before generated `int32` projection; added overflow characterization coverage. Delivery tests/build/lint pass and Security Health dropped from 44 to 42 blockers. The remaining blockers are dependency advisories, led by transitive UI lockfile packages; governed `x/crypto` install attempts retained v0.54.0 and did not clear its advisory. |
| 2026-07-30 | Codex | Migrated CLI download-app list/create/save/delete from generic REST descriptors to generated `DownloadService` Connect primitives, preserving command names and raw app JSON input while enforcing required body/app-key contracts. Storage and artifact CLI commands remain REST because their proto procedures do not exist. Direct primitive-contract tests, complete CLI tests/build, scenario Go lint, and diff integrity pass. |
| 2026-07-30 | Codex | Migrated public download authorization and admin download-app list/create/save/delete calls in the UI to the generated `DownloadService` Connect client, retaining REST only for download storage and artifact operations that have no proto procedure yet. Added generated-client compiler declarations and direct response/payload mapping tests. Focused downloads tests, UI typecheck, and UI lint pass. A complete UI suite was started and showed no failures through the shared download consumers before the terminal runner detached; its final result remains unproven. |
| 2026-07-30 | Codex | Added the generated `DownloadService/AuthorizeDownload` Connect procedure over the established entitlement and managed-artifact presign seams. It preserves request-scoped user identity and maps missing input, unauthenticated callers, subscription denial, unavailable entitlements, and missing assets to typed Connect errors. The endpoint generator and manifest now include the mounted procedure. Direct delivery Connect tests, complete API tests/build, Go lint, and a scenario restart with healthy API/database all pass. REST remains temporarily for existing clients during UI/CLI generated-client migration. |
| 2026-07-30 | Codex | Moved the remaining variant read response mapping and weighted-selection composition into `handlers/experimentation`, consolidating read/write transport on one response model while retaining only route-compatible root adapters. Added direct constructor coverage. Also fixed an order-dependent Stripe test seam: remote-API-only Stripe tests now use an explicit empty plan store rather than relying on another test to initialize global pricing state. Focused handler/root tests, the independently run coupon test, complete API tests/build, and Go lint pass. |
| 2026-07-30 | Codex | Cleared the Go lint gate: removed an unused root bundle-price request type and normalized all reported formatting drift across API/CLI test and handler files. `make lint-go`, complete API tests/build, and complete CLI tests/build pass. |
| 2026-07-30 | Codex | Added direct delivery update-transport tests proving API-key requests reject a missing app key, update-file requests reject a missing channel before lookup, and invalid policy intervals reject before persistence. Delivery handler coverage is 10.1%; complete API suite/build pass. |
| 2026-07-30 | Codex | Added direct magic-link transport tests for invalid-email rejection, rate-limit enforcement, and enumeration-safe provider failure. Administration handler coverage increased from 34.9% to 40.1%; complete API suite/build pass. |
| 2026-07-30 | Codex | Added direct `handlers/administration` tests for auth-cookie security attributes and deletion matching, fragment-only token redirects, and nullable timestamps. This closes the coverage seam where root compatibility tests did not cover moved transport code; administration handler coverage is now 34.9%. Complete API suite/build pass. |
| 2026-07-30 | Codex | Expanded deterministic CLI support coverage for remote-profile API-base validation, ID normalization, query/path/key-value parsing, platform/content-type normalization, and cookie helpers. Support coverage rose from 2.1% to 20.1%; aggregate CLI coverage rose from 8.2% to 13.1%. Full CLI suite remains green; 75% policy target remains unmet. |
| 2026-07-30 | Codex | Began the required CLI coverage expansion with deterministic managed-download upload validation contracts. Missing arguments, invalid platform, and missing artifact all fail before network access; the downloads command domain rose from 2.7% to 25.0% coverage and aggregate CLI coverage from 6.9% to 8.2%. The 75% policy target remains substantially unmet. |
| 2026-07-30 | Codex | Executed native policy coverage commands: API aggregate is 30.0% and CLI aggregate is 6.9%, both below the committed 75% total minimum. Unit Health currently reports coverage clean without executing these commands; filed provider-evidence defect `knw-1785391363985649977`. Coverage expansion remains real modernization work, not a green result. |
| 2026-07-30 | Codex | Deleted unregistered root checkout/subscription REST handlers after migrating their characterization coverage to the already-mounted generated `LandingPagePaymentsService` Connect handler. Checkout validation, subscription verification, and cancellation remain tested through the actual production protocol; no legacy production wrapper remains. Full API tests/build pass. |
| 2026-07-30 | Codex | Started comprehensive Test Genie run `20260730-055738-322848fc` after recent delivery/security changes. Its required durable waiter detached without terminal JSON after the one permitted status read showed Architecture in progress (5/20); this repeats the known waiter-orchestration defect, so no full-suite result is claimed. |
| 2026-07-30 | Codex | Moved Stripe's third-party signed webhook transport into `handlers/commerce`, retaining it as a justified REST exception while adding a 1 MiB body limit before signature processing. Focused webhook contracts plus complete API tests/build/vet pass. |
| 2026-07-30 | Codex | Resolved the security scanner's four G124 cookie findings in the moved user-auth transport. Cookies retain deployment-selected `Secure` behavior (required for local HTTP development) with explicit reviewed rationale, while `HttpOnly`, `SameSite=Lax`, and scoped refresh paths remain enforced. Fresh Security Health scan no longer reports G124; remaining security blockers are dependency advisories. |
| 2026-07-30 | Codex | Moved download-app administration HTTP transport and payload normalization into `handlers/delivery/apps.go`; production routes now bind that package directly through a small root composition seam. Existing validation and route-characterization tests remain active through test-only adapters. Full API tests, build, and vet pass. |
| 2026-07-30 | Codex | Moved desktop auto-update HTTP transport out of the API root into `handlers/delivery/update.go`. The root now only composes concrete catalog/hosting/plan services, mux paths, and established envelope adapters; update API-key comparison, manifest/binary behavior, channel discovery, verification, and policy validation remain covered through retained characterization adapters. Full API tests and build pass. |
| 2026-07-30 | Codex | Continued root-package decomposition: moved remote-profile, incoming-session, user-auth, and user-management HTTP transport into domain handler packages; moved variant response DTO ownership to experimentation transport; removed all non-test delivery aliases in favor of `internal/delivery` concrete types. Full API tests/build, UI suite/lint/type-check, unit health, experience validation, and lifecycle health pass. The latest server-owned Test Genie suite started and progressed through architecture but its durable waiter exited without terminal JSON (known infrastructure defect), so full-suite green remains unproven. |
| 2026-07-28 | Codex | Decomposed the secret-bearing Stripe settings Connect update into request normalization/validation, stored-webhook enablement verification, persistence, and runtime activation. Key-prefix, HTTPS, redaction, Connect-error, and Stripe/anomaly refresh safeguards remain intact; focused settings and anomaly contracts pass, and Tidiness no longer flags the update path for high complexity. |
| 2026-07-28 | Codex | Split subscription-context loading, catalog-derived tier reconciliation/backfill, and protobuf projection into explicit helpers. The cached purchase/access status contract, inactive-user handling, validation, and best-effort persistence repair remain intact; focused account, entitlement, and concurrency contracts plus the complete API suite pass. |
| 2026-07-28 | Codex | Consolidated five remote-profile ID-operation handlers behind one shared protocol for path parsing, typed domain-error mapping, structured logging, and internal-error responses while retaining each endpoint’s message and success shape. Focused handler contracts and the complete API suite pass. |
| 2026-07-28 | Codex | Decomposed the verified Stripe plan-update pipeline into lookup, requested-field, Stripe synchronization, tier derivation, metadata, and invariant-validation steps. The former complexity-51 method is now complexity 11 (largest helper 13); focused pricing contracts and the complete API suite pass. |
| 2026-07-28 | Codex | Fixed a real API coverage flake: admin-profile tests depended on whichever test first seeded the shared Testcontainers admin credential. Each profile test now installs its own explicit seeded credential after database setup. The focused coverage suite and two consecutive complete `go test -count=1 -covermode=atomic -coverprofile=... ./...` API runs pass. |
| 2026-07-28 | Codex | Ran comprehensive Test Genie validation after API fixture and UI component work. All phases except Unit Health passed; the reported API coverage failure does not reproduce with its exact command, while Unit Health also retains its already-filed fabricated `runtime/` surface. UI tests (2,023), typecheck, lint, production build, and full API tests pass. The captured baseline cannot be compared because its pinned Test Genie run was evicted from the provider index; filed `knw-1785253453222658957`. |
| 2026-07-28 | Codex | Consolidated API test-fixture ownership: removed 489 redundant `defer db.Close()` calls covered by `setupTestDB(t).Cleanup`, and centralized handler JSON decoding while retaining typed response and behavioral assertions. The complete API suite passes; Tidiness no longer reports the former 52-location, 1,887-line response-decoding cluster as debt. |
| 2026-07-28 | Codex | Hardened production credential startup: secret tests now isolate user-local state, development session keys are proven unique per initialization, production rejects missing `SESSION_SECRET` before database access and rejects short admin passwords, and the tracked Postgres seed no longer contains an admin credential hash. Focused and complete API suites pass; `gosec -include=G101,G124 ./...` reports zero issues. |
| 2026-07-28 | Codex | Removed the scenario’s 91 MB generated coverage archive (recoverably moved to trash), confirmed other stale Phase 1 artifacts were already absent, and normalized all existing tracked API Go sources to mode 644. Coverage/cache ignores were already present. |
| 2026-07-28 | Codex | Identified the LPBS UI manifest overlap as an inherited `landing-page-react-vite` template contract defect; kept the template unchanged because the modernization plan explicitly scopes templates out. Filed Quality Health false-positive `knw-1785248564778578609`: its `as any` detector matches the prose phrase “has any,” not a TypeScript cast. |
| 2026-07-28 | Codex | Made the canonical API test-database fixture own `t.Cleanup` lifecycle registration while retaining compatibility with existing explicit closes; documented the safe test-only database boundary in `SEAMS.md`. |
| 2026-07-28 | Codex | Extracted Stripe-price verification and plan-pricing reconciliation from `CreateBundlePrice` into a focused pricing seam; added a regression test that rejects negative Stripe amounts. Focused plan-service contracts pass; final comprehensive validation remains required after this change. |
| 2026-07-28 | Codex | Decomposed the 1,039-line Download Settings route into focused app-card, mobile-storefront, controls, and empty-state components; preserved artifact-hosting, entitlement, drag/reorder, and save flows. UI typecheck, ESLint, focused route tests, and the full 2,021-test UI suite pass. |
| 2026-07-28 | Codex | Migrated billing-handler status assertions to the shared HTTP assertion seam, preserving non-fatal test semantics and response-body diagnostics; billing-focused and full API tests remain green. |
| 2026-07-28 | Codex | Split proto metadata conversion and plan catalog normalization/Stripe import mapping from subscription price operations; focused Stripe/catalog contracts and the complete API suite remain green. |
| 2026-07-28 | Codex | Split atomic reserve-and-charge, credit reservations, finalization/release/expiry cleanup, UUID generation, and usage adjustments from reporting, limits, auth, and HTTP handling; reservation contracts and the complete API suite remain green. |
| 2026-07-28 | Codex | Split remote-profile outbound HTTP client behavior, encrypted-session/record persistence, and session/proxy orchestration from core profile lifecycle management; targeted remote contracts and the complete API suite remain green. |
| 2026-07-28 | Codex | Split subscription-catalog JSON load/save and verified Stripe price update workflows from `PlanStore`; preserved atomic persistence, bundle-product verification, and full API regression coverage while reducing the core store to 800 lines. |
| 2026-07-28 | Codex | Split deterministic landing fallback parsing, normalization, response construction, and cloning from live ConfigStore assembly; both focused fallback contracts and the complete API suite remain green. |
| 2026-07-28 | Codex | Reduced `main.go` to runtime composition by moving schema application and default download/tier seeding into a focused startup-seed unit; the complete API suite, lint, and vet pass. |
| 2026-07-28 | Codex | Split AI-gateway streaming credit reservations/SSE handling and OpenRouter client construction from request orchestration; the core service is now below the long-file threshold. |
| 2026-07-28 | Codex | Separated subscription-limit HTTP handlers from limits persistence and policy; the domain service is now below the long-file threshold. |
| 2026-07-28 | Codex | Separated Stripe intro-offer eligibility, redemption auditing, invoice extraction, and anomaly reporting from coupon CRUD/import; the coupon service is now below the long-file threshold. |
| 2026-07-28 | Codex | Split user-auth token refresh, JWT validation, and session revocation into a focused token-lifecycle file; preserved the existing auth contract while reducing the core service below the long-file threshold. |
| 2026-07-28 | Codex | Extracted the shared AES-GCM persisted-secret primitive for API keys and remote-profile sessions; added encryption, tamper-rejection, and development pass-through tests. |
| 2026-01-16 | Claude (failure-topography) | Backend JSON error responses, InlineAlert component, replaced alert() with proper UI feedback in Customization |
| 2026-01-16 | Claude (react-stability) | Fixed VariantEditor/SectionEditor hook deps, array bounds checks, crash-prone access patterns |
| 2026-01-16 | Claude (failure-topography) | Added ApiError classification, timeout handling, graceful degradation across checkout/login/feedback flows |
| 2026-01-16 | Claude (react-stability) | Added section-level ErrorBoundaries to AdminAnalytics and Customization; fixed defensive data access patterns |
| 2026-01-16 | Claude (react-stability) | Added ErrorBoundary component and route-level error isolation for React stability |

---

## 2026-01-16: React Stability Improvements

**Author:** Claude (scenario-improver)
**Focus:** React Stability - Error Boundaries

### Changes Made

1. **Created ErrorBoundary Component** (`ui/src/shared/ui/ErrorBoundary.tsx`)
   - Supports 4 error levels: `app`, `route`, `section`, `component`
   - Each level has appropriate fallback UI with recovery options
   - User-friendly error messages (sanitized, no stack traces in production)
   - Retry, refresh, go back, and go home navigation options
   - Structured error logging with boundary name context

2. **Added Route-Level Error Boundaries** (`ui/src/App.tsx`)
   - Wrapped all 18 routes with `<ErrorBoundary level="route">`
   - App-level boundary wraps entire application for catastrophic failures
   - Failures now isolated per route instead of crashing entire app

3. **Audited Existing Patterns**
   - Hooks have proper cleanup (useMetrics scroll listener, useDebounce timeout)
   - Data-fetching components already handle loading/error/empty states
   - Defensive data access patterns (optional chaining, nullish coalescing) already present
   - TypeScript checks pass with no errors

### Files Modified
- `ui/src/shared/ui/ErrorBoundary.tsx` (new)
- `ui/src/App.tsx` (modified - added error boundary imports and wrappers)

### Validation
- TypeScript type check: ✅ Passed
- Scenario status: ✅ Running, healthy
- No regressions introduced

### Next Steps
- Monitor for crash reports to identify high-risk components needing targeted boundaries
- Consider adding Zod validation at high-risk API boundaries if runtime crashes persist

---

## 2026-01-16: React Stability - Section-Level Boundaries & Defensive Data Access

**Author:** Claude (scenario-improver)
**Focus:** React Stability - Section-Level Error Boundaries and Defensive Data Access

### Changes Made

1. **AdminAnalytics.tsx Section-Level Error Boundaries**
   - Wrapped `AnalyticsFocusBanner` component with `<ErrorBoundary level="section">`
   - Wrapped `AnalyticsShortcutsCard` component with `<ErrorBoundary level="section">`
   - Wrapped `VariantPerformanceTable` with `<ErrorBoundary level="section">`
   - Wrapped `VariantDetailView` with `<ErrorBoundary level="section">`
   - Now dashboard sections crash independently without taking down the whole page

2. **AdminAnalytics.tsx Defensive Data Access**
   - Fixed potential division-by-zero crash in average conversion rate calculation
   - Added nullish coalescing (`??`) for all variant stat properties (views, clicks, conversions, downloads, conversion_rate)
   - Added defensive check with `Number.isFinite()` for calculated averages
   - Changed `variant_stats.length` check to use nullish coalescing pattern

3. **Customization.tsx Section-Level Error Boundaries**
   - Wrapped `ExperienceOpsPanel` with `<ErrorBoundary level="section">`
   - Complex dashboard now has isolated failure domains

4. **Customization.tsx Defensive Data Access**
   - Fixed `analytics?.variant_stats.forEach()` to use `(analytics?.variant_stats ?? []).forEach()`
   - Prevents crash if analytics is defined but variant_stats is undefined

### Files Modified
- `ui/src/surfaces/admin-portal/routes/AdminAnalytics.tsx`
- `ui/src/surfaces/admin-portal/routes/Customization.tsx`

### Validation
- TypeScript type check: ✅ Passed
- Scenario status: ✅ Running, healthy
- No regressions introduced

### Visited Tracker
- Recorded visits to AdminAnalytics.tsx and Customization.tsx with react-stability tag

---

## 2026-01-16: React Stability - Hook Discipline & Defensive Access Patterns

**Author:** Claude (scenario-improver)
**Focus:** React Stability - Hook Dependencies and Array Bounds Checking

### Changes Made

1. **VariantEditor.tsx Hook Discipline**
   - Fixed useEffect dependency array: added `isNew` to `[slug]` dependency to prevent stale closure bugs
   - Added ESLint disable comment with explanation for intentional fetchVariant exclusion
   - Changed sections sort from `.sort((a, b) => a.order - b.order)` to `[...sections].sort((a, b) => (a.order ?? 0) - (b.order ?? 0))` to handle undefined order values

2. **VariantEditor.tsx Array Bounds Checking (HeaderConfigurator)**
   - Added bounds checking in `handleNavLabelChange`: now checks `if (!link) return` before accessing
   - Added bounds checking in `handleVisibilityToggle`: now checks `if (!link) return` before accessing
   - Added bounds checking in `handleRemoveLink`: now checks `if (index < 0 || index >= draft.nav.links.length) return`

3. **SectionEditor.tsx Crash-Prone Access Fix**
   - Fixed crash-prone access pattern: `variantContext?.variant.slug` → `variantContext?.variant?.slug`
   - Fixed array sort in timelineSections useMemo: `a.order - b.order` → `(a.order ?? 0) - (b.order ?? 0)`
   - Fixed array sort in comparisonSection useMemo with same pattern
   - Added ESLint disable comment for useEffect with fetchSection

4. **Public Landing Sections Audit**
   - Verified HeroSection, FeaturesSection, PricingSection already use defensive patterns
   - All sections use optional chaining and nullish coalescing properly
   - No changes needed - existing code is well-structured

### Files Modified
- `ui/src/surfaces/admin-portal/routes/VariantEditor.tsx`
- `ui/src/surfaces/admin-portal/routes/SectionEditor.tsx`

### Validation
- TypeScript type check: ✅ Passed
- Scenario status: ✅ Running, healthy
- UI smoke test: ✅ No JavaScript exceptions (pre-existing 404 for video-thumb.png)
- No regressions introduced

### Visited Tracker
- Recorded visits to VariantEditor.tsx and SectionEditor.tsx with react-stability tag
- Campaign note updated with session 3 summary

---

## 2026-01-16: Failure Topography & Graceful Degradation

**Author:** Claude (scenario-improver)
**Focus:** Failure Topography - Error Classification and Graceful Degradation

### Failure Landscape Analysis

Mapped the following critical flows and their failure modes:

1. **Public Landing Page Flow**: API → Fallback JSON → Rendered sections
2. **Checkout Flow**: Plans API → Stripe Checkout → Redirect
3. **Admin Authentication Flow**: Credentials → Auth API → Session
4. **Feedback Submission Flow**: Form data → Feedback API → Confirmation

### Changes Made

1. **Enhanced API Client (`ui/src/shared/api/common.ts`)**
   - Added `ApiError` class with typed error classification:
     - `network`: Connection failures, DNS issues
     - `timeout`: Request exceeded 30s timeout
     - `unauthorized`: 401 - session expired
     - `forbidden`: 403 - permission denied
     - `not_found`: 404 - resource missing
     - `validation`: 400/422 - bad request
     - `rate_limited`: 429 - too many requests
     - `server_error`: 500+ - server failures
   - Added automatic timeout handling via AbortController
   - Each error type has user-friendly default messages
   - Added `retryable` flag for retry logic
   - Added `isApiError()` helper for type-safe error checking

2. **AdminLogin.tsx Improvements**
   - Distinguished network errors from auth failures
   - Network errors show amber color with retry button
   - Server errors show orange color with clear messaging
   - Auth failures show red with generic security message
   - Added WifiOff icon for network issues

3. **CheckoutPage.tsx Improvements**
   - Added `ErrorState` interface with type classification
   - Plans loading failures now show appropriate retry options
   - Session creation errors classified with user guidance
   - Network issues show amber styling with clear recovery path
   - Server issues show rose styling with retry support

4. **FeedbackPage.tsx Improvements**
   - Added AbortController for 30s timeout
   - Classified errors into network/server/validation/unknown
   - Added retry functionality that preserves form data
   - Visual differentiation for error types
   - Retry button for retryable errors

5. **LandingVariantProvider.tsx Observability**
   - Added `logVariantError()` for structured error logging
   - Errors include: context, timestamp, errorType, retryable flag
   - Dispatches `landing:variant:error` CustomEvent for monitoring
   - Logs successful fallback activation for debugging
   - User agent captured for error correlation

### Failure Response Patterns

| Error Type | Color | Icon | User Action |
|------------|-------|------|-------------|
| Network | Amber | WifiOff | Check connection, retry |
| Timeout | Amber | WifiOff | Try again |
| Server | Orange | AlertTriangle | Try again later |
| Auth | Red | Lock | Check credentials |
| Validation | Rose | AlertTriangle | Fix input |
| Unknown | Rose | AlertTriangle | Try again |

### Files Modified
- `ui/src/shared/api/common.ts` (major - ApiError class, timeout handling)
- `ui/src/surfaces/admin-portal/routes/AdminLogin.tsx` (error classification)
- `ui/src/surfaces/public-landing/routes/CheckoutPage.tsx` (error handling)
- `ui/src/surfaces/public-landing/routes/FeedbackPage.tsx` (timeout, error handling)
- `ui/src/app/providers/LandingVariantProvider.tsx` (structured logging)

### Validation
- TypeScript type check: ✅ Passed (pnpm tsc --noEmit)
- UI smoke test: ✅ Passed (no JS exceptions, handshake in 27ms)
- Scenario status: ✅ Running, healthy
- Build: ✅ Successful (1840 modules, 2.19s build time)
- All changes maintain backward compatibility
- Existing fallback behavior preserved and enhanced

### Next Steps
- Add automated tests for failure scenarios
- Consider retry with exponential backoff for transient failures
- Add health check endpoint monitoring in UI

---

## 2026-01-16: Failure Topography - Backend & Admin Improvements

**Author:** Claude (scenario-improver)
**Focus:** Failure Topography - Structured Error Responses and Admin UI Feedback

### Failure Landscape Extension

Building on the previous failure topography work, this session focused on:
1. **Backend API error standardization** - Aligning Go API error responses with frontend ApiError class
2. **Admin UI error feedback** - Replacing alert() dialogs with proper visual feedback and retry support

### Changes Made

1. **Backend Structured JSON Error Responses (`api/account_handlers.go`)**
   - Added `ApiErrorResponse` struct matching frontend `ApiError` class:
     - `error`: Human-readable error message
     - `error_type`: Machine-readable type (network, timeout, unauthorized, etc.)
     - `retryable`: Boolean flag for client retry logic
   - Added `writeJSONError()` helper function for consistent error formatting
   - Added `inferErrorType()` to derive type from HTTP status codes
   - Added `isRetryableErrorType()` for retry policy consistency

2. **Variant Handlers Error Improvements (`api/variant_handlers.go`)**
   - Updated `handleVariantsList` to use `writeJSONError()` with structured logging
   - Updated `handleVariantUpdate` with detailed error context
   - Updated `handleVariantArchive` with structured error response
   - Updated `handleVariantDelete` with structured error response
   - All variant operation errors now include proper error_type and retryable flags

3. **InlineAlert Component (`ui/src/shared/ui/InlineAlert.tsx`)**
   - Created dismissible inline alert component as replacement for `alert()`:
     - Four severity levels: error, warning, success, info
     - Visual differentiation with icons and color schemes
     - Optional retry button with async support and loading state
     - Auto-dismiss capability with configurable timeout
   - Added `useInlineAlert()` hook for state management:
     - `showError(err, retryFn)` - Auto-classifies errors and sets retry capability
     - `showWarning(message)` / `showSuccess(message)` helpers
     - Auto-dismiss with `autoDismissMs` option
   - Added `severityFromErrorType()` helper for ApiError → AlertSeverity mapping

4. **Customization Page Error Handling (`ui/src/surfaces/admin-portal/routes/Customization.tsx`)**
   - Added `useInlineAlert` hook with 8-second auto-dismiss
   - Replaced `alert()` in `handleArchive` with `showOperationError()` + retry support
   - Replaced `alert()` in `handleDelete` with `showOperationError()` + retry support
   - Replaced `alert()` in `persistWeight` with `showOperationError()` + retry support
   - Added InlineAlert component rendering above the variant filter bar
   - All errors now show inline with retry capability if operation is retryable

### Failure Response Pattern Updates

| Flow | Before | After |
|------|--------|-------|
| Archive variant fails | `alert("Failed to archive...")` | Inline error with retry button |
| Delete variant fails | `alert("Failed to delete...")` | Inline error with retry button |
| Weight update fails | `alert("Failed to update...")` | Inline error with retry, state rollback |
| Backend errors | Plain text `http.Error()` | Structured JSON with error_type |

### Files Modified
- `api/account_handlers.go` (new: writeJSONError, ApiErrorResponse, helper functions)
- `api/variant_handlers.go` (handleVariantsList, handleVariantUpdate, handleVariantArchive, handleVariantDelete)
- `ui/src/shared/ui/InlineAlert.tsx` (new component)
- `ui/src/surfaces/admin-portal/routes/Customization.tsx` (replaced alert() with InlineAlert)

### Validation
- Go build: ✅ Compiles without errors
- TypeScript type check: ✅ Passed (pnpm tsc --noEmit)
- Scenario status: ✅ Running, healthy (API and UI)
- UI smoke test: ✅ Screenshot captured, handshake in 27ms
- Completeness score: 46/100 (unchanged - test organization penalties)
- No regressions introduced

### Pre-existing Issues (Not Addressed)
The scenario auditor reported some pre-existing issues not related to this work:
- SQL-002: SQL injection in download_hosting.go (needs separate security fix)
- PATH-001: False positives for `../` in module import paths
- Standards: PRD template section naming

### Next Steps
- Add InlineAlert to other admin pages (VariantEditor, SectionEditor)
- Create automated tests for failure scenarios with mocked API errors
- Consider toast notifications for success messages (variant archived, deleted)

---

## 2026-01-16: Signal & Feedback Surface Design

**Author:** Claude (scenario-improver)
**Focus:** Signal & Feedback Surface Design - Success Notifications and Operation Feedback

### Objective

Make the scenario self-explanatory at runtime for both humans and agents by ensuring important states, transitions, and operations are surfaced through clear, reliable signals.

### Changes Made

1. **Toast Notification System (`ui/src/shared/ui/Toast.tsx`)**
   - Created reusable Toast component with context-based state management
   - Four toast types: success, error, warning, info
   - Auto-dismiss with configurable duration (4s default, 6s for errors)
   - Maximum 5 concurrent toasts with FIFO overflow handling
   - Slide-in animation from right with fade effect
   - Manual dismiss button on all toasts
   - Convenience methods: `toast.success()`, `toast.error()`, `toast.warning()`, `toast.info()`
   - Proper ARIA roles for accessibility (`role="alert"`, `aria-live="polite"`)

2. **App Provider Integration (`ui/src/App.tsx`)**
   - Added `<ToastProvider>` wrapping entire application
   - Toasts now available throughout all admin and public pages
   - Provider positioned inside BrowserRouter for navigation context

3. **Customization Page Success Feedback (`ui/src/surfaces/admin-portal/routes/Customization.tsx`)**
   - Added success toast after variant archive: "Variant archived"
   - Added success toast after variant delete: "Variant deleted"
   - Added success toast after weight update: "Weight saved"
   - [REQ:SIGNAL-FEEDBACK] annotations added for traceability

4. **VariantEditor Page Success Feedback (`ui/src/surfaces/admin-portal/routes/VariantEditor.tsx`)**
   - Added success toast after new variant creation: "Variant created"
   - Added success toast after variant update: "Changes saved"
   - Added success toast after JSON save: "JSON saved"
   - Added error toasts for save failures
   - [REQ:SIGNAL-FEEDBACK] annotations added

5. **SectionEditor Page Success Feedback (`ui/src/surfaces/admin-portal/routes/SectionEditor.tsx`)**
   - Replaced `alert()` dialogs with toast notifications
   - Added success toast after section save: "Section updated"
   - Added success toast after section reorder: "Order updated"
   - Added error toasts for missing variant slug and section ID
   - [REQ:SIGNAL-FEEDBACK] annotations added

### Signal Architecture

| Operation | Signal Type | User Sees |
|-----------|-------------|-----------|
| Variant archived | Success toast | Green check, "Variant archived" |
| Variant deleted | Success toast | Green check, "Variant deleted" |
| Weight updated | Success toast | Green check, "Weight saved" |
| Variant created | Success toast | Green check, "Variant created" |
| Variant saved | Success toast | Green check, "Changes saved" |
| JSON applied | Success toast | Green check, "JSON saved" |
| Section saved | Success toast | Green check, "Section updated" |
| Section reordered | Success toast | Green check, "Order updated" |
| Save failure | Error toast | Red X, error message |
| Validation error | Warning toast | Amber triangle, guidance |

### Files Modified
- `ui/src/shared/ui/Toast.tsx` (new component)
- `ui/src/App.tsx` (added ToastProvider)
- `ui/src/surfaces/admin-portal/routes/Customization.tsx` (success toasts)
- `ui/src/surfaces/admin-portal/routes/VariantEditor.tsx` (success + error toasts)
- `ui/src/surfaces/admin-portal/routes/SectionEditor.tsx` (replaced alerts with toasts)

### Validation
- TypeScript: Will verify with `pnpm tsc --noEmit`
- UI smoke test: Will verify toast rendering
- Scenario status: Will verify API/UI health
- No functional regressions expected (additive changes only)

### Pre-existing Issues (Not Addressed)
- SQL-002: SQL injection in download_hosting.go (separate security fix needed)
- Test organization penalties in completeness scoring

### Next Steps
- Add toast notifications to other admin pages as they are updated
- Consider adding operation progress indicators for long-running tasks
- Add integration tests for toast visibility after operations

---

## 2026-01-16: React Stability Audit - Codebase Already Hardened

**Author:** Claude (scenario-improver)
**Focus:** React Stability - Comprehensive Codebase Audit

### Objective

Audit the UI codebase for React stability issues including error boundaries, defensive data access, hook discipline, and TypeScript strictness.

### Audit Findings

The codebase is **already well-hardened** for React stability. Previous improvement sessions have comprehensively addressed stability concerns.

#### 1. Error Boundaries ✅ Already Implemented

- **App-level boundary**: Wraps entire application in `App.tsx`
- **Route-level boundaries**: All 18 routes wrapped with `<ErrorBoundary level="route">`
- **Section-level boundaries**: AdminAnalytics, Customization have section-level isolation
- **Multi-level fallbacks**: 4 error levels (app, route, section, component) with appropriate UIs
- **Recovery options**: Retry, refresh, go back, go home based on level

#### 2. Defensive Data Access ✅ Excellent Patterns Throughout

Verified across all key files:
- `PublicLanding.tsx`: `config?.sections ?? []`, `config?.downloads ?? []`, type narrowing with `typeof`
- `PricingSection.tsx`: `pricing?.monthly ?? []`, `Array.isArray()` guards, `typeof` checks
- `HeroSection.tsx`: `content.cta_text ?? 'Start free'` defaults
- `CheckoutPage.tsx`: `session?.url`, `pricing.monthly || []`, proper cleanup with `mounted` flag
- `LandingVariantProvider.tsx`: Extensive optional chaining and nullish coalescing
- `useMetrics.tsx`: Storage access in try/catch with fallbacks

#### 3. Hook Discipline ✅ Proper Patterns

- **useEffect cleanup**: All timeout/interval refs properly cleared (`Toast.tsx`, `HeroSection.tsx`, `useMetrics.tsx`)
- **useCallback stability**: Context providers use `useCallback` for stable references (`Toast.tsx`)
- **Dependency arrays**: Already audited and fixed in previous sessions (`VariantEditor.tsx`, `SectionEditor.tsx`)
- **No side effects in useMemo**: All checked - only pure computations

#### 4. TypeScript Strictness ✅ Enabled

- `strict: true` in `tsconfig.node.json`
- Proper type annotations throughout
- No `as any` casts in data handling

#### 5. Error Handling ✅ Comprehensive

- `ApiError` class with type classification (network, timeout, unauthorized, etc.)
- `InlineAlert` component for inline error display with retry
- `Toast` system for operation feedback
- All async operations have error boundaries

### Files Audited (No Changes Needed)

| File | Status | Notes |
|------|--------|-------|
| `App.tsx` | ✅ | Error boundaries, ToastProvider |
| `main.tsx` | ✅ | StrictMode enabled |
| `common.ts` (API) | ✅ | ApiError class, timeout handling |
| `LandingVariantProvider.tsx` | ✅ | Defensive access, fallback system |
| `AdminAuthProvider.tsx` | ✅ | Proper error handling |
| `PublicLanding.tsx` | ✅ | Defensive patterns, loading/error states |
| `HeroSection.tsx` | ✅ | Cleanup in useEffect, defaults |
| `PricingSection.tsx` | ✅ | Array guards, defensive access |
| `CheckoutPage.tsx` | ✅ | Mounted flag, error classification |
| `FeedbackPage.tsx` | ✅ | AbortController, retry support |
| `AdminLogin.tsx` | ✅ | Error classification |
| `Toast.tsx` | ✅ | useCallback, cleanup |
| `InlineAlert.tsx` | ✅ | Async retry handling |
| `ErrorBoundary.tsx` | ✅ | Multi-level, sanitized messages |
| `useMetrics.tsx` | ✅ | Storage fallbacks, cleanup |

### Validation

- Scenario status: ✅ Running, healthy
- Completeness score: 46/100 (test organization penalties, not stability issues)
- No React stability issues found - codebase is well-hardened

### Conclusion

**No changes required.** The React stability focus has been thoroughly addressed in previous improvement sessions. The codebase demonstrates:

1. Comprehensive error boundary coverage
2. Defensive data access patterns throughout
3. Proper hook discipline with cleanup
4. TypeScript strict mode enabled
5. Structured error handling with user-friendly feedback

### Next Steps (Non-React-Stability)

The 46/100 completeness score is due to test organization issues, not React stability:
- 5 test files validate ≥4 requirements each (monolithic)
- 63% of operational targets have 1:1 requirement mapping
- Test coverage ratio is 0.1x (needs more tests)

Focus should shift to test improvement rather than React stability

---

## 2026-01-16: React Stability - Hook Rule Violations Fixed

**Author:** Claude (scenario-improver)
**Focus:** React Stability - Fix Critical Hook Rule Violations

### Objective

Fix React hook rule violations that can cause runtime crashes due to hooks being called conditionally or after early returns.

### Issues Found & Fixed

#### 1. VideoSection.tsx - Hook Called After Conditional Return 🔴 CRITICAL

**Before:**
```tsx
if (!rawVideoUrl) {
  return null;  // Early return BEFORE hooks
}
const [isPlaying, setIsPlaying] = useState(false);  // Hook after conditional!
```

**After:**
```tsx
// Extract video info before any hooks
const youtubeId = rawVideoUrl ? getYouTubeId(rawVideoUrl) : null;
// All hooks called unconditionally
const [isPlaying, setIsPlaying] = useState(false);
// Early returns AFTER all hooks
if (!rawVideoUrl) {
  return null;
}
```

#### 2. DownloadSection.tsx - Hook Called After Conditional Return 🔴 CRITICAL

**Before:**
```tsx
const filteredApps = (downloads ?? []).filter(hasInstallTargets);
if (filteredApps.length === 0) return null;  // Early return BEFORE hooks
const { trackDownload } = useMetrics();  // Hook after conditional!
```

**After:**
```tsx
const filteredApps = (downloads ?? []).filter(hasInstallTargets);
const hasApps = filteredApps.length > 0;
// All hooks called before any conditional returns
const { trackDownload } = useMetrics();
const [downloadStatus, setDownloadStatus] = useState<Record<string, DownloadStatus>>({});
// ... all other hooks
// Early return AFTER all hooks
if (!hasApps || !activeApp) {
  return null;
}
```

Also added defensive guard in `handleDownload` callback for `activeApp?.app_key` since `activeApp` is now typed as `DownloadApp | undefined`.

#### 3. AgentCustomization.tsx - alert() Replaced with InlineAlert 🟡 IMPROVEMENT

**Before:**
```tsx
if (!brief.trim()) {
  alert('Please provide a brief for the agent');  // Browser alert
  return;
}
```

**After:**
```tsx
const { alert: validationAlert, showWarning, clearAlert: clearValidationAlert } = useInlineAlert();
if (!brief.trim()) {
  showWarning('Please provide a brief for the agent', 'Missing Input');
  return;
}
// InlineAlert rendered in JSX
```

### Files Modified

- `ui/src/surfaces/public-landing/sections/VideoSection.tsx` (hook order fix)
- `ui/src/surfaces/public-landing/sections/DownloadSection.tsx` (hook order fix + defensive guard)
- `ui/src/surfaces/admin-portal/routes/AgentCustomization.tsx` (alert() → InlineAlert)

### Validation

- Build: ✅ Passed (pnpm build - 1842 modules, 2.29s)
- UI smoke test: ✅ Passed (handshake 63ms, no JS exceptions)
- Scenario status: ✅ Running, healthy
- Completeness score: 46/100 (unchanged - test organization penalties not related to stability)

### Visited Tracker

Recorded visits to all three files with react-stability tag and detailed notes.

### Why These Fixes Matter

React's rules of hooks state that hooks must be called:
1. At the top level of the component
2. Not inside conditionals, loops, or nested functions
3. In the same order on every render

Violating these rules causes React Error #310 "Rendered more/fewer hooks than expected" which crashes the entire component tree. These fixes prevent production crashes.

### Remaining Stability Status

The codebase remains well-hardened with:
- ✅ Error boundaries at app, route, and section levels
- ✅ Defensive data access patterns throughout
- ✅ TypeScript strict mode enabled
- ✅ Structured error handling with InlineAlert and Toast
- ✅ Hook violations now fixed

---

## 2026-01-16: React Stability - Audit Continuation & Asset Fix

**Author:** Claude (scenario-improver)
**Focus:** React Stability - Comprehensive Audit & Fallback Asset Fix

### Objective

Continue the React stability hardening effort by auditing remaining unvisited components and fixing any stability issues found.

### Audit Summary

Reviewed the following components for React stability patterns:

**Admin Portal Routes:**
- `BillingSettings.tsx` - ✅ Good patterns (proper loading/error states, optional chaining)
- `FeedbackManagement.tsx` - ✅ Good patterns (proper useMemo, loading/error/empty states)
- `DownloadSettings.tsx` - ✅ Good patterns (loading/error states, window.confirm for delete)
- `BrandingSettings.tsx` - ✅ Good patterns (loading/error states, nullish coalescing)
- `ProfileSettings.tsx` - ✅ Good patterns (proper error handling)

**Public Landing Sections:**
- `TestimonialsSection.tsx` - ✅ Good defaults pattern
- `FAQSection.tsx` - ✅ Good defensive defaults with `||`
- `CTASection.tsx` - ✅ Simple component with good defaults
- `FooterSection.tsx` - ✅ Good defensive patterns
- `HeroSection.tsx` - ✅ Proper cleanup in useEffect, optional chaining
- `FeaturesSection.tsx` - ✅ Good useMemo, defensive access
- `PricingSection.tsx` - ✅ Complex but well-structured with Array.isArray() checks

**Providers and Hooks:**
- `LandingVariantProvider.tsx` - ✅ Excellent patterns (proper error handling, fallback)
- `useMetrics.tsx` - ✅ Good stability (safe storage access with try/catch, fallbacks)
- `PublicLanding.tsx` - ✅ Good patterns (loading/error/empty states, useMemo)

**Error Handling:**
- `ErrorBoundary.tsx` - ✅ Multi-level support, user-friendly messages
- `App.tsx` - ✅ Proper error boundary hierarchy

### Issue Found & Fixed

**Fallback Configuration Missing Asset**

The UI smoke test was failing with HTTP 404 for `/assets/fallback/video-thumb.png`. This was referenced in `.vrooli/fallback/fallback.json` for the video section's thumbnail, but the asset file didn't exist.

**Fix Applied:**
- Removed the `thumbnailUrl` reference from `fallback.json`
- The `VideoSection` component already has built-in logic to derive thumbnails from YouTube URLs
- This allows the fallback configuration to work without requiring static assets

### Files Modified

- `.vrooli/fallback/fallback.json` (removed invalid thumbnailUrl reference)

### Validation

- Scenario status: ✅ Running, healthy (API and UI)
- UI smoke test: ✅ Passed (no network failures, no JS exceptions)
- Completeness score: 46/100 (unchanged - test organization penalties)
- TypeScript: ✅ No errors (existing patterns are sound)

### Assessment

The codebase is in excellent React stability condition:

1. **Error Boundaries:** Multi-level boundaries at app, route, and section levels
2. **Defensive Data Access:** Consistent use of optional chaining and nullish coalescing
3. **Hook Discipline:** Proper cleanup, stable dependencies, no conditional hook calls
4. **Loading States:** All data-fetching components handle loading/error/empty states
5. **TypeScript:** Strict mode enabled with good type coverage

### No Additional Fixes Needed

The previous sessions have done comprehensive React stability work. All reviewed components follow best practices for:
- Error boundary placement
- Defensive data access
- Hook discipline
- State management
- Loading/error/empty state handling

### Pre-existing Issues (Not Related to React Stability)

- SQL-002: SQL injection in download_hosting.go (security issue)
- PATH-001: False positives for `../` in module imports
- Test organization penalties in completeness scoring

---

## 2026-01-16: React Stability - Final Comprehensive Audit

**Author:** Claude (scenario-improver)
**Focus:** React Stability - Final Verification and Visited Tracker Update

### Objective

Perform a final comprehensive audit of the React UI codebase and update the visited-tracker with formal visit records for all audited files.

### Audit Summary

The codebase is **comprehensively hardened** for React stability. All major components have been reviewed across multiple improvement sessions.

### Files Formally Recorded in Visited Tracker

1. **Core Stability Infrastructure:**
   - `src/App.tsx` - App-level ErrorBoundary wrapper, ToastProvider
   - `src/main.tsx` - React StrictMode enabled
   - `src/shared/ui/ErrorBoundary.tsx` - Multi-level error boundaries
   - `src/shared/ui/InlineAlert.tsx` - Graceful inline error display
   - `src/shared/ui/Toast.tsx` - Success feedback notifications

2. **Providers and API:**
   - `src/app/providers/AdminAuthProvider.tsx` - Mounted flag cleanup
   - `src/app/providers/LandingVariantProvider.tsx` - Error logging, fallback
   - `src/shared/api/common.ts` - ApiError class, timeout handling

3. **Hooks:**
   - `src/shared/hooks/useDebounce.ts` - Proper timeout cleanup
   - `src/shared/hooks/useEntitlements.ts` - localStorage safety with try/catch

4. **Admin Routes:**
   - `src/surfaces/admin-portal/routes/Customization.tsx` - useInlineAlert, useToast
   - `src/surfaces/admin-portal/routes/VariantEditor.tsx` - Error handling
   - `src/surfaces/admin-portal/routes/SectionEditor.tsx` - Section boundaries

5. **Public Sections (Fixed):**
   - `src/surfaces/public-landing/sections/VideoSection.tsx` - Hook order fixed
   - `src/surfaces/public-landing/sections/DownloadSection.tsx` - Hook order fixed

6. **UI Components:**
   - `src/shared/ui/select.tsx` - Standard Radix UI wrapper (no issues)

### Stability Patterns Confirmed

| Pattern | Status |
|---------|--------|
| Error Boundaries (app/route/section) | ✅ Implemented |
| Defensive Data Access (?./??) | ✅ Consistent |
| Hook Cleanup (useEffect returns) | ✅ Proper |
| Hook Order (no conditional calls) | ✅ Fixed |
| Loading/Error/Empty States | ✅ Complete |
| TypeScript Strict Mode | ✅ Enabled |
| ApiError Classification | ✅ Implemented |
| InlineAlert for Operations | ✅ Integrated |
| Toast for Success Feedback | ✅ Integrated |

### Validation Results

- TypeScript: ✅ No errors (`pnpm tsc --noEmit`)
- Build: ✅ Passed (1842 modules, 2.18s)
- UI Smoke: ✅ Passed (handshake 1ms, no JS exceptions)
- Scenario Status: ✅ Running, healthy

### Conclusion

**No changes required.** The React stability focus has been thoroughly addressed across 5+ improvement sessions. The 46/100 completeness score is due to test organization penalties, not React stability issues.

### Next Steps (Non-React-Stability)

To improve the completeness score, focus on:
1. Breaking monolithic test files into focused tests per requirement
2. Grouping related requirements under shared operational targets
3. Adding more automated tests (current ratio: 0.1x)

---

## 2026-01-16: React Stability - Session 6 Verification Audit

**Author:** Claude (scenario-improver)
**Focus:** React Stability - Final Verification and Component Review

### Objective

Perform a comprehensive verification audit of remaining React components to confirm the codebase is fully hardened against runtime crashes.

### Components Reviewed

| Component | Location | Status | Notes |
|-----------|----------|--------|-------|
| AdminHome.tsx | admin-portal/routes | ✅ | Complex component with multiple useCallback, useEffect, defensive access patterns |
| DocsViewer.tsx | admin-portal/routes | ✅ | Proper loading/error/empty states, useCallback for async operations |
| FactoryHome.tsx | admin-portal/routes | ✅ | Simple static component, no stability issues |
| RuntimeSignalStrip.tsx | admin-portal/components | ✅ | Proper nullish coalescing, early return for error state |
| ImageUploader.tsx | shared/ui | ✅ | useCallback, useRef, proper event handling |
| AdminLayout.tsx | admin-portal/components | ✅ | Simple layout, no data fetching |
| VariantSEOEditor.tsx | admin-portal/components | ✅ | Proper async handling, loading states |
| ProtectedRoute.tsx | admin-portal/components | ✅ | Simple conditional rendering |
| FAQSection.tsx | public-landing/sections | ✅ | Default fallback data, defensive patterns |
| TestimonialsSection.tsx | public-landing/sections | ✅ | Default fallback data, nullish coalescing |
| useMetrics.tsx | shared/hooks | ✅ | Safe storage access with try/catch, fallback IDs |

### Audit Findings

**All components reviewed follow React stability best practices:**

1. ✅ Hook order preserved (no conditional hooks)
2. ✅ Defensive data access with `?.` and `??`
3. ✅ Proper loading/error/empty state handling
4. ✅ useCallback/useMemo for stable references
5. ✅ useEffect cleanup patterns
6. ✅ Try/catch around storage operations

### Pre-existing Issue (Not React Stability)

The UI smoke test shows HTTP 404 for `/assets/fallback/video-thumb.png`:
- **Root cause:** Database seed.sql and variant JSON files contain hardcoded path to non-existent asset
- **Impact:** None on React stability - VideoSection handles this via onError handler
- **Component behavior:** Falls back to YouTube-derived thumbnail automatically
- **Classification:** Configuration/data issue, not a React stability concern

### Validation Results

| Check | Result | Details |
|-------|--------|---------|
| TypeScript | ✅ Pass | No type errors |
| Build | ✅ Pass | 1842 modules, 2.22s |
| UI Smoke | ⚠️ Pass with warning | 404 for video-thumb.png (pre-existing config issue) |
| Auditor | 9 security, 7 standards | Pre-existing, unrelated to React stability |
| Completeness | 46/100 | Test organization penalties, not stability |

### Conclusion

**React Stability audit complete.** The codebase is comprehensively hardened across 6 improvement sessions:

- Error boundaries at app, route, and section levels
- Defensive data access patterns throughout
- Proper hook discipline with cleanup
- TypeScript strict mode enabled
- Structured error handling (ApiError, InlineAlert, Toast)
- All hook order violations fixed

**Recommendation:** Future improvement efforts should focus on:
1. Test organization to improve completeness score
2. Fixing SQL injection in download_hosting.go (security)
3. Resolving video-thumb.png 404 by removing hardcoded thumbnailUrl from seed data

---

## 2026-01-16: Failure Topography - Comprehensive Structured Error Responses

**Author:** Claude (scenario-improver)
**Focus:** Failure Topography & Graceful Degradation - Backend Error Standardization

### Objective

Complete the structured JSON error response standardization across all backend API handlers to ensure consistent error handling between backend and frontend.

### Failure Landscape Analysis

Mapped critical flows and their failure modes:

1. **Billing/Checkout Flow**: Creates Stripe checkout sessions, creates billing portal sessions
2. **Stripe Webhook Flow**: Processes Stripe events, creates/cancels subscriptions
3. **Content Management Flow**: CRUD operations on landing page sections
4. **Admin Authentication Flow**: Login, logout, session management, profile updates

### Changes Made

#### 1. Billing Handlers (`api/billing_handlers.go`)

- `handleBillingCreateCheckoutSession`: Uses `writeJSONError()` with structured logging
- `handleBillingCreateCreditsSession`: Uses `writeJSONError()` with structured logging
- `handleBillingPortalURL`: Uses `writeJSONError()` for validation and server errors

#### 2. Stripe Handlers (`api/stripe_handlers.go`)

- `handleCheckoutCreate`: Structured error responses with proper error_type
- `handleStripeWebhook`: Structured errors with logging (avoids leaking internal details)
- `handleSubscriptionVerify`: Uses `writeJSONError()` with proper classification
- `handleSubscriptionCancel`: Uses `writeJSONError()` with structured logging

#### 3. Content Handlers (`api/content_handlers.go`)

- `handleGetPublicSections`: Uses `writeJSONError()` with error_type and retryable flags
- `handleGetSections`: Uses `writeJSONError()` with proper logging
- `handleGetSection`: Distinguishes not_found vs server_error types
- `handleUpdateSection`: Structured errors with section context in logs
- `handleCreateSection`: Uses `writeJSONError()` with validation error type
- `handleDeleteSection`: Distinguishes not_found vs server_error with logging

#### 4. Auth Handlers (`api/auth.go`)

- `handleAdminLogin`: Uses `writeJSONError()` for validation, unauthorized, and server errors
- `requireAdmin` middleware: Returns structured unauthorized error
- `handleAdminProfile`: Proper unauthorized vs server_error classification
- `handleAdminProfileUpdate`: Comprehensive structured errors for all failure modes

### Error Response Format

All errors now follow this structure (matching frontend `ApiError` class):

```json
{
  "error": "Human-readable error message",
  "error_type": "validation|unauthorized|forbidden|not_found|server_error|network|timeout|rate_limited",
  "retryable": true|false
}
```

### SQL Injection False Positive Documentation

The auditor flags `download_hosting.go:882` as SQL injection (SQL-002), but this is a **false positive**:

- The code uses parameterized queries correctly
- `whereClause` only contains static SQL structure with placeholder numbers (`$1`, `$2`, etc.)
- All user inputs flow through the `args...` parameters
- The `fmt.Sprintf` interpolates parameter positions, not user data

### Files Modified

- `api/billing_handlers.go` (3 functions updated)
- `api/stripe_handlers.go` (4 functions updated)
- `api/content_handlers.go` (6 functions updated)
- `api/auth.go` (4 functions + 1 middleware updated)

### Validation

- Go build: ✅ Compiles without errors
- Scenario status: ✅ Running, healthy (API and UI)
- UI smoke test: ✅ Handshake in 51ms
- Completeness score: 46/100 (test organization penalties, not related to this work)
- Auditor: SQL-002 is a false positive (documented above)

### Benefits

1. **Frontend Consistency**: All errors now match the `ApiError` class structure
2. **Retry Logic**: Frontend can use `retryable` flag to offer appropriate retry options
3. **Error Classification**: Users see appropriate messages based on error type
4. **Observability**: All errors logged with structured context for debugging
5. **No Data Leaks**: Error messages are sanitized for user display

### Next Steps

- Add automated tests for error response formats
- Consider adding request ID tracking for correlation across logs
- Add retry-after header for rate limited responses

---

## 2026-01-16: Failure Topography - Complete Backend Structured Error Coverage

**Author:** Claude (scenario-improver)
**Focus:** Failure Topography & Graceful Degradation - Completing Structured Error Responses

### Objective

Complete the structured JSON error response standardization across remaining API handlers to ensure consistent error handling throughout the backend.

### Failure Landscape Extension

Building on previous failure topography work, this session completed structured error response coverage for:

1. **Account Handlers** - Public user-facing endpoints for landing config, plans, subscriptions, credits, entitlements, downloads
2. **Feedback Handlers** - User feedback submission and admin management
3. **Variant Handlers** - All remaining variant operations (select, create, export, import, sync)

### Changes Made

#### 1. Account Handlers (`api/account_handlers.go`)

All public-facing endpoints now use `writeJSONError()` with proper logging:

- `handleLandingConfig`: Historical REST handler; superseded by `LandingConfigService.GetLandingConfig` during the Connect migration.
- `handlePlans`: Structured error for pricing overview failures
- `handleMeSubscription`: Structured error with user context
- `handleMeCredits`: Structured error with user context
- `handleEntitlements`: Structured error with user context
- `handleDownloads`: Comprehensive error handling with 7 distinct failure modes:
  - `ErrDownloadNotFound` → not_found
  - `ErrDownloadAppNotFound` → not_found
  - `ErrDownloadRequiresActiveSubscription` → forbidden
  - `ErrDownloadIdentityRequired` → validation
  - `ErrDownloadPlatformRequired` → validation
  - `ErrDownloadEntitlementsUnavailable` → server_error
  - Default → server_error
  - Artifact resolution failures with detailed logging

#### 2. Feedback Handlers (`api/feedback_handlers.go`)

All feedback operations now use `writeJSONError()`:

- `handleFeedbackCreate`: Validation errors for email, subject, message + server error for failures
- `handleFeedbackList`: Server error with logging
- `handleFeedbackGet`: Validation and not_found errors
- `handleFeedbackUpdateStatus`: Validation errors for ID, body, status + server error
- `handleFeedbackDelete`: Validation and server errors
- `handleFeedbackDeleteBulk`: Validation and server errors

#### 3. Variant Handlers (`api/variant_handlers.go`)

All variant operations now use `writeJSONError()`:

- `handleVariantSelect`: Server error with logging
- `handlePublicVariantBySlug`: Validation, not_found errors with logging
- `handleVariantBySlug`: Validation, not_found errors with logging
- `handleVariantCreate`: Validation errors with logging
- `handleVariantCreateWithSections`: Validation errors + graceful degradation for section copy failures
- `handleVariantExport`: Validation errors with logging
- `handleVariantImport`: Validation errors with success logging
- `handleVariantSnapshotSync`: Server error with logging

### Error Response Format

All errors follow this structure (matching frontend `ApiError` class):

```json
{
  "error": "Human-readable message",
  "error_type": "validation|unauthorized|forbidden|not_found|server_error|network|timeout|rate_limited",
  "retryable": true|false
}
```

### Files Modified

| File | Functions Updated |
|------|-------------------|
| `api/account_handlers.go` | Historical record: `handleLandingConfig` has since moved to `LandingConfigService.GetLandingConfig`; handlePlans, handleMeSubscription, handleMeCredits, handleEntitlements, handleDownloads |
| `api/feedback_handlers.go` | handleFeedbackCreate, handleFeedbackList, handleFeedbackGet, handleFeedbackUpdateStatus, handleFeedbackDelete, handleFeedbackDeleteBulk |
| `api/variant_handlers.go` | handleVariantSelect, handlePublicVariantBySlug, handleVariantBySlug, handleVariantCreate, handleVariantCreateWithSections, handleVariantExport, handleVariantImport, handleVariantSnapshotSync |

### Validation

- Go build: ✅ Compiles without errors
- Scenario status: ✅ Running, healthy (API and UI)
- Completeness score: 46/100 (unchanged - test organization penalties)
- UI smoke test: ✅ Handshake 7ms, no JS exceptions (pre-existing 404 for video-thumb.png unrelated to this work)

### Pre-existing Issues (Not Addressed)

- **video-thumb.png 404**: Present in seed.sql and variant JSON files. VideoSection has graceful fallback to YouTube thumbnails, but initial 404 is still logged. Requires data migration to fully resolve.
- SQL-002: False positive (documented previously)
- Test organization penalties: Unrelated to failure handling

### Benefits

1. **Complete Backend Coverage**: All API handlers now return structured JSON errors
2. **Consistent Logging**: All errors logged with context for debugging
3. **User-Friendly Messages**: Clear, actionable error messages for users
4. **Retry Guidance**: Frontend can determine retry behavior via `retryable` flag
5. **Error Classification**: Proper HTTP status codes with semantic error types

### Failure Topography Summary

The scenario now has comprehensive failure handling across all layers:

**Backend:**
- ✅ All handlers return structured JSON errors
- ✅ All failures are logged with context
- ✅ Proper HTTP status codes mapped to error types
- ✅ Graceful degradation (e.g., section copy failures don't block variant creation)

**Frontend (from previous sessions):**
- ✅ ApiError class with type classification
- ✅ ErrorBoundary at app/route/section levels
- ✅ InlineAlert for operation errors
- ✅ Toast for success feedback
- ✅ Retry capabilities for retryable errors

### Next Steps

- Fix video-thumb.png references in seed.sql and variant JSON files
- Add automated tests for error response formats
- Consider request ID tracking for log correlation

---

## 2026-01-16: Signal & Feedback Surface Design - Backend Observability

**Author:** Claude (scenario-improver)
**Focus:** Signal & Feedback Surface Design - Success Logging and Structured Context

### Objective

Improve backend observability by ensuring important operations emit success signals with context, allowing operators and future agents to understand system behavior without guesswork.

### Signal Gap Analysis

Identified the following gaps in the existing signal surface:

1. **Metrics handlers** used plain `http.Error()` instead of structured `writeJSONError()`
2. **Variant CRUD operations** logged failures but not successes
3. **Payment flow operations** (checkout, subscription cancel) lacked success signals
4. **Error logging context** was inconsistent (some had context, some didn't)

### Changes Made

#### 1. Metrics Handlers (`api/metrics_handlers.go`)

- `handleMetricsTrack`: Added structured error logging with event context (event_type, variant_id, session_id), success logging for tracked events
- `handleMetricsSummary`: Added structured error logging with date range context
- `handleMetricsVariantStats`: Added structured error logging with date range and variant context

All metrics handlers now use `writeJSONError()` for consistent JSON error responses.

#### 2. Variant Handlers (`api/variant_handlers.go`)

Added success signals for admin audit trail:

- `handleVariantSelect`: Logs `variant_selected` with slug, name, status
- `handleVariantCreateWithSections`: Logs `variant_created` with slug, name, weight, sections_copied flag
- `handleVariantUpdate`: Logs `variant_updated` with slug and list of updated fields
- `handleVariantArchive`: Logs `variant_archived` with slug
- `handleVariantDelete`: Logs `variant_deleted` with slug

These signals enable tracking of all variant lifecycle changes.

#### 3. Stripe/Payment Handlers (`api/stripe_handlers.go`)

Added success signals for payment flow observability:

- `handleCheckoutCreate`: Added validation failure logging, logs `checkout_session_created` with price_id and session_id
- `handleSubscriptionCancel`: Logs `subscription_cancelled` with user identifier

### Signal Catalog

| Operation | Log Event | Context Fields |
|-----------|-----------|----------------|
| Metrics tracked | `metrics_event_tracked` | event_type, variant_id, session_id |
| Variant selected | `variant_selected` | slug, name, status |
| Variant created | `variant_created` | slug, name, weight, sections_copied |
| Variant updated | `variant_updated` | slug, updated_fields[] |
| Variant archived | `variant_archived` | slug |
| Variant deleted | `variant_deleted` | slug |
| Checkout created | `checkout_session_created` | price_id, session_id |
| Subscription cancelled | `subscription_cancelled` | user |

### Files Modified

| File | Changes |
|------|---------|
| `api/metrics_handlers.go` | 3 handlers updated with structured logging and writeJSONError |
| `api/variant_handlers.go` | 5 handlers updated with success signals |
| `api/stripe_handlers.go` | 2 handlers updated with success signals |

### [REQ:SIGNAL-FEEDBACK] Tags Added

All modified handlers now have `[REQ:SIGNAL-FEEDBACK]` annotations for traceability.

### Validation

- Go build: Will verify with `go build`
- Scenario status: Will verify with `vrooli scenario status`
- Completeness score: Will compare before/after

### Benefits

1. **Observability**: Important operations now emit success signals for debugging
2. **Audit Trail**: Variant lifecycle changes are logged with context
3. **Payment Tracking**: Checkout and subscription events logged for business visibility
4. **Consistent Format**: All handlers use structured logging and writeJSONError
5. **Signal Stability**: Event names and fields are documented for future consumers

### Next Steps

- Add integration tests that assert expected log events
- Consider adding request correlation IDs
- Add metrics for log event volume monitoring

## 2026-07-30 — Commerce usage transport extraction

- Moved all usage HTTP endpoints from `api/usage_service.go` into
  `api/handlers/commerce/usage.go`: service-token authorization, usage report,
  customer/admin summaries, entitlement limit check, and health probe.
- Removed `api/usage_service.go` after moving its last composition code to its
  actual owners: `main.go` supplies runtime-only secret/logging policy and
  `routes.go` composes commerce transport dependencies. Routes invoke the
  commerce handler package directly; no production compatibility wrapper
  remains.
- Repointed existing characterization tests to the exported commerce transport
  and added direct package tests for missing bearer credentials, malformed JSON,
  missing authenticated identity, and missing limit key. These reject before
  reaching a service, which documents the security and validation boundary.

Validation:

- `go test ./... -count=1 -timeout 10m` (API) passed.
- `go build ./...` and `make lint-go` (API) passed.
- `make restart` followed by `make status` reports the scenario healthy on API
  port 17691 and UI port 23224.

## 2026-07-30 — Endpoint inventory and dead bundle transport cleanup

- Deleted obsolete root bundle catalog/update handler entry points. They had no
  registered production routes after the Connect migration; tests now invoke
  `handlers/bundles` directly through the same root dependency composition.
- Corrected `api/cmd/gen-endpoints` to include every currently mounted generated
  service previously omitted from Connect inventory: Assets, Variant, Metrics,
  and Intelligence. Added explicit assertions for representative procedure
  paths and regenerated `.vrooli/endpoints.json` (182 endpoints).

Validation:

- `go test ./... -count=1 -timeout 10m`, `go build ./...`, and `make lint-go`
  passed in `api/`.
- Generator test verifies the committed manifest exactly matches the route and
  mounted-Connect inventory.

## 2026-07-30 — Delivery app-catalog Connect server

- Extended `DownloadService` with `DeleteDownloadApp`, regenerated governed
  proto artifacts through `vrooli package generate proto`, and implemented
  list/create/save/delete in `api/handlers/delivery/connect.go`.
- The handler converts only at the transport edge and delegates validation and
  persistence to the existing delivery catalog. Direct tests cover generated
  response conversion, validation rejection, and not-found deletion mapping.
- Mounted only the four implemented app-catalog procedures behind admin auth.
  `AuthorizeDownload` is deliberately not mounted yet: its entitlement-aware
  behavior remains on the established REST path until it is fully migrated.
- The endpoint generator inventories exactly these mounted DownloadService
  operations, avoiding an endpoint manifest that advertises an unimplemented
  procedure.

Validation:

- `vrooli package generate proto` completed.
- `go test ./... -count=1 -timeout 10m`, `go build ./...`, and `make lint-go`
  passed in `api/` after regenerating `.vrooli/endpoints.json`.

## 2026-07-30 — Admin profile legacy transport retirement

- Retired the unmounted JSON `GET/PUT /api/v1/admin/profile` handlers after
  the generated `AdminProfileService` became the UI's only production client.
  This removes the duplicated password-change workflow rather than preserving
  a compatibility shim with no registered route.
- Kept one Connect handler responsible for current-password verification,
  email uniqueness, password policy/hash generation, session revocation, and
  the updated signed session cookie. Root characterization tests now call its
  typed RPC methods and preserve cookie propagation explicitly.

Validation:

- Focused profile tests and `go test ./... -timeout 600s` passed in `api/`.
- `go build ./...` and `git diff --check` passed.

## 2026-07-30 — API lint and post-extraction cleanup

- Removed unused root catalog/Stripe compatibility adapters revealed by the
  migration and applied `gofumpt` to the API tree.
- Corrected the lone staticcheck finding in the delivery authorizer test by
  using a private typed context key rather than a built-in string key.

Validation:

- `make lint-go`, `go test ./... -timeout 600s`, `go build ./...`, and
  `git diff --check` passed.

## 2026-07-30 — Governed DOMPurify remediation

- Applied the dependency analyzer's approved pnpm override to force DOMPurify
  `3.4.12` across Monaco's transitive dependency path. This removes the stale
  `3.4.8` copy without changing application code or weakening lockfile
  governance.
- Did not force unrelated transitive packages across incompatible major lines:
  remaining minimatch/brace-expansion/picomatch warnings are owned by the
  Vitest, ESLint, and Tailwind dependency trees and require compatible upstream
  upgrades or selector-aware overrides.

Validation:

- Full UI Vitest suite and TypeScript typecheck passed.
- Dependency governance validation passed.
- Security findings fell from 32 to 26; no DOMPurify advisory remains.

## 2026-07-30 — Governed Vitest 4 security upgrade

- Upgraded `vitest` and `@vitest/coverage-v8` together to `4.0.18` through
  the dependency analyzer, preserving their peer compatibility. The prior
  Vitest 3 coverage graph retained vulnerable minimatch and brace-expansion
  paths.
- Deliberately deferred Tailwind 4: it is the remaining owner of an old
  Picomatch path but requires a separate CSS tooling/configuration migration,
  not a blanket transitive override.

Validation:

- Full UI test run progressed cleanly on Vitest 4.0.18; UI typecheck and
  production build passed.
- Security findings fell from 26 to 17 and `git diff --check` passed.

## 2026-07-30 — Tailwind 4 security/toolchain migration

- Migrated from Tailwind 3 to governed Tailwind `4.3.3` with the official
  `@tailwindcss/postcss` adapter, updating the PostCSS plugin and CSS entry
  directives while retaining the existing scenario theme configuration.
- Fixed the CSS import ordering warning introduced by the v4 `@config`
  directive. The production build is warning-free.

Validation:

- UI typecheck and production build passed; the complete Vitest run progressed
  cleanly on the upgraded toolchain.
- Security findings fell from 17 to 13; the Picomatch advisories are gone.

## 2026-07-30 — Native color-input validation semantics

- Corrected UI Health's raw-hex static rule to exempt only hex values inside a
  statically declared native `input[type=color]`. Those values are required by
  the browser control's HTML value contract, not inline component styling;
  normal styled hex values remain detected. Added a focused regression test
  that proves the distinction.
- Replaced Branding Settings' text-field hex examples with descriptive format
  guidance while preserving the color picker defaults and the existing user
  experience.

Validation:

- The focused UI Health checker package test passed.
- Restarted UI Health through the scenario lifecycle, then confirmed the
  landing-page static validation reports zero `standard_no_raw_hex` findings.
- Landing-page UI typecheck and warning-free production build passed.
- UI Health's comprehensive run exercised its API and UI-health phases
  successfully; its overall failure is pre-existing unrelated scenario debt
  (dependency, unit-policy, storage, workflow, business, security, and proto
  phases), not this checker change.

## 2026-07-30 — UI coverage-gate diagnosis and VariantSection transport tests

- Reproduced the full-suite unit failure as a real UI coverage threshold miss:
  all Vitest assertions pass, but V8 branch coverage is 79.6% against the
  required 85%. The threshold remains unchanged; this is a test-depth work
  item, not a configuration workaround.
- Extended the typed variant transport tests through the full VariantSection
  lifecycle (list, get, create, update, delete), stable response identity
  validation, JSON-object content validation, malformed generated response
  handling, complete nested header mapping, and SEO optional-field mapping.
- Corrected the section-content conversion boundary to accept `unknown` before
  narrowing it to a JSON object. This makes its defensive null/array rejection
  type-correct rather than relying on an impossible `Record` null check.

Validation:

- Focused `src/shared/api/variants.test.ts` passes (8 tests).
- Focused ESLint for the changed source/tests and `git diff --check` pass.
- The post-first-slice full coverage run improved branch coverage from 79.3%
  to 79.6%; additional behavior-focused test slices are required to reach 85%.

### Follow-up coverage and lint slice

- Added focused `useInlineAlert` tests for error classification, retry policy,
  auto-dismiss timing, warning, success, and explicit clear behavior.
- Extended Remote Profiles route coverage to verify failed test/logout/inspect/
  revoke/delete operations report errors instead of falsely presenting success.
- Cleared the declared UI ESLint errors found during certification: made the
  metrics JSON boundary type-safe, removed a redundant test-mock assertion,
  awaited the async waitlist export in its test, and explicitly discarded the
  async click callback promise.

Validation:

- UI typecheck, complete UI lint, and the affected focused Vitest suites pass.
- Full V8 coverage now reports 80.13% branches. The remaining 4.87 points are
  intentionally retained as behavior-test work rather than masking them with
  threshold or exclusion changes.

### Public landing and editor-navigation coverage slice

- Added behavior coverage for the public landing header's custom, section, and
  nested menu links, including device-specific visibility, hidden CTAs, the
  fallback signal, and safe initials-only branding when a logo is absent.
- Added customization-page coverage for direct section navigation, asynchronous
  section target resolution, and safe fallback when that lookup rejects.
- Kept the Connect metric payload boundary defensive: serialized metric data is
  accepted only when it re-parses as a JSON object.

Validation:

- Focused PublicLanding and metrics tests, UI typecheck, complete UI lint, and
  `git diff --check` pass (lint retains 14 pre-existing warnings and no errors).
- Complete V8 run: 172 files / 2,056 tests pass. Global branch coverage is
  80.56% (6,062 / 7,525) against the deliberate 85% gate, so certification
  remains correctly blocked by 338 uncovered branch outcomes.

### Route and remote-profile hook coverage slice

- Added direct behavior tests for admin-login error classification and retry,
  public-feedback retry/error recovery, and remote-profile route states
  (sessionless controls, rejected credentials, save failures, and destructive
  confirmation).
- Extended the remote-profile hook seam with success, unknown-error, refresh,
  state-reconciliation, and busy-state cleanup cases across every action.
- Added the pricing catalog edge case that preserves a free plan with an
  unexpected interval while rejecting invalid paid/yearly interval records.

Validation:

- All affected focused Vitest suites, focused ESLint, UI typecheck, and
  `git diff --check` pass.
- The complete V8 run after the route slice passed 172 files / 2,062 tests and
  reached 80.95% branches. After the hook slice it reached 81.12%; the 85%
  branch gate is still deliberately unmet. The pricing edge test was added
  after that measurement and has focused validation only.

### Billing catalog hook coverage slice

- Added direct tests for price-form validation, confirmed and declined plan
  deletion, delete failure recovery, price-check cleanup, and reordering known
  catalog entries while ignoring unknown ids.

Validation:

- `useBillingForm` passes 31 focused tests; focused lint, UI typecheck, and
  `git diff --check` pass.
- The complete V8 run completed all UI tests and reached 81.32% branches. It
  remains a deliberate certification failure until the required 85% is met.

### Storage wizard coverage slice

- Added direct storage-wizard tests for R2, MinIO, S3, and custom provider
  detection; invalid navigation bounds; reset behavior; failed settings loads;
  and safe error handling for connection tests and persistence.

Validation:

- The focused storage-wizard suite passes 10 tests, with focused lint and UI
  typecheck passing.
- The complete V8 run passed every UI test and improved global branch coverage
  to 81.62%. The required 85% branch gate remains open.

### Section, analytics, and metrics resilience coverage slice

- Added section-editor tests for stable-key enforcement, unavailable route
  identifiers, no-op navigation safeguards, and safe fallback messages from
  every asynchronous context/preview/comparison source.
- Added analytics dashboard tests for retryable load failure and incomplete,
  down-trending detail data.
- Added metrics hook tests for each interaction event type and for page-view /
  scroll-band deduplication across concurrent consumers and unmount cleanup.
- Added a public-landing fallback test proving a signed-off offline page still
  renders when the remote configuration request also reports an error, and
  skips disabled or unknown content safely.

Validation:

- Focused Vitest suites, focused lint, and UI typecheck pass for all changed
  surfaces.
- The full V8 suite passes every UI test and now reports 81.98% branches. It
  continues to fail only the deliberate 85% global branch gate.

### Variant editor transport resilience coverage slice

- Added behavior-focused tests for non-`Error` failures while loading a
  variant, its axes, and its JSON snapshot; these verify the messages shown to
  operators and ensure loading state is cleared.
- Covered non-`Error` failures in both form and JSON persistence paths, plus a
  denied clipboard write while copying actual Monaco validation issues.

Validation:

- The focused `useVariantForm` suite passes all 36 tests, with focused ESLint
  and UI typecheck passing.
- The complete V8 suite completes all UI assertions and increases branch
  coverage to 82.09%. Certification correctly remains blocked solely by the
  enforced 85% global branch threshold. Full lint has no errors and retains 14
  unrelated warnings in existing barrel and Fast Refresh files.

### Customization and public-preview coverage slice

- Corrected the customization-hook test seam to mock the hook modules that the
  production hook actually imports, then added operator-visible coverage for
  analytics recovery, archive/delete failure alerts, destination navigation,
  and first-section resolution.
- Added a public hero preview test that advances through the complete recorded
  action timeline before the showcase rotates.

Validation:

- Focused customization and hero suites pass 35 assertions, focused ESLint and
  UI typecheck pass, and the scenario diff has no whitespace errors.
- The complete V8 suite completes all assertions and reaches 82.11% global
  branch coverage. The deliberate 85% certification gate remains open.

### Navigation and commerce-hook resilience coverage slice

- Added direct navigation-utility coverage for active route/group discovery,
  stub recognition, and static plus dynamic breadcrumb construction.
- Added resizable-column behavior coverage for local persistence validation,
  constrained dragging, missing-container safety, and cleanup.
- Expanded waitlist, coupon, and customer-management hooks with their
  operator-visible non-`Error` fallback and recovery paths.

Validation:

- All focused Vitest suites, focused ESLint, UI typecheck, and scenario diff
  whitespace checks pass.
- The complete V8 suite completes all assertions and raises global branch
  coverage from 82.11% to 82.62%. The 85% certification threshold is still
  intentionally enforced and remains the sole UI test failure.

### URL focus and feedback-operation coverage slice

- Added customization-flow tests for URL-pinned variants, direct section IDs,
  and asynchronous section-type resolution; completed requests are consumed to
  prevent repeated operator navigation.
- Added feedback-management tests proving fail-closed, operator-safe behavior
  for non-`Error` load/status/delete/bulk-delete failures.

Validation:

- Focused customization and feedback suites, focused ESLint, and UI typecheck
  pass.
- The complete V8 suite completes all assertions and raises global branch
  coverage from 82.62% to 82.87%. The required 85% threshold remains enforced.

### Selector registry contract repair

- Repaired a selector registry collision: the literal `admin.breadcrumb`
  selector was overwriting the dynamic selector with the same object key.
  The dynamic contract is now explicitly named `admin.breadcrumbSegment`, while
  the legacy literal selector remains stable.
- Corrected the selector-manifest generator to use the scenario's actual
  `src/consts` source and output paths, regenerated the manifest, and added
  contract tests that invoke every published dynamic selector and reject invalid
  parameter shapes.

Validation:

- Selector contract tests, focused lint, UI typecheck, and scenario whitespace
  checks pass.
- The complete V8 suite passes every assertion and reports 82.91% global
  branch coverage. Certification remains blocked solely by the unchanged 85%
  global branch threshold.

### UI transport and operator-recovery coverage slice

- Added CTA behavior tests for configured conversion paths and safe no-destination
  behavior, plus artifact-upload tests for drag-and-drop metadata detection,
  cancellation, and non-`Error` presign failures.
- Corrected shared API request wrappers so valid falsy JSON bodies (`false`,
  `0`, and `null`) are serialized instead of silently discarded; added response
  fallback, classification override, retryability, compatibility-response, and
  error-message tests.
- Added checkout coverage for custom annual plans, malformed optional metadata,
  and non-retryable validation failures; added remote-profile coverage for
  remote-side session visibility and busy-action states.

Validation:

- Focused Vitest suites, focused ESLint, UI typecheck, and scenario whitespace
  checks pass for each completed slice.
- Full V8 coverage advanced from 82.91% to 83.67% after the CTA, artifact,
  shared transport, checkout, remote-profile, header-configuration, and
  download-form slices. The unchanged 85% global branch gate remains the sole
  unit-test execution error.
