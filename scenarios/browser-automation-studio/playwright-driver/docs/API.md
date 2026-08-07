# Playwright Driver API Specification

## Overview

HTTP API for browser automation instruction execution. All endpoints accept and return JSON.

**Base URL**: `http://localhost:39400` (default)

---

## Endpoints

### GET /health

Health check endpoint.

**Response** (200 OK):
```json
{
  "status": "ok",
  "timestamp": "2025-01-26T12:34:56.789Z",
  "sessions": 3
}
```

---

### POST /session/start

Create a new browser automation session.

**Request Body**:
```json
{
  "execution_id": "550e8400-e29b-41d4-a716-446655440000",
  "workflow_id": "650e8400-e29b-41d4-a716-446655440000",
  "viewport": {
    "width": 1280,
    "height": 720
  },
  "reuse_mode": "fresh",  // "fresh" | "clean" | "reuse"
  "base_url": "https://example.com",  // optional
  "labels": {  // optional
    "environment": "test"
  },
  "required_capabilities": {  // optional
    "har": true,
    "video": true,
    "tracing": true,
    "tabs": false,
    "iframes": true
  },
  "electron_target": {  // optional; attach to a scenario-to-desktop target
    "target_id": "target-opaque-id",
    "cdp_endpoint": "http://127.0.0.1:43123",
    "renderer_id": "renderer-opaque-id",
    "renderer_url": "file:///controlled/index.html",
    "renderer_title": "Controlled Desktop",
    "context_id": "validation-context-id",
    "cdp_transport": "loopback-authenticated"
  },
  "validation_context": {  // optional run/cell binding
    "context_id": "validation-context-id",
    "scenario_name": "example-app",
    "artifact_digest": "sha256:...",
    "target_id": "target-opaque-id",
    "workflow_id": "workflow-id",
    "profile_id": "normal"
  }
}
```

When `electron_target` is present, BAS connects to the already-running,
scenario-to-desktop-owned Electron app. The endpoint must be an explicit
numeric loopback HTTP endpoint with no credentials or query data. BAS verifies
the renderer identity through `/json/list`, requires the target's context ID,
reuses the normal workflow and
instruction machinery, and detaches on session close without closing the
desktop page or context. The desktop target remains responsible for process,
profile, port, and artifact cleanup.

When supplied, `validation_context` must match the target, artifact, scenario,
workflow, and profile selected by the cell. Existing BAS `browser_profile`
extra headers (including the `X-Vrooli-Test-Mode` marker installed by
workflow-health) are applied to the attached browser context; BAS does not
acquire or bypass the workflow-health isolation lease.

**Response** (200 OK):
```json
{
  "session_id": "sess_abc123"
}
```

**Errors**:
- 400: Invalid request body
- 503: Session limit exceeded

---

### POST /session/:id/run

Execute an instruction in the session.

**Request Body**:
```json
{
  "instruction": {
    "index": 0,
    "node_id": "node-123",
    "type": "click",
    "params": {
      "selector": "#submit-button",
      "timeoutMs": 15000
    },
    "context": {},
    "metadata": {}
  }
}
```

**Response** (200 OK):
```json
{
  "schema_version": "automation-step-outcome-v1",
  "payload_version": "1",
  "step_index": 0,
  "node_id": "node-123",
  "step_type": "click",
  "success": true,
  "started_at": "2025-01-26T12:34:56.000Z",
  "completed_at": "2025-01-26T12:34:56.250Z",
  "duration_ms": 250,
  "final_url": "https://example.com/success",
  "console_logs": [
    {
      "type": "log",
      "text": "Button clicked",
      "timestamp": "2025-01-26T12:34:56.100Z"
    }
  ],
  "network": [
    {
      "type": "request",
      "url": "https://api.example.com/submit",
      "method": "POST",
      "timestamp": "2025-01-26T12:34:56.150Z"
    }
  ],
  "screenshot_base64": "iVBORw0KGgoAAAANS...",
  "screenshot_media_type": "image/png",
  "screenshot_width": 1280,
  "screenshot_height": 720,
  "dom_html": "<!DOCTYPE html><html>...",
  "dom_preview": "<!DOCTYPE html><html>..."
}
```

**Response** (200 OK with failure):
```json
{
  "schema_version": "automation-step-outcome-v1",
  "payload_version": "1",
  "step_index": 0,
  "node_id": "node-123",
  "step_type": "click",
  "success": false,
  "started_at": "2025-01-26T12:34:56.000Z",
  "completed_at": "2025-01-26T12:34:56.500Z",
  "duration_ms": 500,
  "failure": {
    "kind": "engine",
    "code": "SELECTOR_NOT_FOUND",
    "message": "Selector '#submit-button' not found",
    "retryable": true,
    "occurred_at": "2025-01-26T12:34:56.500Z",
    "source": "engine"
  }
}
```

**Errors**:
- 400: Invalid instruction
- 404: Session not found
- 500: Execution error

---

### POST /session/:id/reset

Reset session state (clear cookies, storage, navigate to about:blank).

**Response** (200 OK):
```json
{
  "status": "ok"
}
```

**Errors**:
- 404: Session not found

---

### POST /session/:id/close

Close and cleanup session.

**Response** (200 OK):
```json
{
  "status": "closed"
}
```

**Errors**:
- 404: Session not found

---

## Instruction Types

All 28 supported instruction types with their parameters:

### Navigation
- **navigate**: `{ url, timeoutMs?, waitUntil? }`
- **frame-switch**: `{ action: "enter" | "exit" | "parent", selector?, frameId?, frameUrl? }`

### Interaction
- **click**: `{ selector, timeoutMs?, button?, clickCount?, modifiers? }`
- **hover**: `{ selector, timeoutMs? }`
- **type**: `{ selector, text, timeoutMs?, delay? }`
- **focus**: `{ selector, timeoutMs? }`
- **blur**: `{ selector? }`

### Input
- **uploadfile**: `{ selector, filePath | filePaths }`
- **select**: `{ selector, value? | label? | index?, timeoutMs? }`
- **keyboard**: `{ key? | keys?, modifiers?, action? }`

### Wait
- **wait**: `{ selector?, timeoutMs?, state? }`

### Assertion
- **assert**: `{ selector, mode, expected?, timeoutMs? }`

### Extraction
- **extract**: `{ selector, attribute?, timeoutMs? }`
- **evaluate**: `{ script, args? }`

### Screenshot
- **screenshot**: `{ selector?, fullPage? }`

### Download
- **download**: `{ selector | url, timeoutMs? }`

### Scroll
- **scroll**: `{ selector? | x?, y? }`

### Storage
- **cookie-storage**: `{ operation, storageType, key?, value?, cookieOptions? }`

### Advanced
- **drag-drop**: `{ sourceSelector, targetSelector | offsetX, offsetY?, steps? }`
- **swipe**: `{ selector?, direction, distance? }`
- **pinch**: `{ selector?, scale }`
- **tab-switch**: `{ action, url? | index? | title? | urlPattern? }`
- **network-mock**: `{ operation, urlPattern, statusCode?, headers?, body?, delayMs? }`
- **rotate**: `{ orientation: "portrait" | "landscape", angle? }`

See `playwright-driver/src/handlers/` for detailed parameter documentation.

---

## Error Codes

### Session Errors
- `SESSION_NOT_FOUND`: Session ID doesn't exist
- `SESSION_LIMIT_EXCEEDED`: Max concurrent sessions reached

### Instruction Errors
- `SELECTOR_NOT_FOUND`: Element selector not found
- `NAVIGATION_FAILED`: Page navigation failed
- `TIMEOUT`: Operation timed out
- `VALIDATION_ERROR`: Invalid parameters

### System Errors
- `BROWSER_LAUNCH_FAILED`: Could not start browser
- `CONFIGURATION_ERROR`: Invalid configuration

---

## Common Patterns

### Execute Workflow
```javascript
// 1. Start session
const { session_id } = await fetch('/session/start', {
  method: 'POST',
  body: JSON.stringify({
    execution_id: uuid(),
    workflow_id: uuid(),
    viewport: { width: 1280, height: 720 },
    reuse_mode: 'fresh'
  })
}).then(r => r.json());

// 2. Execute instructions
for (const instruction of instructions) {
  const outcome = await fetch(`/session/${session_id}/run`, {
    method: 'POST',
    body: JSON.stringify({ instruction })
  }).then(r => r.json());

  if (!outcome.success) {
    console.error('Instruction failed:', outcome.failure);
    break;
  }
}

// 3. Close session
await fetch(`/session/${session_id}/close`, { method: 'POST' });
```

### Error Handling
```javascript
try {
  const outcome = await executeInstruction(instruction);

  if (!outcome.success) {
    const { kind, code, retryable } = outcome.failure;

    if (retryable && kind === 'timeout') {
      // Retry with longer timeout
      return retry(instruction, { timeoutMs: 60000 });
    }

    throw new Error(`${code}: ${outcome.failure.message}`);
  }
} catch (error) {
  console.error('Request failed:', error);
}
```

---

## Versioning

### Schema Versions
- **StepOutcome**: `automation-step-outcome-v1`
- **Payload**: `1`

Clients should check `schema_version` and `payload_version` for compatibility.

### API Version
Current: **v2.0** (TypeScript rewrite, all 28 instruction types)

---

## Rate Limiting

None currently. Future: max requests per second per client.

## Authentication

None. Assumes trusted local network. Do not expose to internet.

---

## Monitoring

### Metrics Endpoint
`GET /metrics` (if `METRICS_ENABLED=true`)

Returns Prometheus-format metrics.

### Health Check
`GET /health`

Returns session count and server status.
