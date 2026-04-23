---
title: "Seams & Architecture"
description: "Testability boundaries, responsibility zones, and substitution points"
category: "technical"
order: 6
audience: ["developers"]
---

# Seams & Architecture

> **Last Updated**: 2026-02-05
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
- Baked fallback content loads once from `.vrooli/fallback/fallback.json` (or `defaultFallbackLandingJSON`). Each response clones the payload, so mutations in one request cannot leak into the next.  
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
  - Subscription persistence preserves existing `plan_tier`/`bundle_key` when Stripe payloads omit or invalidate fields, and uses price ID token inference as a fallback.

- **Payment anomaly dispatch seam** (`payment_anomaly_service.go`, `anomaly_alert_dispatcher.go`)
  - `PaymentAnomalyService.Log(ctx, anomaly)` is the sole entrypoint for recording payment-pipeline anomalies into `payment_anomaly_log`. Callers outside this file should prefer the package-level helper `LogPaymentAnomaly(ctx, srv, anomaly)`.
  - `PaymentAnomalyService.WaitForDispatch(ctx, rowID)` polls the row's `dispatch_status` column at 25 ms intervals until it transitions out of `pending` or ctx cancels. Used by integration tests for deterministic async synchronization and available to admin tooling for manual-insert confirmation (no test-only code path).
  - `PaymentAnomalyService.RefreshConfig(ctx)` re-reads the three `payment_settings` columns (`anomaly_webhook_url`, `anomaly_webhook_enabled`, `anomaly_rate_limits`) and atomically swaps the in-memory snapshot via `atomic.Pointer[anomalyConfig]`. `handleUpdateStripeSettings` calls `RefreshConfig` after every successful save so the next dispatch uses the new config without restart.
  - `AnomalyAlertDispatcher.Dispatch` is the outbound HTTP seam: 3 attempts × 5 s per-attempt timeout with 1 s/2 s/4 s backoff, retries on 5xx + transport errors, does not retry on 4xx. Test seam: `AnomalyAlertDispatcher.UseHTTPClient(httpDoer)` and `UseBackoff(...)` swap the client/backoffs for deterministic tests.
  - Rate limiter: in-process token bucket per `anomaly_type`, guarded by `sync.Mutex`, with defaults of `burst=5` + `refill=1/60s`. Per-type overrides are read from `payment_settings.anomaly_rate_limits` JSONB. Known constraint: the bucket is **in-process** and therefore single-instance; multi-replica deployments would need a postgres-backed limiter, but LPBS runs as one instance today.
  - Dispatch lifecycle: `Log(...)` returns after the row is committed and fires `go s.dispatcher.Dispatch(s.shutdownCtx, payload)` on an unbounded goroutine. Each goroutine exits within ~7 s worst-case (3 attempts × 5 s + 1 s/2 s backoffs); the per-type rate limiter caps spawn rate. HTTP requests use `http.NewRequestWithContext(shutdownCtx, ...)` so in-flight POSTs abort on server shutdown.
  - Historical note: the legacy `intro_anomaly_log` table has been removed. Its rows were migrated into `payment_anomaly_log` with `subject_kind='intro_coupon'` by a one-time boot-time migration in `ensureSchema`.

---

## Other Seams

- **Plan/pricing lookup + integrity** (`plan_service.go`, `plan_store.go`)  
  Centralizes bundle/price metadata and enforces plan integrity rules (valid tiers, kind↔tier alignment, currency, billing interval, unique Stripe price IDs, bundle↔Stripe product matching).  
  - Additional tier invariants: **free** plans require `amount_cents = 0`; **credits/donation** tiers must use `one_time` billing intervals.  
  - Admin creation now flows through `PlanService.CreateBundlePrice(...)`, keeping validation + Stripe verification in the domain layer.
  - Admin updates flow through `PlanService.UpdateBundlePriceWithStripe(...)`, which verifies Stripe price changes before applying updates.
  - Stripe import flows through `PlanService.ImportStripePrices(...)` → `PlanStore.ApplyStripeImportSelections(...)` to batch updates and persist once (rejecting prices that do not belong to the bundle's Stripe product).
  - Plan catalog writes are atomic (temp file + rename) and re-normalize bundle/plan fields on save to keep `.vrooli/plans.json` consistent.

- **Metrics ingestion** (`metrics_service.go`, `metrics_handlers.go`)  
  Validation and storage happen in the service; handlers only marshal/unmarshal requests.

- **Content & variants** (`content_service.go`, `variant_service.go`)  
  UI content and A/B variants are isolated behind services so new presentations do not touch SQL.
- **SEO composition** (`seo_service.go`)  
  Combines site branding defaults with per-variant SEO config and drives sitemap/robots responses. Handlers call the service; admin UI uses the `seoController` + shared API client instead of direct `fetch` calls to keep transport concerns separate from editing logic.

---

## Remote Profiles & Proxy Seam (NEW)

- **Remote profile service** (`api/remote_profiles_service.go`)  
  Centralizes remote admin sessions in one place and encrypts stored `admin_session` cookies at rest. The service owns validation of remote API bases, connector IDs, remote session linkage (`remote_session_id`), and status updates so handlers remain transport-only.

- **HTTP client seam** (`RemoteProfileService.httpClient`)  
  All remote admin calls flow through an injected `HTTPDoer`, making login/test/proxy flows testable with `httptest.Server` or mock clients.

- **Handler boundary seam** (`RemoteProfileManager` in `remote_profiles_handlers.go`)  
  Admin handlers depend on a narrow interface instead of the concrete service. This keeps transport code decoupled from persistence and lets tests stub behavior without standing up the full service.

- **Clock seam** (`RemoteProfileService.now`)  
  Session-expiry evaluation and cookie-derived expirations use an injected clock, keeping time-sensitive logic deterministic in tests.

- **Proxy allowlist seam** (`remoteProfileProxyAllowlist`)  
  Only explicitly allowlisted `/admin/*` endpoints can be proxied, preventing accidental exposure of unrelated admin routes.

- **Incoming session observability seam** (`api/remote_profile_sessions_handlers.go`)  
  Incoming connector sessions are discovered and revoked via dedicated admin endpoints (`/admin/remote-profile-sessions*`) by parsing connector metadata from admin-session user-agent values. This keeps remote-profile ingress controls separate from generic auth and avoids coupling UI flows directly to auth storage internals.

---

## Desktop Auto-Update Seams

The update endpoints (`/api/v1/updates/{app_key}/{channel}/{file}`) serve electron-updater manifests and binary download redirects.

- **Handler interface seam** (`api/update_handlers.go`)
  `handleUpdateFile` is a public endpoint (no auth middleware) that dispatches manifest vs. binary requests based on the `{file}` path segment. Four narrow interfaces are injected:
  - `updateAppLookup` — app retrieval for API key gating
  - `updateAssetLookup` — asset retrieval by variant for manifest serving
  - `updateArtifactResolver` — artifact retrieval, filename lookup, and presigned URL generation
  - `updateBundleKeyProvider` — bundle key resolution

  In production, `*DownloadService` satisfies both `updateAppLookup` and `updateAssetLookup`; `*DownloadHostingService` satisfies `updateArtifactResolver`; `*PlanService` satisfies `updateBundleKeyProvider`. In tests, lightweight mocks (`mockUpdateAppLookup`, etc.) substitute without a database.

- **Per-app API key gating** (`DownloadApp.UpdateAPIKey`)
  Optional per-app access control via `X-Update-Key` header. When `update_api_key` is empty on the app, the endpoint is fully public. Validation lives in the handler, checked before any manifest or download logic.

- **Channel-to-variant mapping** (`channelToVariantKey`)
  Pure function mapping electron-updater channel names to download_assets `variant_key` values. `"stable"` and `""` map to `"default"`; all other channel names pass through. Easily testable without dependencies.

- **Manifest generation** (`buildElectronManifest`)
  Pure function generating electron-updater YAML from a `DownloadArtifact`. No yaml library needed — fixed format via `fmt.Sprintf`. Requires `SHA512` to be present; endpoint returns 404 if missing.

- **Download redirect** (`updateArtifactResolver.GetCurrentArtifactByFilename` + `PresignGetArtifact`)
  Binary download requests resolve the current artifact via a join query (download_artifacts + download_assets), then redirect to a presigned S3 URL. The existing `DownloadStorage` interface seam for presigning applies here.

### Testability Status

| Concern | Status | Notes |
|---------|--------|-------|
| Pure functions | Strong | `manifestFilenameToPlatform`, `channelToVariantKey`, `buildElectronManifest` — all tested in `update_handlers_test.go` |
| Handler flow | Strong | Mock-based unit tests cover manifest serving, binary redirect, missing SHA512, nil artifact, presign failure, and API key gating. Integration tests cover DB-backed flows. |
| Storage redirect | Inherited | Uses existing `DownloadStorage` interface seam for presigning |

---

## Scan Helper Utilities

`artifactScanTargets` (download_hosting.go), `assetScanTargets` (download_service.go), and `appScanTargets` (download_service.go) consolidate the repeated SQL NullString/JSON → struct hydration pattern across all artifact, asset, and app query methods. Each helper provides `scanDest()` for ordered scan destinations and `hydrate()` for populating the struct. Extra columns (e.g., `is_current`, `artifact_count`) are appended to `scanDest()` at the call site.

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

## AI Gateway Seams (NEW)

The AI Gateway provides centralized AI access with credit management for all Vrooli applications.

### OpenRouterClient Seam (Strong)

**Location:** `api/openrouter_client.go`

**Interface:**
```go
type OpenRouterClient interface {
    Chat(ctx context.Context, req OpenRouterChatRequest) (*OpenRouterChatResponse, error)
    ChatStream(ctx context.Context, req OpenRouterChatRequest, onChunk func(content string)) (*OpenRouterUsage, error)
    VerifyAPIKey(ctx context.Context) error
}
```

**Test Doubles:**
- `MockOpenRouterClient` - Configurable mock with function fields for custom behavior

**Status:** Strong
- Clean interface separating HTTP concerns from business logic
- Mock enables testing AI gateway service without network calls
- Compile-time interface enforcement

**Testing Pattern:**
```go
mockClient := &MockOpenRouterClient{
    ChatFn: func(ctx context.Context, req OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
        return &OpenRouterChatResponse{
            Content: "Test response",
            Usage: OpenRouterUsage{PromptTokens: 10, CompletionTokens: 20},
        }, nil
    },
}
svc := NewAIGatewayService(AIGatewayServiceOptions{
    OpenRouterClient: mockClient,
    // ...
})
```

### AIGateway Seam (Strong)

**Location:** `api/ai_gateway_interface.go`

**Interface:**
```go
type AIGateway interface {
    ExecuteChat(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error)
    ExecuteChatStream(ctx context.Context, userIdentity string, req AIRequest, w http.ResponseWriter) error
    GetAvailableModels() []string
    HealthCheck(ctx context.Context) error
}
```

**Test Doubles:**
- `MockAIGateway` - Full interface mock for handler testing

**Status:** Strong
- Handlers accept `AIGateway` interface, not concrete `*AIGatewayService`
- Enables isolated handler testing without real service
- Compile-time interface enforcement

**Testing Pattern:**
```go
mock := &MockAIGateway{
    ExecuteChatFn: func(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error) {
        return &AIResponse{Content: "Mock response"}, nil
    },
}
handler := handleAIChat(mock)
```

### AI Gateway Service Injection Seams

**Location:** `api/ai_gateway_service.go`

The service has multiple injection points:

1. **OpenRouter Client** - `UseOpenRouterClient(client OpenRouterClient)`
   - Primary seam for testing AI provider communication

2. **Logger** - Via `AIGatewayServiceOptions.Logger`
   - Enables capturing logs in tests

**Status:** Strong

### AI Gateway Rate Limiter Seam

**Location:** `api/ai_gateway_handlers.go`

**Design:**
- Package-level `aiGatewayRateLimiter` with `UseTimeProvider()` injection
- Enables controlling time progression in tests

**Testing Pattern:**
```go
// Control time for rate limit testing
fixedTime := time.Now()
aiGatewayRateLimiter.UseTimeProvider(func() time.Time { return fixedTime })
defer aiGatewayRateLimiter.UseTimeProvider(nil)
```

### AI Gateway Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                      AI Gateway Service                          │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────┐    ┌─────────────────────────────────┐   │
│  │ OpenRouterClient │───▶ Chat(), ChatStream()              │   │
│  │ (testing seam)   │    VerifyAPIKey()                     │   │
│  └──────────────────┘    └─────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────┐    ┌─────────────────────────────────┐   │
│  │ UsageService     │───▶ ReserveAndCharge() (atomic)       │   │
│  │                  │    CheckLimit(), RecordUsage()        │   │
│  └──────────────────┘    └─────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────┐    ┌─────────────────────────────────┐   │
│  │ AccountService   │───▶ GetSubscription() (tier lookup)   │   │
│  │                  │                                        │   │
│  └──────────────────┘    └─────────────────────────────────┘   │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

**Credit Flow:**

1. **Non-streaming (atomic):**
   - Estimate cost → ReserveAndCharge (atomic) → Call OpenRouter → Adjust if needed

2. **Streaming (check-then-charge):**
   - Estimate cost → CheckLimit → Stream response → RecordUsage after completion

**Credit Units:**
- Internal unit = 1/1,000,000 of a cent
- $1.00 = 100,000,000 internal units
- Pricing stored as cost per 1K tokens in internal units

---

## See Also

- `docs/STRIPE_RESTRICTED_KEYS.md`
- `docs/STRIPE_WEBHOOKS.md`
- `docs/api/payments.md`
- `docs/AI_GATEWAY.md`
