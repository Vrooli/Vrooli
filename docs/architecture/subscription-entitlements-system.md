# Subscription Entitlements System Architecture

**Last Updated:** 2026-01-21
**Status:** Phase 3 Complete - Production Ready for Core Usage Tracking
**Related Scenarios:** `browser-automation-studio`, `landing-page-business-suite`

This document captures the full architecture, data flows, known gaps, and recommendations for the subscription entitlements/usage/limits system that spans two scenarios.

## Quick Status

| Component | LPBS | BAS | Notes |
|-----------|------|-----|-------|
| Usage Recording | ✅ Complete | ✅ Complete | Idempotent, atomic UPSERT |
| LPBS Reporting | ✅ Endpoint ready | ✅ Reporter ready | Async with retry logic |
| Health Endpoints | ✅ `/api/v1/usage/health` | ✅ `CheckLPBSHealth()` | Not HTTP-exposed in BAS |
| Admin UI - Usage Dashboard | ✅ Complete | N/A | Monthly view, top users |
| Admin UI - Tier Limits | ✅ Complete | N/A | Editable dollar values |
| Admin UI - App Limits | ✅ Complete | N/A | Per-app quotas |
| Admin UI - API Keys | ✅ Complete | N/A | AI provider keys |
| Feature-Aware Gating | N/A | ✅ Complete | Features array + tier fallback |
| Billing Cycle | ✅ Stored | ✅ Used | Custom day (1-28) support |
| Test Coverage | ✅ 943+ lines | ✅ 50+ tests | Comprehensive |

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [System Overview](#system-overview)
3. [Architecture Diagram](#architecture-diagram)
4. [Component Deep Dive](#component-deep-dive)
   - [Landing-Page-Business-Suite (LPBS)](#landing-page-business-suite-lpbs)
   - [Browser-Automation-Studio (BAS)](#browser-automation-studio-bas)
5. [Data Flow](#data-flow)
6. [API Contract](#api-contract)
7. [Known Gaps](#known-gaps)
8. [What Works Today](#what-works-today)
9. [Recommendations](#recommendations)
10. [Key Files Reference](#key-files-reference)
11. [Testing Checklist](#testing-checklist)
12. [Change Log](#change-log)

---

## Executive Summary

The subscription system spans two scenarios:
- **LPBS** (landing-page-business-suite): Source of truth for subscriptions, plans, and credit balances
- **BAS** (browser-automation-studio): Consumer that fetches entitlements and enforces feature gating

**Critical Finding:** The system has **two incompatible credit models running in parallel**:
1. LPBS tracks "purchased balance pool" (`credit_wallets.balance_credits`)
2. BAS tracks "tier-based monthly limits" locally

**Neither system deducts from the other.** BAS never reports usage back to LPBS, and LPBS's `balance_credits` only increases (via Stripe webhooks) but never decreases.

---

## System Overview

### The Two Scenarios

| Scenario | Role | Database | Key Responsibility |
|----------|------|----------|-------------------|
| `landing-page-business-suite` | Source of Truth | PostgreSQL | Subscription status, plan tiers, credit balances, Stripe integration |
| `browser-automation-studio` | Consumer | SQLite/PostgreSQL | Fetch entitlements, enforce limits, track local usage |

### Subscription Tiers

Five tiers in ascending order of capability:

| Tier | Order | Execution Limit | AI Credits | Recording | Watermark |
|------|-------|-----------------|------------|-----------|-----------|
| `free` | 1 | 50/month | 0 (no access) | No | Yes |
| `solo` | 2 | 200/month | 50/month | Yes | Yes |
| `pro` | 3 | Unlimited | 500/month | Yes | No |
| `studio` | 4 | Unlimited | 2000/month | Yes | No |
| `business` | 5 | Unlimited | Unlimited | Yes | No |

### Subscription Statuses

```go
const (
    StatusActive   = "active"    // Subscription is current
    StatusTrialing = "trialing"  // In trial period
    StatusPastDue  = "past_due"  // Payment failed, grace period
    StatusCanceled = "canceled"  // User canceled
    StatusInactive = "inactive"  // No subscription / default
)
```

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                          LANDING-PAGE-BUSINESS-SUITE                             │
│                           (vrooli.com - Source of Truth)                         │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐ │
│  │    subscriptions    │    │   credit_wallets    │    │     plan_options    │ │
│  ├─────────────────────┤    ├─────────────────────┤    ├─────────────────────┤ │
│  │ subscription_id     │    │ customer_email      │    │ stripe_price_id     │ │
│  │ status (active...)  │    │ balance_credits ────│───►│ plan_tier           │ │
│  │ plan_tier           │    │ bonus_credits       │    │ features[] (JSON)   │ │
│  │ price_id            │    │ updated_at          │    │ monthly_credits     │ │
│  │ customer_email      │    └─────────────────────┘    └─────────────────────┘ │
│  │ bundle_key          │                                                        │
│  └──────────┬──────────┘           ▲                                           │
│             │                      │ Credits ADDED only                         │
│             │                      │ (via Stripe webhook)                       │
│             │                ┌─────┴─────────────────┐                          │
│             │                │  stripe_service.go    │                          │
│             │                │  checkout.completed   │                          │
│             │                │  → ADD balance_credits│                          │
│             │                └───────────────────────┘                          │
│             │                                                                    │
│             ▼                                                                    │
│  ┌──────────────────────────────────────────────────────────────────────────┐  │
│  │                    GET /api/v1/entitlements?user=email                    │  │
│  └──────────────────────────────────────────────────────────────────────────┘  │
│             │                                                                    │
└─────────────│────────────────────────────────────────────────────────────────────┘
              │
              │  HTTP Response:
              │  {
              │    "status": "active",
              │    "plan_tier": "pro",
              │    "features": ["ai", "recording"],
              │    "credits": { "balance_credits": 5000 },
              │    "subscription": { ... }
              │  }
              │
              ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         BROWSER-AUTOMATION-STUDIO                                │
│                            (Desktop App - Consumer)                              │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │                      entitlement/service.go                              │   │
│  │  ┌─────────────────────────────────────────────────────────────────┐    │   │
│  │  │  IN-MEMORY CACHE (5-min TTL, 5-hr offline grace)                │    │   │
│  │  │  cache[userEmail] = { tier, status, features, credits, ... }    │    │   │
│  │  └─────────────────────────────────────────────────────────────────┘    │   │
│  │                                │                                         │   │
│  │                                ▼                                         │   │
│  │  ┌─────────────────────────────────────────────────────────────────┐    │   │
│  │  │          TIER-BASED FEATURE GATING (hardcoded config)           │    │   │
│  │  │  ┌──────────────┬──────────────┬──────────────┬──────────────┐  │    │   │
│  │  │  │ Watermark    │ AI Access    │ Recording    │ Exec Limits  │  │    │   │
│  │  │  ├──────────────┼──────────────┼──────────────┼──────────────┤  │    │   │
│  │  │  │ free: YES    │ free: NO     │ free: NO     │ free: 50/mo  │  │    │   │
│  │  │  │ solo: YES    │ solo: NO     │ solo: YES    │ solo: 200/mo │  │    │   │
│  │  │  │ pro+: NO     │ pro+: YES    │ pro+: YES    │ pro+: ∞      │  │    │   │
│  │  │  └──────────────┴──────────────┴──────────────┴──────────────┘  │    │   │
│  │  └─────────────────────────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │                       credits/service.go                                 │   │
│  │                   LOCAL USAGE TRACKING (SQLite/Postgres)                 │   │
│  │                                                                          │   │
│  │  ┌────────────────────────┐    ┌────────────────────────┐               │   │
│  │  │     credit_usage       │    │     operation_log      │               │   │
│  │  ├────────────────────────┤    ├────────────────────────┤               │   │
│  │  │ user_identity          │    │ operation_type         │               │   │
│  │  │ billing_month (YYYY-MM)│    │ credits_charged        │               │   │
│  │  │ total_credits_used     │    │ success                │               │   │
│  │  │ credits_by_operation{} │    │ metadata (JSON)        │               │   │
│  │  │ operations_by_type{}   │    │ created_at             │               │   │
│  │  └────────────────────────┘    └────────────────────────┘               │   │
│  │                                                                          │   │
│  │  AI Credit Limits (by tier):                                            │   │
│  │  free: 0 │ solo: 50/mo │ pro: 500/mo │ studio: 2000/mo │ business: ∞   │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│                           ⚠️  NO SYNC BACK TO LPBS  ⚠️                          │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## Component Deep Dive

### Landing-Page-Business-Suite (LPBS)

#### Database Tables

**subscriptions** - Stores Stripe subscription data
```sql
CREATE TABLE subscriptions (
    subscription_id TEXT PRIMARY KEY,
    status TEXT,                    -- active, trialing, past_due, canceled
    customer_email TEXT,
    customer_id TEXT,
    plan_tier TEXT,                 -- free, solo, pro, studio, business
    price_id TEXT,                  -- Stripe price ID
    bundle_key TEXT,                -- Product bundle identifier
    billing_cycle_start INTEGER DEFAULT 0,  -- Day of month (1-28) for billing cycle
    canceled_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

**credit_wallets** - Stores purchased credit balances
```sql
CREATE TABLE credit_wallets (
    customer_email TEXT PRIMARY KEY,
    balance_credits BIGINT DEFAULT 0,  -- ⚠️ Only ADDED, never decremented
    bonus_credits BIGINT DEFAULT 0,
    updated_at TIMESTAMP
);
```

**plan_options** - Plan metadata including features
```sql
CREATE TABLE plan_options (
    stripe_price_id TEXT PRIMARY KEY,
    plan_name TEXT,
    plan_tier TEXT,
    billing_interval TEXT,          -- MONTH, YEAR, ONE_TIME
    amount_cents BIGINT,
    monthly_included_credits BIGINT,
    one_time_bonus_credits BIGINT,
    metadata JSONB                  -- Contains features array
);
```

#### Key Services

| File | Purpose |
|------|---------|
| `api/account_service.go` | GetSubscription, GetCredits, GetEntitlements |
| `api/account_handlers.go` | HTTP handlers for `/api/v1/entitlements` |
| `api/plan_service.go` | Plan metadata and pricing |
| `api/stripe_service.go` | Stripe webhooks, credit additions |

#### Entitlements Endpoint

**Request:** `GET /api/v1/entitlements?user={email}`

**Response:**
```json
{
  "status": "active",
  "plan_tier": "pro",
  "price_id": "price_xxx",
  "features": ["ai", "recording", "watermark-free"],
  "credits": {
    "customer_email": "user@example.com",
    "balance_credits": 5000,
    "bundle_key": "bas",
    "updated_at": "2026-01-21T12:00:00Z"
  },
  "subscription": {
    "state": "SUBSCRIPTION_STATE_ACTIVE",
    "subscription_id": "sub_xxx",
    "user_identity": "user@example.com",
    "plan_tier": "pro",
    "stripe_price_id": "price_xxx"
  }
}
```

---

### Browser-Automation-Studio (BAS)

#### Database Tables (Local)

**credit_usage** - Monthly usage aggregates
```sql
CREATE TABLE credit_usage (
    id UUID PRIMARY KEY,
    user_identity TEXT,
    billing_month TEXT,             -- YYYY-MM format
    total_credits_used INT,
    total_operations INT,
    credits_by_operation JSONB,     -- {"ai.workflow_generate": 15, ...}
    operations_by_type JSONB,       -- {"ai.workflow_generate": 3, ...}
    last_operation_at TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    UNIQUE(user_identity, billing_month)
);
```

**operation_log** - Individual operation audit trail
```sql
CREATE TABLE operation_log (
    id UUID PRIMARY KEY,
    user_identity TEXT,
    operation_type TEXT,            -- ai.workflow_generate, execution.run, etc.
    credits_charged INT,
    success BOOLEAN,
    metadata JSONB,
    error_message TEXT,
    duration_ms INT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### Key Services

| File | Purpose |
|------|---------|
| `api/services/entitlement/service.go` | Fetch & cache entitlements from LPBS |
| `api/services/entitlement/types.go` | Tier, Status, Entitlement types |
| `api/services/credits/service.go` | Local usage tracking & credit charging |
| `api/services/credits/interface.go` | CreditService interface |
| `api/services/credits/costs.go` | Operation cost definitions |
| `api/middleware/entitlement.go` | Request middleware for feature gating |
| `api/handlers/entitlement.go` | HTTP endpoints for status & usage |

#### Entitlement Service Configuration

```go
type EntitlementConfig struct {
    ServiceURL           string         // Default: https://vrooli.com
    CacheTTL             time.Duration  // Default: 5 minutes
    RequestTimeout       time.Duration  // Default: 5 seconds
    OfflineGracePeriod   time.Duration  // Default: 5 hours (reduced from 24h on 2026-01-21)
    DefaultTier          string         // Default: "free"
    TierLimits           map[string]int // Execution limits per tier
    AICreditsLimits      map[string]int // AI credits per tier
    WatermarkTiers       []string       // ["free", "solo"]
    AITiers              []string       // ["pro", "studio", "business"]
    RecordingTiers       []string       // ["solo", "pro", "studio", "business"]
}
```

#### Environment Variables

```bash
# Service URL
BAS_ENTITLEMENT_SERVICE_URL=https://vrooli.com

# Cache & Offline
BAS_ENTITLEMENT_CACHE_TTL_MS=300000                    # 5 minutes
BAS_ENTITLEMENT_REQUEST_TIMEOUT_MS=5000                # 5 seconds
BAS_ENTITLEMENT_OFFLINE_GRACE_PERIOD_MS=18000000       # 5 hours (was 24h)

# Tier Limits (JSON)
BAS_ENTITLEMENT_TIER_LIMITS_JSON='{"free":50,"solo":200,"pro":-1,"studio":-1,"business":-1}'
BAS_ENTITLEMENT_AI_CREDITS_LIMITS_JSON='{"free":0,"solo":50,"pro":500,"studio":2000,"business":-1}'

# Feature Gates (comma-separated)
BAS_ENTITLEMENT_WATERMARK_TIERS=free,solo
BAS_ENTITLEMENT_AI_TIERS=pro,studio,business
BAS_ENTITLEMENT_RECORDING_TIERS=solo,pro,studio,business
```

#### Operation Types & Costs

```go
const (
    OpAIWorkflowGenerate  OperationType = "ai.workflow_generate"
    OpAIWorkflowEdit      OperationType = "ai.workflow_edit"
    OpAIChat              OperationType = "ai.chat"
    OpExecutionRun        OperationType = "execution.run"
    OpExportVideo         OperationType = "export.video"
    OpExportGIF           OperationType = "export.gif"
    // ... more operations
)

// Costs are hardcoded to prevent bypass
func DefaultOperationCosts() OperationCosts {
    return OperationCosts{
        OpAIWorkflowGenerate: 5,
        OpAIWorkflowEdit:     3,
        OpAIChat:             1,
        OpExecutionRun:       1,
        OpExportVideo:        2,
        // ...
    }
}
```

---

## Data Flow

### Purchase Flow

```
User clicks "Subscribe" on vrooli.com
         │
         ▼
┌─────────────────────────┐
│   Stripe Checkout       │
│   (payment processed)   │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  Stripe Webhook         │
│  checkout.session.completed
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  LPBS stripe_service.go │
│  - INSERT subscription  │
│  - ADD balance_credits  │◄── Credits only go UP
└─────────────────────────┘
            │
            │  (No notification to BAS)
            ▼
         (done)
```

### Entitlement Check Flow

```
BAS needs to check if user can use AI
         │
         ▼
┌─────────────────────────┐
│  Check in-memory cache  │
│  (TTL: 5 minutes)       │
└───────────┬─────────────┘
            │
    cache miss or expired
            │
            ▼
┌─────────────────────────┐
│  HTTP GET to LPBS       │
│  /api/v1/entitlements   │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  LPBS returns:          │
│  - tier: "pro"          │
│  - balance_credits: 5000│◄── This value is ignored by BAS!
│  - status: "active"     │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  BAS caches result      │
│  Checks tier vs config  │
│  tier in AITiers? → YES │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  Check LOCAL usage      │◄── Uses tier-based limits, NOT balance_credits
│  credits_used < limit?  │
└─────────────────────────┘
```

### Usage Tracking Flow

```
User generates AI workflow
         │
         ▼
┌─────────────────────────┐
│  CanPerformAIOperation  │
│  1. Check BYOK (bypass) │
│  2. Check tier has AI   │
│  3. Check local credits │
└───────────┬─────────────┘
            │
        allowed
            │
            ▼
┌─────────────────────────┐
│  Perform AI operation   │
└───────────┬─────────────┘
            │
        success
            │
            ▼
┌─────────────────────────┐
│  Charge local credits   │
│  - UPDATE credit_usage  │
│  - INSERT operation_log │
│  ⚠️ LPBS never knows!   │
└─────────────────────────┘
```

---

## API Contract

### BAS → LPBS

**Endpoint:** `GET /api/v1/entitlements`

**Query Parameters:**
- `user` - Email or customer ID

**Headers:**
- `X-User-Email: user@example.com`
- `Accept: application/json`

**Success Response (200):**
```json
{
  "status": "active",
  "plan_tier": "pro",
  "price_id": "price_1ABC123",
  "features": ["ai", "recording", "watermark-free"],
  "billing_cycle_start": 15,
  "credits": {
    "customer_email": "user@example.com",
    "balance_credits": 5000,
    "bundle_key": "bas",
    "updated_at": "2026-01-21T12:00:00Z"
  },
  "subscription": {
    "state": "SUBSCRIPTION_STATE_ACTIVE",
    "subscription_id": "sub_1ABC123",
    "user_identity": "user@example.com",
    "plan_tier": "pro",
    "stripe_price_id": "price_1ABC123",
    "bundle_key": "bas",
    "cached_at": "2026-01-21T12:00:00Z",
    "cache_age_ms": 1234
  }
}
```

**No Subscription Response (200):**
```json
{
  "status": "inactive",
  "plan_tier": "",
  "billing_cycle_start": 0,
  "credits": {
    "customer_email": "user@example.com",
    "balance_credits": 0,
    "bundle_key": "bas"
  },
  "subscription": {
    "state": "SUBSCRIPTION_STATE_INACTIVE",
    "user_identity": "user@example.com"
  }
}
```

### BAS Internal Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/entitlement/status` | GET | Current entitlements & usage |
| `/api/v1/entitlement/identity` | GET/POST/DELETE | Manage stored user identity |
| `/api/v1/entitlement/refresh` | POST | Force refresh cached entitlements |
| `/api/v1/entitlement/override` | GET/POST/DELETE | Tier override (dev mode) |
| `/api/v1/entitlement/api-source` | GET/POST/DELETE | API source config (prod/local/disabled) |
| `/api/v1/entitlement/usage` | GET | Current month usage |
| `/api/v1/entitlement/usage/history` | GET | Multiple months history |
| `/api/v1/entitlement/usage/operations` | GET | Operation log (paginated) |

---

## Known Gaps

### Critical (Must Fix)

#### GAP-001: No Usage Synchronization ✅ FIXED

**Problem:** BAS tracks usage locally but never reports back to LPBS. The `balance_credits` in LPBS only increases (via Stripe webhooks) but never decreases.

**Impact:**
- LPBS shows incorrect credit balance
- Cannot implement usage-based billing
- Cannot aggregate usage across devices

**Evidence:**
- `scenarios/browser-automation-studio/api/services/credits/service.go` - No sync code
- `scenarios/landing-page-business-suite/api/` - No usage ingestion endpoint

**Fix Applied (2026-01-21):**
- LPBS implemented `POST /api/v1/usage/report` endpoint
- LPBS implemented `GET /api/v1/usage/health` endpoint for monitoring
- BAS `reportUsageToLPBS()` with retry logic (exponential backoff: 500ms, 1s, 2s)
- BAS sends usage reports to LPBS after each charge
- Report includes: user_identity, limit_key, usage_amount (cents × 1M), app_bundle_key, operation_id, metadata

**Idempotency (2026-01-21):**
- Added `operation_id` field (UUID) to prevent double-counting on retries
- BAS generates operation_id ONCE per charge, reuses across all retry attempts
- LPBS checks for duplicate operation_id before recording usage
- Database: Added `operation_id` column and unique partial index

**Documentation:**
- See `scenarios/browser-automation-studio/docs/LPBS_INTEGRATION.md` for integration guide

---

#### GAP-002: Multi-Instance Exploit

**Problem:** Each BAS instance tracks usage in its own local database. A user can run multiple instances and get N× the monthly credits.

**Impact:** Credits can be trivially "multiplied" by running multiple app instances.

**Evidence:**
- BAS uses SQLite by default (local to each instance)
- No device/instance registration or limits

**Recommendation:** Either:
- Server-side usage tracking (preferred)
- Device fingerprinting with instance limits
- Cryptographic usage receipts

---

#### GAP-003: Two Incompatible Credit Systems

**Problem:**
- LPBS has `balance_credits` (purchased pool, one-time or subscription)
- BAS has tier-based monthly limits (reset each billing cycle)

These don't interact. BAS fetches `balance_credits` but never uses or decrements it.

**Impact:** The credit purchase system in LPBS is effectively non-functional.

**Evidence:**
- `account_service.go:217` - Returns `balance_credits` in entitlements
- `service.go:281-283` - BAS stores it in `ent.Credits` but never reads it
- `service.go:574-589` - `getUserCreditsLimit` uses tier config, not `balance_credits`

**Decision Made (2026-01-21):** Hybrid tier-based model
- **Cost-Based Limits:** Shared across bundle, for operations that cost Vrooli (AI API calls)
- **App-Specific Limits:** Arbitrary units, per-app (exports, workflow runs)
- BYOK operations: Log with 0 cost for analytics
- Local AI: Don't count against cost-based limits

**Remaining Work:**
- Once LPBS usage endpoint is implemented (GAP-001), BAS usage can deduct from centralized limits

---

### Significant (Should Fix)

#### GAP-004: Features Array Ignored ✅ FIXED

**Problem:** LPBS returns `features: ["ai", "recording"]` extracted from plan metadata, but BAS ignores this and uses hardcoded tier-based configuration.

**Impact:** Plan-specific features in LPBS are decorative, not functional.

**Evidence:**
- `account_service.go:230-233` - Extracts features from plan metadata
- `service.go:447-456` - `tierCanUseAI` checks hardcoded config, not features

**Fix Applied (2026-01-21):**
- Added `CanUseAIWithEntitlement(ent)` - checks features array first, tier fallback for backwards compatibility
- Added `CanUseRecordingWithEntitlement(ent)` - checks features array first
- Added `RequiresWatermarkWithEntitlement(ent)` - checks features array first (looks for `watermark_free` feature)
- Updated `CanPerformAIOperation` in credits service to use `CanUseAIWithEntitlement`

**Files Modified:**
- `bas/services/entitlement/service.go` - Added `*WithEntitlement` methods (lines 470-509)
- `bas/services/credits/service.go` - Updated to use feature-aware methods

---

#### GAP-005: Calendar Month Reset ✅ FIXED

**Problem:** BAS resets usage on the 1st of each calendar month, not the subscription anniversary date.

**Impact:** User subscribing on the 25th loses 24 days of credits in the first month.

**Evidence:**
- `service.go:147` - `time.Now().Format("2006-01")` for billing month
- No subscription start date tracking

**Fix Applied (2026-01-21):**
- LPBS: Added `billing_cycle_start` column to subscriptions table
- LPBS: Added `BillingCycleAnchor` parsing from Stripe subscriptions
- LPBS: `extractBillingCycleDay()` converts Unix timestamp to day (1-28, capped for short months)
- LPBS: `GetEntitlements` now returns `billing_cycle_start` in response
- BAS: `entitlementResponse` parses `billing_cycle_start`
- BAS: Added `GetBillingPeriod(t)` and `GetBillingMonth(t)` methods to Entitlement
- BAS: `getBillingMonth()` in credits service uses custom billing cycle
- BAS: `GetUsage()` returns correct `PeriodStart`, `PeriodEnd`, `ResetDate` boundaries

**Files Modified:**
- `lpbs/api/main.go` - Database migration for billing_cycle_start column
- `lpbs/api/stripe_service.go` - BillingCycleAnchor parsing, extractBillingCycleDay helper
- `lpbs/api/account_service.go` - getBillingCycleStart helper, updated GetEntitlements
- `lpbs/initialization/postgres/schema.sql` - Schema documentation
- `bas/services/entitlement/types.go` - GetBillingPeriod, GetBillingMonth methods
- `bas/services/entitlement/service.go` - Parse BillingCycleStart from response
- `bas/services/credits/service.go` - getBillingMonth helper, updated Charge/GetUsage/getUsageFromDB

---

#### GAP-006: Separate Execution Limits

**Problem:** Execution limits (50/200/∞) are tracked separately from credit limits.

**Impact:** Two independent enforcement systems that could conflict or confuse users.

**Evidence:**
- `service.go:115-131` - `CanExecuteWorkflow` uses `getTierLimit`
- `service.go:88-126` - `CanCharge` uses `getUserCreditsLimit`
- Different tracking paths

**Recommendation:** Unify into single credit system where executions cost credits.

---

#### GAP-007: No Overage Handling

**Problem:** When credits are exhausted, operations simply fail with an error.

**Impact:** Poor UX, no grace period, no overage billing option.

**Evidence:**
- `service.go:178-180` - Returns `ErrInsufficientCredits`
- No soft limits or overage tracking

**Recommendation:** Implement soft limits, grace periods, or overage billing.

---

### Minor (Nice to Fix)

#### GAP-008: BYOK Not Tracked ✅ FIXED

**Problem:** AI operations with user's own API key bypass credit checks entirely.

**Impact:** Usage analytics miss BYOK operations, can't measure actual platform usage.

**Evidence:**
- `service.go:189-192` - `if hasBYOK { return true, "", "", -1, nil }`

**Fix Applied (2026-01-21):**
- Added `IsBYOK bool` field to `ChargeRequest` struct
- Modified `Charge()` to set cost=0 when IsBYOK=true, while still logging the operation
- Updated all AI handlers (`vision_navigation.go`, `ai_analysis.go`, `element_analysis.go`) to pass IsBYOK flag

---

#### GAP-009: No Proration

**Problem:** Credits don't prorate on mid-cycle tier changes.

**Impact:** Upgrade/downgrade mid-month may over/under-allocate.

**Recommendation:** Track tier changes, prorate remaining credits.

---

#### GAP-010: Stale Cache Fails Open ✅ MITIGATED

**Problem:** 24-hour offline grace period uses potentially very stale data.

**Impact:** User could have canceled but still use premium features for 24 hours.

**Evidence:**
- `service.go:96-101` - Uses stale cache within grace period
- `service.go:319-324` - `withinOfflineGrace` checks 24-hour window

**Fix Applied (2026-01-21):**
- Reduced default offline grace period from 24 hours to 5 hours
- Changed `BAS_ENTITLEMENT_OFFLINE_GRACE_PERIOD_MS` default from 86400000 to 18000000

---

## What Works Today

The system is **production-ready** for core usage tracking. Here's what's operational:

### Core Functionality

| Feature | Status | Notes |
|---------|--------|-------|
| Tier-based feature gating | ✅ Working | Watermarks, AI, recording correctly gated |
| Feature-aware gating | ✅ Working | `*WithEntitlement` methods check features array first |
| Billing cycle awareness | ✅ Working | Uses subscription anniversary date (day 1-28), calendar fallback |
| Entitlement caching | ✅ Working | 5-min TTL, 5-hr offline grace |
| Local usage tracking | ✅ Working | Operations logged, credits charged locally |
| Operation audit trail | ✅ Working | Full operation_log with metadata |
| BYOK tracking | ✅ Working | BYOK operations logged with cost=0 for analytics |

### LPBS Usage API

| Feature | Status | Notes |
|---------|--------|-------|
| `POST /api/v1/usage/report` | ✅ Working | Idempotent with operation_id, service-to-service auth |
| `GET /api/v1/usage/summary` | ✅ Working | User's usage with remaining calculation |
| `GET /api/v1/usage/check` | ✅ Working | Pre-check if operation allowed |
| `GET /api/v1/usage/health` | ✅ Working | Unauthenticated monitoring endpoint |
| `GET /api/v1/admin/usage` | ✅ Working | Admin aggregation view |
| Idempotency | ✅ Working | Duplicate operation_id returns success without incrementing |
| Concurrent safety | ✅ Working | Atomic UPSERT, tested with 100 concurrent requests |

### BAS→LPBS Integration

| Feature | Status | Notes |
|---------|--------|-------|
| `reportUsageToLPBS()` | ✅ Working | Async, non-blocking, generates unique operation_id |
| Retry logic | ✅ Working | 3 attempts with exponential backoff (500ms, 1s, 2s) |
| `CheckLPBSHealth()` | ✅ Working | Method exists but not HTTP-exposed |
| Cost conversion | ✅ Working | `actualCostCents * 1,000,000` for precision |

### Admin UI (LPBS)

| Page | Status | Features |
|------|--------|----------|
| `UsageDashboard.tsx` | ✅ Working | Period selector, stats, app breakdown, top users |
| `TierLimitsSettings.tsx` | ✅ Working | Editable limits, dollar values, unlimited toggle |
| `AppLimitsSettings.tsx` | ✅ Working | Per-app quotas per tier |
| `APIKeysSettings.tsx` | ✅ Working | AI provider API key management |

### Infrastructure

| Feature | Status | Notes |
|---------|--------|-------|
| Stripe subscription sync | ✅ Working | LPBS updates on webhooks |
| Plan/pricing metadata | ✅ Working | Features, prices, tiers stored correctly |
| Tier override (dev) | ✅ Working | Test different tiers without subscription |
| API source switching | ✅ Working | production/local/disabled modes |
| Usage history & reports | ✅ Working | Historical data available |

---

## Recommendations

### Credit Model Decision (2026-01-21)

**Decision:** Formalize a **hybrid tier-based model** with two distinct limit types:

1. **Cost-Based Limits (Shared Across Bundle)**
   - For operations that incur real costs to Vrooli (AI API calls, etc.)
   - Unit: cents × multiplier (e.g., cents × 1,000,000 for precision)
   - Configured at **subscription tier level** (same limit for all apps)
   - Only charged when using vrooli.com endpoints (not local AI or BYOK)
   - Shared across all apps in the subscription bundle

2. **App-Specific Limits (Arbitrary)**
   - For operations that don't cost Vrooli money (exports, workflow runs, etc.)
   - Unit: arbitrary (each app defines their own)
   - Configured at **app level** within subscription tier
   - Apps decide whether to display/enforce these limits

**Key Rules:**
- BYOK operations: Log with 0 cost (for analytics), don't deduct from limits
- Local AI operations: Don't count against cost-based limits
- Only vrooli.com API calls that incur fees count against cost-based limits

---

### Phased Implementation Checklist

#### Phase 1: Short-Term (Quick Wins) ✅ COMPLETE

| # | Task | Status | Files | Notes |
|---|------|--------|-------|-------|
| 1.1 | Add BYOK 0-cost tracking | ✅ DONE | `credits/service.go`, `credits/interface.go`, `handlers/ai/*.go` | IsBYOK field added to ChargeRequest, operations logged with cost=0 |
| 1.2 | Reduce offline grace period | ✅ DONE | `config/config.go`, `entitlement/service_test.go` | Changed default from 24h to 5h (18000000ms) |
| 1.3 | Document credit model architecture | ✅ DONE | `lpbs/docs/IMPLEMENTATION_PLAN.md` | Detailed plan for cost-based system with database schema, API endpoints, admin UI |
| 1.4 | Update this doc with decision | ✅ DONE | This file | Recorded hybrid model decision |

#### Phase 2: Medium-Term (Feature-Aware Gating & Billing Cycle) ✅ COMPLETE

| # | Task | Status | Files | Notes |
|---|------|--------|-------|-------|
| 2.1 | LPBS retry logic with exponential backoff | ✅ DONE | BAS `services/credits/service.go` | 3 retries with 500ms/1s/2s backoff |
| 2.2 | Feature-aware methods | ✅ DONE | BAS `services/entitlement/service.go` | CanUseAIWithEntitlement, etc. |
| 2.3 | Billing cycle fix (GAP-005) | ✅ DONE | LPBS + BAS (multiple files) | billing_cycle_start stored and used |

#### Phase 2b: Admin Infrastructure ✅ COMPLETE

| # | Task | Status | Files | Notes |
|---|------|--------|-------|-------|
| 2b.1 | Add API keys admin page | ✅ DONE | `ui/src/surfaces/admin-portal/routes/APIKeysSettings.tsx` | AI provider key management |
| 2b.2 | Add tier limits admin page | ✅ DONE | `ui/src/surfaces/admin-portal/routes/TierLimitsSettings.tsx` | Editable dollar values |
| 2b.3 | Add app limits admin page | ✅ DONE | `ui/src/surfaces/admin-portal/routes/AppLimitsSettings.tsx` | Per-app quotas |
| 2b.4 | Add usage dashboard | ✅ DONE | `ui/src/surfaces/admin-portal/routes/UsageDashboard.tsx` | Monthly stats, top users |
| 2b.5 | Database schema for limits | ✅ DONE | `initialization/postgres/schema.sql` | `subscription_tier_limits` table |
| 2b.6 | Limits API endpoints | ✅ DONE | `api/limits_service.go`, `api/main.go` | Full CRUD for tier/app limits |

#### Phase 3: Medium-Term (Usage Sync) ✅ COMPLETE

| # | Task | Status | Files | Notes |
|---|------|--------|-------|-------|
| 3.1 | Usage reporting API in LPBS | ✅ DONE | LPBS `api/usage_service.go` | `RecordUsage()` with idempotency |
| 3.2 | Usage reporter in BAS | ✅ DONE | BAS `services/credits/service.go` | `reportUsageToLPBS()` with retry logic |
| 3.3 | Idempotency keys | ✅ DONE | Both scenarios | `operation_id` prevents double-counting |
| 3.4 | Health endpoints | ✅ DONE | Both scenarios | LPBS `/usage/health`, BAS `CheckLPBSHealth()` |
| 3.5 | Documentation | ✅ DONE | BAS `docs/LPBS_INTEGRATION.md` | Integration guide with examples |

**Implementation Details:**
- LPBS `POST /api/v1/usage/report` receives and records usage
- LPBS `GET /api/v1/usage/health` provides monitoring endpoint
- BAS generates `operation_id` (UUID) per charge, reuses across retries
- LPBS checks for duplicate `operation_id` before recording (idempotent)
- Retry logic: 3 attempts with exponential backoff (500ms, 1s, 2s)

#### Phase 4: Long-Term (Full Solution)

| # | Task | Status | Files | Notes |
|---|------|--------|-------|-------|
| 4.1 | Billing cycle awareness | ✅ DONE | Both scenarios | Uses subscription anniversary date (see GAP-005 fix) |
| 4.2 | Overage handling | ⬜ TODO | Both scenarios | Soft limits, grace periods |
| 4.3 | Multi-device coordination | ⬜ TODO | Both scenarios | Device registration, instance limits |
| 4.4 | Real-time usage dashboard | ⬜ TODO | LPBS UI | Show usage across all apps |

---

### Legacy Recommendations (Pre-Decision)

The following were the original recommendations before the credit model decision was made:

#### Short-Term (Quick Wins)
1. ~~**Decide on credit model**~~ → DONE: Hybrid tier-based model
2. **Add BYOK tracking** - Log operations with 0 cost for analytics
3. **Reduce offline grace** - 4-6 hours instead of 24 hours

#### Medium-Term (Required for Production)
4. **Implement usage sync** - BAS reports usage to LPBS after each operation
5. **Add usage ingestion API** - LPBS endpoint to receive usage reports
6. **Unify execution/credit limits** - Single credit system

#### Long-Term (Full Solution)
7. **Server-side usage tracking** - All usage tracked in LPBS, BAS is thin client
8. **Billing cycle awareness** - Use subscription anniversary date
9. **Overage handling** - Soft limits, grace periods, or overage billing
10. **Multi-device coordination** - Device registration, instance limits

---

## Key Files Reference

### Landing-Page-Business-Suite

| File | Purpose |
|------|---------|
| `api/account_service.go` | GetSubscription, GetCredits, GetEntitlements |
| `api/account_handlers.go` | HTTP handlers: /api/v1/entitlements |
| `api/plan_service.go` | Plan/pricing metadata from database |
| `api/stripe_service.go` | Stripe webhooks, credit additions |
| `api/stripe_handlers.go` | Stripe webhook HTTP handlers |
| `initialization/postgres/schema.sql` | Database schema |
| `initialization/postgres/seed.sql` | Initial plan data |

### Browser-Automation-Studio

| File | Purpose |
|------|---------|
| `api/services/entitlement/types.go` | Core types: Tier, Entitlement, Status |
| `api/services/entitlement/service.go` | Entitlement fetching, caching, tier limits |
| `api/services/credits/interface.go` | CreditService interface |
| `api/services/credits/service.go` | Credit deduction & usage tracking |
| `api/services/credits/costs.go` | Operation cost definitions |
| `api/handlers/entitlement.go` | HTTP endpoints for entitlements & usage |
| `api/middleware/entitlement.go` | Request middleware for feature gating |
| `api/config/config.go` | Configuration for limits & tiers |

---

## Testing Checklist

### Subscription Flow
- [ ] New subscription creates record in LPBS
- [ ] Subscription webhook adds credits to wallet
- [x] BAS can fetch entitlements for subscribed user
- [x] BAS correctly identifies tier from response

### Feature Gating
- [x] Free tier cannot access AI features
- [x] Free tier gets watermarked exports
- [x] Solo tier can record but not use AI
- [x] Pro+ tier has full access
- [x] Feature array is respected when present (GAP-004 fix)
- [x] Tier fallback works when features array is empty

### Usage Tracking (Local)
- [x] Operations are logged in operation_log
- [x] Credits are deducted from monthly usage
- [x] Usage resets at billing cycle boundary (not calendar month)
- [x] Usage history shows correct data
- [x] BYOK operations logged with cost=0

### Usage Sync (BAS→LPBS)
- [x] BAS generates unique operation_id per charge
- [x] LPBS records usage with operation_id
- [x] Duplicate operation_id returns success without incrementing
- [x] Retry logic uses same operation_id across attempts
- [x] 100 concurrent requests handled atomically (LPBS)
- [x] LPBS health endpoint returns correct status

### Billing Cycle (GAP-005 fix)
- [x] `billing_cycle_start` stored from Stripe subscription
- [x] BAS receives `billing_cycle_start` in entitlements response
- [x] `GetBillingPeriod()` returns correct boundaries
- [x] `getBillingMonth()` uses custom billing cycle
- [x] Fallback to calendar month when billing_cycle_start is 0
- [x] February edge cases handled (leap year, non-leap year)
- [x] Year boundary transitions handled correctly

### Admin UI
- [x] Usage dashboard shows period stats and top users
- [x] Tier limits page allows editing dollar values
- [x] App limits page allows per-app configuration
- [x] API keys page allows managing provider keys

### Edge Cases
- [x] Offline mode uses cached entitlements
- [x] Expired cache fetches fresh data
- [x] Invalid user gets free tier defaults
- [x] Tier override works in dev mode

### Gap Verification
- [ ] Verify balance_credits never decrements (GAP-003) - still true, intentional
- [ ] Verify multi-instance can exceed limits (GAP-002) - still true
- [x] Features array is now respected (GAP-004) - FIXED

### Integration Testing (Manual)
- [ ] Deploy LPBS with `LPBS_SERVICE_SECRET=test-secret`
- [ ] Deploy BAS with `LPBS_URL` and `LPBS_SECRET` configured
- [ ] Perform AI operation in BAS
- [ ] Verify usage appears in LPBS admin dashboard
- [ ] Simulate network failure, verify retry and idempotency

---

## Change Log

| Date | Author | Changes |
|------|--------|---------|
| 2026-01-21 | Claude | Initial analysis and documentation |
| 2026-01-21 | Claude | Phase 1 complete: BYOK tracking, reduced offline grace (24h→5h), IMPLEMENTATION_PLAN.md |
| 2026-01-21 | Claude | Phase 2 complete: Feature-aware gating (GAP-004), Billing cycle fix (GAP-005), LPBS retry logic |
| 2026-01-21 | Claude | Phase 2b complete: Admin UI pages (UsageDashboard, TierLimitsSettings, AppLimitsSettings, APIKeysSettings) |
| 2026-01-21 | Claude | Phase 3 complete: Usage sync (GAP-001), idempotency keys, health endpoints, integration docs |
| 2026-01-21 | Claude | Updated documentation with accurate current state assessment |

---

## Next Steps

### Immediate (Verification)
1. **End-to-end integration test** - Deploy both services, verify BAS→LPBS flow works in real environment
2. **Expose BAS health endpoint** - Wire `CheckLPBSHealth()` to `/api/v1/credits/lpbs-health` route

### Short-Term (Hardening)
3. **Rate limiting** - Add per-app token rate limits on `/api/v1/usage/report`
4. **Service token enforcement** - Make `LPBS_SERVICE_SECRET` required in production (fail startup if not set)
5. **Metrics/alerting** - Add instrumentation for LPBS failure rates, retry counts, latency

### Medium-Term (Robustness)
6. **Circuit breaker** - Prevent thundering herd on LPBS failures
7. **Overage handling** - Soft limits, grace periods, overage billing option
8. **Audit logging** - Track who changed tier limits and when

### Long-Term (Scale)
9. **Multi-device coordination** - Device registration, instance limits (addresses GAP-002)
10. **Real-time usage dashboard** - WebSocket updates for live usage monitoring
