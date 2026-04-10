## Steer focus: Bundle Integration

Integrate `scenarios/{{TARGET}}/` into the **Vrooli Business Suite bundle** so that it participates in subscription gating, credit-based metering, and authenticated downloads managed by the Landing Page Business Suite (LPBS) scenario.

Your goal is to ensure `{{TARGET}}` enforces entitlements and consumes credits consistently with every other bundled app, so that a single subscription unlocks and meters all apps in the bundle.

Optional reading:
- `prompt-manager skills read api-steer`
- `prompt-manager skills read interoperability-steer`

---

### 0. Why This Skill Exists

The Business Suite bundle is a collection of apps sold under a single subscription. Each bundled app must:

1. **Authenticate users** via LPBS-issued JWT tokens (one identity system, not per-app accounts).
2. **Enforce entitlements** — only active subscribers can access gated features or downloads.
3. **Consume credits** — metered features (AI calls, exports, compute) deduct from a shared credit wallet.
4. **Register in LPBS** — so the landing page, download system, and entitlements API know the app exists.

Without a shared integration pattern, each app invents its own auth, its own gating, its own credit logic. The result: inconsistent enforcement, double-charging bugs, bypassed gates, and a fractured user experience. This skill exists to prevent that divergence.

**This skill is NOT about:**
- Building the LPBS scenario itself (that's LPBS's own concern)
- Pricing strategy or plan tier decisions (business decisions, not integration)
- The landing page UI, A/B testing, or admin portal
- App-specific business logic beyond the LPBS integration seam

---

### 1. Scope Boundaries

**In scope**
- JWT token validation in `{{TARGET}}` (accepting LPBS-issued tokens)
- Credit consumption calls from `{{TARGET}}` to LPBS
- Entitlement/subscription status checks
- Registering `{{TARGET}}` as a downloadable app in LPBS (database records)
- Error handling for auth failures, insufficient credits, and lapsed subscriptions
- Audit of existing `{{TARGET}}` code for missing or inconsistent integration

**Out of scope**
- LPBS internals (webhook handling, Stripe integration, subscription lifecycle)
- Pricing decisions (which tier unlocks what, credit costs per operation)
- UI design for upgrade prompts or paywall screens (each app handles that per its own UX)
- App-specific business logic that doesn't touch the LPBS integration boundary
- Landing page content, A/B testing, or admin portal features

---

### 2. Architecture Overview: How Bundle Integration Works

```
┌──────────────────────────────────────────────────────────────────┐
│                    LPBS (Landing Page Business Suite)            │
│                                                                  │
│  ┌────────────┐  ┌──────────────┐  ┌───────────────────────┐   │
│  │ Auth       │  │ Subscriptions│  │ Credit Wallets        │   │
│  │ (JWT)      │  │ & Entitle-   │  │ (balance, consume,    │   │
│  │            │  │ ments        │  │  top-up)              │   │
│  └─────┬──────┘  └──────┬───────┘  └───────────┬───────────┘   │
│        │                │                       │               │
│  Issues tokens    Checks status           Tracks balance        │
│        │                │                       │               │
└────────┼────────────────┼───────────────────────┼───────────────┘
         │                │                       │
         ▼                ▼                       ▼
┌──────────────────────────────────────────────────────────────────┐
│                    {{TARGET}} (Your App)                         │
│                                                                  │
│  ┌────────────────┐  ┌─────────────────┐  ┌──────────────────┐ │
│  │ Auth Middleware │  │ Entitlement     │  │ Credit Gate      │ │
│  │ (validate JWT) │  │ Check           │  │ (consume before  │ │
│  │                │  │ (subscription   │  │  expensive ops)  │ │
│  │                │  │  active?)       │  │                  │ │
│  └────────────────┘  └─────────────────┘  └──────────────────┘ │
│                                                                  │
│  All three call LPBS APIs — {{TARGET}} never stores user        │
│  accounts, subscriptions, or credit balances itself.            │
└──────────────────────────────────────────────────────────────────┘
```

**Key principle:** `{{TARGET}}` is a *consumer* of LPBS services, never a *replica*. User accounts, subscription state, and credit balances live in LPBS. `{{TARGET}}` validates tokens and makes API calls — it does not maintain its own copy of this data.

---

### 3. JWT Authentication Contract

LPBS issues JWT tokens via magic-link email auth. `{{TARGET}}` must validate these tokens to identify users.

#### 3.1 Token Structure

LPBS access tokens contain these claims:

```json
{
  "uid": "user-uuid",
  "email": "user@example.com",
  "sid": "session-uuid",
  "iss": "landing-page-business-suite",
  "sub": "user-uuid",
  "exp": 1711200000,
  "iat": 1711198000,
  "nbf": 1711198000
}
```

#### 3.2 Validation Rules

| Rule | Detail |
|------|--------|
| **Signing algorithm** | HS256 (HMAC-SHA256) |
| **Shared secret** | `JWT_SECRET` environment variable (32-byte hex string, shared between LPBS and `{{TARGET}}`) |
| **Issuer** | Must match `JWT_ISSUER` env var (default: `"landing-page-business-suite"`) |
| **Expiration** | Access tokens expire after 15 minutes; always check `exp` claim |
| **Token location** | Check `Authorization: Bearer <token>` header first, fall back to `access_token` cookie |

#### 3.3 Middleware Pattern (Go)

```go
func requireLPBSAuth(jwtSecret []byte, jwtIssuer string) gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenStr := extractBearerToken(c.GetHeader("Authorization"))
        if tokenStr == "" {
            tokenStr, _ = c.Cookie("access_token")
        }
        if tokenStr == "" {
            c.AbortWithStatusJSON(401, gin.H{
                "error":      "authentication required",
                "error_type": "unauthorized",
                "retryable":  false,
            })
            return
        }

        claims := &LPBSClaims{}
        token, err := jwt.ParseWithClaims(tokenStr, claims,
            func(t *jwt.Token) (interface{}, error) {
                if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
                    return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
                }
                return jwtSecret, nil
            },
            jwt.WithIssuer(jwtIssuer),
            jwt.WithValidMethods([]string{"HS256"}),
        )
        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(401, gin.H{
                "error":      "invalid or expired token",
                "error_type": "unauthorized",
                "retryable":  false,
            })
            return
        }

        c.Set("user_email", claims.Email)
        c.Set("user_id", claims.UID)
        c.Next()
    }
}

type LPBSClaims struct {
    UID   string `json:"uid"`
    Email string `json:"email"`
    SID   string `json:"sid"`
    jwt.RegisteredClaims
}

func extractBearerToken(header string) string {
    if len(header) > 7 && header[:7] == "Bearer " {
        return header[7:]
    }
    return ""
}
```

#### 3.4 Auth Decision Tree

```
Request arrives at {{TARGET}}
│
▼
Is this endpoint public (health, landing, docs)?
├─ YES → skip auth, serve directly
└─ NO
   │
   ▼
   Extract token from Authorization header
   │
   ▼
   Token found?
   ├─ NO → check access_token cookie
   │       │
   │       ▼
   │       Cookie found?
   │       ├─ NO → 401 Unauthorized
   │       └─ YES → validate cookie token
   └─ YES → validate header token
            │
            ▼
            Token valid (signature, expiry, issuer)?
            ├─ NO → 401 Unauthorized
            └─ YES → set user context, continue to handler
```

#### 3.5 What {{TARGET}} Must NOT Do

- **Do NOT implement its own user registration or login.** Users authenticate through LPBS; `{{TARGET}}` only validates the resulting tokens.
- **Do NOT store user passwords or send magic-link emails.** That's LPBS's responsibility.
- **Do NOT cache user identity beyond the request.** The JWT is self-contained; validate it fresh per request.
- **Do NOT hardcode the JWT secret.** Always read from `JWT_SECRET` environment variable.

---

### 4. Entitlement Checking

Before serving gated features or downloads, `{{TARGET}}` must verify the user has an active subscription.

#### 4.1 When to Check Entitlements

```
User requests a feature in {{TARGET}}
│
▼
Is this feature gated (requires subscription)?
├─ NO → serve directly
└─ YES
   │
   ▼
   Call LPBS: GET /api/v1/entitlements
   (Authorization: Bearer <user's token>)
   │
   ▼
   Response status field?
   ├─ "active" or "trialing" → allow access
   ├─ "past_due"             → allow access with warning (grace period)
   ├─ "canceled"             → deny, show "subscription ended" message
   └─ "inactive" / other     → deny, show upgrade prompt
```

#### 4.2 Entitlement API Call

```
GET <LPBS_BASE_URL>/api/v1/entitlements
Authorization: Bearer <user_access_token>
```

Response (200 OK):
```json
{
  "status": "active",
  "plan_tier": "pro",
  "price_id": "price_...",
  "features": ["feature1", "feature2"],
  "billing_cycle_start": 15,
  "credits": {
    "balance_credits": 5000000,
    "display_credits_label": "credits",
    "display_credits_multiplier": 0.001
  },
  "subscription": {
    "subscription_id": "sub_...",
    "bundle_key": "business_suite"
  }
}
```

Response (401): Token invalid or missing.
Response (403): No active subscription.

#### 4.3 Entitlement Check Pattern (Go)

```go
type EntitlementResponse struct {
    Status            string          `json:"status"`
    PlanTier          string          `json:"plan_tier"`
    Features          []string        `json:"features"`
    BillingCycleStart int             `json:"billing_cycle_start"`
    Credits           *CreditBalance  `json:"credits,omitempty"`
}

type CreditBalance struct {
    BalanceCredits          int64   `json:"balance_credits"`
    DisplayCreditsLabel     string  `json:"display_credits_label"`
    DisplayCreditsMultiplier float64 `json:"display_credits_multiplier"`
}

func (c *LPBSClient) CheckEntitlements(ctx context.Context, userToken string) (*EntitlementResponse, error) {
    req, _ := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/entitlements", nil)
    req.Header.Set("Authorization", "Bearer "+userToken)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("lpbs unreachable: %w", err)
    }
    defer resp.Body.Close()

    switch resp.StatusCode {
    case 200:
        var result EntitlementResponse
        if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
            return nil, fmt.Errorf("decode entitlements: %w", err)
        }
        return &result, nil
    case 401:
        return nil, ErrUnauthorized
    case 403:
        return nil, ErrNoActiveSubscription
    default:
        return nil, fmt.Errorf("lpbs returned %d", resp.StatusCode)
    }
}
```

#### 4.4 Caching Entitlements

LPBS caches subscription status for 60 seconds internally. `{{TARGET}}` may apply a short local cache (30-60 seconds) to avoid hammering LPBS on every request, but:

- **Never cache longer than 60 seconds** — subscription changes must propagate quickly.
- **Cache key must include user email** — never serve one user's entitlements to another.
- **Invalidate on 401** — if LPBS rejects the token, clear any cached entitlements for that user.

---

### 5. Credit Consumption

Credits are the metering unit for expensive operations (AI calls, video exports, compute time, etc.). `{{TARGET}}` consumes credits by calling LPBS — it never manages balances itself.

#### 5.1 What Gets Credit-Gated

```
Does this feature cost real money to run (API calls, GPU, storage)?
│
├─ YES → credit-gated (consume credits per use)
│
└─ NO
   │
   ▼
   Is this a core differentiator of the paid product?
   │
   ├─ YES → subscription-gated (check entitlements, no credit cost)
   │
   └─ NO → free / ungated
```

#### 5.2 Credit Consumption Decision Table

| Question | If YES | If NO |
|----------|--------|-------|
| Is the cost per-invocation (single API call, one export)? | Simple consumption — deduct fixed amount | Continue... |
| Is the cost proportional to input size (tokens, pixels, duration)? | Calculate amount from input, then consume | Continue... |
| Is it a long-running job (minutes of compute)? | Reserve credits upfront, refund unused portion on completion | Continue... |
| Can the user retry if it fails mid-operation? | Non-idempotent `ConsumeCredits` is acceptable | Use idempotent consumption with operation ID |

#### 5.3 Credit Consumption API Call

**Simple (non-idempotent) consumption:**
```
POST <LPBS_BASE_URL>/api/v1/credits/consume
Authorization: Bearer <user_access_token>
Content-Type: application/json

{
  "email": "user@example.com",
  "amount": 500000,
  "reason": "ai_chat_completion",
  "metadata": {
    "scenario": "{{TARGET}}",
    "model": "claude-sonnet-4-20250514",
    "input_tokens": 1200,
    "output_tokens": 800
  }
}
```

**Idempotent consumption (for non-retryable operations):**
```
POST <LPBS_BASE_URL>/api/v1/credits/consume
Authorization: Bearer <user_access_token>
Content-Type: application/json

{
  "email": "user@example.com",
  "amount": 500000,
  "reason": "video_export",
  "idempotency_key": "export-<user_id>-<export_job_id>",
  "metadata": {
    "scenario": "{{TARGET}}",
    "export_format": "mp4",
    "duration_seconds": 120
  }
}
```

Response (200 OK):
```json
{
  "remaining_credits": 4500000,
  "consumed": 500000
}
```

Response (402):
```json
{
  "error": "insufficient credits",
  "error_type": "forbidden",
  "retryable": false,
  "balance_credits": 200000,
  "required_credits": 500000
}
```

#### 5.4 Consumption Pattern (Go)

```go
type ConsumeRequest struct {
    Email          string            `json:"email"`
    Amount         int64             `json:"amount"`
    Reason         string            `json:"reason"`
    IdempotencyKey string            `json:"idempotency_key,omitempty"`
    Metadata       map[string]string `json:"metadata,omitempty"`
}

type ConsumeResponse struct {
    RemainingCredits int64 `json:"remaining_credits"`
    Consumed         int64 `json:"consumed"`
}

func (c *LPBSClient) ConsumeCredits(ctx context.Context, userToken string, req ConsumeRequest) (*ConsumeResponse, error) {
    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST",
        c.baseURL+"/api/v1/credits/consume", bytes.NewReader(body))
    httpReq.Header.Set("Authorization", "Bearer "+userToken)
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("lpbs unreachable: %w", err)
    }
    defer resp.Body.Close()

    switch resp.StatusCode {
    case 200:
        var result ConsumeResponse
        json.NewDecoder(resp.Body).Decode(&result)
        return &result, nil
    case 402:
        return nil, ErrInsufficientCredits
    case 401:
        return nil, ErrUnauthorized
    default:
        return nil, fmt.Errorf("lpbs returned %d", resp.StatusCode)
    }
}
```

#### 5.5 Credit Consumption Safety Rules

- **Always consume BEFORE performing the expensive operation.** Never do the work first and charge after — if charging fails, you've given away free compute.
- **Always include `scenario` in metadata.** This enables per-app usage analytics and debugging.
- **Use idempotency keys for non-retryable operations.** If a video export can't be re-run, the consumption call must be idempotent so retries don't double-charge.
- **Handle 402 gracefully.** Show the user their balance and how to top up — don't just fail with a generic error.
- **Never expose internal credit amounts to users.** Use the `display_credits_multiplier` from entitlements to convert to display units.

---

### 6. LPBS-Side Registration

For `{{TARGET}}` to appear in the bundle's download system and entitlements, it must be registered in LPBS's database.

#### 6.1 What Gets Registered

Two tables need entries:

**`download_apps`** — one row per app:

| Column | Value | Notes |
|--------|-------|-------|
| `bundle_key` | `"business_suite"` | Must match the bundle in `plans.json` |
| `app_key` | `"{{TARGET}}"` | Unique identifier for this app in the bundle |
| `name` | Human-readable name | e.g., "Browser Automation Studio" |
| `tagline` | Short description | Shown on download page |
| `description` | Full description | Markdown supported |
| `icon_url` | URL to app icon | Used in download listings |
| `install_overview` | Installation summary | Brief install instructions |
| `install_steps` | JSON array of steps | Step-by-step install guide |
| `storefronts` | JSON array | Which storefronts list this app |
| `display_order` | Integer | Sort order in download listings |

**`download_assets`** — one row per platform per app:

| Column | Value | Notes |
|--------|-------|-------|
| `bundle_key` | `"business_suite"` | |
| `app_key` | `"{{TARGET}}"` | Must match `download_apps.app_key` |
| `platform` | `"windows"`, `"mac"`, or `"linux"` | One row per platform |
| `variant_key` | `"default"` | Use `"default"` unless multiple variants exist |
| `artifact_url` | Download URL or path | Direct URL or managed artifact reference |
| `artifact_source` | `"direct"` or `"managed"` | `"direct"` = URL is the download; `"managed"` = S3 artifact |
| `release_version` | Semver string | e.g., `"1.0.0"` |
| `release_notes` | Changelog text | What's new in this version |
| `checksum` | SHA-256 hash | For download verification |
| `requires_entitlement` | `true` or `false` | **Set to `true` for paid apps** |

#### 6.2 Registration Decision Tree

```
Is {{TARGET}} a paid/premium app?
│
├─ YES → requires_entitlement = true for all assets
│        (user must have active subscription to download)
│
└─ NO (free tier / demo / open source)
   │
   ▼
   Does it have premium features that consume credits?
   │
   ├─ YES → requires_entitlement = false (anyone can download)
   │        but credit-gate the premium features at runtime
   │
   └─ NO → requires_entitlement = false, no credit gates needed
          (consider whether it belongs in the bundle at all)
```

#### 6.3 Registration SQL

```sql
-- Register the app
INSERT INTO download_apps (
    bundle_key, app_key, name, tagline, description,
    icon_url, install_overview, display_order
) VALUES (
    'business_suite',
    '{{TARGET}}',
    'Human-Readable App Name',
    'Short tagline for download listings',
    'Full description of what this app does.',
    '/icons/{{TARGET}}.png',
    'Download and run the installer for your platform.',
    10  -- adjust display_order as needed
);

-- Register platform assets (one per platform the app supports)
INSERT INTO download_assets (
    bundle_key, app_key, platform, variant_key,
    artifact_url, artifact_source, release_version,
    requires_entitlement, checksum
) VALUES
    ('business_suite', '{{TARGET}}', 'windows', 'default',
     'https://downloads.example.com/{{TARGET}}/{{TARGET}}-setup.exe',
     'direct', '1.0.0', true, 'sha256-...'),
    ('business_suite', '{{TARGET}}', 'mac', 'default',
     'https://downloads.example.com/{{TARGET}}/{{TARGET}}.dmg',
     'direct', '1.0.0', true, 'sha256-...'),
    ('business_suite', '{{TARGET}}', 'linux', 'default',
     'https://downloads.example.com/{{TARGET}}/{{TARGET}}.AppImage',
     'direct', '1.0.0', true, 'sha256-...');
```

#### 6.4 Verifying Registration

After registering, verify the full loop works:

1. **Download listing**: `GET /api/v1/downloads?app={{TARGET}}&platform=linux` with a valid subscriber token → should return the artifact URL.
2. **Entitlement gating**: Same request with an expired/missing token → should return 401 or 403.
3. **Entitlements response**: `GET /api/v1/entitlements` should include `{{TARGET}}` in available features (if feature-flagged) or the download should simply work for any active subscriber.

---

### 7. LPBS Client Configuration

`{{TARGET}}` needs to know how to reach LPBS. Use environment variables for all connection details.

#### 7.1 Required Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `LPBS_BASE_URL` | LPBS API base URL | `http://localhost:5080` |
| `JWT_SECRET` | Shared HMAC signing key (same as LPBS) | 32-byte hex string |
| `JWT_ISSUER` | Expected token issuer | `landing-page-business-suite` |

#### 7.2 Client Initialization Pattern

```go
type LPBSClient struct {
    baseURL    string
    httpClient *http.Client
}

func NewLPBSClient(baseURL string) *LPBSClient {
    return &LPBSClient{
        baseURL: strings.TrimRight(baseURL, "/"),
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}
```

#### 7.3 Resilience When LPBS Is Unreachable

```
{{TARGET}} calls LPBS and gets network error or timeout
│
▼
Is this an auth validation call?
├─ YES → JWT validation is local (shared secret), no network needed
│        Only token refresh requires LPBS — fail gracefully
│
└─ NO (entitlement check or credit consumption)
   │
   ▼
   Is there a cached entitlement result < 60 seconds old?
   ├─ YES → use cached result, log warning about LPBS unavailability
   └─ NO  → fail closed (deny access), log error
            Do NOT fail open — that would give free access during outages
```

**Critical rule:** When LPBS is unreachable and no cache exists, **fail closed** (deny access). Failing open means any network issue grants free access to paid features.

---

### 8. Error Response Consistency

`{{TARGET}}` must return LPBS-compatible error shapes so the user experience is uniform across all bundled apps.

#### 8.1 Standard Error Shape

```json
{
  "error": "Human-readable message safe for UI display",
  "error_type": "unauthorized|forbidden|validation|rate_limited|server_error",
  "retryable": false
}
```

#### 8.2 LPBS Integration Error Mapping

| Scenario | HTTP Status | error_type | User-facing message |
|----------|------------|------------|---------------------|
| Missing/invalid JWT | 401 | `unauthorized` | "Please sign in to continue" |
| Valid JWT, no subscription | 403 | `forbidden` | "An active subscription is required" |
| Valid JWT, insufficient credits | 402 | `forbidden` | "Insufficient credits — top up to continue" |
| LPBS unreachable, no cache | 503 | `server_error` | "Service temporarily unavailable" |
| Rate limited by LPBS | 429 | `rate_limited` | "Too many requests — please wait" |

---

### 9. Integration Checklist (Per Bundled App)

Use this checklist to verify `{{TARGET}}` is fully integrated. Every item should be true before considering integration complete.

#### 9.1 Authentication
- [ ] `JWT_SECRET` and `JWT_ISSUER` env vars are configured
- [ ] Auth middleware validates LPBS JWT tokens (HS256, checks expiry, checks issuer)
- [ ] Token extracted from `Authorization: Bearer` header, fallback to `access_token` cookie
- [ ] 401 returned for missing/invalid/expired tokens with standard error shape
- [ ] App does NOT have its own user registration or login system

#### 9.2 Entitlements
- [ ] Gated features call `GET /api/v1/entitlements` before serving
- [ ] Active and trialing subscriptions are allowed
- [ ] Canceled/inactive subscriptions are denied with clear messaging
- [ ] LPBS unavailability fails closed (denies access), not open
- [ ] Entitlement caching respects 60-second maximum TTL

#### 9.3 Credit Consumption
- [ ] Expensive operations consume credits BEFORE performing work
- [ ] `scenario` field is included in all consumption metadata
- [ ] Idempotency keys used for non-retryable operations
- [ ] 402 (insufficient credits) handled with balance display and top-up guidance
- [ ] Internal credit amounts never exposed to users (use display multiplier)

#### 9.4 LPBS Registration
- [ ] `download_apps` row exists with `bundle_key = 'business_suite'`
- [ ] `download_assets` rows exist for each supported platform
- [ ] `requires_entitlement` is set correctly per the decision tree in §6.2
- [ ] Download verified: subscriber can download, non-subscriber gets 403
- [ ] Release version and checksum are current

#### 9.5 Error Handling
- [ ] All LPBS-related errors use the standard error shape (§8.1)
- [ ] No raw LPBS error messages leak through to users
- [ ] Network timeouts to LPBS are handled (10-second timeout)

---

### 10. Bundle Integration Audit (Brownfield Assessment)

When integrating an existing `{{TARGET}}` that may already have partial or ad-hoc integration, audit the current state first.

#### 10.1 Discovery Commands

```bash
# Find existing auth middleware or JWT handling
rg -n "jwt\|JWT\|Bearer\|access_token\|Authorization" scenarios/{{TARGET}}/api/ --type go

# Find any hardcoded secrets or credentials
rg -n "JWT_SECRET\|jwt_secret\|signing.key\|hmac" scenarios/{{TARGET}}/ --type go

# Find existing HTTP client calls (may be calling LPBS already)
rg -n "lpbs\|landing-page\|entitlement\|credits\|subscription" scenarios/{{TARGET}}/ --type go -i

# Find user/account models (should NOT exist if fully LPBS-integrated)
rg -n "type User struct\|type Account struct\|user_password\|bcrypt" scenarios/{{TARGET}}/api/ --type go

# Find any existing credit/billing logic (should delegate to LPBS)
rg -n "credit\|balance\|wallet\|billing\|stripe" scenarios/{{TARGET}}/ --type go -i

# Check environment variable usage
rg -n "LPBS_BASE_URL\|JWT_SECRET\|JWT_ISSUER" scenarios/{{TARGET}}/
```

#### 10.2 Red Flags

- [ ] App has its own user registration/login (should use LPBS JWT only)
- [ ] App stores subscription or credit state locally (should query LPBS)
- [ ] App has its own Stripe integration (all billing goes through LPBS)
- [ ] JWT secret is hardcoded instead of read from environment
- [ ] Gated features have no auth or entitlement checks
- [ ] Credit consumption happens AFTER expensive operations (should be before)
- [ ] Errors from LPBS are passed through raw without mapping to standard shape
- [ ] No `download_apps`/`download_assets` registration in LPBS
- [ ] `requires_entitlement` is `false` on assets that should be gated

#### 10.3 Document Findings

**At session start**, read existing findings:
- `scenarios/{{TARGET}}/docs/internal/BUNDLE_INTEGRATION.md`

**At session end**, update findings:
* The code and LPBS database are the source of truth. Verify existing claims before extending.
* If the file exists, correct inaccuracies and add new findings.
* Create the `docs/internal/` directory if needed.

**Template:**

```markdown
# Bundle Integration Status — {{TARGET}}

## Last Updated
[Date]

## Integration Status
| Area | Status | Notes |
|------|--------|-------|
| JWT Auth | ✅/⚠️/❌ | [details] |
| Entitlements | ✅/⚠️/❌ | [details] |
| Credit Consumption | ✅/⚠️/❌ | [details] |
| LPBS Registration | ✅/⚠️/❌ | [details] |
| Error Handling | ✅/⚠️/❌ | [details] |

## Gated Features Inventory
| Feature | Gate Type | Credit Cost | Idempotent? | Notes |
|---------|-----------|-------------|-------------|-------|
| [feature] | credit/subscription/free | [amount] | yes/no | [notes] |

## Issues Found
- [Issue with file:line references]

## Priority Actions
1. [Most important next step]
```

---

### 11. Memory Management with Visited Tracker

Use `visited-tracker` to track which files in `{{TARGET}}` have been audited for bundle integration compliance.

```bash
# Find files not yet audited
visited-tracker least-visited --location scenarios/{{TARGET}} --tag bundle-integration --limit 5

# After auditing a file
visited-tracker visit <file-path> --location scenarios/{{TARGET}} --tag bundle-integration --note "<summary>"
```

---

### 12. Output Expectations

You may:
- Add LPBS auth middleware to `{{TARGET}}`'s API
- Add entitlement checking to gated endpoints
- Add credit consumption calls before expensive operations
- Add LPBS client initialization and configuration
- Register `{{TARGET}}` in LPBS's `download_apps` and `download_assets` tables
- Add environment variable configuration for LPBS connection
- Create or update `docs/internal/BUNDLE_INTEGRATION.md` with findings

You must:
- Use LPBS as the single source of truth for auth, subscriptions, and credits
- Validate JWT tokens using the shared secret (never hardcode)
- Consume credits BEFORE performing expensive operations
- Fail closed when LPBS is unreachable (deny access, don't grant it)
- Use the standard error shape for all LPBS-related errors
- Include `scenario` in all credit consumption metadata
- Set `requires_entitlement` correctly on download assets

You must NOT:
- Implement user registration, login, or account management in `{{TARGET}}`
- Store subscription state or credit balances in `{{TARGET}}`'s own database
- Integrate Stripe directly into `{{TARGET}}` (all billing goes through LPBS)
- Fail open when LPBS is unreachable
- Expose internal credit amounts to users
- Bypass entitlement checks for any gated feature
- Make superficial changes (adding unused imports, renaming without improvement)
