# System Invariants

## Last Updated
2026-04-05

## Critical Invariants

| Invariant | Domain Concept | Enforcement | Test Coverage |
|-----------|----------------|-------------|---------------|
| Event IDs follow `{scenario}.{domain}.{action}.{version}` format | Structured event identification | Regex validation in `handleIngest` | `api/handlers_test.go` [REQ:REQ-API-001] |
| SQLite WAL mode enabled on store open | Concurrent read access | `PRAGMA journal_mode=wal` in `NewSQLiteStore` | `internal/store/sqlite_test.go` [REQ:REQ-ES-001] |
| Subscriber channel buffer = 64 | Backpressure boundary | Constant `subscriberBufSize` in broker | `internal/broker/broker_test.go` [REQ:REQ-PS-004] |
| Events are immutable after insertion | Durable audit trail | No UPDATE/DELETE on events table (prune only deletes by retention) | `internal/store/pruner_test.go` [REQ:REQ-ES-003] |
| Glob matching uses segment-aware `*`/`**` | Pattern filtering consistency | `Match()` in `internal/broker/matcher.go` | `internal/broker/matcher_test.go` [REQ:REQ-PS-002] |

## Important Invariants

| Invariant | Domain Concept | Enforcement | Test Coverage |
|-----------|----------------|-------------|---------------|
| Health endpoint always returns JSON with `status`, `timestamp`, `readiness` | Health check contract | `handleHealth` handler | `api/handlers_test.go` [REQ:REQ-API-002] |
| Pruning respects both time-based and size-based retention | Dual-trigger pruning | `Prune()` checks `retention_days` and `max_payload_bytes` from settings | `internal/store/pruner_test.go` [REQ:REQ-ES-003] |
| SSE heartbeat fires every 30 seconds | Connection keepalive | `heartbeatInterval` constant | Verified in broker tests |

## Replay/Idempotency Invariants

| Operation | Idempotent? | Key | Test Coverage |
|-----------|-------------|-----|---------------|
| Event ingestion (POST) | No — UNIQUE constraint rejects duplicate event_id | `event_id` column | `TestDuplicateEventID`, `TestConcurrentDuplicateInsert` |
| SSE Last-Event-ID resume | Yes — replays events with ID > lastID | Store autoincrement ID | `TestGetSinceIdempotent` |
| Pruning | Yes — re-running deletes 0 if nothing new qualifies | Time/size thresholds | `TestPruneIdempotent` |
| broker.cleanup() | Yes — second call is no-op | Subscriber map key | `TestCleanupIdempotent` |
| context.CancelFunc | Yes — Go guarantees idempotent cancel | Context tree | `TestContextCancellationStopsDelivery` |
| Meta reconciliation | Yes — recalculates from actual data | SUM(LENGTH(payload)) | `TestReconcileMeta`, `TestMetaConsistencyThroughPruneCycle` |
| Webhook admission | Yes — one durable delivery row per subscription/event pair | `(subscription_id, event_id)` unique key | `TestIngest_FansOutDurablyWithSignatureAndHealth` |
| Dead-letter retry | Yes for a request replay — only a fully exhausted dead-letter row with the expected cumulative failed-attempt count can start one new bounded cycle | `(subscription_id, event_id, expected_attempts)` | `TestSQLiteStore_RetryDeadLetterUsesExpectedGenerationAndNewBoundedCycle` |
| Direct webhook delivery | No queue reconciliation — repeats send the supplied payload and leave durable delivery/health rows unchanged | None | `TestDeliverSubscription...` |

### Safe vs Unsafe Retry Patterns

- **Safe to retry**: Query, GetSince, Stats, Health, Prune, and an exact dead-letter retry with the same expected attempt count. The first exact retry changes `dead_letter` to `pending`; repeated requests conflict and do not start a second cycle.
- **Unsafe to retry without dedup**: Insert (duplicate event_id causes error, which is the correct behavior — callers should handle UNIQUE constraint errors as "already ingested")
- **One-shot, not queue retry**: `POST /subscriptions/:id/deliver` sends caller-supplied payload directly. It does not update a queued row or health counters and must not be used to claim a dead-letter was recovered.
- **Bounded delivery retry**: The dispatcher allows five failures per cycle using cumulative failed attempts modulo `DeliveryAttemptsPerCycle`. An explicit retry is accepted only for an exhausted cycle, so retry starts at a multiple of five without another stored boundary. Early terminal failures such as invalid queued payloads remain non-retryable because the endpoint cannot replace their immutable payload. `attempts` and `subscription_health.total_failed` remain cumulative; backoff resets for the newly opened cycle. A successful queued delivery increments `total_delivered` but does not increment the existing failed-attempt counter, retains historical failures, resets consecutive failures, and returns health to `active`.
- **Fire-and-forget (no retry)**: Broadcast to SSE subscribers from `handleIngest` goroutine. Failed broadcasts are not retried; subscribers catch up via Last-Event-ID on reconnect.

### Idempotency Keys

- **Event deduplication**: `event_id` UNIQUE constraint on the `events` table. The database enforces exactly-once storage per event_id.
- **SSE replay cursor**: Store autoincrement `id` column. Monotonically increasing, gap-free within a single process lifetime.
- **Pruner**: No explicit idempotency key — idempotency is structural (time/size thresholds are stable between runs).

## Enforcement Mechanisms

- **Type enforcement**: Go's type system prevents misuse of `Store` interface (e.g., `Insert` requires fully populated `Event` struct).
- **SQL constraints**: `events` table uses `NOT NULL` on `event_id`, `source_scenario`, `event_type`, `created_at`. `UNIQUE` on `event_id` prevents duplicate ingestion.
- **Transactional writes**: Insert and Prune use SQL transactions with `defer tx.Rollback()` — partial writes are impossible.
- **Atomic counters**: `atomic.Int64` for dropped count tracking — no lock contention between publishers and heartbeat.
- **Constant-based limits**: Buffer sizes and intervals are constants, not configurable at runtime, preventing misconfiguration.
