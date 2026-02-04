---
title: "Seams & Architecture"
description: "Testability boundaries, responsibility zones, and substitution points"
category: "technical"
order: 6
audience: ["developers"]
---

# Seams & Architecture

> **Last Updated**: 2026-02-04
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
  Centralizes remote admin sessions in one place and encrypts stored `admin_session` cookies at rest. The service owns validation of remote API bases, session storage, and status updates so handlers remain transport-only.

- **HTTP client seam** (`RemoteProfileService.httpClient`)  
  All remote admin calls flow through an injected `HTTPDoer`, making login/test/proxy flows testable with `httptest.Server` or mock clients.

- **Handler boundary seam** (`RemoteProfileManager` in `remote_profiles_handlers.go`)  
  Admin handlers depend on a narrow interface instead of the concrete service. This keeps transport code decoupled from persistence and lets tests stub behavior without standing up the full service.

- **Clock seam** (`RemoteProfileService.now`)  
  Session-expiry evaluation and cookie-derived expirations use an injected clock, keeping time-sensitive logic deterministic in tests.

- **Proxy allowlist seam** (`remoteProfileProxyAllowlist`)  
  Only explicitly allowlisted `/admin/*` endpoints can be proxied, preventing accidental exposure of unrelated admin routes.

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
