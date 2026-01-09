# Agent Inbox Failure Modes and Recovery Paths

This document catalogs the failure modes of the Agent Inbox scenario, their impact, detection signals, and recovery strategies.

## Failure Topography Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Agent Inbox API                                │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                │
│  │   Database  │    │  OpenRouter │    │   Ollama    │                │
│  │  (Required) │    │  (Required) │    │  (Optional) │                │
│  └─────────────┘    └─────────────┘    └─────────────┘                │
│         │                  │                  │                         │
│         ▼                  ▼                  ▼                         │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │                    Failure Categories                            │  │
│  ├───────────────┬──────────────────┬───────────────────────────────┤  │
│  │  Unhealthy    │     Degraded     │        Healthy               │  │
│  │  - DB down    │  - Ollama down   │  - All systems go            │  │
│  │  - No writes  │  - No auto-name  │  - Full functionality        │  │
│  └───────────────┴──────────────────┴───────────────────────────────┘  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

## Error Categories

### 1. Validation Errors (CategoryValidation)

**When:** User input fails validation

**Error Codes:**
- `V001` - Invalid input format
- `V002` - Required field missing
- `V003` - Invalid UUID format
- `V004` - Invalid message role
- `V005` - Invalid view mode
- `V006` - Empty content
- `V007` - Missing tool_call_id for tool messages
- `V008` - Malformed JSON
- `V009` - No fields to update
- `V011` - No messages in chat

**Recovery:** `correct_input` - User should fix the input and retry

**HTTP Status:** 400 Bad Request

### 2. Not Found Errors (CategoryNotFound)

**When:** Requested resource doesn't exist

**Error Codes:**
- `N001` - Chat not found
- `N002` - Message not found
- `N003` - Label not found
- `N004` - Tool not found

**Recovery:** `verify_resource` - Verify the resource ID is correct

**HTTP Status:** 404 Not Found

### 3. Dependency Errors (CategoryDependency)

**When:** External services fail or are unavailable

**Error Codes:**
- `D001` - Database unavailable
- `D002` - Database query failed
- `D003` - OpenRouter unavailable
- `D004` - OpenRouter API error
- `D005` - Ollama unavailable
- `D006` - Agent Manager error
- `D007` - Tool execution failed

**Recovery:** `retry_with_backoff` or `check_dependency`

**HTTP Status:** 502 Bad Gateway

### 4. Configuration Errors (CategoryConfiguration)

**When:** Service is misconfigured

**Error Codes:**
- `C001` - Missing API key
- `C002` - Invalid configuration
- `C003` - Service not enabled

**Recovery:** `check_configuration`

**HTTP Status:** 503 Service Unavailable

### 5. Internal Errors (CategoryInternal)

**When:** Unexpected server errors

**Error Codes:**
- `I001` - Internal error
- `I002` - Streaming error
- `I003` - Serialization error

**Recovery:** `escalate` or `retry`

**HTTP Status:** 500 Internal Server Error

---

## Critical Failure Paths

### Database Connection Failure

**Flow:**
```
Request → Handler → Repository.DB().Ping() → FAILURE
```

**Signals:**
- Health endpoint: status = "unhealthy"
- Health endpoint: dependencies.database.status = "disconnected"
- Logs: `[ERROR] [request_id] ... 500 ...`

**Impact:**
- All CRUD operations fail
- Chat completion fails
- Service cannot accept traffic

**Recovery:**
1. Check database container/service status
2. Verify connection string in environment
3. Check network connectivity
4. Restart service if connection is restored

**Graceful Degradation:** None - database is required

### OpenRouter API Failure

**Flow:**
```
ChatComplete → CreateCompletion → HTTP Request → FAILURE
```

**Signals:**
- API returns 4xx/5xx
- SSE error event: code = "D003" or "D004"
- Logs: `[ERROR] [request_id] POST /api/v1/chats/.../complete 502 ...`

**Impact:**
- Chat completion fails
- No AI responses generated

**Recovery:**
1. Check OPENROUTER_API_KEY is set
2. Verify API key is valid
3. Check OpenRouter service status
4. Retry with exponential backoff

**Error Response:**
```json
{
  "error": {
    "code": "D003",
    "category": "dependency",
    "message": "AI service temporarily unavailable",
    "recovery": "retry_with_backoff"
  },
  "request_id": "abc12345"
}
```

### Ollama Service Unavailable

**Flow:**
```
AutoName → OllamaClient.GenerateChatName() → FAILURE → FallbackName()
```

**Signals:**
- Health endpoint: status = "degraded"
- Health endpoint: capabilities.auto_naming = false
- Logs: `auto-name failed, using fallback | error=...`

**Impact:**
- Auto-naming disabled
- Chats use fallback name "New Conversation"

**Recovery:**
1. Start Ollama service
2. Verify Ollama is accessible on configured port
3. Service auto-recovers when Ollama becomes available

**Graceful Degradation:** ✅ Yes
- Fallback to default name
- User can manually rename

### Agent Manager Unavailable

**Flow:**
```
ExecuteToolCalls → AgentManagerClient → FAILURE → Record with status="failed"
```

**Signals:**
- Tool call record: status = "failed", error_message = "..."
- SSE event: type = "tool_call_result", status = "failed"
- Logs: `[WARN] [request_id] ... tool execution failed ...`

**Impact:**
- Coding agent tools fail
- spawn_coding_agent, check_agent_status, etc. unavailable

**Recovery:**
1. Start agent-manager scenario
2. Check AGENT_MANAGER_API_URL configuration
3. Verify agent-manager health endpoint

**Error Response (SSE):**
```json
{
  "type": "tool_call_result",
  "tool_name": "spawn_coding_agent",
  "tool_id": "call_123",
  "status": "failed",
  "error": "agent-manager not available"
}
```

### Streaming Interruption

**Flow:**
```
handleStreamingResponse → parseStreamingChunks → NETWORK FAILURE
```

**Signals:**
- SSE stream terminates unexpectedly
- Client receives partial content
- No "done" event received

**Impact:**
- Partial AI response
- Message may not be saved
- Client must handle incomplete state

**Recovery:**
1. Client should detect missing "done" event
2. Client can refetch chat messages
3. Retry completion if appropriate

**Graceful Degradation:**
- Partial content already sent is preserved
- Client should handle gracefully

---

## Recovery Action Reference

| Action | Description | Automated? |
|--------|-------------|------------|
| `retry` | Immediate retry may succeed | Yes |
| `retry_with_backoff` | Wait before retrying | Yes |
| `correct_input` | User must fix input | No |
| `check_configuration` | Admin must fix config | No |
| `check_dependency` | Verify external service | No |
| `verify_resource` | Check resource exists | No |
| `escalate` | Manual intervention needed | No |
| `none` | No recovery possible | - |

---

## Signal and Feedback Surfaces

### For Users (UI)

1. **Loading states** - Show when operations are in progress
2. **Error messages** - Display user-friendly messages with guidance
3. **Streaming indicators** - Show content as it arrives
4. **Tool call status** - Show running/completed/failed states

### For Agents (API)

1. **Structured errors** - Machine-readable codes and recovery hints
2. **Health endpoint** - Capability reporting for feature availability
3. **SSE events** - Typed events with status information
4. **Request IDs** - Correlation between errors and logs

### For Operators (Logs)

1. **Structured logs** - `[LEVEL] [request_id] METHOD path status duration`
2. **Health checks** - Dependency status with latency
3. **Error context** - Operation name, error details
4. **Degradation events** - When optional services fail

---

## Testing Failure Paths

### Simulate Database Failure
```bash
# Stop postgres container
docker stop agent-inbox-postgres

# Verify health endpoint shows unhealthy
curl http://localhost:PORT/health | jq '.status'
# Expected: "unhealthy"
```

### Simulate Ollama Unavailable
```bash
# Stop Ollama
vrooli resource stop ollama

# Verify degraded status
curl http://localhost:PORT/health | jq '.status'
# Expected: "degraded"

# Verify auto-naming uses fallback
curl -X POST http://localhost:PORT/api/v1/chats/ID/auto-name
# Expected: name = "New Conversation" (or similar fallback)
```

### Simulate OpenRouter Failure
```bash
# Unset API key
unset OPENROUTER_API_KEY

# Attempt completion
curl -X POST http://localhost:PORT/api/v1/chats/ID/complete
# Expected: 503 with code "C001"
```

---

## Adding New Failure Modes

When adding new failure paths:

1. **Define error code** in `domain/errors.go`
2. **Choose category** based on error source
3. **Specify recovery action** for callers
4. **Add graceful degradation** if possible
5. **Update health endpoint** if it affects capabilities
6. **Document in this file** with signals and recovery steps
