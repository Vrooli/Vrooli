# Implementation Plan: Cost-Based Credit System

## Overview

Implement a hybrid credit system with:
1. **Cost-based limits** (shared across bundle, tied to real money)
2. **App-specific limits** (arbitrary, per-app)

This system enables monetization by tracking actual costs incurred when users use Vrooli's AI services, while also allowing individual apps to enforce their own usage limits.

## Architecture

### Two Limit Types

#### Cost-Based Limits
- **Unit**: cents x 1,000,000 (for precision without floating point)
- **Scope**: Subscription tier level (shared across all apps in the bundle)
- **When charged**: Only when using vrooli.com API endpoints that incur real costs
- **Not charged for**: BYOK (Bring Your Own Key), local AI, or operations that don't cost Vrooli money

**Example**: If a user's tier includes $10/month of AI credits:
- Internal representation: 10 * 100 * 1,000,000 = 1,000,000,000 units
- An AI call costing $0.002: 0.002 * 100 * 1,000,000 = 200,000 units

#### App-Specific Limits
- **Unit**: Arbitrary (defined per app)
- **Scope**: App level within subscription tier
- **Examples**: Workflow exports, execution counts, storage quotas

**Relationship**: A user might have:
- $10/month cost-based credits (shared across BAS, future apps)
- 100 workflow exports/month in BAS (app-specific)
- 50 report generations/month in a future analytics app (app-specific)

---

## Database Schema

### Table: `subscription_tier_limits`

Defines the limits for each subscription tier.

```sql
CREATE TABLE subscription_tier_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tier_id VARCHAR(50) NOT NULL,           -- 'free', 'solo', 'pro', 'studio', 'business'
    limit_type VARCHAR(20) NOT NULL,         -- 'cost_based' or 'app_specific'
    limit_key VARCHAR(100) NOT NULL,         -- e.g., 'ai_credits', 'workflow_exports'
    limit_value BIGINT NOT NULL,             -- In base units (-1 = unlimited)
    cost_multiplier BIGINT DEFAULT 1000000,  -- For cost-based: cents x multiplier
    app_bundle_key VARCHAR(100),             -- NULL for cost-based, app key for app-specific
    reset_period VARCHAR(20) DEFAULT 'monthly', -- 'monthly', 'daily', 'never'
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(tier_id, limit_type, limit_key, app_bundle_key)
);

-- Example data:
-- Cost-based AI credits (shared across all apps)
INSERT INTO subscription_tier_limits (tier_id, limit_type, limit_key, limit_value, cost_multiplier, app_bundle_key)
VALUES
    ('free', 'cost_based', 'ai_credits', 0, 1000000, NULL),
    ('solo', 'cost_based', 'ai_credits', 500000000, 1000000, NULL),  -- $5/month
    ('pro', 'cost_based', 'ai_credits', 2000000000, 1000000, NULL),  -- $20/month
    ('studio', 'cost_based', 'ai_credits', 10000000000, 1000000, NULL), -- $100/month
    ('business', 'cost_based', 'ai_credits', -1, 1000000, NULL);     -- Unlimited

-- App-specific limits for BAS
INSERT INTO subscription_tier_limits (tier_id, limit_type, limit_key, limit_value, app_bundle_key)
VALUES
    ('free', 'app_specific', 'workflow_exports', 5, 'browser-automation-studio'),
    ('solo', 'app_specific', 'workflow_exports', 50, 'browser-automation-studio'),
    ('pro', 'app_specific', 'workflow_exports', -1, 'browser-automation-studio');
```

### Table: `usage_records`

Tracks actual usage per user per billing period.

```sql
CREATE TABLE usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_identity VARCHAR(255) NOT NULL,     -- Email or customer ID
    billing_period VARCHAR(20) NOT NULL,     -- 'YYYY-MM' or subscription cycle
    limit_key VARCHAR(100) NOT NULL,         -- e.g., 'ai_credits', 'workflow_exports'
    usage_amount BIGINT NOT NULL DEFAULT 0,  -- Current usage in base units
    app_bundle_key VARCHAR(100),             -- Which app reported this (NULL for cost-based)
    last_operation_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(user_identity, billing_period, limit_key, app_bundle_key)
);

-- Indexes for efficient lookups
CREATE INDEX idx_usage_records_user_period ON usage_records(user_identity, billing_period);
CREATE INDEX idx_usage_records_app ON usage_records(app_bundle_key, billing_period);
```

### Table: `api_keys`

Stores encrypted API keys for AI providers (admin-managed).

```sql
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL,           -- 'openrouter', 'openai', 'anthropic', etc.
    encrypted_key TEXT NOT NULL,             -- Encrypted with server key
    key_hint VARCHAR(20),                    -- Last 4 chars for display
    is_active BOOLEAN DEFAULT true,
    last_verified_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(provider)  -- One key per provider for now
);
```

---

## API Endpoints

### Limits Configuration (Admin)

```
GET  /api/v1/admin/tiers/:tier/limits     - Get all limits for a tier
PUT  /api/v1/admin/tiers/:tier/limits     - Update limits for a tier
GET  /api/v1/admin/apps/:app/limits       - Get app-specific limits
PUT  /api/v1/admin/apps/:app/limits       - Update app-specific limits
```

**Request/Response Examples:**

```typescript
// GET /api/v1/admin/tiers/pro/limits
{
  "tier": "pro",
  "limits": [
    {
      "type": "cost_based",
      "key": "ai_credits",
      "value": 2000000000,
      "display_value": "$20.00/month",
      "app_bundle_key": null
    },
    {
      "type": "app_specific",
      "key": "workflow_exports",
      "value": -1,
      "display_value": "Unlimited",
      "app_bundle_key": "browser-automation-studio"
    }
  ]
}

// PUT /api/v1/admin/tiers/pro/limits
{
  "limits": [
    {
      "key": "ai_credits",
      "value_dollars": 25.00  // API converts to internal units
    }
  ]
}
```

### API Keys Management (Admin)

```
GET    /api/v1/admin/api-keys              - List configured keys (masked)
POST   /api/v1/admin/api-keys              - Add new key
DELETE /api/v1/admin/api-keys/:id          - Remove key
POST   /api/v1/admin/api-keys/:id/test     - Test key validity
```

**Request/Response Examples:**

```typescript
// GET /api/v1/admin/api-keys
{
  "keys": [
    {
      "id": "uuid",
      "provider": "openrouter",
      "key_hint": "...sk-4f2a",
      "is_active": true,
      "last_verified_at": "2024-01-15T10:30:00Z"
    }
  ]
}

// POST /api/v1/admin/api-keys
{
  "provider": "openrouter",
  "key": "sk-or-v1-..."
}

// POST /api/v1/admin/api-keys/:id/test
// Response:
{
  "valid": true,
  "provider": "openrouter",
  "balance": "$45.23",  // If provider supports balance check
  "models_available": 150
}
```

### Usage Reporting (Apps)

```
POST /api/v1/usage/report                  - Report usage from an app
GET  /api/v1/usage/summary                 - Get usage summary for user
```

**Request/Response Examples:**

```typescript
// POST /api/v1/usage/report (called by BAS, other apps)
{
  "user_identity": "user@example.com",
  "operation": "ai_vision_navigate",
  "cost_cents": 0.2,                       // Actual cost if known
  "app_bundle_key": "browser-automation-studio",
  "metadata": {
    "model": "gpt-4o-mini",
    "tokens": 1500
  }
}

// GET /api/v1/usage/summary?user=user@example.com
{
  "user_identity": "user@example.com",
  "billing_period": "2024-01",
  "cost_based": {
    "ai_credits": {
      "used": 150000000,
      "limit": 2000000000,
      "used_display": "$1.50",
      "limit_display": "$20.00",
      "percentage": 7.5
    }
  },
  "app_specific": {
    "browser-automation-studio": {
      "workflow_exports": {
        "used": 12,
        "limit": -1,
        "used_display": "12",
        "limit_display": "Unlimited"
      }
    }
  }
}
```

---

## Admin UI Pages

### 1. API Keys Page (`/admin/api-keys`)

**Purpose**: Manage AI provider API keys

**UI Components**:
- **Key List Table**:
  - Provider name (OpenRouter, OpenAI, Anthropic, etc.)
  - Key hint (last 4 chars)
  - Status indicator (active/inactive)
  - Last verified date
  - Actions (Test, Delete)

- **Add Key Form**:
  - Provider dropdown
  - Key input (password field)
  - Add button

- **Test Results Modal**:
  - Shows validation result
  - Balance (if supported)
  - Available models count

**Security**:
- Keys are never displayed in full
- Require admin authentication
- Audit log for all key operations

### 2. Tier Limits Page (`/admin/tiers/limits`)

**Purpose**: Configure cost-based limits per subscription tier

**UI Components**:
- **Tier Selector**: Dropdown to select tier (free, solo, pro, studio, business)
- **Cost-Based Limits Section**:
  - AI Credits: Dollar input with automatic unit conversion
  - Display: "$X.XX / month" format
  - Unlimited checkbox option
- **Conversion Display**: Shows internal units for transparency
- **Save Button**: With confirmation for changes

**UX Notes**:
- Show warning when reducing limits below current usage
- Highlight tiers with users currently at or near limit

### 3. App Limits Page (`/admin/apps/limits`)

**Purpose**: Configure app-specific limits

**UI Components**:
- **App Selector**: Dropdown of registered apps
- **Limits List**:
  - Limit name
  - Value per tier (table format)
  - Edit inline
- **Add Limit Button**: Create new app-specific limit
- **Unit Configuration**: For each limit type

---

## Admin Header Refactor

Current admin header has many top-level tabs. Reorganize into dropdowns:

```
Dashboard  |  Content ▼  |  Users ▼  |  Config ▼  |  Analytics
                 │              │           │
                 ├─ Pages       ├─ Accounts  ├─ API Keys
                 ├─ Blog        ├─ Subscriptions  ├─ Tier Limits
                 └─ Media       └─ Usage     └─ App Limits
                                             └─ Settings
```

---

## Implementation Order

### Phase 1: Database & Backend Foundation
1. [ ] Create database migrations for new tables
2. [ ] Implement API key encryption/decryption service
3. [ ] Create tier limits repository
4. [ ] Create usage records repository

### Phase 2: API Endpoints
5. [ ] Implement API keys CRUD endpoints
6. [ ] Implement tier limits CRUD endpoints
7. [ ] Implement app limits CRUD endpoints
8. [ ] Implement usage reporting endpoint
9. [ ] Implement usage summary endpoint

### Phase 3: Admin UI
10. [ ] Refactor admin header with dropdowns
11. [ ] Create API Keys page
12. [ ] Create Tier Limits page
13. [ ] Create App Limits page

### Phase 4: Integration
14. [ ] Update BAS to report usage to LPBS
15. [ ] Create shared client library for usage reporting
16. [ ] Add usage analytics dashboard

### Phase 5: Testing & Documentation
17. [ ] Unit tests for all new services
18. [ ] Integration tests for API endpoints
19. [ ] E2E tests for admin flows
20. [ ] Update API documentation

---

## Security Considerations

### API Key Storage
- Keys MUST be encrypted at rest using AES-256-GCM
- Encryption key stored in environment variable, not in code
- Key rotation support required
- Audit logging for all key access

### Admin Endpoints
- Require admin role authentication
- Rate limiting on sensitive operations
- IP allowlist option for production

### Usage Reporting
- Apps must authenticate with service-to-service tokens
- Validate app_bundle_key against registered apps
- Rate limit to prevent abuse
- Idempotency keys for duplicate prevention

### Data Protection
- User identities normalized and validated
- PII handling compliant with privacy policy
- Usage data retention policy (default: 2 years)

---

## Testing Checklist

### API Keys
- [ ] Admin can add a new API key
- [ ] API keys are encrypted in database
- [ ] API keys can be tested for validity
- [ ] API keys can be deactivated
- [ ] API keys can be deleted
- [ ] Invalid keys are rejected

### Tier Limits
- [ ] Admin can view limits for each tier
- [ ] Admin can update cost-based limits
- [ ] Dollar values correctly convert to internal units
- [ ] Unlimited (-1) works correctly
- [ ] Changes take effect immediately

### App Limits
- [ ] Admin can view app-specific limits
- [ ] Admin can add new limit types
- [ ] Admin can update limit values per tier
- [ ] Limits are correctly scoped to apps

### Usage Reporting
- [ ] Apps can report usage successfully
- [ ] Usage is correctly aggregated
- [ ] Cost-based limits are shared across apps
- [ ] App-specific limits are isolated per app
- [ ] Usage summary returns correct data
- [ ] Billing period boundaries work correctly

### Integration
- [ ] BAS correctly reports AI usage to LPBS
- [ ] BYOK operations are logged with 0 cost
- [ ] Users see accurate usage in their dashboard
- [ ] Admins see accurate usage across all users

---

## Migration Notes

### From Current System

The current system in BAS has:
- Credit tracking in SQLite (`credit_usage`, `operation_log` tables)
- Entitlement service fetching from LPBS
- Per-tier AI credit limits in config

Migration path:
1. Keep BAS local tracking for offline/BYOK scenarios
2. Add reporting to LPBS for online usage
3. LPBS becomes source of truth for billing
4. BAS syncs with LPBS on reconnect

### Backwards Compatibility

- Existing BAS installations continue working with local tracking
- New features require LPBS connection
- Graceful degradation when LPBS unavailable
