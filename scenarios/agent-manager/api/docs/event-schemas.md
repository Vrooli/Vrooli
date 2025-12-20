# Agent-Manager Event Schemas

This document describes the event types emitted by agent-manager runners during execution.

## Event Structure

All events share a common envelope:

```json
{
  "id": "uuid",
  "runId": "uuid",
  "sequence": 1,
  "eventType": "tool_call",
  "timestamp": "2025-12-20T04:18:42.709878Z",
  "data": { ... }
}
```

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Unique event identifier |
| runId | UUID | The run this event belongs to |
| sequence | int | Sequential order within the run (0-indexed) |
| eventType | string | Event type (see below) |
| timestamp | datetime | When the event occurred |
| data | object | Event-specific payload |

## Event Types

### tool_call

Emitted when an agent invokes a tool.

**Payload: ToolCallEventData**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| toolName | string | Yes | Name of the tool being called |
| input | object | Yes | Tool arguments as key-value pairs |

**Example:**
```json
{
  "eventType": "tool_call",
  "data": {
    "toolName": "Write",
    "input": {
      "file_path": "/tmp/test.txt",
      "content": "Hello world"
    }
  }
}
```

### tool_result

Emitted after a tool execution completes.

**Payload: ToolResultEventData**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| toolName | string | Yes | Name of the tool that was executed |
| toolCallId | string | No | Tool invocation ID for correlating with tool_call events |
| output | string | No | Tool output on success |
| error | string | No | Error message on failure |
| success | bool | Yes | Whether the tool call succeeded |

**Example (success):**
```json
{
  "eventType": "tool_result",
  "data": {
    "toolName": "Write",
    "toolCallId": "toolu_01GXZ12345",
    "output": "File created successfully at: /tmp/test.txt",
    "success": true
  }
}
```

**Example (failure):**
```json
{
  "eventType": "tool_result",
  "data": {
    "toolName": "Bash",
    "toolCallId": "call_01361367",
    "error": "command not found: foobar",
    "success": false
  }
}
```

### message

Emitted for conversation messages.

**Payload: MessageEventData**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| role | string | Yes | "user", "assistant", or "system" |
| content | string | Yes | Message text content |

**Example:**
```json
{
  "eventType": "message",
  "data": {
    "role": "assistant",
    "content": "I've created the file with the specified content."
  }
}
```

### metric

Emitted for usage and cost data.

**Payload: CostEventData**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| inputTokens | int | Yes | Input tokens used |
| outputTokens | int | Yes | Output tokens generated |
| cacheReadTokens | int | No | Tokens read from cache |
| cacheCreationTokens | int | No | Tokens written to cache |
| totalCostUsd | float | Yes | Estimated cost in USD |
| model | string | No | Model used (if available) |

**Example:**
```json
{
  "eventType": "metric",
  "data": {
    "inputTokens": 3562,
    "outputTokens": 401,
    "cacheReadTokens": 6144,
    "cacheCreationTokens": 0,
    "totalCostUsd": 0.00197528
  }
}
```

### status

Emitted when run status changes.

**Payload: StatusEventData**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| oldStatus | string | Yes | Previous status |
| newStatus | string | Yes | New status |
| reason | string | No | Reason for the change |

**Example:**
```json
{
  "eventType": "status",
  "data": {
    "oldStatus": "running",
    "newStatus": "complete",
    "reason": "Execution completed successfully"
  }
}
```

### log

Emitted for informational log messages.

**Payload: LogEventData**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| level | string | Yes | "debug", "info", "warn", or "error" |
| message | string | Yes | Log message content |

**Example:**
```json
{
  "eventType": "log",
  "data": {
    "level": "info",
    "message": "phase: Agent is executing"
  }
}
```

### error

Emitted when an error occurs.

**Payload: ErrorEventData**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| code | string | Yes | Machine-readable error code |
| message | string | Yes | Human-readable error message |
| retryable | bool | Yes | Whether the error is retryable |

**Example:**
```json
{
  "eventType": "error",
  "data": {
    "code": "rate_limit",
    "message": "Rate limit exceeded. Please wait 60 seconds.",
    "retryable": true
  }
}
```

## Runner-Specific Notes

### Claude Code

- Tool names match Claude Code tool names: `Read`, `Write`, `Edit`, `Bash`, `Glob`, `Grep`, etc.
- `tool_result.toolCallId` contains the tool invocation ID (e.g., `toolu_01GXZ...`) for correlation
- Full token breakdown including cache tokens
- Cost estimates in USD

**Sample tool_call:**
```json
{
  "toolName": "Write",
  "input": {
    "file_path": "/tmp/test.txt",
    "content": "Hello from Claude Code\n"
  }
}
```

### Codex

- Uses `file_change` as the tool name for file operations
- Input contains `files` array with `kind` ("add", "modify", "delete") and `path`
- Includes reasoning in log events (level: "debug")
- Token breakdown includes reasoning tokens
- `tool_result.toolCallId` is not available (empty string)

**Sample tool_call:**
```json
{
  "toolName": "file_change",
  "input": {
    "files": [
      {
        "kind": "add",
        "path": "/tmp/test.txt"
      }
    ],
    "status": "completed"
  }
}
```

### OpenCode

- Uses lowercase tool names: `write`, `bash`, `read`, etc.
- Input fields use camelCase: `filePath`, `content`
- Cost tracking via `step_finish` events
- Message events may not always be emitted
- `tool_result.toolCallId` contains the call ID (e.g., `call_01361367`)

**Sample tool_call:**
```json
{
  "toolName": "write",
  "input": {
    "content": "Hello from OpenCode",
    "filePath": "/tmp/test.txt"
  }
}
```

## Validation

Events are validated when emitted. Warnings are logged for:

- `tool_call` with empty or "unknown_tool" toolName
- `tool_call` with empty input
- `tool_result` failure with no error details
- `message` with empty content
- `metric` with zero tokens

These warnings help detect parsing issues and can be monitored in logs.
