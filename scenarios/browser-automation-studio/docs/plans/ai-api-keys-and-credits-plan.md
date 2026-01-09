# AI API Keys & Credits Implementation Plan

## Overview

This plan implements Option B: OpenRouter as the single AI gateway with BYOK (Bring Your Own Key) support through OpenRouter's dashboard, integrated with subscription tier entitlements and usage tracking.

## Current State Analysis

### What Exists
- **Entitlement System**: Mature tier-based feature gating (`api/services/entitlement/`)
  - Tiers: free, solo, pro, studio, business
  - AI access gated by `AITiers` config (default: pro, studio, business)
  - Usage tracking for workflow executions in `entitlement_usage` table
  - UI components: `UsageMeterSection`, `SubscriptionStatusCard`, `TierBadge`

- **API Keys Settings** (`ui/src/views/SettingsView/sections/ApiKeysSection.tsx`)
  - `browserlessApiKey` - **OBSOLETE** (now using Playwright)
  - `openaiApiKey` - **NOT USED** by backend
  - `anthropicApiKey` - **NOT USED** by backend
  - `customApiEndpoint` - unclear purpose

- **Backend AI Client** (`api/services/ai/client.go`)
  - Uses `resource-openrouter` CLI command
  - Reads `OPENROUTER_API_KEY` from environment/Vault
  - No integration with frontend API key settings

### Problems to Solve
1. Frontend API keys are disconnected from backend
2. Browserless key is obsolete
3. No AI credits/usage tracking
4. No visibility into AI usage or limits on the settings page
5. No upgrade prompts for AI features

---

## Implementation Phases

### Phase 1: Clean Up Obsolete Settings

**Goal**: Remove dead code and prepare for new structure

#### 1.1 Update ApiKeySettings Interface
**File**: `ui/src/stores/settingsStore.ts`

```typescript
// BEFORE
export interface ApiKeySettings {
  browserlessApiKey: string;    // Remove
  openaiApiKey: string;         // Remove
  anthropicApiKey: string;      // Remove
  customApiEndpoint: string;    // Remove
}

// AFTER
export interface ApiKeySettings {
  openrouterApiKey: string;     // Single key for all AI
}
```

#### 1.2 Update Default Settings
**File**: `ui/src/stores/settingsStore.ts`

```typescript
const getDefaultApiKeySettings = (): ApiKeySettings => ({
  openrouterApiKey: '',
});
```

#### 1.3 Add Migration for Existing Users
**File**: `ui/src/stores/settingsStore.ts`

```typescript
// In loadApiKeySettings(), handle migration from old keys
const loadApiKeySettings = (): ApiKeySettings => {
  try {
    const stored = safeGetItem(API_KEYS_KEY);
    if (!stored) return getDefaultApiKeySettings();
    const parsed = JSON.parse(stored);

    // Migrate old format: if old keys exist, keep openrouterApiKey only
    if ('browserlessApiKey' in parsed || 'openaiApiKey' in parsed) {
      const migrated = { openrouterApiKey: parsed.openrouterApiKey || '' };
      saveApiKeySettings(migrated);
      return migrated;
    }

    return { ...getDefaultApiKeySettings(), ...parsed };
  } catch {
    return getDefaultApiKeySettings();
  }
};
```

---

### Phase 2: Backend AI Credits Tracking

**Goal**: Track AI API usage per user with monthly limits by tier

#### 2.1 Database Schema
**File**: `initialization/storage/postgres/schema.sql`

```sql
-- AI Credits Usage Tracking
CREATE TABLE IF NOT EXISTS ai_credits_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_identity VARCHAR(255) NOT NULL,
    billing_month VARCHAR(7) NOT NULL,  -- YYYY-MM format
    credits_used INTEGER NOT NULL DEFAULT 0,
    requests_count INTEGER NOT NULL DEFAULT 0,
    last_request_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_ai_user_month UNIQUE (user_identity, billing_month)
);

CREATE INDEX idx_ai_credits_user ON ai_credits_usage(user_identity);
CREATE INDEX idx_ai_credits_month ON ai_credits_usage(billing_month);

-- AI Request Log (for detailed tracking and debugging)
CREATE TABLE IF NOT EXISTS ai_request_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_identity VARCHAR(255) NOT NULL,
    request_type VARCHAR(50) NOT NULL,  -- 'workflow_generate', 'element_analyze', etc.
    model VARCHAR(100),
    prompt_tokens INTEGER,
    completion_tokens INTEGER,
    credits_charged INTEGER NOT NULL DEFAULT 1,
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    duration_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ai_log_user ON ai_request_log(user_identity);
CREATE INDEX idx_ai_log_created ON ai_request_log(created_at);
CREATE INDEX idx_ai_log_type ON ai_request_log(request_type);
```

#### 2.2 AI Credits Tracker Service
**File**: `api/services/entitlement/ai_credits.go` (new file)

```go
package entitlement

import (
    "context"
    "database/sql"
    "time"
)

// AICreditsTracker tracks AI credit usage per user.
type AICreditsTracker struct {
    db  *sql.DB
    log *logrus.Logger
}

type AICreditsUsage struct {
    UserIdentity   string    `json:"user_identity"`
    BillingMonth   string    `json:"billing_month"`
    CreditsUsed    int       `json:"credits_used"`
    RequestsCount  int       `json:"requests_count"`
    CreditsLimit   int       `json:"credits_limit"`   // -1 for unlimited
    CreditsRemaining int     `json:"credits_remaining"` // -1 for unlimited
    PeriodStart    time.Time `json:"period_start"`
    PeriodEnd      time.Time `json:"period_end"`
    ResetDate      time.Time `json:"reset_date"`
}

func NewAICreditsTracker(db *sql.DB, log *logrus.Logger) *AICreditsTracker

func (t *AICreditsTracker) GetAICreditsUsage(ctx context.Context, userIdentity string) (*AICreditsUsage, error)

func (t *AICreditsTracker) ChargeCredits(ctx context.Context, userIdentity string, credits int, requestType string, model string) error

func (t *AICreditsTracker) CanUseCredits(ctx context.Context, userIdentity string, tier Tier) (bool, int, error)

func (t *AICreditsTracker) LogRequest(ctx context.Context, req *AIRequestLog) error
```

#### 2.3 Update Entitlement Config
**File**: `api/config/config.go`

Add AI credits limits configuration:

```go
type EntitlementConfig struct {
    // ... existing fields ...

    // AICreditsLimits defines the AI credits limit per tier per calendar month.
    // Parsed from JSON: {"free": 0, "solo": 50, "pro": 500, "studio": 2000, "business": -1}
    // -1 means unlimited. 0 means no access.
    // Env: BAS_ENTITLEMENT_AI_CREDITS_LIMITS_JSON
    AICreditsLimits map[string]int
}
```

#### 2.4 Integrate AI Credits with Entitlement Service
**File**: `api/services/entitlement/service.go`

Add methods:

```go
// GetAICreditsForTier returns the AI credits limit for a tier.
func (s *Service) GetAICreditsForTier(tier Tier) int

// CanChargeAICredits checks if user has credits available and charges them.
func (s *Service) CanChargeAICredits(ctx context.Context, userIdentity string, creditsToCharge int) (bool, error)
```

---

### Phase 3: Backend API Key Integration

**Goal**: Allow frontend-provided OpenRouter key to be used by backend

#### 3.1 API Endpoint for AI Configuration
**File**: `api/handlers/ai_config.go` (new file)

```go
// POST /api/v1/ai/config - Set AI configuration (OpenRouter API key)
// GET /api/v1/ai/config - Get AI configuration status (not the key itself)
// GET /api/v1/ai/credits - Get AI credits usage for current user
// POST /api/v1/ai/test - Test OpenRouter key validity

type AIConfigRequest struct {
    OpenRouterAPIKey string `json:"openrouter_api_key"`
}

type AIConfigStatusResponse struct {
    Configured      bool   `json:"configured"`
    KeySource       string `json:"key_source"` // "user", "server", "none"
    CanUseAI        bool   `json:"can_use_ai"`
    RequiredTier    string `json:"required_tier,omitempty"`
}

type AICreditsResponse struct {
    UserIdentity     string    `json:"user_identity"`
    CreditsUsed      int       `json:"credits_used"`
    CreditsLimit     int       `json:"credits_limit"`      // -1 for unlimited
    CreditsRemaining int       `json:"credits_remaining"`  // -1 for unlimited
    RequestsCount    int       `json:"requests_count"`
    PeriodStart      time.Time `json:"period_start"`
    PeriodEnd        time.Time `json:"period_end"`
    ResetDate        time.Time `json:"reset_date"`
    Tier             string    `json:"tier"`
    HasAccess        bool      `json:"has_access"`
}
```

#### 3.2 Session-Based Key Storage
**File**: `api/services/ai/key_manager.go` (new file)

Store user-provided keys securely in session (not database for security):

```go
// KeyManager handles user-provided API keys with session-based storage.
// Keys are encrypted in memory and tied to session cookies.
type KeyManager struct {
    sessions sync.Map // sessionID -> encryptedKey
    cipher   cipher.AEAD
}

func (m *KeyManager) SetKey(sessionID, apiKey string) error
func (m *KeyManager) GetKey(sessionID string) (string, bool)
func (m *KeyManager) ClearKey(sessionID string)
func (m *KeyManager) HasKey(sessionID string) bool
```

#### 3.3 Update OpenRouter Client
**File**: `api/services/ai/client.go`

Modify to accept user-provided key:

```go
type OpenRouterClient struct {
    log        *logrus.Logger
    model      string
    keyManager *KeyManager
    serverKey  string  // Fallback server-side key
}

func (c *OpenRouterClient) ExecutePromptWithKey(ctx context.Context, prompt string, userKey string) (string, error)
```

---

### Phase 4: Frontend API Keys Section Redesign

**Goal**: Create clear, informative UI for AI configuration and usage

#### 4.1 New AI Settings Section
**File**: `ui/src/views/SettingsView/sections/AISettingsSection.tsx` (new file)

Replace `ApiKeysSection.tsx` with a comprehensive AI settings section:

```tsx
export function AISettingsSection() {
  return (
    <div className="space-y-6">
      {/* Header */}
      <SectionHeader
        icon={<Sparkles />}
        title="AI Features"
        description="Configure AI-powered workflow generation and analysis"
      />

      {/* Access Status Card */}
      <AIAccessStatusCard />

      {/* Usage Meter (if has access) */}
      <AIUsageMeter />

      {/* API Key Configuration */}
      <OpenRouterKeySection />

      {/* Tier Comparison / Upgrade Prompt */}
      <AITierComparisonSection />

      {/* BYOK Info */}
      <BYOKInfoSection />
    </div>
  );
}
```

#### 4.2 AI Access Status Card
**File**: `ui/src/views/SettingsView/sections/ai/AIAccessStatusCard.tsx` (new file)

Shows current AI access status based on tier and key configuration:

```tsx
interface AIAccessStatusCardProps {}

export function AIAccessStatusCard() {
  const { status } = useEntitlementStore();
  const aiCredits = useAICredits(); // New hook

  // States:
  // 1. Has tier access + has credits remaining = ✅ Full access
  // 2. Has tier access + no credits remaining = ⚠️ Credits exhausted
  // 3. No tier access but has personal key = ✅ Using personal key
  // 4. No tier access, no personal key = 🔒 Upgrade required

  return (
    <div className={`rounded-xl border ${borderColor} ${bgColor} p-6`}>
      <div className="flex items-center gap-3">
        <StatusIcon />
        <div>
          <h3 className="font-medium">{statusTitle}</h3>
          <p className="text-sm text-gray-400">{statusDescription}</p>
        </div>
      </div>
    </div>
  );
}
```

#### 4.3 AI Usage Meter
**File**: `ui/src/views/SettingsView/sections/ai/AIUsageMeter.tsx` (new file)

Visual usage display with progress bar and reset date:

```tsx
export function AIUsageMeter() {
  const aiCredits = useAICredits();

  if (!aiCredits || aiCredits.credits_limit === 0) {
    return null; // No access to show
  }

  const isUnlimited = aiCredits.credits_limit === -1;
  const percentage = isUnlimited ? 0 :
    Math.round((aiCredits.credits_used / aiCredits.credits_limit) * 100);

  return (
    <div className="rounded-xl border border-gray-700 bg-gray-800/50 p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="font-medium flex items-center gap-2">
          <Zap className="text-amber-400" />
          AI Credits
        </h3>
        <span className="text-sm text-gray-400">
          Resets {formatDate(aiCredits.reset_date)}
        </span>
      </div>

      {/* Progress Bar */}
      <div className="space-y-2">
        <div className="flex justify-between text-sm">
          <span>{aiCredits.credits_used} used</span>
          <span>{isUnlimited ? 'Unlimited' : `${aiCredits.credits_limit} limit`}</span>
        </div>
        <ProgressBar
          value={percentage}
          variant={percentage >= 90 ? 'danger' : percentage >= 75 ? 'warning' : 'default'}
        />
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-3 gap-4">
        <StatCard label="Used" value={aiCredits.credits_used} />
        <StatCard label="Remaining" value={isUnlimited ? '∞' : aiCredits.credits_remaining} />
        <StatCard label="Requests" value={aiCredits.requests_count} />
      </div>

      {/* Warning if approaching limit */}
      {!isUnlimited && percentage >= 80 && (
        <WarningBanner percentage={percentage} />
      )}
    </div>
  );
}
```

#### 4.4 OpenRouter Key Section
**File**: `ui/src/views/SettingsView/sections/ai/OpenRouterKeySection.tsx` (new file)

```tsx
export function OpenRouterKeySection() {
  const { apiKeys, setApiKey } = useSettingsStore();
  const [showKey, setShowKey] = useState(false);
  const [testStatus, setTestStatus] = useState<'idle' | 'testing' | 'valid' | 'invalid'>('idle');

  return (
    <div className="rounded-xl border border-gray-700 bg-gray-800/50 p-6 space-y-4">
      <h3 className="font-medium">OpenRouter API Key</h3>
      <p className="text-sm text-gray-400">
        Optional: Provide your own OpenRouter API key for AI features.
        This bypasses subscription limits.
      </p>

      <div className="relative">
        <input
          type={showKey ? 'text' : 'password'}
          value={apiKeys.openrouterApiKey}
          onChange={(e) => setApiKey('openrouterApiKey', e.target.value)}
          placeholder="sk-or-v1-..."
          className="..."
        />
        <VisibilityToggle show={showKey} onToggle={() => setShowKey(!showKey)} />
      </div>

      <div className="flex items-center gap-3">
        <button onClick={handleTestKey} disabled={!apiKeys.openrouterApiKey}>
          {testStatus === 'testing' ? <Loader2 className="animate-spin" /> : 'Test Key'}
        </button>
        <TestStatusIndicator status={testStatus} />
      </div>

      <p className="text-xs text-gray-500">
        Get an API key from <a href="https://openrouter.ai/keys" target="_blank">openrouter.ai/keys</a>
      </p>
    </div>
  );
}
```

#### 4.5 AI Tier Comparison Section
**File**: `ui/src/views/SettingsView/sections/ai/AITierComparisonSection.tsx` (new file)

Shows what's available at each tier with upgrade prompts:

```tsx
const AI_TIER_FEATURES = [
  { tier: 'free', credits: 0, label: 'No AI access' },
  { tier: 'solo', credits: 50, label: '50 credits/month' },
  { tier: 'pro', credits: 500, label: '500 credits/month', recommended: true },
  { tier: 'studio', credits: 2000, label: '2,000 credits/month' },
  { tier: 'business', credits: -1, label: 'Unlimited credits' },
];

export function AITierComparisonSection() {
  const currentTier = useCurrentTier();

  return (
    <div className="rounded-xl border border-gray-700 bg-gray-800/50 p-6 space-y-4">
      <h3 className="font-medium">AI Credits by Plan</h3>

      <div className="space-y-2">
        {AI_TIER_FEATURES.map((tier) => (
          <TierRow
            key={tier.tier}
            tier={tier.tier}
            label={tier.label}
            isCurrent={tier.tier === currentTier}
            isRecommended={tier.recommended}
          />
        ))}
      </div>

      {currentTier === 'free' && (
        <UpgradePrompt
          title="Unlock AI Features"
          description="Upgrade to Solo or higher to access AI-powered workflow generation"
          targetTier="solo"
        />
      )}
    </div>
  );
}
```

#### 4.6 BYOK Info Section
**File**: `ui/src/views/SettingsView/sections/ai/BYOKInfoSection.tsx` (new file)

```tsx
export function BYOKInfoSection() {
  return (
    <div className="rounded-xl border border-blue-800/50 bg-blue-900/20 p-6 space-y-3">
      <div className="flex items-center gap-2 text-blue-400">
        <Info size={20} />
        <h3 className="font-medium">Using Your Own Provider Keys</h3>
      </div>

      <p className="text-sm text-gray-300">
        OpenRouter supports <strong>Bring Your Own Key (BYOK)</strong> for providers like
        Anthropic, OpenAI, AWS Bedrock, Google Vertex, and Azure.
      </p>

      <ul className="text-sm text-gray-400 space-y-1 list-disc list-inside">
        <li>Add your provider keys in OpenRouter's dashboard</li>
        <li>First 1M BYOK requests/month are free</li>
        <li>Your keys are used automatically when routing to those providers</li>
      </ul>

      <a
        href="https://openrouter.ai/settings/keys"
        target="_blank"
        className="inline-flex items-center gap-1 text-sm text-blue-400 hover:text-blue-300"
      >
        Configure BYOK in OpenRouter <ExternalLink size={14} />
      </a>
    </div>
  );
}
```

---

### Phase 5: Frontend Stores and Hooks

#### 5.1 AI Credits Store
**File**: `ui/src/stores/aiCreditsStore.ts` (new file)

```typescript
interface AICreditsState {
  credits: AICreditsResponse | null;
  isLoading: boolean;
  error: string | null;
  lastFetched: Date | null;

  fetchCredits: () => Promise<void>;
  refreshCredits: () => Promise<void>;
}

interface AICreditsResponse {
  user_identity: string;
  credits_used: number;
  credits_limit: number;      // -1 for unlimited
  credits_remaining: number;  // -1 for unlimited
  requests_count: number;
  period_start: string;       // ISO date
  period_end: string;         // ISO date
  reset_date: string;         // ISO date
  tier: string;
  has_access: boolean;
}

export const useAICreditsStore = create<AICreditsState>((set, get) => ({
  // ... implementation
}));

// Convenience hook
export const useAICredits = () => {
  const store = useAICreditsStore();

  useEffect(() => {
    const shouldFetch = !store.isLoading &&
      (!store.lastFetched || Date.now() - store.lastFetched.getTime() > 60000);
    if (shouldFetch) {
      void store.fetchCredits();
    }
  }, [store.lastFetched]);

  return store.credits;
};
```

#### 5.2 Update aiCapabilityStore
**File**: `ui/src/stores/aiCapabilityStore.ts`

Enhance to check both tier access and credits:

```typescript
export interface AICapability {
  available: boolean;
  reason: 'has_credits' | 'has_api_key' | 'disabled' | 'no_credits' | 'no_tier_access' | 'checking' | 'error';
  creditsRemaining?: number;
  creditsLimit?: number;
  apiKeyConfigured?: boolean;
  requiredTier?: string;
  lastChecked?: Date;
}
```

---

### Phase 6: Update Entitlement Status Response

#### 6.1 Backend Response Enhancement
**File**: `api/handlers/entitlement.go`

Add AI credits to the status response:

```go
type EntitlementStatusResponse struct {
    // ... existing fields ...

    // AI-specific fields
    AICreditsUsed      int    `json:"ai_credits_used"`
    AICreditsLimit     int    `json:"ai_credits_limit"`      // -1 for unlimited
    AICreditsRemaining int    `json:"ai_credits_remaining"`  // -1 for unlimited
    AIRequestsCount    int    `json:"ai_requests_count"`
    AIResetDate        string `json:"ai_reset_date"`         // ISO date
    HasAIAPIKey        bool   `json:"has_ai_api_key"`
}
```

#### 6.2 Frontend Type Updates
**File**: `ui/src/stores/entitlementStore.ts`

```typescript
export interface EntitlementStatusResponse {
  // ... existing fields ...

  // AI credits
  ai_credits_used: number;
  ai_credits_limit: number;      // -1 for unlimited
  ai_credits_remaining: number;  // -1 for unlimited
  ai_requests_count: number;
  ai_reset_date: string;
  has_ai_api_key: boolean;
}
```

---

### Phase 7: Integration Points

#### 7.1 Charge Credits on AI Operations
**Files to modify**:
- `api/handlers/ai/workflow_generate.go` - Charge credits on workflow generation
- `api/handlers/ai/element_analysis.go` - Charge credits on element analysis
- `api/handlers/ai/` (any other AI handlers)

```go
// Before executing AI operation:
if err := h.entitlementSvc.CanChargeAICredits(ctx, userIdentity, 1); err != nil {
    return fmt.Errorf("AI credits: %w", err)
}

// After successful execution:
if err := h.aiCreditsTracker.ChargeCredits(ctx, userIdentity, 1, "workflow_generate", model); err != nil {
    h.log.WithError(err).Warn("Failed to charge AI credits")
}
```

#### 7.2 Check Credits Before AI Features
Update middleware to check AI access:

**File**: `api/middleware/entitlement.go`

```go
// AIGate middleware checks both tier access AND credits availability
func AIGate(svc *entitlement.Service, tracker *entitlement.AICreditsTracker) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userIdentity := entitlement.GetUserIdentity(r.Context())

            // Check tier access
            if !svc.CanUseAI(r.Context(), userIdentity) {
                respondError(w, 403, "AI features require a Pro subscription or higher")
                return
            }

            // Check credits
            canUse, remaining, err := tracker.CanUseCredits(r.Context(), userIdentity)
            if err != nil {
                respondError(w, 500, "Failed to check AI credits")
                return
            }
            if !canUse {
                respondError(w, 403, "AI credits exhausted for this billing period")
                return
            }

            // Add remaining credits to context for handlers
            ctx := context.WithValue(r.Context(), "ai_credits_remaining", remaining)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

---

## File Summary

### New Files
| File | Purpose |
|------|---------|
| `api/services/entitlement/ai_credits.go` | AI credits tracking service |
| `api/services/ai/key_manager.go` | Session-based API key management |
| `api/handlers/ai_config.go` | AI configuration endpoints |
| `ui/src/stores/aiCreditsStore.ts` | AI credits state management |
| `ui/src/views/SettingsView/sections/AISettingsSection.tsx` | Main AI settings component |
| `ui/src/views/SettingsView/sections/ai/AIAccessStatusCard.tsx` | Access status display |
| `ui/src/views/SettingsView/sections/ai/AIUsageMeter.tsx` | Credits usage meter |
| `ui/src/views/SettingsView/sections/ai/OpenRouterKeySection.tsx` | API key input |
| `ui/src/views/SettingsView/sections/ai/AITierComparisonSection.tsx` | Tier comparison |
| `ui/src/views/SettingsView/sections/ai/BYOKInfoSection.tsx` | BYOK documentation |

### Modified Files
| File | Changes |
|------|---------|
| `initialization/storage/postgres/schema.sql` | Add AI credits tables |
| `api/config/config.go` | Add AI credits limits config |
| `api/services/entitlement/service.go` | Add AI credits methods |
| `api/services/ai/client.go` | Support user-provided keys |
| `api/handlers/entitlement.go` | Include AI credits in status |
| `api/middleware/entitlement.go` | Add AI gate middleware |
| `ui/src/stores/settingsStore.ts` | New ApiKeySettings interface |
| `ui/src/stores/entitlementStore.ts` | Add AI credits types |
| `ui/src/stores/aiCapabilityStore.ts` | Enhanced capability checking |
| `ui/src/views/SettingsView/SettingsView.tsx` | Replace ApiKeysSection with AISettingsSection |

### Deleted Files
| File | Reason |
|------|--------|
| `ui/src/views/SettingsView/sections/ApiKeysSection.tsx` | Replaced by AISettingsSection |

---

## Configuration

### Environment Variables

```bash
# AI Credits Limits (JSON format)
# {"free": 0, "solo": 50, "pro": 500, "studio": 2000, "business": -1}
BAS_ENTITLEMENT_AI_CREDITS_LIMITS_JSON='{"free":0,"solo":50,"pro":500,"studio":2000,"business":-1}'

# Credit costs per operation type (optional fine-tuning)
BAS_AI_CREDIT_COST_WORKFLOW_GENERATE=1
BAS_AI_CREDIT_COST_ELEMENT_ANALYZE=1
BAS_AI_CREDIT_COST_DOM_EXTRACT=1
```

### Default Tier Limits

| Tier | AI Credits/Month | Notes |
|------|-----------------|-------|
| Free | 0 | No AI access |
| Solo | 50 | Basic usage |
| Pro | 500 | Power users |
| Studio | 2,000 | Teams |
| Business | Unlimited | Enterprise |

---

## Testing Checklist

### Unit Tests
- [ ] `ai_credits.go` - Credit tracking logic
- [ ] `key_manager.go` - Key storage/retrieval
- [ ] `aiCreditsStore.ts` - Store actions
- [ ] `settingsStore.ts` - Migration logic

### Integration Tests
- [ ] API endpoints return correct credits
- [ ] Credits decrement on AI operations
- [ ] Rate limiting when credits exhausted
- [ ] Personal API key bypasses limits
- [ ] Reset date calculation

### E2E Tests
- [ ] Settings page displays correct usage
- [ ] Progress bar updates after AI operations
- [ ] Upgrade prompts show at correct times
- [ ] API key test functionality works
- [ ] BYOK link opens correctly

---

## Migration Notes

1. **Existing Users**: The localStorage migration in `settingsStore.ts` handles clearing old obsolete keys
2. **Database**: Run schema migration to create new tables before deploying
3. **Backwards Compatibility**: Old API key fields are simply ignored; no breaking changes
4. **Feature Flags**: Consider gating new AI UI behind feature flag for gradual rollout

---

## Future Enhancements

1. **Credit Top-ups**: Allow purchasing additional credits mid-cycle
2. **Usage Alerts**: Email notifications when approaching limits
3. **Team Credits**: Shared credit pools for studio/business tiers
4. **Credit Rollover**: Carry unused credits to next month (optional feature)
5. **Usage Analytics**: Dashboard showing credit usage patterns
