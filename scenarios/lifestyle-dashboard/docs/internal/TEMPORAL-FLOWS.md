# Temporal Flows Documentation

This document describes the **temporal patterns** (async operations, polling, scheduling, time-dependent behavior) in the Lifestyle Dashboard scenario. Understanding these flows helps developers write robust, predictable code.

## Last Updated
2026-03-10 (Phase 12: Temporal Flow Audit)

## Async Operation Inventory

### API Layer (Go Backend)

| Operation | Location | Trigger | Duration | State Changed |
|-----------|----------|---------|----------|---------------|
| Event Create | `repository/sqlite_events.go:Create` | POST /api/v1/events | <10ms | events table |
| Domain Upsert | `repository/sqlite_domains.go:Upsert` | POST /api/v1/domains | <10ms | domains table |
| Domain Health Check | `handlers/domains.go:GetDomainHealth` | GET /api/v1/domains/{name}/health | configurable (default 5s) | domains.status |
| Score Calculation | `repository/sqlite_stats.go:GetLifestyleScore` | GET /api/v1/stats/score | <100ms | none (read-only) |
| Timeline Query | `repository/sqlite_stats.go:GetTimeline` | GET /api/v1/stats/timeline | <50ms | none (read-only) |

### UI Layer (React)

| Operation | Location | Interval | Query Key | Dependencies |
|-----------|----------|----------|-----------|--------------|
| Domains List | `DashboardPage.tsx` | 60s | `["domains"]` | None |
| Summary Stats | `DashboardPage.tsx` | 30s | `["summary"]` | None |
| Timeline | `DashboardPage.tsx` | 60s | `["timeline", period]` | period state |
| Events List | `DashboardPage.tsx` | 30s | `["events"]` | None |
| Score | `DashboardPage.tsx` | 60s | `["score"]` | None |
| Domain Detail | `DomainDetailPage.tsx` | 30s | `["domain", name]` | name param |
| Domain Health | `DomainDetailPage.tsx` | 60s | `["domain-health", name]` | name param |
| Domain Events | `DomainDetailPage.tsx` | 30s | `["events", {domain}]` | name param |

## Ordering Assumptions

### Database Operations

**SQLite Single-Writer Constraint**: All database operations are serialized via `MaxOpenConns=1`. This provides:
- ✅ No read-your-writes issues
- ✅ No concurrent update conflicts
- ⚠️ Potential bottleneck under high load

**Timestamp Generation**: Timestamps are generated server-side in UTC:
- `sqlite_events.go:Create` - Sets `created_at` and `timestamp` if not provided
- `sqlite_domains.go:Upsert` - Sets `registered_at` (first time) and `updated_at`
- `sqlite_domains.go:UpdateStatus` - Sets `last_health_at` and `updated_at`

**Current State**: Using direct `time.Now()` calls. Future improvement available via `internal/clock.Clock` interface.

### UI State Updates

**React Query Behavior**:
- Queries run independently (no explicit coordination)
- `refetchInterval` triggers background refetches
- Stale data shows immediately while refetch happens
- Multiple queries can fire simultaneously on page load

**Period Selection Flow** (TimelineChart):
1. User clicks period button (7d/30d/90d)
2. `setTimelinePeriod` updates local state
3. React Query key changes `["timeline", newPeriod]`
4. New query fires, old data cleared
5. Loading state shown briefly
6. New data rendered

**Error Handling Flow**:
1. Query fails with error
2. Error stored in query state
3. `ErrorAlert` component renders
4. User can click "Retry" to refetch all queries
5. Individual queries may succeed/fail independently

## Race Conditions

### Identified & Mitigated

| Race | Mitigation | Status |
|------|------------|--------|
| Concurrent domain registration | SQLite `ON CONFLICT` upsert | ✅ Safe |
| Overlapping health checks | Last-write-wins for status | ⚠️ Acceptable |
| Concurrent event creation | UUIDs ensure uniqueness | ✅ Safe |
| UI rapid period switching | React Query cancels stale requests | ✅ Safe |

### Potential Issues

| Scenario | Risk | Severity | Notes |
|----------|------|----------|-------|
| Multiple health checks for same domain | Redundant work | Low | User-triggered, self-limiting |
| Rapid retry button clicks | Multiple concurrent refetches | Low | React Query deduplicates |
| Score history sequential queries | N+1 query pattern | Medium | See Performance section |

## Initialization & Teardown

### API Server Startup

```
1. Preflight checks (rebuild if stale)
2. Configure SQLITE_PATH with WAL mode
3. database.Connect() with retry (3 attempts, 100ms base, 500ms max)
4. domain.InitSchema() creates tables if not exist
5. NewServer() creates repositories and routes
6. server.Run() starts HTTP with graceful shutdown
```

**Graceful Shutdown**:
- `server.Run` handles SIGINT/SIGTERM
- Cleanup function closes database connection
- In-flight requests complete before shutdown

### UI Initialization

```
1. Check if in iframe (window.parent !== window)
2. If iframe: initialize bridge with parentOrigin
3. Set idempotency guard: window.__lifestyleDashboardBridgeInitialized
4. Create QueryClient
5. Render React tree
```

**Idempotency Guard**: The `__lifestyleDashboardBridgeInitialized` flag prevents double-initialization if React StrictMode or hot reload re-runs main.tsx.

### Cleanup

**Browser**: QueryClient handles cache cleanup on unmount. No explicit timers or subscriptions to clear.

**Server**: Database connection closed via cleanup function. No background goroutines or timers running.

## Polling & Retry Behavior

### UI Polling Configuration

| Query | Interval | Purpose |
|-------|----------|---------|
| domains | 60s | Catch new domain registrations |
| summary | 30s | Keep event counts fresh |
| timeline | 60s | Refresh chart data |
| events | 30s | Show recent activity |
| score | 60s | Update lifestyle score |
| domain-health | 60s | Refresh health status |

**Design Rationale**:
- 30s for frequently changing data (events, summary)
- 60s for slowly changing data (domains, score, timeline)
- No exponential backoff (polling continues at fixed interval)

### Database Connection Retry

```go
Retry: &retry.Config{
    MaxAttempts: 3,
    BaseDelay:   100 * time.Millisecond,
    MaxDelay:    500 * time.Millisecond,
}
```

Only applies during startup. If database becomes unavailable after startup, health check will report unhealthy but no automatic reconnection.

### Health Check Timeout

```go
cfg := config.DefaultHealthCheckConfig()
// Timeout: 5s (configurable via LD_HEALTH_CHECK_TIMEOUT_SECS)
// UnhealthyThreshold: 300 (HTTP status >= 300 = unhealthy)
```

Context timeout ensures health checks don't hang indefinitely.

## Checkpoint Flows

### Event Creation Checkpoint

The event creation flow is atomic - either the event is persisted or an error is returned. No partial state.

```
1. Parse request JSON
2. Validate required fields (domain, event_type)
3. Generate UUID (if not provided)
4. Set timestamps (server-side)
5. INSERT into SQLite
6. Return created event with ID
```

**Recovery**: If client doesn't receive response, it should retry with a new request. Events are not idempotent by default (new UUID each time).

### Domain Registration Checkpoint

Domain registration uses UPSERT, making it naturally idempotent.

```
1. Parse request JSON
2. Validate required fields (name, display_name)
3. Set timestamps
4. INSERT ON CONFLICT UPDATE
5. Return domain
```

**Idempotency**: Re-registering the same domain updates `updated_at` and other fields but preserves `registered_at`. Safe to retry.

### Score Calculation Flow

Score calculation is read-only but involves multiple database queries:

```
1. Get today's domain scores (1 query)
2. Calculate composite score (in-memory)
3. Get yesterday's score (1 query)
4. Calculate trend (in-memory)
5. Get history (N queries, one per day)
6. Return score response
```

**Known Issue**: History calculation uses N+1 query pattern. For 7-day history, this is 8 total queries. For longer periods, consider batch query optimization.

### UI Page Load Flow

```
1. Route matched, page component mounts
2. useQuery hooks register queries
3. All enabled queries fire in parallel
4. Loading states shown independently
5. Data arrives, components render
6. Background refetch timers start
```

**Interruption Recovery**: If user navigates away during load, queries are cancelled. On return, queries refetch from scratch (no progress preservation).

## Time Provider Abstraction

### Current State

Direct `time.Now()` calls exist in:
- `repository/sqlite_events.go:33-34` (event timestamps)
- `repository/sqlite_domains.go:26, 119, 137` (domain timestamps)
- `repository/sqlite_stats.go:113-114` (score date calculation)
- `handlers/domains.go:164` (health check timestamp)

### Clock Interface

A new `internal/clock.Clock` interface provides testable time:

```go
// Production
clock := clock.Real()
now := clock.Now()

// Tests
clock := clock.Fixed(time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC))
now := clock.Now()  // Always returns fixed time
```

**Methods**:
- `Now()` - Current time in UTC
- `Today()` - Today's date as YYYY-MM-DD
- `Yesterday()` - Yesterday's date
- `DaysAgo(n)` - Date N days ago

**Integration Status**: Interface defined, not yet injected into repositories. This is prepared for future phases that need deterministic time testing.

## Performance Considerations

### Score History N+1 Query

The `getScoreHistory` method queries each day individually:

```go
for i := days - 1; i >= 0; i-- {
    date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
    score, err := r.calculateDayScore(ctx, date)
    // ...
}
```

**Impact**: 7 queries for 7-day history, 30 for 30-day.

**Future Optimization**: Batch query with GROUP BY date, calculate scores in-memory:
```sql
SELECT date(timestamp) as day, domain, count(*) as event_count
FROM events
WHERE timestamp >= datetime('now', '-7 days')
GROUP BY date(timestamp), domain
```

### UI Query Coordination

Five queries fire independently on DashboardPage load. React Query handles this efficiently but network latency may cause visual "pop-in" as data arrives.

**Possible Improvement**: Use `useQueries` hook to coordinate loading states, show skeleton until critical data arrives.

## Related Documentation

- [SEAMS.md](SEAMS.md) - Integration seams including Time Provider
- [ARCHITECTURE.md](../concepts/ARCHITECTURE.md) - Overall system design
- [STORAGE_AUDIT.md](STORAGE_AUDIT.md) - Database patterns and constraints
- [ERROR_SEMANTICS.md](ERROR_SEMANTICS.md) - Error handling flows
