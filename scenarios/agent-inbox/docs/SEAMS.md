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

## Streaming Protocol Specification

### SSE Event Types

The completion endpoint (`POST /chats/{id}/complete?stream=true`) returns Server-Sent Events (SSE) with the following event structure:

```
data: {"type": "<event_type>", ...fields}
```

#### Event Type Reference

| Type | Description | Key Fields |
|------|-------------|------------|
| `content` | Streamed text chunk | `content`, `completion_id` |
| `image_generated` | AI-generated image | `image_url` |
| `tool_call_start` | Tool execution begins | `tool_id`, `tool_name`, `arguments` |
| `tool_call_result` | Tool execution complete | `tool_id`, `status`, `result`, `error`, `deactivate_template` |
| `tool_calls_complete` | All tools done, continuing | `continuing: true` |
| `tool_pending_approval` | Tool awaits user approval | `tool_call_id`, `tool_name`, `arguments` |
| `awaiting_approvals` | Stream paused for approvals | (no extra fields) |
| `async_waiting` | Paused for async tool | `operations: [{tool_call_id, tool_name, run_id}]` |
| `async_progress` | Async operation update | `tool_call_id`, `progress`, `phase`, `message` |
| `async_completed` | Async operation finished | `tool_call_id`, `status`, `result`, `error` |
| `error` | Structured error | `code`, `error`, `request_id` |
| `warning` | Non-fatal issue | `code`, `message` |
| `progress` | Phase/status update | `phase`, `message` |

### Event Flow Diagrams

#### Simple Completion (No Tools)
```
Client                                Server
  │                                     │
  │  POST /complete?stream=true         │
  │ ──────────────────────────────────► │
  │                                     │
  │     data: {"type":"content","content":"H"}
  │ ◄────────────────────────────────── │
  │     data: {"type":"content","content":"ello"}
  │ ◄────────────────────────────────── │
  │     data: [DONE]                    │
  │ ◄────────────────────────────────── │
```

#### Completion with Tool Calls
```
Client                                Server                          External Tool
  │                                     │                                    │
  │  POST /complete?stream=true         │                                    │
  │ ──────────────────────────────────► │                                    │
  │                                     │                                    │
  │     data: {"type":"tool_call_start", "tool_id":"...", "tool_name":"..."}
  │ ◄────────────────────────────────── │                                    │
  │                                     │  Execute tool                      │
  │                                     │ ─────────────────────────────────► │
  │                                     │                                    │
  │                                     │  Tool result                       │
  │                                     │ ◄───────────────────────────────── │
  │     data: {"type":"tool_call_result", "status":"completed", "result":...}
  │ ◄────────────────────────────────── │                                    │
  │     data: {"type":"tool_calls_complete", "continuing":true}
  │ ◄────────────────────────────────── │                                    │
  │                                     │                                    │
  │     (auto-continue: AI responds to tool results)                         │
  │     data: {"type":"content","content":"Based on the tool result..."}
  │ ◄────────────────────────────────── │                                    │
  │     data: [DONE]                    │                                    │
  │ ◄────────────────────────────────── │                                    │
```

#### Completion with Approval Required
```
Client                                Server
  │                                     │
  │  POST /complete?stream=true         │
  │ ──────────────────────────────────► │
  │                                     │
  │     data: {"type":"tool_pending_approval", "tool_call_id":"...", ...}
  │ ◄────────────────────────────────── │
  │     data: {"type":"awaiting_approvals"}
  │ ◄────────────────────────────────── │
  │     data: [DONE]                    │  (stream ends, waiting for action)
  │ ◄────────────────────────────────── │
  │                                     │
  │  POST /tool-calls/{id}/approve      │
  │ ──────────────────────────────────► │
  │     {"auto_continued": true, ...}   │
  │ ◄────────────────────────────────── │
  │                                     │
  │  (new completion triggered server-side if auto_continued)
```

#### Completion with Async Tool
```
Client                                Server                     Async Tool
  │                                     │                             │
  │  POST /complete?stream=true         │                             │
  │ ──────────────────────────────────► │                             │
  │                                     │                             │
  │     data: {"type":"tool_call_start", ...}
  │ ◄────────────────────────────────── │                             │
  │     data: {"type":"tool_call_result", "is_async":true, ...}
  │ ◄────────────────────────────────── │                             │
  │     data: {"type":"async_waiting", "operations":[...]}
  │ ◄────────────────────────────────── │                             │
  │                                     │  Start polling              │
  │                                     │ ──────────────────────────► │
  │                                     │                             │
  │  (stream stays open, waiting)       │  Poll status                │
  │                                     │ ◄────────────────────────── │
  │     data: {"type":"async_progress", "progress":50, ...}
  │ ◄────────────────────────────────── │                             │
  │                                     │                             │
  │                                     │  Operation complete         │
  │                                     │ ◄────────────────────────── │
  │     data: {"type":"async_completed", "status":"completed", ...}
  │ ◄────────────────────────────────── │                             │
  │                                     │                             │
  │     (auto-continue with async result)
  │     data: {"type":"content","content":"The operation completed..."}
  │ ◄────────────────────────────────── │                             │
  │     data: [DONE]                    │                             │
  │ ◄────────────────────────────────── │                             │
```

### Request ID Correlation

Each streaming request includes identifiers for correlation:

- **completion_id**: Unique per completion request, included in events
- **request_id**: Server-side trace ID, included in error events

Client-side usage (from `useCompletion.ts`):
```typescript
// Guard against stale events from cancelled requests
if (currentRequestIdRef.current !== requestId) {
  return; // Stale event from cancelled request
}
```

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

---

## UI Layer Seams

### Hook Boundaries

The UI uses React hooks to encapsulate domain logic. Each hook has clear boundaries:

| Hook | Responsibility | Seam Point |
|------|----------------|------------|
| `useCompletion` | AI streaming state, tool calls, approvals | `completeChat()` from api module |
| `useChats` | Chat list, selection, CRUD | `fetchChats()`, `createChat()`, etc. |
| `useAsyncStatus` | Async operation polling via SSE | SSE connection to `/async-status` |
| `useAttachments` | File upload state | `uploadAttachment()` |
| `useLabels` | Label management | `fetchLabels()`, `createLabel()`, etc. |
| `useTools` | Tool registry state | `fetchToolSet()` |

### Testing Seams

For testing UI components, hooks can be mocked at the API module level:

```typescript
// Test file
jest.mock('../lib/api', () => ({
  completeChat: jest.fn().mockImplementation(async (chatId, options) => {
    // Simulate streaming events
    options?.onEvent?.({ type: 'content', content: 'Test response' });
  }),
  // ... other mocks
}));
```

Alternatively, for integration tests, the `fetch` function can be intercepted:

```typescript
// Using MSW (Mock Service Worker)
import { rest } from 'msw';

const handlers = [
  rest.post('/api/v1/chats/:id/complete', (req, res, ctx) => {
    return res(ctx.status(200), ctx.body('data: {"type":"content","content":"Hello"}\n\n'));
  }),
];
```

### State Management Boundaries

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              App                                         │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                       Global State                                  │ │
│  │  - ToolsContext (tool registry, refresh)                           │ │
│  │  - No Redux/MobX - minimal global state                            │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  ┌─────────────────────────┐  ┌─────────────────────────────────────┐  │
│  │     useChats            │  │           useCompletion              │  │
│  │  (chat list, selection) │  │  (streaming, tool calls, approvals) │  │
│  │                         │  │                                      │  │
│  │  State:                 │  │  State:                              │  │
│  │  - chats[]              │  │  - isGenerating                      │  │
│  │  - selectedChat         │  │  - streamingContent                  │  │
│  │  - viewState            │  │  - activeToolCalls[]                 │  │
│  │                         │  │  - pendingApprovals[]                │  │
│  │  Seam: React Query      │  │  - awaitingApprovals                 │  │
│  │  (cache invalidation)   │  │                                      │  │
│  └─────────────────────────┘  │  Seam: AbortController               │  │
│                               │  (request cancellation)              │  │
│                               └─────────────────────────────────────┘  │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                        API Client Layer                            │ │
│  │                      (ui/src/lib/api.ts)                           │ │
│  │                                                                     │ │
│  │  Primary Seam: All HTTP calls go through this module               │ │
│  │  - Type-safe request/response handling                             │ │
│  │  - SSE streaming abstraction                                       │ │
│  │  - URL building via buildApiUrl()                                  │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

### React Patterns for Stability

The `useCompletion` hook uses several patterns to prevent render storms:

1. **Stable Empty Arrays**: Prevents reference changes
   ```typescript
   const EMPTY_IMAGES: string[] = [];
   const EMPTY_TOOL_CALLS: ActiveToolCall[] = [];
   ```

2. **Request ID Guards**: Prevents stale event handling
   ```typescript
   if (currentRequestIdRef.current !== requestId) {
     return; // Stale event
   }
   ```

3. **startTransition**: Batches non-urgent state updates
   ```typescript
   startTransition(() => {
     setStreamingContent((prev) => prev + event.content);
   });
   ```

4. **Memoized Return**: Prevents object reference changes
   ```typescript
   return useMemo(() => ({ ...state, ...actions }), [dependencies]);
   ```

---

## Handler Responsibilities

### API Handler Boundaries

| Handler File | Responsibility | Should NOT Do |
|--------------|----------------|---------------|
| `ai.go` | HTTP for completions, SSE setup | Business logic (delegates to services) |
| `chat.go` | Chat CRUD HTTP endpoints | Tool execution, message tree logic |
| `message.go` | Message HTTP endpoints | AI completion orchestration |
| `async_status.go` | SSE for async updates | Polling logic (delegates to tracker) |
| `tools.go` | Tool discovery/config HTTP | Tool execution |
| `upload.go` | File upload HTTP | Storage implementation |

### Service Layer Boundaries

| Service | Owns | Delegates To |
|---------|------|--------------|
| `CompletionService` | Completion orchestration, tool call coordination | `ToolExecutor`, `AsyncTracker`, `Repository` |
| `AsyncTrackerService` | Operation lifecycle, polling, notifications | `ToolExecutor` (for status calls), `Repository` |
| `ToolRegistry` | Tool metadata cache, enabled state | `Repository` (for configs) |
| `ReconciliationService` | Startup recovery | `Repository` |

### Handler → Service → Repository Pattern

```
Handler                    Service                     Repository
   │                          │                            │
   │  Validate HTTP input     │                            │
   │  ──────────────────►     │                            │
   │                          │  Orchestrate workflow      │
   │                          │  ──────────────────────►   │
   │                          │                            │  SQL
   │                          │                            │ ─────►
   │                          │                            │
   │                          │  ◄──────────────────────   │
   │  Format HTTP response    │                            │
   │  ◄──────────────────     │                            │
```

### Error Translation Boundaries

| Layer | Error Type | Responsibility |
|-------|------------|----------------|
| Repository | `sql.ErrNoRows`, DB errors | Wrap in domain errors |
| Service | `ErrChatNotFound`, `ErrNoMessages` | Business rule validation |
| Handler | `AppError` → HTTP status | Map to status codes |

From `handlers/ai.go`:
```go
func mapCompletionErrorToStatus(err error) int {
    switch {
    case errors.Is(err, services.ErrChatNotFound):
        return http.StatusNotFound
    case errors.Is(err, services.ErrNoMessages):
        return http.StatusBadRequest
    // ...
    }
}
```

---

## Testability Seams Summary

### API-Side Seams

| Seam | Interface | Mock Strategy |
|------|-----------|---------------|
| Database | `CompletionRepository` | In-memory implementation |
| Tool execution | `ToolExecutorInterface` | Mock executor returning canned results |
| Async tracking | `AsyncTrackerInterface` | Mock tracker with immediate completion |
| LLM client | (not injectable) | Consider extracting `LLMClientInterface` |
| Ollama client | `OllamaClientInterface` | Mock for auto-naming tests |

### UI-Side Seams

| Seam | Injection Point | Mock Strategy |
|------|-----------------|---------------|
| API calls | `api.ts` module | Jest mock or MSW |
| SSE streaming | `completeChat()` | Call `onEvent` directly |
| Fetch | Global `fetch` | MSW intercept |

### Recommended Improvements

1. **Extract LLM Client Interface**: Currently `integrations.NewOpenRouterClient()` is called directly in handlers. Extracting to an interface would improve testability.

2. **Dependency Injection for Hooks**: Consider a `useApiClient()` hook that can be provided via context for easier testing.

3. **State Machine for Completion**: The boolean flags in `useCompletion` (`isGenerating`, `awaitingApprovals`) could be replaced with an explicit state machine (e.g., XState) for clearer state transitions.
