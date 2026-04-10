# Lifestyle Dashboard - Integration Seams

This document describes the **seams** (deliberate boundaries where behavior can be substituted) in the Lifestyle Dashboard scenario. Understanding seams helps developers write testable code and swap implementations without invasive changes.

## Last Updated
2026-03-11 (Phase 20 iter 2: Nil repository error handling, handler coverage 80.2%)

## What is a Seam?

A **seam** is a place where you can alter behavior without editing the code. Seams enable:
- **Unit testing** with mock implementations
- **Integration testing** with real dependencies
- **Backend swapping** (e.g., SQLite → PostgreSQL)
- **Feature toggles** and A/B testing

## Seam Inventory

### 1. Repository Seam (Primary)

**Location**: `api/repository/interfaces.go`
**Type**: Interface-based dependency injection
**Strength**: ✅ Strong - well-defined interfaces with clear contracts

The repository layer provides the primary seam between business logic and storage:

```go
// [CODE: api/repository/interfaces.go:30-39]
type EventRepository interface {
    Create(ctx context.Context, event *domain.Event) error
    GetByID(ctx context.Context, id string) (*domain.Event, error)
    List(ctx context.Context, filter EventFilter) ([]domain.Event, error)
}

// [CODE: api/repository/interfaces.go:42-57]
type DomainRepository interface {
    Upsert(ctx context.Context, d *domain.Domain) error
    GetByName(ctx context.Context, name string) (*domain.Domain, error)
    List(ctx context.Context) ([]domain.Domain, error)
    UpdateStatus(ctx context.Context, name, status, lastHealthAt string) error
    Update(ctx context.Context, name string, updates map[string]interface{}) error
}

// [CODE: api/repository/interfaces.go:60-66]
type StatsRepository interface {
    GetTimeline(ctx context.Context, days int) ([]domain.TimelineEntry, error)
    GetSummary(ctx context.Context) (*domain.SummaryResponse, error)
}
```

**Implementations**:
| Implementation | Location | Purpose |
|---------------|----------|---------|
| SQLiteEventRepository | `api/repository/sqlite_events.go` | Production SQLite |
| SQLiteDomainRepository | `api/repository/sqlite_domains.go` | Production SQLite |
| SQLiteStatsRepository | `api/repository/sqlite_stats.go` | Production SQLite |
| MockEventRepository | `api/internal/testutil/mocks.go` | Unit testing |
| MockDomainRepository | `api/internal/testutil/mocks.go` | Unit testing |
| MockStatsRepository | `api/internal/testutil/mocks.go` | Unit testing |

**Usage Pattern**:
```go
// Production (main.go)
eventRepo := repository.NewSQLiteEventRepository(db)
handler := handlers.New(eventRepo, domainRepo, statsRepo)

// Testing (with mocks)
eventRepo := testutil.NewMockEventRepository()
handler := handlers.New(eventRepo, domainRepo, statsRepo)
```

### 2. Handler Seam

**Location**: `api/handlers/handlers.go`
**Type**: Constructor injection
**Strength**: ✅ Strong - handlers accept interfaces, not concrete types

The Handler struct accepts repository interfaces via constructor:

```go
// [CODE: api/handlers/handlers.go:25-29]
type Handler struct {
    Events  repository.EventRepository
    Domains repository.DomainRepository
    Stats   repository.StatsRepository
}

// [CODE: api/handlers/handlers.go:32-38]
func New(events repository.EventRepository, domains repository.DomainRepository, stats repository.StatsRepository) *Handler {
    return &Handler{
        Events:  events,
        Domains: domains,
        Stats:   stats,
    }
}
```

This enables testing handlers with mock repositories.

### 3. Database Connection Seam

**Location**: `api/main.go:207-223`
**Type**: Environment-driven configuration
**Strength**: ✅ Strong - uses api-core/database.Connect with driver selection

The database connection is configurable via environment variables:

```go
// [CODE: api/main.go:207-223]
db, err := database.Connect(ctx, database.Config{
    Driver:       database.DriverSQLite,
    MaxOpenConns: 1,
    MaxIdleConns: 1,
    Retry: &retry.Config{
        MaxAttempts: 3,
        BaseDelay:   100 * time.Millisecond,
        MaxDelay:    500 * time.Millisecond,
    },
})
```

**Configuration**:
| Env Var | Purpose |
|---------|---------|
| SQLITE_PATH | Path to SQLite database file |
| SQLITE_DB | Alternative path variable |
| SCENARIO_DATA_DIR | Default data directory |

### 4. HTTP Routing Seam

**Location**: `api/main.go:62-89`
**Type**: Router configuration
**Strength**: ✅ Strong - routes are registered centrally

Routes are configured in `setupRoutes()`:

```go
// [CODE: api/main.go:62-89]
func (s *Server) setupRoutes() {
    // Health endpoints
    s.router.HandleFunc("/health", healthHandler).Methods("GET")
    s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

    // Events API
    s.router.HandleFunc("/api/v1/events", s.handler.CreateEvent).Methods("POST")
    // ...
}
```

### 5. Time Provider Seam (Ready for Integration)

**Location**: `api/internal/clock/clock.go`
**Type**: Interface-based abstraction
**Strength**: ✅ Strong - interface defined, ready for injection

A Clock interface has been implemented for testable time operations:

```go
// [CODE: api/internal/clock/clock.go]
type Clock interface {
    Now() time.Time
    Today() string
    Yesterday() string
    DaysAgo(n int) string
}
```

**Implementations**:
| Implementation | Location | Purpose |
|---------------|----------|---------|
| realClock | `internal/clock/clock.go` | Production - uses system time |
| fixedClock | `internal/clock/clock.go` | Testing - returns configured time |

**Usage Pattern**:
```go
// Production
clock := clock.Real()
now := clock.Now()

// Testing
clock := clock.Fixed(time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC))
now := clock.Now()  // Always returns fixed time
```

**Current state**: Interface defined and tested. Direct `time.Now()` calls still exist in repositories (integration pending). Locations:
- `api/repository/sqlite_events.go:34`
- `api/repository/sqlite_domains.go:26, 119, 137`
- `api/repository/sqlite_stats.go:113-114`
- `api/handlers/domains.go:164`

**See also**: [TEMPORAL-FLOWS.md](TEMPORAL-FLOWS.md) for time-related patterns

### 6. UUID Generation Seam (Weak)

**Location**: `api/repository/sqlite_events.go:29`
**Type**: Direct `uuid.New()` calls
**Strength**: ⚠️ Weak - no interface abstraction

UUIDs are generated directly via `uuid.New()`. This is generally acceptable since UUIDs don't need to be deterministic in most tests, but for full reproducibility, an `IDGenerator` interface could be introduced.

### 7. Health Check Seam

**Location**: `api/main.go:100-131`
**Type**: Interface implementation
**Strength**: ✅ Strong - uses api-core/health.Check interface

Health checks use the `health.Check` interface from api-core:

```go
// [CODE: api/main.go:100-131]
type sqliteChecker struct {
    db *sql.DB
}

func (c *sqliteChecker) Check(ctx context.Context) health.CheckResult {
    // ...
}
```

## Testing Seams Summary

### API Testing Seams

| Seam | Test Type | Mock Available | Location |
|------|-----------|----------------|----------|
| Repository | Unit | ✅ Yes | `api/internal/testutil/mocks.go` |
| Handler | Unit | ✅ Yes (via repo mocks) | `api/handlers/*_test.go` |
| Database | Integration | ✅ Yes (in-memory) | `api/internal/testutil/db.go` |
| Time | Unit | ✅ Yes | `api/internal/clock/clock.go` |
| UUID | Unit | ❌ No | N/A |
| Health | Integration | ✅ Yes | via api-core |

### UI Testing Seams

| Seam | Test Type | Mock Available | Location |
|------|-----------|----------------|----------|
| API Functions | Unit | ✅ Via fetch mock | `ui/src/lib/api.test.ts` |
| Error Handling | Unit | ✅ Yes | `ui/src/components/ErrorAlert.test.tsx` |
| Formatting | Unit | ✅ Pure functions | `ui/src/lib/format.test.ts` |
| Time | Unit | ✅ Via vi.useFakeTimers | Vitest time mocking |

**UI Test Coverage** (64 tests):
- `ErrorAlert.test.tsx`: 22 tests - error categorization, recovery actions, rendering
- `format.test.ts`: 22 tests - formatRelativeTime, formatDate, formatBytes, etc.
- `api.test.ts`: 20 tests - APIError class, error categories, retryability

**Nil Repository Error Handling** (17 tests added Phase 20 iter 2):
All handlers now check for nil repositories and return 503 Service Unavailable:
- `briefs_test.go`: 3 tests - GetCurrentBrief, GetMorningBrief, GetEveningBrief
- `digest_test.go`: 2 tests - GetCurrentDigest, GetDigestByWeek
- `score_config_test.go`: 3 tests - GetScoreConfig, GetDomainWeight, UpdateDomainWeight
- `stats_test.go`: 3 tests - GetTimeline, GetSummary, GetScore
- `storage_test.go`: 2 tests - GetStorageInfo, CleanupEvents
Plus 4 additional validation tests for missing domain names in score config handlers.

## Seam Usage Guidelines

### When to Use Mock Repositories

Use mocks when:
- Testing handler logic in isolation
- Testing error handling paths
- Verifying specific repository method calls
- Speeding up test execution

Use real SQLite when:
- Testing SQL query correctness
- Testing transaction behavior
- Testing schema constraints
- Integration/E2E tests

### Example: Handler Test with Mocks

```go
func TestCreateEvent_Success(t *testing.T) {
    // Arrange - use mock repositories
    eventRepo := testutil.NewMockEventRepository()
    domainRepo := testutil.NewMockDomainRepository()
    statsRepo := testutil.NewMockStatsRepository()
    h := handlers.New(eventRepo, domainRepo, statsRepo)

    // Act
    req := httptest.NewRequest("POST", "/api/v1/events", ...)
    rr := httptest.NewRecorder()
    h.CreateEvent(rr, req)

    // Assert
    if rr.Code != http.StatusCreated {
        t.Errorf("Expected 201, got %d", rr.Code)
    }
}
```

### Example: Repository Test with Real SQLite

```go
func TestEventRepository_Create(t *testing.T) {
    // Arrange - use real in-memory database
    db := testutil.SetupInMemoryDB(t)
    repo := repository.NewSQLiteEventRepository(db)

    // Act
    event := &domain.Event{Domain: "test", EventType: "test.event"}
    err := repo.Create(context.Background(), event)

    // Assert
    if err != nil {
        t.Fatalf("Create failed: %v", err)
    }
}
```

## Strengthening Weak Seams

### Priority 1: Time Provider (Recommended)

If time-sensitive behavior becomes important:

1. Define interface in `api/internal/clock/clock.go`
2. Inject into repositories via constructor
3. Use mock in tests for deterministic timestamps

### Priority 2: UUID Generator (Optional)

Only needed if UUID reproducibility matters for testing:

1. Define `IDGenerator` interface
2. Inject into repositories
3. Use sequential/predictable IDs in tests

## Observability Surface

This section documents the key observable states, signals, and transitions for debugging and agent decision-making.

### API Error Signals

The API uses structured error responses with explicit categories and recovery hints:

| Category | HTTP Status | When Used | Recovery Path |
|----------|-------------|-----------|---------------|
| `validation` | 400 | Invalid input | Fix input and retry |
| `not_found` | 404 | Resource doesn't exist | Check resource ID |
| `conflict` | 409 | State conflict | Resolve conflict |
| `internal` | 500 | Server error | Retry later |
| `unavailable` | 503 | Dependency down | Check scenario status |

**Error Response Structure**:
```json
{
  "error": true,
  "category": "validation",
  "code": "MISSING_FIELD",
  "message": "Field 'domain' is required",
  "details": { "field": "domain" },
  "recovery": "Check the request body and fix validation errors"
}
```

**Error Codes**:
- Validation: `INVALID_JSON`, `MISSING_FIELD`, `INVALID_FIELD`, `INVALID_TIME_RANGE`
- Not Found: `EVENT_NOT_FOUND`, `DOMAIN_NOT_FOUND`
- Internal: `DATABASE_ERROR`, `HEALTH_CHECK_ERROR`
- Unavailable: `DEPENDENCY_UNAVAILABLE`

### Log Signals

All errors are logged with structured format for grep/parsing:

```
[ERROR] CreateEvent: database error: <details>
[WARN] GetDomainHealth(domain-name): health check failed: <details>
```

Log levels:
- `[ERROR]` - Operation failed, requires attention
- `[WARN]` - Degraded behavior, non-critical
- `[GET]`, `[POST]` - HTTP request logging (from gorilla/mux)

### Health Check Signals

**API Health Endpoint**: `GET /health`

Returns:
- `status`: "healthy" or "unhealthy"
- `readiness`: true/false for load balancer checks
- `dependencies.sqlite.connected`: database connectivity
- `uptime_seconds`: process uptime

**Domain Health Checks**: `GET /api/v1/domains/{name}/health`

Returns health status for individual domain scenarios with `last_check` timestamp.

### UI Error Surfaces

The UI extracts structured error information and displays:
- Error category-specific titles (e.g., "Invalid Request", "Not Found")
- User-friendly messages without technical details
- Recovery actions: Retry (for retryable), Back (for not_found), Help link

### Signal Gaps (Known Debt)

1. **Metrics**: No Prometheus/OpenMetrics endpoint yet
2. **Request IDs**: No correlation IDs for tracing across systems
3. **Audit Log**: No persistent record of mutations

## Change Axes

This section identifies the **primary ways this scenario is likely to change** and documents how localized those changes are. Understanding change axes helps future agents make modifications safely and efficiently.

### Change Axis 1: New Domain Integrations

**Likelihood**: High (core use case)
**Cost of Change**: Low ✅

Adding a new health/wellness domain (e.g., nutrition tracker, medication tracker) requires:
1. Domain registers itself via `POST /api/v1/domains` - **no code changes**
2. Domain emits events via `POST /api/v1/events` - **no code changes**

The system is designed for additive domain growth without code modification.

**Stable Core**:
- `api/domain/types.go` - Event and Domain structures are generic envelopes
- `api/repository/interfaces.go` - CRUD interfaces are domain-agnostic
- `api/handlers/` - Handlers process any domain/event_type combination

### Change Axis 2: New Event Types & Payload Schemas

**Likelihood**: High
**Cost of Change**: Low ✅

The `Event.Payload` field is `json.RawMessage`, allowing any JSON structure without schema changes.

**Volatile Edge**: Payload validation would require changes in:
- `api/handlers/events.go:CreateEvent` - Add validation logic
- `api/domain/types.go` - Add payload schema types if needed

**Current State**: No payload validation - events accept any JSON payload. This is intentional for flexibility but may need tightening as the system matures.

### Change Axis 3: Query & Filtering Capabilities

**Likelihood**: Medium
**Cost of Change**: Medium ⚠️

Adding new query filters requires changes in:
1. `api/repository/interfaces.go:EventFilter` - Add new filter fields
2. `api/repository/sqlite_events.go:List` - Add SQL WHERE clauses
3. `api/handlers/events.go:QueryEvents` - Parse new query params
4. `ui/src/lib/api.ts:QueryEventsParams` - Add UI params

**Localization**: Changes span 4 files but follow clear patterns. Each layer has a single responsibility.

### Change Axis 4: UI Visualization Variations

**Likelihood**: Medium
**Cost of Change**: Low ✅

UI components are isolated in `ui/src/components/dashboard/`:
- `TimelineChart.tsx` - Timeline visualization
- `DomainBreakdown.tsx` - Domain statistics
- `DomainCard.tsx` - Domain display cards
- `EventRow.tsx` - Event list items
- `StatCard.tsx` - Summary statistics

**Extension Point**: New visualizations can be added as new components without modifying existing ones. Pages compose components from the barrel export (`index.ts`).

### Change Axis 5: Storage Backend

**Likelihood**: Low (SQLite is suitable for single-server deployment)
**Cost of Change**: Low ✅

Repository interfaces abstract storage entirely:
1. Add new implementation (e.g., `repository/postgres_events.go`)
2. Change `main.go:NewServer` to use new implementation

**No handler changes required** - Storage Architecture skill already applied.

### Change Axis 6: Error Handling Policies

**Likelihood**: Low
**Cost of Change**: Low ✅

Error categories are centralized in `api/errors/errors.go`:
- Adding new error codes: Add constant, create pre-built error
- Changing recovery hints: Update `RecoveryHint` constants
- UI updates category titles in `ui/src/components/ErrorAlert.tsx:getErrorTitle`

**Protective Comment**: The errors package includes a warning header about category stability.

### Change Axis 7: Configuration & Tuning

**Likelihood**: Medium
**Cost of Change**: Low ✅

All tunable parameters are centralized in `api/config/config.go`:
- `QueryConfig` - Event limits, timeline defaults
- `HealthCheckConfig` - Timeout, unhealthy threshold
- `DatabaseConfig` - Connection pool, retry settings
- `CORSConfig` - Allowed origins

**Extension Point**: Add new config structs following existing pattern. Environment overrides are applied in `Default*Config()` functions.

### Change Axis Summary

| Axis | Likelihood | Cost | Files Touched | Extension Point |
|------|------------|------|---------------|-----------------|
| New domains | High | Low | 0 | REST API |
| New event types | High | Low | 0-2 | json.RawMessage payload |
| Query filters | Medium | Medium | 4 | EventFilter struct |
| UI visualizations | Medium | Low | 1 | New component file |
| Storage backend | Low | Low | 2 | Repository interface |
| Error policies | Low | Low | 2 | errors package |
| Configuration | Medium | Low | 1 | config package |

## Decision Points

This section documents **important decisions** (branches, thresholds, rules) and where they live. Well-located decision points make the system easier to understand and modify.

### Decision Point 1: Error Categorization

**Location**: `api/errors/errors.go:30-48`
**Type**: Enumeration with HTTP mapping
**Strength**: ✅ Well-extracted

```go
// [CODE: api/errors/errors.go:33-48]
const (
    CategoryValidation ErrorCategory = "validation"
    CategoryNotFound   ErrorCategory = "not_found"
    CategoryConflict   ErrorCategory = "conflict"
    CategoryInternal   ErrorCategory = "internal"
    CategoryUnavailable ErrorCategory = "unavailable"
)
```

**Decision Criteria** (documented in ERROR_SEMANTICS.md):
1. Can the user fix it? → `validation`
2. Does the resource not exist? → `not_found`
3. Is it a state conflict? → `conflict`
4. Is a dependency unreachable? → `unavailable`
5. Everything else → `internal`

**Who Uses This Decision**:
- `api/handlers/*` - Select appropriate category when creating errors
- `ui/src/lib/api.ts:APIError` - Determines `isRetryable`, `isValidation`, `isNotFound`
- `ui/src/components/ErrorAlert.tsx` - Selects UI title and recovery action

### Decision Point 2: Health Check Status Determination

**Location**: `api/handlers/domains.go:155-174`
**Type**: Status code threshold + timeout handling
**Strength**: ✅ Well-extracted

The health check decides `healthy` vs `unhealthy` based on:
1. **Timeout**: `cfg.Timeout` (default 5s) - exceeding → `unhealthy`
2. **Status code**: `cfg.UnhealthyThreshold` (default 300) - status >= threshold → `unhealthy`
3. **Error**: Any error during request → `unhealthy`

```go
// [CODE: api/handlers/domains.go:167-174]
if healthErr != nil {
    status = "unhealthy"
    message = "Health check failed: " + healthErr.Error()
} else if resp.StatusCode >= cfg.UnhealthyThreshold {
    status = "unhealthy"
    message = "Health check returned unhealthy status code"
}
```

**Tunability**: Both threshold values are configurable via `api/config/config.go`.

### Decision Point 3: Event Query Limit Enforcement

**Location**: `api/repository/sqlite_events.go` (uses `config.QueryConfig`)
**Type**: Default + max limit clamping
**Strength**: ✅ Well-extracted

The query limit decision follows this logic:
1. No limit specified → use `DefaultEventLimit` (100)
2. Limit specified → clamp to `MaxEventLimit` (1000)

**Tunable via**:
- `LD_DEFAULT_EVENT_LIMIT` environment variable
- `LD_MAX_EVENT_LIMIT` (would need to add)

### Decision Point 4: Timeline Days Enforcement

**Location**: `api/handlers/stats.go:17-32`
**Type**: Default + max limit clamping
**Strength**: ✅ Well-extracted

```go
// [CODE: api/handlers/stats.go:21-31]
days := cfg.DefaultTimelineDays
if daysStr != "" {
    if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
        days = d
        if days > cfg.MaxTimelineDays {
            days = cfg.MaxTimelineDays
        }
    }
}
```

**Tunability**: `LD_DEFAULT_TIMELINE_DAYS` environment variable.

### Decision Point 5: UI Recovery Action Selection

**Location**: `ui/src/components/ErrorAlert.tsx:46-57`
**Type**: Category-based switch
**Strength**: ✅ Well-extracted

```typescript
// [CODE: ui/src/components/ErrorAlert.tsx:46-57]
function getRecoveryAction(error: Error): "retry" | "back" | "help" {
  if (error instanceof APIError) {
    if (error.isRetryable) return "retry";  // internal, unavailable
    if (error.isNotFound) return "back";    // not_found
    if (error.isValidation) return "help";  // validation
  }
  // Network errors are retryable
  if (error.message.includes("Failed to fetch")) {
    return "retry";
  }
  return "retry";  // default
}
```

**Decision Flow**:
| Error Category | `isRetryable` | Action | Button |
|---------------|---------------|--------|--------|
| `internal` | true | retry | "Try Again" |
| `unavailable` | true | retry | "Try Again" |
| `not_found` | false | back | "Go Back" |
| `validation` | false | help | "Get Help" |
| Network error | N/A | retry | "Try Again" |

### Decision Point 6: Domain Status Values

**Location**: `api/domain/types.go:41` (implicit)
**Type**: String enumeration (not yet strongly typed)
**Strength**: ⚠️ Scattered

Domain status can be:
- `active` - Domain is responding
- `inactive` - Domain never checked or no health URL
- `unhealthy` - Health check failed

**Current State**: Status is a plain `string` field. Transitions happen in:
- `api/handlers/domains.go:GetDomainHealth` - Sets `healthy` or `unhealthy`
- `api/repository/sqlite_domains.go:Upsert` - Defaults to `active`

**Improvement Opportunity**: Extract status transitions to a dedicated function with clear rules.

### Decision Point 7: Trend Direction Determination (NEW - Phase 19)

**Location**: `api/domain/decisions.go`
**Type**: Threshold-based classification
**Strength**: ✅ Strong - centralized with configurable thresholds

Direction determination logic has been extracted to centralized helpers:

```go
// [CODE: api/domain/decisions.go:50-60]
func DetermineDirectionWithThreshold(percentChange, threshold float64) Direction {
    if math.Abs(percentChange) < threshold {
        return DirectionStable
    }
    if percentChange > 0 {
        return DirectionUp
    }
    return DirectionDown
}
```

**Thresholds** (from `api/config/config.go:ScoringConfig`):
| Threshold | Default | Purpose |
|-----------|---------|---------|
| TrendThreshold | 5.0% | Composite score direction |
| DomainTrendThreshold | 10.0% | Individual domain direction |
| NotableChangeThreshold | 20.0% | Highlight-worthy changes |

**Who Uses This Decision**:
- `api/repository/sqlite_stats.go:GetLifestyleScore` - Score trend via `DetermineDirection`
- `api/repository/sqlite_digest.go:getDomainChanges` - Domain trend via `DetermineDomainDirection`
- `api/repository/sqlite_digest.go:getScoreTrend` - Digest trend via `DetermineDirection`

### Decision Point 8: Data Quality Assessment (NEW - Phase 19)

**Location**: `api/domain/decisions.go`
**Type**: Threshold-based classification
**Strength**: ✅ Strong - centralized with configurable thresholds

```go
// [CODE: api/domain/decisions.go:100-110]
func DetermineDataQuality(activeDomains int) DataQuality {
    if activeDomains >= 3 { return DataQualityGood }
    if activeDomains >= 1 { return DataQualityLimited }
    return DataQualityInsufficient
}
```

**Thresholds** (from `api/config/config.go:ScoringConfig`):
- `DataQualityGoodThreshold`: 3 (default)
- `DataQualityLimitedThreshold`: 1 (default)

### Decision Point 9: Score Level Messaging (NEW - Phase 19)

**Location**: `api/domain/decisions.go`
**Type**: Threshold-based classification
**Strength**: ✅ Strong - centralized with user-friendly messages

```go
// [CODE: api/domain/decisions.go:130-145]
func DetermineScoreLevel(score int) ScoreLevel
func ScoreLevelMessage(level ScoreLevel) string
func TrendMessage(direction Direction) string
```

**Thresholds** (from `api/config/config.go:ScoringConfig`):
| Level | Threshold | Message |
|-------|-----------|---------|
| Excellent | ≥80 | "Excellent day!" |
| Good | ≥60 | "Good progress today." |
| Moderate | ≥40 | "Moderate activity." |
| Light | <40 | "Light activity today." |

### Decision Point 10: Health Check Result (NEW - Phase 19 iter 2)

**Location**: `api/domain/decisions.go`
**Type**: Dual-status response
**Strength**: ✅ Strong - separates API response from storage status

```go
// [CODE: api/domain/decisions.go:270-300]
type HealthCheckResult struct {
    ResponseStatus HealthStatus  // For API: "healthy"/"unhealthy"
    DomainStatus   DomainStatus  // For storage: "active"/"unhealthy"
    Message        string
}
func DetermineHealthCheckResult(healthError error, statusCode, threshold int) HealthCheckResult
```

**Decision Criteria**:
- `healthError != nil` → unhealthy
- `statusCode >= threshold (default 300)` → unhealthy
- otherwise → healthy (API) / active (storage)

**Unique feature**: Returns separate statuses for API response ("healthy"/"unhealthy") and storage ("active"/"unhealthy") to maintain API contract while using correct domain status internally.

### Decision Point 11: Highlight Generation (NEW - Phase 19 iter 2)

**Location**: `api/domain/decisions.go`
**Type**: Threshold + direction classification
**Strength**: ✅ Strong - centralized highlight logic

```go
// [CODE: api/domain/decisions.go:320-340]
func ShouldHighlightDomainChange(percentChange float64, direction Direction) (bool, HighlightType)
func ShouldHighlightScoreImprovement(percentChange float64, direction Direction) bool
func GenerateFocusRecommendation(displayName string, percentChange float64, direction Direction) (string, bool)
```

**Who Uses This Decision**:
- `repository/sqlite_digest.go:generateHighlights` - Weekly digest highlights
- `repository/sqlite_digest.go:generateNextWeekFocus` - Focus recommendations

### Decision Points Summary

| Decision | Location | Extracted? | Tested? |
|----------|----------|------------|---------|
| Error categorization | `errors/errors.go` | ✅ Yes | ✅ Yes |
| Health check result | `domain/decisions.go` | ✅ Yes | ✅ Yes |
| Event limit | `repository/sqlite_events.go` | ✅ Yes | ✅ Yes |
| Timeline days | `handlers/stats.go` | ✅ Yes | ✅ Yes |
| UI recovery action | `ErrorAlert.tsx` | ✅ Yes | ✅ Yes |
| **Domain status** | `domain/decisions.go` | ✅ Yes | ✅ Yes |
| **Trend direction** | `domain/decisions.go` | ✅ Yes | ✅ Yes |
| **Data quality** | `domain/decisions.go` | ✅ Yes | ✅ Yes |
| **Score level** | `domain/decisions.go` | ✅ Yes | ✅ Yes |
| **Notable change** | `domain/decisions.go` | ✅ Yes | ✅ Yes |
| **Highlight generation** | `domain/decisions.go` | ✅ Yes | ✅ Yes |
| **Focus recommendations** | `domain/decisions.go` | ✅ Yes | ✅ Yes |

### Decision Debt

1. ~~**UI Recovery Actions**: Add unit tests for `getRecoveryAction` function.~~ ✅ Resolved Phase 19 iteration 3 - 22 tests added in `ErrorAlert.test.tsx`
2. **Payload Validation**: No decision logic yet - if added, should be in `domain/` package.

## CLI Surface

This section documents the CLI as a thin wrapper over the API, following the `cli-steer` pattern from Vrooli's cli-core package.

### CLI Architecture

The CLI (`cli/app.go`) is built on cli-core's `ScenarioApp` scaffolding:
- **Cross-platform**: Go binary with `install.sh` (Unix) and `install.ps1` (Windows)
- **Auto-rebuild**: Stale detection triggers rebuild when source changes
- **API-first**: All commands are thin wrappers over HTTP API calls
- **Consistent output**: Human-readable default with `--json` for machine-readable

### CLI-API Parity

| API Endpoint | HTTP Method | CLI Command | Status |
|--------------|-------------|-------------|--------|
| `/health` | GET | `status` | ✅ Complete |
| `/api/v1/events` | POST | `event create` | ✅ Complete |
| `/api/v1/events` | GET | `event list` | ✅ Complete |
| `/api/v1/events/{id}` | GET | `event get` | ✅ Complete |
| `/api/v1/domains` | POST | `domain register` | ✅ Complete |
| `/api/v1/domains` | GET | `domain list` | ✅ Complete |
| `/api/v1/domains/{name}` | GET | `domain get` | ✅ Complete |
| `/api/v1/domains/{name}` | PATCH | `domain update` | ✅ Complete |
| `/api/v1/domains/{name}/health` | GET | `domain health` | ✅ Complete |
| `/api/v1/stats/timeline` | GET | `stats timeline` | ✅ Complete |
| `/api/v1/stats/summary` | GET | `stats summary` | ✅ Complete |

### Command Groups

```
lifestyle-dashboard <command>

Health:
  status              Check API health

Events:
  event create        Create a new event
  event list          List events with optional filters
  event get           Get event by ID

Domains:
  domain register     Register a new domain
  domain list         List all registered domains
  domain get          Get domain by name
  domain update       Update domain attributes
  domain health       Check domain health status

Statistics:
  stats timeline      Get event timeline
  stats summary       Get aggregated statistics

Configuration:
  configure           Set API base URL and token
```

### Output Contracts

All commands follow the CLI-steer output contracts:

| Command Type | Contract | Example |
|--------------|----------|---------|
| `status` | Operational | Status → Triage → Next Steps |
| `list`, `get` | Data Retrieval | Summary → Results → Hints |
| `create`, `update`, `register` | Mutation Result | Result → What Changed → Next |

**Default output** is human-friendly with clear structure:
```
Created event: abc123def456
  Domain: sleep
  Type: session.logged
  Timestamp: 2026-03-10T12:00:00Z
```

**JSON output** (`--json`) provides full API response fidelity:
```json
{
  "id": "abc123def456",
  "domain": "sleep",
  "event_type": "session.logged",
  "timestamp": "2026-03-10T12:00:00Z",
  "payload": {...}
}
```

### Environment Variables

Derived via `StandardScenarioEnv("lifestyle-dashboard")`:

| Purpose | Variables (precedence order) |
|---------|------------------------------|
| API Base URL | `LIFESTYLE_DASHBOARD_API_BASE`, `API_BASE_URL`, `VITE_API_BASE_URL` |
| API Port | `LIFESTYLE_DASHBOARD_API_PORT`, `API_PORT` |
| API Token | `LIFESTYLE_DASHBOARD_API_TOKEN`, `VROOLI_API_TOKEN` |
| Config Dir | `LIFESTYLE_DASHBOARD_CONFIG_DIR`, `VROOLI_CLI_CONFIG_DIR` |

### CLI Tests

Location: `cli/app_test.go`

| Test | Purpose |
|------|---------|
| `TestNewApp` | CLI construction |
| `TestAppConstants` | Constant values |
| `TestAPIPath` | Path construction helper |
| `TestRegisterCommands` | All command groups registered |
| `TestCommandsNeedAPI` | API commands have `NeedsAPI: true` |
| `TestCommandDescriptions` | All commands have descriptions |

## Tech Debt

This section tracks known technical debt and simplification opportunities for future work.

### Resolved (Phase 15)

1. **Handler constructor proliferation** (✅ Resolved)
   - **Was**: Three constructors (`New`, `NewWithStorage`, `NewComplete`) evolved organically as repositories were added
   - **Now**: Single `New` constructor accepting all required repositories
   - **Files**: `api/handlers/handlers.go`, `api/handlers/handlers_test.go`

2. **Monolithic handlers_test.go** (✅ Resolved - Phase 15 iteration 2)
   - **Was**: Single `handlers_test.go` validated 5 requirements, causing 2-point completeness penalty
   - **Now**: Tests split into focused files by domain:
     - `events_test.go` - Event CRUD tests [REQ:LD-EVENT-STORAGE]
     - `domains_test.go` - Domain registration/discovery tests [REQ:LD-DOMAIN-REGISTER, LD-DOMAIN-DISCOVER]
     - `stats_test.go` - Query/aggregation tests [REQ:LD-QUERY-FILTER, LD-QUERY-AGGREGATE]
     - `handlers_test.go` - Shared test setup and utility tests
   - **Impact**: Validation penalty eliminated (+2 points), better test organization and traceability
   - **Requirements updated**: events/schema.json, domains/registration.json, queries/api.json

3. **Repository code duplication** (✅ Resolved - Phase 15 iteration 3)
   - **sqlite_briefs.go**: Extracted common brief generation into `generateBrief()` helper
     - `briefConfig` struct for type-safe configuration
     - `countSectionActivity()` and `formatBriefSummary()` helpers
     - Reduced ~80 lines of duplication between morning/evening brief methods
   - **sqlite_stats.go**: Extracted score calculation into `calculateDomainScore()` helper
     - Named constants: `scoreMultiplier=20`, `maxDomainScore=100`
     - Unified logic between `getTodayDomainScores` and `calculateDayScore`
   - **Files**: `api/repository/sqlite_briefs.go`, `api/repository/sqlite_stats.go`

4. **Decision boundary extraction** (✅ Resolved - Phase 19)
   - **Was**: Direction determination, percent change, data quality, and score messaging logic duplicated across `sqlite_stats.go` and `sqlite_digest.go` with scattered magic numbers
   - **Now**: Centralized in `api/domain/decisions.go` with:
     - `DetermineDirection()`, `DetermineDomainDirection()` - trend direction helpers
     - `CalculatePercentChange()` - percent change calculation
     - `IsNotableChange()` - notable change detection
     - `DetermineDataQuality()` - data quality assessment
     - `CalculateDomainScore()` - score calculation with configurable params
     - `DetermineScoreLevel()`, `ScoreLevelMessage()`, `TrendMessage()` - messaging helpers
   - **Configuration**: Thresholds moved to `api/config/config.go:ScoringConfig`
   - **Tests**: 15 new tests in `api/domain/decisions_test.go`
   - **Impact**: All decision boundaries are now explicit, testable, and configurable

5. **Domain status transitions** (✅ Resolved - Phase 19 iteration 2)
   - **Was**: Health check status determination scattered in `handlers/domains.go` with separate logic for API responses vs storage
   - **Now**: Centralized in `api/domain/decisions.go` with:
     - `HealthCheckResult` struct separating API response status from storage status
     - `DetermineHealthCheckResult()` - single source of truth for health check → status mapping
     - `HealthStatusHealthy`/`HealthStatusUnhealthy` - API response statuses
     - `DomainStatusActive`/`DomainStatusUnhealthy` - storage statuses
   - **Tests**: 6 new tests for health check result determination
   - **Impact**: Clear separation between API contracts (healthy/unhealthy) and storage (active/unhealthy)

6. **Highlight/recommendation generation** (✅ Resolved - Phase 19 iteration 2)
   - **Was**: Highlight and focus recommendation logic embedded in `sqlite_digest.go` with inline thresholds
   - **Now**: Centralized in `api/domain/decisions.go` with:
     - `ShouldHighlightDomainChange()` - determines if a domain change warrants highlighting
     - `ShouldHighlightScoreImprovement()` - determines if a score improvement is significant
     - `GenerateFocusRecommendation()` - generates focus recommendations based on direction/magnitude
     - `HighlightType` enum - positive/warning/info highlight categorization
   - **Tests**: 5 new tests for highlight and recommendation generation
   - **Impact**: Digest generation logic is clearer, thresholds are explicit and testable

7. **UI Recovery Action decision tests** (✅ Resolved - Phase 19 iteration 3)
   - **Was**: `getRecoveryAction`, `getErrorTitle`, and `getErrorMessage` functions in `ErrorAlert.tsx` untested
   - **Now**: Comprehensive test coverage in `ui/src/components/ErrorAlert.test.tsx`:
     - 7 tests for `getRecoveryAction` (internal→retry, unavailable→retry, not_found→back, validation→help, network errors→retry, unknown→retry)
     - 7 tests for `getErrorTitle` (all 5 categories + network + generic)
     - 4 tests for error message/recovery hints
     - 4 tests for component rendering behavior
   - **Tests**: 22 new UI tests with vitest + @testing-library/react
   - **Impact**: All decision points in the scenario (API + UI) now have test coverage

### Current Debt

1. **Direct time.Now() calls** (Priority: Low)
   - **Location**: `api/repository/sqlite_*.go` files
   - **Impact**: Difficult to test time-dependent behavior deterministically
   - **Status**: Clock interface exists (`api/internal/clock/`), integration pending
   - **Recommendation**: Inject Clock into repositories when time-sensitive tests are needed

2. ~~**Monolithic handlers_test.go**~~ (✅ Resolved - see Resolved section above)

3. **UUID generation not injectable** (Priority: Very Low)
   - **Location**: `api/repository/sqlite_events.go:29`
   - **Impact**: Cannot test with deterministic IDs
   - **Status**: Acceptable for current needs; UUIDs don't need reproducibility in most tests

### Architecture Clarity Notes

**What's clean now:**
- Handler→Repository→Domain layering is clear and consistent
- All P0 requirements have test coverage with [REQ:ID] tags
- Error handling uses centralized categories with recovery hints
- Configuration is centralized in `api/config/` with env overrides
- UI utilities are properly tiered (core/framework/domain)
- **Decision boundaries are explicit** - trend, quality, and score decisions centralized in `domain/decisions.go`
- **Scoring thresholds are configurable** - all magic numbers extracted to `config/config.go:ScoringConfig`
- **All decision points tested** - 12 API decision points + 3 UI decision functions have test coverage

**Where to look for common changes:**
- New API endpoints: `api/handlers/*.go` + `api/main.go:setupRoutes()`
- New storage queries: `api/repository/sqlite_*.go`
- New domain types: `api/domain/types.go`
- **Decision boundaries**: `api/domain/decisions.go` (trend, quality, score level)
- **Scoring thresholds**: `api/config/config.go:ScoringConfig`
- New UI pages: `ui/src/pages/*.tsx` + `ui/src/App.tsx` routes
- Configuration: `api/config/config.go`
- Test utilities: `api/internal/testutil/`

## Related Documentation

- [ARCHITECTURE.md](../concepts/ARCHITECTURE.md) - Overall system design
- [STORAGE_AUDIT.md](STORAGE_AUDIT.md) - Database architecture decisions
- [UNIT_TEST_ARCHITECTURE.md](UNIT_TEST_ARCHITECTURE.md) - Test organization patterns
- [ERROR_SEMANTICS.md](ERROR_SEMANTICS.md) - Error category design decisions
