# Playwright Driver v2.0 - Architecture

## Overview

The Playwright Driver is a TypeScript-based HTTP service that executes browser automation instructions using Playwright. It serves as the execution engine for the Vrooli Ascension, providing a clean abstraction over Playwright's API while maintaining full compatibility with the Go `AutomationEngine` contract.

**Key Characteristics**:
- **Type-safe**: 100% TypeScript with strict mode
- **Modular**: Clear separation of concerns across 50+ files
- **Contract-compliant**: Implements all Go automation contracts
- **Production-ready**: Structured logging, metrics, error handling
- **Comprehensive**: All 28 instruction types supported

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Go API (automation/engine/playwright_engine.go)            │
│  - PlaywrightEngine: HTTP client                           │
│  - Translates CompiledInstruction → HTTP JSON              │
│  - Decodes StepOutcome ← HTTP JSON                         │
└────────────────┬────────────────────────────────────────────┘
                 │ HTTP/JSON
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ Playwright Driver (Node.js/TypeScript)                     │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ HTTP Server (src/server.ts)                          │  │
│  │  - Express-like request routing                      │  │
│  │  - Middleware pipeline                               │  │
│  │  - Error handling                                    │  │
│  └────────────┬─────────────────────────────────────────┘  │
│               │                                             │
│  ┌────────────▼──────────────┬──────────────┬───────────┐  │
│  │ Routes                    │ Middleware   │ Utils     │  │
│  │  - /health                │  - Body      │  - Logger │  │
│  │  - /session/start         │    parser    │  - Metrics│  │
│  │  - /session/:id/run       │  - Error     │  - Errors │  │
│  │  - /session/:id/reset     │    handler   │           │  │
│  │  - /session/:id/close     │              │           │  │
│  └───────────────────────────┴──────────────┴───────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Session Manager (src/session/manager.ts)             │  │
│  │  - Browser/context/page lifecycle                    │  │
│  │  - Session pooling and reuse                         │  │
│  │  - Resource limits enforcement                       │  │
│  │  - Idle cleanup                                      │  │
│  └────────────┬─────────────────────────────────────────┘  │
│               │                                             │
│  ┌────────────▼──────────────┬──────────────┬───────────┐  │
│  │ Handlers                  │ Telemetry    │ Types     │  │
│  │  - Base handler           │  - Console   │  - Contracts│
│  │  - Registry               │  - Network   │  - Session│
│  │  - 28 implementations     │  - Screenshot│  - Instruction│
│  │                           │  - DOM       │           │  │
│  └───────────────────────────┴──────────────┴───────────┘  │
│                                                             │
└────────────────┬────────────────────────────────────────────┘
                 │ Playwright API
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ Playwright (Node.js library)                                │
│  - Chromium browser automation                              │
│  - CDP protocol                                             │
│  - Native screenshot/DOM/network capture                    │
└─────────────────────────────────────────────────────────────┘
```

---

## Directory Structure

```
playwright-driver/
├── src/
│   ├── server.ts                 # HTTP server entry point
│   ├── config.ts                 # Configuration management
│   ├── constants.ts              # Shared constants
│   │
│   ├── types/                    # Type definitions
│   │   ├── contracts.ts          # Go contract types
│   │   ├── session.ts            # Session types
│   │   ├── instruction.ts        # Instruction parameter schemas
│   │   └── index.ts              # Type exports
│   │
│   ├── session/                  # Session lifecycle
│   │   ├── manager.ts            # SessionManager class
│   │   ├── context-builder.ts   # BrowserContext configuration
│   │   ├── cleanup.ts            # Idle session cleanup
│   │   └── index.ts
│   │
│   ├── handlers/                 # Instruction handlers (28 total)
│   │   ├── base.ts               # BaseHandler interface
│   │   ├── registry.ts           # Handler registration
│   │   ├── navigation.ts         # navigate
│   │   ├── interaction.ts        # click, hover, type, focus, blur
│   │   ├── wait.ts               # wait
│   │   ├── assertion.ts          # assert
│   │   ├── extraction.ts         # extract, evaluate
│   │   ├── screenshot.ts         # screenshot
│   │   ├── upload.ts             # uploadfile
│   │   ├── download.ts           # download
│   │   ├── scroll.ts             # scroll
│   │   ├── frame.ts              # frame-switch (iframe navigation)
│   │   ├── cookie-storage.ts    # cookie/localStorage ops
│   │   ├── select.ts             # select (dropdown)
│   │   ├── keyboard.ts           # keyboard, shortcut
│   │   ├── gesture.ts            # drag-drop, swipe, pinch/zoom
│   │   ├── tab.ts                # tab-switch (multi-tab)
│   │   ├── network.ts            # network-mock (interception)
│   │   ├── device.ts             # rotate (orientation)
│   │   └── index.ts
│   │
│   ├── telemetry/                # Observability data collection
│   │   ├── collector.ts          # Console/network collectors
│   │   ├── screenshot.ts         # Screenshot capture
│   │   ├── dom.ts                # DOM snapshot
│   │   └── index.ts
│   │
│   ├── outcome/                  # Wire-format transformation
│   │   ├── outcome-builder.ts    # HandlerResult → StepOutcome → DriverOutcome
│   │   └── index.ts
│   │
│   ├── routes/                   # HTTP endpoints
│   │   ├── health.ts             # GET /health
│   │   ├── session-start.ts     # POST /session/start
│   │   ├── session-run.ts       # POST /session/:id/run
│   │   ├── session-reset.ts     # POST /session/:id/reset
│   │   ├── session-close.ts     # POST /session/:id/close
│   │   └── index.ts
│   │
│   ├── middleware/               # HTTP middleware
│   │   ├── body-parser.ts       # JSON parsing
│   │   ├── error-handler.ts     # Error handling
│   │   └── index.ts
│   │
│   └── utils/                    # Utilities
│       ├── logger.ts             # Winston logger
│       ├── metrics.ts            # Prometheus metrics
│       ├── errors.ts             # Error hierarchy
│       └── index.ts
│
├── tests/                        # Test suite
│   ├── helpers/                  # Test utilities
│   ├── unit/                     # Unit tests
│   └── integration/              # Integration tests
│
├── docs/                         # Documentation
│   ├── ARCHITECTURE.md           # This file
│   ├── API.md                    # API specification
│   ├── HANDLERS.md               # Handler guide
│   ├── DEPLOYMENT.md             # Deployment guide
│   └── CONFIGURATION.md          # Configuration reference
│
├── package.json                  # Dependencies
├── tsconfig.json                 # TypeScript config
├── jest.config.js                # Jest config
└── README.md                     # Quick start
```

---

## Core Components

### 1. HTTP Server (`src/server.ts`)

**Responsibilities**:
- Accept HTTP requests
- Route to appropriate handlers
- Apply middleware pipeline
- Send HTTP responses
- Graceful shutdown

**Key Features**:
- Simple request routing (no Express dependency)
- Middleware support (body parsing, error handling)
- Signal handling (SIGTERM, SIGINT)
- Health endpoint for monitoring

### 2. Session Manager (`src/session/manager.ts`)

**Responsibilities**:
- Manage browser/context/page lifecycle
- Enforce resource limits (max concurrent sessions)
- Implement session reuse strategies (fresh/clean/reuse)
- Track session activity
- Clean up idle sessions

**Session Lifecycle**:
```
create() → browser.launch() → newContext() → newPage() → SessionState
                                                              ↓
                                                         run instructions
                                                              ↓
reset() → clear cookies/storage → navigate to about:blank
close() → page.close() → context.close() → browser.close()
```

**Reuse Modes**:
- **fresh**: New browser/context/page per session
- **clean**: Reuse browser, new context/page, reset storage
- **reuse**: Keep existing session state intact

### 3. Handler Registry (`src/handlers/registry.ts`)

**Responsibilities**:
- Register instruction handlers
- Look up handlers by instruction type
- Validate handler exists before execution

**Handler Pattern**:
```typescript
class MyHandler implements BaseHandler {
  getSupportedTypes(): string[] {
    return ['my-instruction-type'];
  }

  async execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    // 1. Parse and validate params
    const params = MyParamsSchema.parse(instruction.params);

    // 2. Execute Playwright action
    await context.page.doSomething(params);

    // 3. Return normalized result
    return { success: true };
  }
}
```

### 4. Telemetry Collectors (`src/telemetry/`)

**Responsibilities**:
- Capture console logs during instruction execution
- Capture network events (requests/responses)
- Capture screenshots
- Capture DOM snapshots

**Collection Pattern**:
```
Before instruction:
  - Attach event listeners (console, network)

During instruction:
  - Buffer events in memory

After instruction:
  - Capture screenshot and DOM
  - Remove listeners
  - Return collected telemetry
```

### 5. Type System (`src/types/`)

**Key Type Categories**:

**Contracts** (`contracts.ts`):
- Mirror Go automation/contracts types exactly
- `CompiledInstruction` - Input from Go
- `StepOutcome` - Output to Go
- `StepFailure` - Error taxonomy

**Session** (`session.ts`):
- `SessionState` - Internal session representation
- `SessionSpec` - Session creation parameters
- `ReuseMode` - Session reuse strategy

**Instruction** (`instruction.ts`):
- Zod schemas for all 28 instruction types
- Runtime parameter validation
- Type-safe parameter access

---

## Request Flow

### Session Start

```
1. POST /session/start
   ├─ Body parser extracts JSON
   ├─ Middleware validates request
   └─ Route handler:
       ├─ SessionManager.create()
       │  ├─ Launch browser (if needed)
       │  ├─ Create new context
       │  ├─ Create new page
       │  ├─ Configure HAR/video/tracing
       │  └─ Store SessionState
       ├─ Generate session ID
       └─ Return { session_id }
```

### Instruction Execution

```
2. POST /session/:id/run
   ├─ Body parser extracts instruction JSON
   ├─ Middleware validates request
   └─ Route handler:
       ├─ SessionManager.get(id)
       ├─ HandlerRegistry.getHandler(instruction.type)
       ├─ Start telemetry collection (console/network)
       ├─ Handler.execute(instruction, context)
       │  ├─ Validate params with Zod
       │  ├─ Execute Playwright actions
       │  └─ Return HandlerResult
       ├─ Stop telemetry collection
       ├─ Capture screenshot
       ├─ Capture DOM snapshot
       ├─ Build StepOutcome
       └─ Return StepOutcome JSON
```

### Session Close

```
3. POST /session/:id/close
   ├─ SessionManager.close(id)
   │  ├─ Stop tracing (if enabled)
   │  ├─ page.close()
   │  ├─ context.close()
   │  ├─ browser.close() (if last session)
   │  └─ Remove from session map
   └─ Return { status: "closed" }
```

---

## Error Handling

### Error Hierarchy

```
PlaywrightDriverError (base)
├─ SessionNotFoundError
├─ SessionLimitError
├─ InstructionError
│  ├─ SelectorNotFoundError
│  ├─ TimeoutError
│  ├─ NavigationError
│  └─ ValidationError
└─ ConfigurationError
```

### Error Mapping

Errors are normalized to Go `FailureKind`:
- **engine**: Playwright errors, selector not found, navigation failed
- **timeout**: Timeout errors
- **user**: Validation errors, misconfiguration
- **infra**: Session limit exceeded, browser launch failed

### Error Response Format

```json
{
  "success": false,
  "error": {
    "kind": "engine",
    "code": "SELECTOR_NOT_FOUND",
    "message": "Selector '#element' not found",
    "retryable": true,
    "occurred_at": "2025-01-26T12:34:56Z"
  }
}
```

---

## Observability

### Logging (Winston)

**Structured JSON logging**:
```typescript
logger.info('Instruction executed', {
  sessionId: 'sess_123',
  instructionType: 'click',
  duration: 250,
  success: true
});
```

**Log Levels**:
- **debug**: Detailed execution traces
- **info**: Normal operations
- **warn**: Non-fatal issues
- **error**: Failures

### Metrics (Prometheus)

**Available Metrics**:
- `playwright_driver_sessions_total` (gauge) - Active session count
- `playwright_driver_instruction_duration_ms` (histogram) - Instruction execution time
- `playwright_driver_instruction_errors_total` (counter) - Error count by type
- `playwright_driver_screenshot_size_bytes` (histogram) - Screenshot size distribution

**Metrics Endpoint**: `GET /metrics` (if metrics.enabled)

---

## Configuration

**Environment Variables**:
- `PLAYWRIGHT_DRIVER_PORT` - HTTP port (default: 39400)
- `PLAYWRIGHT_DRIVER_HOST` - Bind address (default: 127.0.0.1)
- `HEADLESS` - Run headless (default: true)
- `MAX_SESSIONS` - Max concurrent sessions (default: 10)
- `LOG_LEVEL` - Logging level (default: info)

See [CONTROL-SURFACE.md](../CONTROL-SURFACE.md) for full reference.

---

## Performance Characteristics

### Typical Performance

- **Session startup**: < 2s (browser launch + context creation)
- **Instruction overhead**: < 100ms (parsing + validation + telemetry)
- **Screenshot capture**: < 500ms (PNG, full page)
- **DOM snapshot**: < 50ms (HTML extraction)

### Resource Usage

- **Memory per session**: ~50-150 MB (depends on page complexity)
- **CPU**: Varies with instruction type (screenshots are CPU-intensive)
- **Disk**: Temporary screenshots/traces (cleaned up on close)

### Concurrency

- **Max concurrent sessions**: Configurable (default: 10)
- **Session pooling**: Reuse mode minimizes overhead
- **Idle cleanup**: Automatic cleanup after configurable timeout

---

## Security Considerations

### Input Validation

- All instruction params validated with Zod schemas
- JSON size limits (5 MB default)
- Timeout limits on all operations

### Network Isolation

- Binds to 127.0.0.1 by default (localhost only)
- No authentication (assumes trusted network)
- HTTPS not required (local communication)

### Resource Limits

- Max concurrent sessions
- Instruction timeouts
- Memory limits per session (future)

---

## Testing Strategy

### Unit Tests

- Handler logic (mocked Playwright)
- Session management
- Telemetry collection
- Error handling
- Configuration parsing

### Integration Tests

- Full request → response flows
- Session lifecycle
- Instruction execution
- Error scenarios

### Contract Tests

- StepOutcome format validation
- Schema version compatibility
- Failure taxonomy correctness

---

## Future Enhancements

### Planned Features

- **Session pooling**: Warm pool of browser contexts
- **Screenshot caching**: Deduplicate identical screenshots
- **OpenTelemetry**: Distributed tracing support
- **Health checks**: Deep health validation
- **Resource monitoring**: Per-session memory/CPU tracking

### Extensibility

- **Custom handlers**: Easy to add new instruction types
- **Plugin system**: Hook into handler execution
- **Custom telemetry**: Additional event collectors
- **Storage backends**: Pluggable artifact storage

---

## References

- [API Documentation](API.md)
- [Extending the Driver](EXTENDING.md)
- [Driver Assumptions](ASSUMPTIONS.md)
- [Deployment Notes](../../docs/playwright-driver-v2-implementation.md)
- [Go AutomationEngine Contract](../../docs/automation-engine-analysis.md)
