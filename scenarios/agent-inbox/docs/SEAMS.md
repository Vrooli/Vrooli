# Agent Inbox - Seams & Boundaries

This document describes the architectural seams and responsibility boundaries in the agent-inbox scenario, with particular focus on the async tool tracking system.

## Responsibility Zones

### 1. Entry/Presentation Layer

**Location:** `api/handlers/`

**Responsibilities:**
- HTTP request/response handling
- SSE streaming setup and teardown
- Input validation (chat IDs, tool call IDs)
- JSON serialization for responses

**Key Files:**
- `async_status.go` - SSE endpoint for async operation streaming
- `chat.go` - Chat CRUD operations
- `ai.go` - AI completion endpoints

**Boundaries:**
- Handlers call service methods, never domain logic directly
- Handlers do NOT build domain objects - they receive them from services
- SSE connection lifecycle is managed here, not in services

### 2. Coordination/Orchestration Layer

**Location:** `api/services/`

**Responsibilities:**
- Orchestrating multi-step workflows
- Managing operation lifecycles
- Coordinating between components (AI, tools, storage)

**Key Files:**
- `completion.go` - Orchestrates AI completion flow including tool execution
- `async_tracker.go` - Coordinates async operation polling and notifications

**Boundaries:**
- Services own the "how" of workflows, not the "what" (domain rules)
- Services coordinate between repositories, executors, and trackers
- Services do NOT handle HTTP/transport concerns

### 3. Domain Layer

**Location:** `api/domain/`, `api/services/` (types)

**Responsibilities:**
- Domain types and their invariants
- Status constants and validation
- Data structures representing business concepts

**Key Files:**
- `domain/types.go` - Core domain types (Chat, Message, ToolCallRecord)
- `services/async_config.go` - Async status constants and configuration
- `services/async_tracker.go` (types only) - AsyncOperation, AsyncStatusUpdate

**Boundaries:**
- Domain types are pure data containers with minimal behavior
- Status transitions are validated by services, not in domain types
- Configuration constants live alongside the code that uses them

### 4. Integration Layer

**Location:** `api/integrations/`

**Responsibilities:**
- External service communication (LLM APIs, scenario tools)
- Protocol handling (JSON-RPC, HTTP)
- Error translation from external systems

**Key Files:**
- `tool_executor.go` - Executes tools on external scenarios
- `scenario_client.go` - HTTP client for scenario APIs
- `protocol_handler.go` - JSON-RPC protocol implementation

**Boundaries:**
- Integration layer translates external errors to domain errors
- Never exposes external response structures to services
- Handles retries and circuit breaking

### 5. Cross-Cutting Concerns

**Location:** `api/services/`, `api/resilience/`

**Responsibilities:**
- Logging
- Error handling patterns
- Retry/circuit breaker logic
- Configuration management

**Key Files:**
- `resilience/circuit_breaker.go` - Circuit breaker implementation
- `resilience/retry.go` - Retry logic with backoff
- `config/config.go` - Configuration loading

---

## Async Tool Tracking Architecture

### Component Diagram

```
                    ┌─────────────────────────────────────────────────────────┐
                    │                    HTTP Layer                            │
                    │  ┌───────────────────────────────────────────────────┐  │
                    │  │ handlers/async_status.go                          │  │
                    │  │  - StreamAsyncStatus (SSE endpoint)               │  │
                    │  │  - GetAsyncOperations                             │  │
                    │  │  - CancelAsyncOperation                           │  │
                    │  └───────────────────────────────────────────────────┘  │
                    └─────────────────────────┬───────────────────────────────┘
                                              │
                                              │ calls
                                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Service Layer                                      │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │ services/async_tracker.go                                              │ │
│  │                                                                         │ │
│  │  AsyncTrackerService                                                   │ │
│  │  ├── StartTracking()      - Begin tracking async operation             │ │
│  │  ├── StopTracking()       - Cancel and mark as cancelled               │ │
│  │  ├── GetOperation()       - Retrieve single operation                  │ │
│  │  ├── GetActiveOperations()- List non-completed operations              │ │
│  │  ├── SubscribeWithID()    - Subscribe to SSE updates                   │ │
│  │  ├── UnsubscribeByID()    - Clean up subscription                      │ │
│  │  ├── RegisterCompletionCallback() - AI loop notification               │ │
│  │  ├── CancelOperation()    - Execute cancel tool + stop                 │ │
│  │  ├── RecoverOperations()  - Load & resume from DB on startup           │ │
│  │  ├── Shutdown()           - Graceful shutdown of polling loops         │ │
│  │  └── GetCompletionEvents()- Query completion events since timestamp    │ │
│  │                                                                         │ │
│  │  Internal (unexported):                                                │ │
│  │  ├── pollLoop()           - Background polling goroutine               │ │
│  │  ├── processStatusResult()- Handle status tool response                │ │
│  │  ├── pushUpdateData()     - Send to all subscribers                    │ │
│  │  └── triggerCompletionCallback() - Notify AI loop                      │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
│                                                                              │
│  ┌─────────────────────────┐  ┌─────────────────────────────────────────┐  │
│  │ services/async_config.go│  │ services/field_extractor.go             │  │
│  │  - Poll intervals       │  │  - ExtractField()                       │  │
│  │  - Buffer sizes         │  │  - ExtractStringField()                 │  │
│  │  - Status constants     │  │  - ExtractIntField()                    │  │
│  │  - Cleanup config       │  │  - ContainsString()                     │  │
│  └─────────────────────────┘  └─────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                                              │
                                              │ uses
                                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Integration Layer                                    │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │ integrations/tool_executor.go                                          │ │
│  │  - ExecuteTool() - Executes tools on external scenarios                │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
                                              │
                 ┌────────────────────────────┼────────────────────────────┐
                 │ HTTP calls                 │ SQL                        │
                 ▼                            ▼                            │
┌────────────────────────────────┐  ┌────────────────────────────────────────┐
│      External Scenarios        │  │          Persistence Layer             │
│  - agent-manager (agent runs)  │  │  ┌──────────────────────────────────┐  │
│  - browser-automation-studio   │  │  │ persistence/async_operation.go   │  │
│  - Any scenario with async     │  │  │  - CreateAsyncOperation()        │  │
│    tools                       │  │  │  - UpdateAsyncOperation()        │  │
└────────────────────────────────┘  │  │  - GetAllActiveAsyncOperations() │  │
                                    │  │  - CreateCompletionEvent()       │  │
                                    │  │  - GetCompletionEventsSince()    │  │
                                    │  └──────────────────────────────────┘  │
                                    │                                        │
                                    │  Tables:                               │
                                    │  - async_operations                    │
                                    │  - async_completion_events             │
                                    └────────────────────────────────────────┘
```

### Data Flow: Async Operation Lifecycle

```
1. TOOL EXECUTION
   ─────────────────────────────────────────────────────────────────────────
   CompletionService.ExecuteToolCalls()
       │
       ├─► toolExecutor.ExecuteTool("run-agent", {...})
       │       │
       │       └─► Returns: {"run_id": "run_abc123", "status": "pending"}
       │
       └─► maybeStartAsyncTracking()
               │
               ├─► toolRegistry.GetToolByName() - check for AsyncBehavior
               │
               └─► asyncTracker.StartTracking()

2. TRACKING INITIALIZATION
   ─────────────────────────────────────────────────────────────────────────
   AsyncTrackerService.StartTracking()
       │
       ├─► extractOperationID() - get "run_abc123" from result
       │
       ├─► Create AsyncOperation record
       │
       ├─► Store in operations map (mutex protected)
       │
       ├─► pushUpdateData() - notify SSE subscribers of new operation
       │
       └─► go pollLoop() - start background polling goroutine

3. BACKGROUND POLLING
   ─────────────────────────────────────────────────────────────────────────
   pollLoop(ctx, op)
       │
       ├─► snapshotOperation() - copy immutable config for thread safety
       │
       └─► Loop until terminal or timeout:
               │
               ├─► Wait poll interval (default 5s)
               │
               ├─► callStatusToolWithSnapshot()
               │       │
               │       └─► toolExecutor.ExecuteTool("get-run-status", {"run_id": "run_abc123"})
               │
               ├─► processStatusResult()
               │       │
               │       ├─► ExtractStringField(result, "data.run.status")
               │       ├─► Check against success_values / failure_values
               │       ├─► Extract progress, message, phase if configured
               │       ├─► Update operation (mutex protected)
               │       └─► pushUpdateData() - notify subscribers
               │
               └─► If terminal:
                       │
                       ├─► triggerCompletionCallback() - notify AI loop
                       └─► Return (goroutine exits)

4. SSE DELIVERY TO UI
   ─────────────────────────────────────────────────────────────────────────
   StreamAsyncStatus handler
       │
       ├─► SubscribeWithID(chatID) - get buffered channel
       │
       ├─► Send initial operations snapshot
       │
       └─► Loop until client disconnects:
               │
               └─► Select on sub.Channel or r.Context().Done()
                       │
                       └─► Marshal update to JSON, write SSE event

5. AI LOOP CONTINUATION
   ─────────────────────────────────────────────────────────────────────────
   Streaming completion handler
       │
       ├─► RegisterCompletionCallback(chatID) - get callback channel
       │
       └─► Select on callback channel or timeout:
               │
               └─► Receive AsyncCompletionEvent
                       │
                       └─► Continue AI conversation with result
```

### Concurrency Model

**Thread Safety Mechanisms:**

1. **Mutex Protection (`sync.RWMutex`)**
   - All map operations (operations, subscribers, subscriptions, callbacks)
   - Read lock for queries, write lock for modifications

2. **Immutable Snapshots (`OperationSnapshot`)**
   - Polling goroutine captures config once at start
   - Avoids repeated lock acquisitions during long-running polls
   - Proto AsyncBehavior is immutable after creation

3. **Non-Blocking Channel Sends**
   - `pushUpdateData()` uses `select` with `default` case
   - Full channels cause dropped updates (logged), not deadlocks
   - Buffer sizes tuned in `async_config.go`

4. **Context Cancellation**
   - Each poll goroutine has its own cancellable context
   - `StopTracking()` cancels context, goroutine exits cleanly
   - `cancelFuncs` map tracks cancel functions by toolCallID

**Race Conditions Avoided:**
- Building updates while holding mutex, then releasing before push
- Using snapshot for config reads in poll loop
- Non-blocking sends prevent subscriber slowdown from blocking tracker

### Notification Systems

| System | Purpose | Consumer | Buffer Size | Behavior on Full |
|--------|---------|----------|-------------|------------------|
| SSE Subscribers | Real-time UI updates | Web clients | 100 | Drop update, log warning |
| Completion Callbacks | AI loop continuation | CompletionService | 10 | Drop event, log warning |

### Configuration Constants

| Constant | Default | Purpose |
|----------|---------|---------|
| `DefaultPollInterval` | 5s | Time between status checks (initial) |
| `MinPollInterval` | 1s | Minimum allowed interval |
| `DefaultMaxPollDuration` | 1h | Max time before timeout |
| `SubscriberChannelBufferSize` | 100 | SSE channel buffer |
| `CompletionCallbackBufferSize` | 10 | AI callback buffer |
| `DefaultCleanupInterval` | 5m | Cleanup routine frequency |
| `DefaultCleanupRetention` | 30m | How long to keep completed ops |

### Backoff Configuration (Proto Defaults)

| Field | Default | Purpose |
|-------|---------|---------|
| `initial_interval_seconds` | 5 | Starting poll interval |
| `max_interval_seconds` | 30 | Maximum poll interval after backoff |
| `multiplier` | 1.5 | Interval growth factor per poll |

---

## Interface Contracts

### AsyncTrackerInterface

Defined in `services/interfaces.go` for dependency injection and testing:

```go
type AsyncTrackerInterface interface {
    GetActiveOperations(chatID string) []*AsyncOperation
    GetOperation(toolCallID string) *AsyncOperation
    StartTracking(ctx, toolCallID, chatID, toolName, scenario string,
                  toolResult interface{}, asyncBehavior *toolspb.AsyncBehavior) error
}
```

### Proto-Based Configuration

Tool async behavior is defined in `packages/proto/schemas/agent-inbox/v1/domain/tool.proto`:

- `AsyncBehavior` - Top-level async configuration
- `StatusPolling` - How to poll for status
- `CompletionConditions` - When to consider operation done
- `ProgressTracking` - Optional progress extraction paths
- `CancellationBehavior` - Optional cancel tool configuration

---

## Testing Strategy

### Unit Tests (`async_tracker_test.go`)

- Initialization verification
- Input validation (missing config, missing operation ID)
- Operation lifecycle (start, stop, remove)
- Subscription management (subscribe, unsubscribe)
- Field extraction (dot-notation paths)
- Status processing (success, failure, in-progress)

### Integration Tests (`async_tracker_integration_test.go`)

- Full operation lifecycle with subscribers and callbacks
- Multiple concurrent operations
- Cleanup of stale operations
- Concurrent subscribe/unsubscribe

### Concurrency Tests (run with `-race`)

- Concurrent StartTracking calls
- Concurrent Subscribe/Unsubscribe
- Concurrent update pushing
- Goroutine leak detection

---

## Persistence Layer

### Database Tables

The async tracking system persists state for crash recovery and multi-consumer callbacks:

| Table | Purpose |
|-------|---------|
| `async_operations` | Stores active and completed operations for crash recovery |
| `async_completion_events` | Stores completion events for multi-consumer querying |

### Crash Recovery

On server startup, `RecoverOperations()` is called to:
1. Load all active (non-terminal) operations from the database
2. Perform a fresh status check for each operation
3. Resume polling loops for operations that are still running

This ensures that long-running operations survive server restarts without losing progress.

### Graceful Shutdown

On server shutdown, `Shutdown()` is called to:
1. Cancel all active polling goroutines
2. Mark in-progress operations as "interrupted" in the database
3. Allow cleanup before database connection closes

### Multi-Consumer Callbacks

In addition to channel-based callbacks, completion events are persisted to `async_completion_events`:
- Multiple consumers can query `GetCompletionEvents(chatID, since)` for events since a timestamp
- Events are retained in the database for later retrieval
- Useful for AI handlers that start after an operation completes

---

## Exponential Backoff

The polling system supports exponential backoff to reduce load on external services:

### Configuration (via Proto)

Backoff is configured per-tool in the `PollingBackoff` proto message:
- `initial_interval_seconds` - Starting interval (default: 5s)
- `max_interval_seconds` - Maximum interval (default: 30s)
- `multiplier` - Growth factor (default: 1.5)

### Behavior

1. First poll uses `initial_interval_seconds`
2. Each subsequent poll: `interval = min(interval * multiplier, max_interval)`
3. Polling continues at `max_interval_seconds` until completion

### Example

With defaults (5s initial, 30s max, 1.5x multiplier):
```
Poll 1: 5s
Poll 2: 7.5s
Poll 3: 11.25s
Poll 4: 16.875s
Poll 5: 25.3s
Poll 6+: 30s (capped)
```
