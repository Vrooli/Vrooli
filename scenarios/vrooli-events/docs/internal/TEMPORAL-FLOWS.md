# Temporal Flows & Async Patterns

## Last Updated
2026-04-05

## Async Flows Identified

| Flow | Entry Point | Async Operations | Completion Signal |
|------|-------------|------------------|-------------------|
| Event ingestion → SSE delivery | `handleIngest` | `go func()` marshals + calls `broker.Publish()` which fans out to subscriber channels | Message appears in subscriber channel (or dropped count increments) |
| Background pruning | `store.StartPruner` goroutine in `main.go` | Periodic `Prune()` call on ticker (default 6h) | Prune result logged; next tick scheduled |
| SSE heartbeat | `broker.heartbeat` goroutine per subscriber | 30s ticker sends heartbeat message with dropped count | Message delivered to subscriber channel |
| SSE subscription lifecycle | `broker.Subscribe` | Channel created, heartbeat goroutine started | `cleanup()` function cancels context, removes subscriber |
| UI SSE connection | `subscribeSSE` in `ui/src/lib/api.ts` | `EventSource` connects, receives messages | `onEvent` callback fires; `close()` returned for teardown |
| SSE Last-Event-ID replay | `replayMissedEvents` in `handleSubscribe` | Synchronous store query + write before live stream | Replay flushed, then `streamLiveEvents` begins |

## Ordering Guarantees

- **Ingestion → broadcast**: The event is stored (with autoincrement ID) *before* the async broadcast goroutine launches. This ensures the event is durable and queryable even if the broadcast fails.
- **SSE replay → live stream**: `replayMissedEvents()` runs synchronously before `streamLiveEvents()` begins. This ensures no gap between replayed and live events.
- **Pruner ticks**: `time.NewTicker` guarantees ticks at the configured interval. If a prune operation takes longer than one interval, the next tick fires immediately but operations are serialized by the SQLite writer lock.

## Race Conditions

- **Publish during subscriber removal**: `Publish()` holds `RLock`, `cleanup()` acquires write lock. Safe — publish skips removed subscriber on next iteration. Channel write uses `select/default` so a closed channel after cleanup won't panic (subscriber is deleted from map first). **Verified**: `TestConcurrentPublishAndCleanup` (race-detector clean).
- **Concurrent inserts**: SQLite WAL allows concurrent reads but serializes writes via `db.SetMaxOpenConns(1)`. Under high throughput, writes queue behind the connection. Not a race condition but a latency concern at scale. **Verified**: `TestConcurrentUniqueInserts`, `TestConcurrentDuplicateInsert` (race-detector clean).
- **Heartbeat vs. close**: Heartbeat goroutine checks `ctx.Done()` before sending. `cleanup()` cancels context then removes subscriber. Order is safe — heartbeat exits before next tick. **Verified**: `TestBrokerCloseStopsHeartbeats`.
- **Concurrent subscribe**: Multiple goroutines can subscribe simultaneously; the write lock serializes map mutations. **Verified**: `TestConcurrentSubscribe` (race-detector clean).
- **Dropped count atomics**: `atomic.Int64.Add()` in publish path and `atomic.Int64.Swap()` in heartbeat are lock-free and race-safe. **Verified**: `TestDroppedCountConcurrency` (race-detector clean).

## Timing Assumptions

- **Pruner interval**: Defaults to 6 hours (configurable via `prune_interval_minutes` setting). Assumes pruning completes well within one interval. **Verified**: `TestStartPrunerDefaultInterval`.
- **SSE heartbeat**: 30-second default interval (configurable via `BrokerConfig.HeartbeatInterval`). Assumes clients treat missed heartbeats as connection loss after ~60s (2 missed). **Verified**: `TestHeartbeatTiming`.
- **Event delivery latency**: `Publish()` is synchronous fan-out — all matching subscribers get the message in the same goroutine. Fast for small subscriber counts; could block the publisher if many subscribers.
- **Broadcast goroutine**: Launched fire-and-forget from `handleIngest`. If marshaling fails, the error is logged but the event is already stored. No retry mechanism — subscribers catch up via Last-Event-ID on reconnect.

## Concurrency Concerns

- **Broker subscriber map**: Protected by `sync.RWMutex`. Read-heavy workload (publish) uses `RLock`; write-rare operations (subscribe/unsubscribe) use `Lock`.
- **Dropped count tracking**: Uses `atomic.Int64` — lock-free increment in publish path, swap-to-zero in heartbeat. No contention between publisher and heartbeat goroutines.
- **Store access**: The `Store` interface is used concurrently by HTTP handlers (insert, query) and the pruner goroutine. SQLite serializes writes internally via WAL. `db.SetMaxOpenConns(1)` prevents "database is locked" errors.
- **Cleanup idempotency**: Calling `cleanup()` twice is safe — the second call is a no-op (delete from map is idempotent, cancel() is idempotent). **Verified**: `TestCleanupIdempotent`.

## Initialization & Teardown

- **Startup sequence** (`main.go`): context.WithCancel() → NewSQLiteStore → reconcileMeta → NewBroker → go StartPruner → http.Server.Run. All dependencies are initialized before the HTTP server starts accepting requests.
- **Shutdown sequence**: Server.Run() returns → Cleanup callback fires → ctx cancel (stops pruner) → store.Close(). The broker's active subscribers are cleaned up when their request contexts are cancelled by the HTTP server shutdown.
- **broker.Close()**: Closes the `done` channel (stops all heartbeats), cancels all subscriber contexts, closes all subscriber channels, and resets the map. **Verified**: `TestBrokerCloseStopsHeartbeats`.

## Checkpoint Flows

- **SSE reconnection**: Clients reconnect with `Last-Event-ID` header containing the last received event's store ID. The server replays all events with ID > Last-Event-ID before switching to live stream. This is the primary progress continuity mechanism.
- **Pruner progress**: The pruner has no checkpoint — each run independently evaluates time and size retention. This is safe because prune is idempotent. **Verified**: `TestPruneIdempotent`.
- **Meta tracking**: `store_meta.total_payload_bytes` is updated transactionally with inserts and prunes. If drift occurs, `reconcileMeta()` recalculates from actual data on startup. **Verified**: `TestMetaConsistencyThroughPruneCycle`, `TestReconcileMeta`.
- **Context cancellation**: Insert operations respect context cancellation — a cancelled context rolls back the transaction cleanly with no partial state. **Verified**: `TestInsertContextCancellation`.
