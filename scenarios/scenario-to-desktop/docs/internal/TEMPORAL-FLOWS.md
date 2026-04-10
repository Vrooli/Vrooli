# Temporal Flows

## Last Updated
2026-04-06

## Pipeline Execution Flow

The core async pattern is the multi-stage pipeline orchestrator in [CODE: api/pipeline/orchestrator.go].

### Execution Modes

```
Client Request
    │
    ├─ ?block=true ──→ RunPipelineBlocking(ctx, config, timeoutSecs)
    │                   - Extends HTTP write deadline
    │                   - Returns when pipeline completes or timeout
    │
    └─ (default) ────→ RunPipeline(context.Background(), config)
                        - Returns pipeline ID immediately
                        - Pipeline runs independent of HTTP lifecycle
                        - Client polls GET /status/{id} for updates
```

### Stage Progression

```
Bundle → Preflight → Generate → Build → SmokeTest → Deploy
  │                                                     │
  └── Any failure ──→ Pipeline marked Failed ───────────┘
                      (no automatic retry)
```

Each stage updates `PipelineStatus` atomically via `InMemoryStore.UpdateStage()`.

## Concurrency Model

### Store Thread Safety

`InMemoryStore` in [CODE: api/pipeline/store.go] uses `sync.RWMutex`:

| Operation | Lock Type | Notes |
|-----------|-----------|-------|
| `Get()` | RLock | Concurrent reads safe |
| `Save()` | Lock | Exclusive write |
| `Update(fn)` | Lock | Callback executes under write lock |
| `UpdateStage()` | Lock | Atomic stage update |
| `GetByIdempotencyKey()` | RLock | Concurrent lookup safe |
| `Cleanup()` | Lock | Removes old entries |

### Goroutine Patterns

- **Async pipeline**: Launched as a goroutine with `context.Background()` — survives HTTP connection close
- **Blocking pipeline**: Runs in HTTP handler goroutine with extended deadline
- **Runtime health polling**: `main.ts` polls `/ready`, `/ports`, `/secrets` endpoints from Electron main process
- **Child process management**: Bundled runtime spawned via Node.js `fork()` with lifecycle monitoring

## Race Condition Guards

| Scenario | Guard | Location |
|----------|-------|----------|
| Duplicate pipeline submission | Idempotency key check before start | `orchestrator.go` |
| UI double-click | `isSubmitting` flag + idempotency key | `pipelineStore.ts` |
| Concurrent status reads during update | RWMutex on store | `store.go` |
| Pipeline timeout vs completion race | Context cancellation + status check | `orchestrator.go` |

## Checkpoint Flows

Pipeline stages are checkpointed to the store after each stage completes. On resume:
1. Load pipeline status from store
2. Determine last completed stage
3. Resume from next stage (stages are not re-run)
4. `resumePipeline()` in UI resets `isSubmitting` with fresh idempotency key
