# Database Backoff Rule - Implementation Notes

## Current Scope (v1.0)

The `database_backoff` rule currently checks:
- ✅ Database connection retry loops containing `db.Ping()` calls
- ✅ Presence of exponential backoff (via `math.Pow()` or bit shifts)
- ✅ Presence of random jitter (via `rand.Float64()`, `rand.Intn()`, or `time.Now().UnixNano()`)
- ✅ Presence of `time.Sleep()` between retry attempts

## Intentional Scope Limitations

### What This Rule Does NOT Check

1. **Redis/Other Stateful Stores**: Currently only detects PostgreSQL/MySQL `Ping()` calls
   - 16 Redis connections exist in codebase (as of 2025-10-01)
   - Neo4j, Elasticsearch, RabbitMQ, Kafka not present in current codebase
   - **Rationale**: Adding these would create 16+ immediate violations across many scenarios

2. **Missing Retry Logic**: Does not flag connections without any retry loop
   - Example: `scenario-authenticator/api/db/connection.go:99` - Redis connection with no retries
   - **Rationale**: This is a different concern - needs separate rule for "missing retry logic"

3. **Retry Library Detection**: Cannot verify jitter in abstracted retry libraries
   - Example: `backoff.Retry(func() error { return db.Ping() })` passes without verifying library implementation
   - **Rationale**: We trust established libraries like `github.com/cenkalti/backoff`

## Design Decisions

### Context-Aware Severity

The rule uses **dynamic severity** based on file path:

**High Severity** (default):
- Production API code
- Long-running services
- Multi-instance deployments

**Medium Severity** (reduced):
- `/migrate/` or `/migration/` - Migration scripts (single-instance)
- `_test.go` or `/test/` - Test files (not production)
- `/scripts/` or `/tools/` - CLI utilities
- `/initialization/` or `/init/` - Setup scripts

**Rationale**: Thundering herd is only a concern when multiple instances retry simultaneously. Single-instance scripts cannot create a thundering herd, even without jitter.

### Accepted Jitter Implementations

The rule accepts these randomness sources:

1. **`math/rand.Float64()`** / **`rand.Intn()`**
   - ✅ Standard approach
   - ⚠️ Requires seeding in Go < 1.20 (not currently checked)
   - ✅ Good statistical distribution

2. **`time.Now().UnixNano()`**
   - ✅ No import required (already have `time`)
   - ✅ Sufficient entropy for jitter use case
   - ✅ No seeding required
   - ⚠️ Theoretically could collide if instances start at exact same nanosecond (extremely unlikely)

3. **`crypto/rand`**
   - ✅ Maximum entropy
   - ❌ Overkill for non-cryptographic jitter
   - ❌ Performance overhead
   - Currently not explicitly detected but would work via randomness checks

### Rejected Jitter Patterns

The rule **correctly rejects** these deterministic patterns:

```go
// ❌ REJECTED: Deterministic based on attempt number
jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))

// ❌ REJECTED: Zero jitter
jitter := 0 * time.Millisecond

// ❌ REJECTED: Constant jitter
jitter := 100 * time.Millisecond
```

**Why rejected**: All instances retry at mathematically identical intervals, causing thundering herd.

## Future Enhancements (v2.0)

### Recommended Additions

1. **New Rule: `stateful_store_retry_required`**
   - Flag **all** stateful store connections (Redis, Neo4j, etc.)
   - Detect connections without retry loops
   - Suggest appropriate retry strategy
   - **Separate from** `database_backoff` to avoid confusion

2. **Enhanced Detection**
   - Check for `rand.Seed()` usage in Go < 1.20
   - Detect retry library patterns (`backoff.Retry`, `retry.Do`, etc.)
   - Warn about unseeded `math/rand` (produces deterministic sequences)

3. **Configuration Support**
   - Allow per-scenario severity overrides
   - Configurable file path patterns for context awareness
   - Adjustable jitter requirements (e.g., minimum randomness percentage)

### Why Not Expand Now?

**Incremental > Aggressive**:
- Current rule works correctly for stated purpose
- Expanding scope would create 16+ violations immediately
- Would require coordinated fix across many scenarios
- Better to gather data first, then decide on separate rules

**Separation of Concerns**:
- "Missing retry logic" is different from "bad jitter in existing retry"
- Different violations require different fixes
- Separate rules provide clearer actionable guidance

## Production Fixes Applied

### Fixed Files (2025-10-01)

1. **`scenario-authenticator/api/db/connection.go:59`**
   - Changed from: `jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))`
   - Changed to: `jitter := time.Duration(time.Now().UnixNano() % int64(jitterRange))`

2. **`contact-book/api/main.go:223`**
   - Changed from: `jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))`
   - Changed to: `jitter := time.Duration(time.Now().UnixNano() % int64(jitterRange))`

Both files now pass the rule with zero violations.

## Test Coverage

### Current Test Cases (8 total)

**Passing Tests** (should-fail="false"):
- `PASS-postgres-backoff-real-random-jitter` - Standard `rand.Float64()` implementation
- `PASS-time-based-jitter-unixnano` - Using `time.Now().UnixNano()` for entropy
- `PASS-rand-intn-jitter` - Integer-based random jitter via `rand.Intn()`

**Failing Tests** (should-fail="true"):
- `FAIL-postgres-backoff-deterministic-jitter` - Fake jitter based on attempt number
- `FAIL-missing-jitter` - Exponential backoff without any jitter
- `FAIL-missing-exponential-backoff` - Linear delay (no exponential growth)
- `FAIL-missing-sleep-call` - No sleep between retry attempts
- `FAIL-zero-jitter-constant` - Jitter variable set to constant zero

All tests passing as of 2025-10-01.

## References

- [Exponential Backoff and Jitter](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/)
- [Thundering Herd Problem](https://en.wikipedia.org/wiki/Thundering_herd_problem)
- [Go math/rand Seeding](https://pkg.go.dev/math/rand#Seed)

---

**Last Updated**: 2025-10-01
**Rule Version**: 1.0
**Status**: ✅ Production Ready
