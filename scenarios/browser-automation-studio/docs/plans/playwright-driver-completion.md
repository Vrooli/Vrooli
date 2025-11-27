# Playwright Driver Completion Plan

## Executive Summary

Transform the current 503-line monolithic `server.js` into a professional, type-safe, modular TypeScript-based Playwright driver that:
- Implements all 28 documented node types
- Provides production-grade reliability and observability
- Maintains full compatibility with the Go `AutomationEngine` contract
- Enables future extensibility and maintenance

**Timeline**: 4 phases, ~3-4 weeks
**Priority**: High (frame-switch is marked as supported but missing!)

---

## Current State Analysis

### What Exists (13/28 node types)
✅ navigate, click, hover, type, scroll, wait, evaluate, extract, assert, uploadfile, download, screenshot, (plus control flow handled by executor)

### Critical Gap
❌ **frame-switch** - Marked as `SupportsIframes: true` in capabilities but NOT IMPLEMENTED
  - This is a contract violation that could cause silent failures

### Missing Features (14 node types)
❌ blur, focus, conditional, cookie-storage, drag-drop, gesture, keyboard, network-mock, rotate, select, shortcut, tab-switch, set-variable, use-variable

### Architectural Issues
- ❌ Single 503-line file (unmaintainable)
- ❌ No type safety (JavaScript)
- ❌ No structured logging
- ❌ Minimal error handling
- ❌ No metrics/observability
- ❌ No session timeout/cleanup
- ❌ No resource limits
- ❌ No testing infrastructure

---

## Target Architecture

### Directory Structure
```
playwright-driver/
├── src/
│   ├── server.ts                     # Entry point, HTTP server setup
│   ├── config.ts                     # Configuration management
│   │
│   ├── types/                        # Type definitions
│   │   ├── contracts.ts              # Go contract types (CompiledInstruction, StepOutcome)
│   │   ├── session.ts                # Session types
│   │   ├── instruction.ts            # Instruction parameter types
│   │   ├── telemetry.ts              # Console/network event types
│   │   └── index.ts                  # Type exports
│   │
│   ├── session/                      # Session lifecycle management
│   │   ├── manager.ts                # SessionManager class
│   │   ├── pool.ts                   # Session pooling/reuse strategies
│   │   ├── cleanup.ts                # Idle timeout and cleanup
│   │   ├── context-builder.ts        # BrowserContext configuration
│   │   └── index.ts
│   │
│   ├── handlers/                     # Instruction handlers (one per category)
│   │   ├── registry.ts               # Handler registration and dispatch
│   │   ├── base.ts                   # BaseHandler interface and helpers
│   │   ├── navigation.ts             # navigate
│   │   ├── interaction.ts            # click, hover, type, blur, focus
│   │   ├── wait.ts                   # wait
│   │   ├── assertion.ts              # assert
│   │   ├── extraction.ts             # extract, evaluate
│   │   ├── screenshot.ts             # screenshot
│   │   ├── upload.ts                 # uploadfile
│   │   ├── download.ts               # download
│   │   ├── scroll.ts                 # scroll
│   │   ├── cookie-storage.ts         # cookie/localStorage/sessionStorage ops
│   │   ├── frame.ts                  # frame-switch (CRITICAL - missing!)
│   │   ├── tab.ts                    # tab-switch (multi-tab support)
│   │   ├── keyboard.ts               # keyboard, shortcut
│   │   ├── gesture.ts                # drag-drop, gesture
│   │   ├── network.ts                # network-mock
│   │   ├── select.ts                 # select (dropdown)
│   │   ├── device.ts                 # rotate (device orientation)
│   │   └── index.ts
│   │
│   ├── telemetry/                    # Event collection and formatting
│   │   ├── collector.ts              # ConsoleLogCollector, NetworkCollector
│   │   ├── screenshot-capture.ts     # Screenshot logic + optimization
│   │   ├── dom-capture.ts            # DOM snapshot logic
│   │   └── index.ts
│   │
│   ├── routes/                       # HTTP route handlers
│   │   ├── health.ts                 # GET /health
│   │   ├── session-start.ts          # POST /session/start
│   │   ├── session-run.ts            # POST /session/:id/run
│   │   ├── session-reset.ts          # POST /session/:id/reset
│   │   ├── session-close.ts          # POST /session/:id/close
│   │   ├── capabilities.ts           # GET /capabilities (NEW - for introspection)
│   │   └── index.ts
│   │
│   ├── middleware/                   # HTTP middleware
│   │   ├── body-parser.ts            # JSON parsing with size limits
│   │   ├── error-handler.ts          # Centralized error handling
│   │   ├── request-logger.ts         # Request/response logging
│   │   └── index.ts
│   │
│   ├── utils/                        # Utilities
│   │   ├── logger.ts                 # Structured logging (winston/pino)
│   │   ├── metrics.ts                # Metrics collection (Prometheus-compatible)
│   │   ├── timeout.ts                # Timeout helpers
│   │   ├── validation.ts             # Input validation
│   │   └── index.ts
│   │
│   └── constants/                    # Constants
│       ├── defaults.ts               # Default timeouts, limits
│       ├── errors.ts                 # Error codes and messages
│       └── index.ts
│
├── tests/
│   ├── unit/
│   │   ├── handlers/                 # Handler unit tests
│   │   │   ├── navigation.test.ts
│   │   │   ├── interaction.test.ts
│   │   │   └── ... (one per handler)
│   │   ├── session/
│   │   │   ├── manager.test.ts
│   │   │   └── pool.test.ts
│   │   └── telemetry/
│   │       ├── collector.test.ts
│   │       └── screenshot.test.ts
│   ├── integration/
│   │   ├── smoke.test.ts             # Basic smoke tests
│   │   ├── workflows/                # End-to-end workflow tests
│   │   │   ├── basic-navigation.test.ts
│   │   │   ├── form-interaction.test.ts
│   │   │   ├── iframe-navigation.test.ts
│   │   │   └── multi-tab.test.ts
│   │   └── contract/                 # Contract compliance tests
│   │       └── step-outcome.test.ts
│   ├── fixtures/
│   │   ├── instructions.ts           # Sample instructions
│   │   ├── pages/                    # Test HTML pages
│   │   │   ├── basic.html
│   │   │   ├── iframe.html
│   │   │   └── multi-tab.html
│   │   └── expected-outcomes.ts
│   └── helpers/
│       ├── test-server.ts            # Test HTTP server
│       └── session-factory.ts
│
├── docs/
│   ├── architecture.md               # Architecture overview
│   ├── api.md                        # API specification
│   ├── handlers.md                   # Handler implementation guide
│   ├── contracts.md                  # Go contract compatibility
│   ├── configuration.md              # Configuration reference
│   ├── development.md                # Development guide
│   ├── testing.md                    # Testing guide
│   └── deployment.md                 # Production deployment guide
│
├── scripts/
│   ├── build.sh                      # Build script
│   ├── dev.sh                        # Development server
│   └── test.sh                       # Test runner
│
├── package.json
├── tsconfig.json
├── tsconfig.build.json
├── .eslintrc.js
├── .prettierrc
├── jest.config.js
└── README.md
```

---

## Phase 1: Foundation & Migration (Week 1)

### Goals
- Set up TypeScript infrastructure
- Migrate existing functionality without regression
- Establish testing foundation
- Implement session management improvements

### Tasks

#### 1.1 Project Setup
```bash
# Initialize TypeScript project
- [ ] Create package.json with dependencies
- [ ] Configure TypeScript (tsconfig.json)
- [ ] Set up ESLint + Prettier
- [ ] Configure Jest for testing
- [ ] Set up build pipeline
```

**Dependencies**:
```json
{
  "dependencies": {
    "playwright": "^1.40.0",
    "winston": "^3.11.0",        // Structured logging
    "prom-client": "^15.1.0",    // Metrics
    "zod": "^3.22.4"             // Runtime validation
  },
  "devDependencies": {
    "@types/node": "^20.10.0",
    "typescript": "^5.3.0",
    "eslint": "^8.55.0",
    "@typescript-eslint/eslint-plugin": "^6.15.0",
    "@typescript-eslint/parser": "^6.15.0",
    "prettier": "^3.1.1",
    "jest": "^29.7.0",
    "@types/jest": "^29.5.11",
    "ts-jest": "^29.1.1"
  }
}
```

#### 1.2 Type System
```typescript
// src/types/contracts.ts - Mirror Go contracts
- [ ] Define CompiledInstruction type
- [ ] Define StepOutcome type
- [ ] Define StepFailure type
- [ ] Define Screenshot type
- [ ] Define DOMSnapshot type
- [ ] Define ConsoleLogEntry type
- [ ] Define NetworkEvent type
- [ ] Define AssertionOutcome type
- [ ] Define BoundingBox, Point, etc.
- [ ] Add Zod schemas for runtime validation
```

```typescript
// src/types/session.ts
- [ ] Define SessionConfig type
- [ ] Define SessionState type
- [ ] Define SessionMetrics type
- [ ] Define ReuseMode enum
```

```typescript
// src/types/instruction.ts
- [ ] Define per-instruction parameter types
- [ ] Create discriminated union for all instruction types
- [ ] Add validation schemas
```

#### 1.3 Core Infrastructure
```typescript
// src/config.ts
- [ ] Environment variable parsing
- [ ] Configuration validation
- [ ] Default value management
- [ ] Type-safe config object
```

```typescript
// src/utils/logger.ts
- [ ] Set up Winston logger
- [ ] Define log levels (debug, info, warn, error)
- [ ] Add structured logging helpers
- [ ] Add request correlation IDs
```

```typescript
// src/utils/metrics.ts
- [ ] Set up Prometheus metrics
- [ ] Define core metrics:
  - [ ] session_count (gauge)
  - [ ] instruction_duration_ms (histogram)
  - [ ] instruction_errors_total (counter)
  - [ ] screenshot_size_bytes (histogram)
```

#### 1.4 Session Management
```typescript
// src/session/manager.ts
- [ ] SessionManager class
- [ ] Session creation with timeout
- [ ] Session retrieval
- [ ] Session deletion
- [ ] Idle session cleanup
- [ ] Resource limits (max sessions)
```

```typescript
// src/session/context-builder.ts
- [ ] Build BrowserContext from SessionSpec
- [ ] Configure viewport
- [ ] Configure HAR recording
- [ ] Configure video recording
- [ ] Configure tracing
- [ ] Handle baseURL
```

#### 1.5 HTTP Layer
```typescript
// src/middleware/body-parser.ts
- [ ] JSON parsing with size limits (5MB)
- [ ] Error handling for malformed JSON
```

```typescript
// src/middleware/error-handler.ts
- [ ] Centralized error handling
- [ ] Map errors to HTTP status codes
- [ ] Structured error responses
- [ ] Error logging
```

```typescript
// src/server.ts
- [ ] HTTP server setup
- [ ] Route registration
- [ ] Graceful shutdown
- [ ] Signal handling (SIGTERM, SIGINT)
```

#### 1.6 Migrate Existing Handlers
```typescript
// src/handlers/base.ts
- [ ] Define HandlerContext interface
- [ ] Define HandlerResult interface
- [ ] Define BaseHandler abstract class
- [ ] Add telemetry hooks (beforeExecute, afterExecute)
```

```typescript
// Migrate existing functionality:
- [ ] src/handlers/navigation.ts (navigate)
- [ ] src/handlers/interaction.ts (click, hover, type)
- [ ] src/handlers/wait.ts (wait)
- [ ] src/handlers/assertion.ts (assert)
- [ ] src/handlers/extraction.ts (extract, evaluate)
- [ ] src/handlers/screenshot.ts (screenshot)
- [ ] src/handlers/upload.ts (uploadfile)
- [ ] src/handlers/download.ts (download)
- [ ] src/handlers/scroll.ts (scroll)
```

```typescript
// src/handlers/registry.ts
- [ ] Handler registration
- [ ] Handler lookup by instruction type
- [ ] Handler validation
```

#### 1.7 Telemetry Collection
```typescript
// src/telemetry/collector.ts
- [ ] ConsoleLogCollector class
- [ ] NetworkCollector class
- [ ] Event buffering
- [ ] Timestamp normalization
```

```typescript
// src/telemetry/screenshot-capture.ts
- [ ] Screenshot capture with error handling
- [ ] Base64 encoding
- [ ] Size optimization
- [ ] Viewport size extraction
```

```typescript
// src/telemetry/dom-capture.ts
- [ ] DOM HTML extraction
- [ ] Preview generation (first 512 chars)
- [ ] Error handling
```

#### 1.8 Routes
```typescript
// Implement all routes:
- [ ] src/routes/health.ts
- [ ] src/routes/session-start.ts
- [ ] src/routes/session-run.ts
- [ ] src/routes/session-reset.ts
- [ ] src/routes/session-close.ts
- [ ] src/routes/capabilities.ts (NEW)
```

#### 1.9 Testing
```typescript
// Set up test infrastructure:
- [ ] tests/helpers/test-server.ts
- [ ] tests/helpers/session-factory.ts
- [ ] tests/fixtures/instructions.ts
- [ ] tests/fixtures/pages/*.html
```

```typescript
// Unit tests for migrated code:
- [ ] tests/unit/session/manager.test.ts
- [ ] tests/unit/telemetry/collector.test.ts
- [ ] tests/unit/handlers/navigation.test.ts
- [ ] tests/unit/handlers/interaction.test.ts
```

```typescript
// Integration tests:
- [ ] tests/integration/smoke.test.ts
- [ ] tests/integration/contract/step-outcome.test.ts
```

#### 1.10 Documentation
```markdown
- [ ] docs/architecture.md
- [ ] docs/development.md
- [ ] Update README.md
```

### Deliverables
- ✅ Full TypeScript project with type safety
- ✅ All existing functionality migrated (no regressions)
- ✅ Test coverage >80% for core modules
- ✅ Structured logging and metrics
- ✅ Improved session management

---

## Phase 2: Critical Missing Features (Week 2)

### Goals
- Fix the contract violation (iframe support)
- Implement high-value missing features
- Achieve feature parity with documented node types

### Priority 1: Frame Operations (CRITICAL)

#### 2.1 Frame-Switch Handler
```typescript
// src/handlers/frame.ts
/**
 * CRITICAL: This is marked as supported (SupportsIframes: true) but not implemented!
 *
 * Frame-switch allows navigation into iframes and back to main frame.
 *
 * Instruction params:
 * - selector?: string       // CSS selector for iframe element
 * - frameId?: string        // Frame ID (if known)
 * - frameUrl?: string       // Frame URL (for matching)
 * - action: 'enter' | 'exit' | 'parent'
 *
 * State management:
 * - Track frame stack per session
 * - Maintain current frame context
 * - Support nested iframes
 */

interface FrameSwitchParams {
  selector?: string;
  frameId?: string;
  frameUrl?: string;
  action: 'enter' | 'exit' | 'parent';
  timeoutMs?: number;
}

class FrameSwitchHandler extends BaseHandler {
  async execute(params: FrameSwitchParams, context: HandlerContext): Promise<HandlerResult> {
    // Implementation
  }
}
```

**Tasks**:
- [ ] Implement frame entry (by selector, frameId, URL)
- [ ] Implement frame exit (to main frame)
- [ ] Implement parent frame navigation
- [ ] Track frame stack in session state
- [ ] Handle frame not found errors
- [ ] Add timeout handling
- [ ] Write comprehensive tests
- [ ] Document frame stack semantics

#### 2.2 Focus Management
```typescript
// src/handlers/interaction.ts - extend existing
/**
 * Focus and blur operations
 *
 * focus params:
 * - selector: string        // Element to focus
 * - scroll?: boolean        // Scroll into view first
 *
 * blur params:
 * - selector?: string       // Specific element to blur (optional)
 *                           // If omitted, blur currently focused element
 */
```

**Tasks**:
- [ ] Add focus() method to interaction handler
- [ ] Add blur() method to interaction handler
- [ ] Support scrollIntoView option
- [ ] Track focused element in telemetry
- [ ] Tests for focus/blur
- [ ] Document focus management

#### 2.3 Cookie & Storage Operations
```typescript
// src/handlers/cookie-storage.ts
/**
 * Cookie and Web Storage manipulation
 *
 * Supported operations:
 * - getCookie(name)
 * - setCookie(name, value, options)
 * - deleteCookie(name)
 * - clearCookies()
 * - getLocalStorage(key)
 * - setLocalStorage(key, value)
 * - removeLocalStorage(key)
 * - clearLocalStorage()
 * - getSessionStorage(key)
 * - setSessionStorage(key, value)
 * - removeSessionStorage(key)
 * - clearSessionStorage()
 */

interface CookieStorageParams {
  operation: 'get' | 'set' | 'delete' | 'clear';
  storageType: 'cookie' | 'localStorage' | 'sessionStorage';
  key?: string;
  value?: string;
  cookieOptions?: {
    domain?: string;
    path?: string;
    expires?: number;
    httpOnly?: boolean;
    secure?: boolean;
    sameSite?: 'Strict' | 'Lax' | 'None';
  };
}
```

**Tasks**:
- [ ] Implement cookie operations (get/set/delete/clear)
- [ ] Implement localStorage operations
- [ ] Implement sessionStorage operations
- [ ] Handle cookie options (domain, path, etc.)
- [ ] Return extracted data for get operations
- [ ] Tests for all storage types
- [ ] Document storage semantics

#### 2.4 Select (Dropdown) Handler
```typescript
// src/handlers/select.ts
/**
 * Dropdown selection
 *
 * Supports:
 * - Select by value
 * - Select by label
 * - Select by index
 * - Multi-select
 */

interface SelectParams {
  selector: string;
  value?: string | string[];
  label?: string | string[];
  index?: number | number[];
  timeoutMs?: number;
}
```

**Tasks**:
- [ ] Implement select by value
- [ ] Implement select by label
- [ ] Implement select by index
- [ ] Support multi-select
- [ ] Validate option exists
- [ ] Return selected values in extracted_data
- [ ] Tests for all selection modes
- [ ] Document select semantics

#### 2.5 Keyboard & Shortcuts
```typescript
// src/handlers/keyboard.ts
/**
 * Raw keyboard events and shortcuts
 *
 * keyboard:
 * - Press individual keys
 * - Hold modifiers
 * - Release keys
 *
 * shortcut:
 * - Execute common shortcuts (Ctrl+C, etc.)
 * - Cross-platform handling (Cmd on Mac, Ctrl on Win/Linux)
 */

interface KeyboardParams {
  key?: string;              // Single key
  keys?: string[];           // Key sequence
  modifiers?: string[];      // 'Control', 'Shift', 'Alt', 'Meta'
  action?: 'press' | 'down' | 'up';
}

interface ShortcutParams {
  shortcut: string;          // e.g., 'Ctrl+C', 'Cmd+V'
  selector?: string;         // Optional: focus element first
}
```

**Tasks**:
- [ ] Implement keyboard key press
- [ ] Implement key down/up
- [ ] Handle modifier keys
- [ ] Implement shortcut executor
- [ ] Cross-platform modifier mapping
- [ ] Tests for keyboard operations
- [ ] Document key names and shortcuts

### Deliverables
- ✅ Frame-switch implemented (fixes contract violation!)
- ✅ Focus/blur implemented
- ✅ Cookie/storage operations implemented
- ✅ Select dropdown implemented
- ✅ Keyboard/shortcuts implemented
- ✅ Test coverage >85%
- ✅ Updated handler documentation

---

## Phase 3: Advanced Features (Week 3)

### Goals
- Implement complex interaction features
- Enable multi-tab support
- Add network mocking

### 3.1 Drag & Drop / Gestures
```typescript
// src/handlers/gesture.ts
/**
 * Complex mouse gestures
 *
 * drag-drop:
 * - Drag from source to target
 * - Drag by offset
 * - Custom drag path
 *
 * gesture:
 * - Swipe
 * - Pinch/zoom
 * - Multi-touch (if supported)
 */

interface DragDropParams {
  sourceSelector: string;
  targetSelector?: string;
  offsetX?: number;
  offsetY?: number;
  steps?: number;           // Animation steps
  delayMs?: number;         // Delay between steps
}

interface GestureParams {
  type: 'swipe' | 'pinch' | 'zoom';
  selector?: string;
  direction?: 'up' | 'down' | 'left' | 'right';
  distance?: number;
  scale?: number;           // For pinch/zoom
}
```

**Tasks**:
- [ ] Implement drag and drop (selector to selector)
- [ ] Implement drag by offset
- [ ] Implement swipe gestures
- [ ] Support animation steps
- [ ] Capture bounding boxes in telemetry
- [ ] Tests for drag/drop scenarios
- [ ] Document gesture semantics

### 3.2 Multi-Tab Support
```typescript
// src/handlers/tab.ts
/**
 * Multi-tab management
 *
 * Operations:
 * - Open new tab
 * - Switch to tab (by index, URL, title)
 * - Close tab
 * - List tabs
 *
 * State:
 * - Track tab stack per session
 * - Maintain current tab context
 */

interface TabSwitchParams {
  action: 'open' | 'switch' | 'close' | 'list';
  url?: string;              // For open
  index?: number;            // For switch/close
  title?: string;            // For switch (match by title)
  urlPattern?: string;       // For switch (match by URL regex)
}
```

**Tasks**:
- [ ] Extend session state to track multiple pages
- [ ] Implement tab open
- [ ] Implement tab switch (by index, URL, title)
- [ ] Implement tab close
- [ ] Implement tab list
- [ ] Update capabilities: `AllowsParallelTabs: true`
- [ ] Handle tab not found errors
- [ ] Tests for multi-tab workflows
- [ ] Document tab management

### 3.3 Network Mocking
```typescript
// src/handlers/network.ts
/**
 * Request interception and mocking
 *
 * Operations:
 * - Mock response for URL pattern
 * - Block requests
 * - Modify request headers
 * - Modify response
 * - Clear mocks
 */

interface NetworkMockParams {
  operation: 'mock' | 'block' | 'modifyRequest' | 'modifyResponse' | 'clear';
  urlPattern: string | RegExp;
  method?: string;
  statusCode?: number;
  headers?: Record<string, string>;
  body?: string | object;
  delayMs?: number;         // Simulate latency
}
```

**Tasks**:
- [ ] Implement route interception
- [ ] Implement response mocking
- [ ] Implement request blocking
- [ ] Implement header modification
- [ ] Support URL patterns and regex
- [ ] Track active mocks per session
- [ ] Tests for network mocking
- [ ] Document network mock semantics

### 3.4 Device Rotation
```typescript
// src/handlers/device.ts
/**
 * Device orientation and emulation
 *
 * Operations:
 * - Rotate device (portrait/landscape)
 * - Set device emulation
 * - Set geolocation
 * - Set timezone
 */

interface RotateParams {
  orientation: 'portrait' | 'landscape';
  angle?: 0 | 90 | 180 | 270;
}

interface DeviceParams {
  operation: 'rotate' | 'emulate' | 'geolocation' | 'timezone';
  orientation?: 'portrait' | 'landscape';
  device?: string;          // Device name (e.g., 'iPhone 12')
  latitude?: number;
  longitude?: number;
  accuracy?: number;
  timezone?: string;        // e.g., 'America/New_York'
}
```

**Tasks**:
- [ ] Implement device rotation
- [ ] Implement device emulation (predefined devices)
- [ ] Implement geolocation setting
- [ ] Implement timezone setting
- [ ] Update viewport on rotation
- [ ] Tests for device operations
- [ ] Document device capabilities

### Deliverables
- ✅ Drag/drop and gestures implemented
- ✅ Multi-tab support enabled
- ✅ Network mocking implemented
- ✅ Device rotation implemented
- ✅ All 28 node types implemented
- ✅ Test coverage >90%

---

## Phase 4: Production Readiness (Week 4)

### Goals
- Performance optimization
- Production-grade reliability
- Comprehensive monitoring
- Deployment preparation

### 4.1 Performance Optimization
```typescript
// Screenshot optimization
- [ ] Implement screenshot caching/deduplication
- [ ] Add quality/compression options
- [ ] Lazy screenshot capture (only when needed)
- [ ] Configurable screenshot strategy
```

```typescript
// Session pooling
- [ ] Implement session reuse strategies
- [ ] Browser process pooling
- [ ] Context reuse for 'clean' mode
- [ ] Warm pool maintenance
```

```typescript
// Resource management
- [ ] Memory monitoring per session
- [ ] Browser process cleanup
- [ ] Temp file cleanup (videos, traces)
- [ ] Resource limits enforcement
```

**Tasks**:
- [ ] Profile instruction execution times
- [ ] Optimize screenshot capture
- [ ] Implement efficient session pooling
- [ ] Add resource cleanup
- [ ] Load testing
- [ ] Performance benchmarks

### 4.2 Enhanced Error Handling
```typescript
// src/utils/errors.ts
/**
 * Structured error hierarchy
 *
 * - PlaywrightDriverError (base)
 *   - SessionNotFoundError
 *   - InstructionError
 *     - SelectorNotFoundError
 *     - TimeoutError
 *     - NavigationError
 *   - ResourceError
 *     - MemoryLimitError
 *     - SessionLimitError
 *   - ConfigurationError
 */
```

**Tasks**:
- [ ] Define error hierarchy
- [ ] Map errors to Go FailureKind (engine, infra, timeout, etc.)
- [ ] Add error codes
- [ ] Include actionable error messages
- [ ] Error recovery strategies
- [ ] Tests for error handling

### 4.3 Observability
```typescript
// Enhanced metrics
- [ ] Add percentile histograms (p50, p95, p99)
- [ ] Per-handler metrics
- [ ] Session lifecycle metrics
- [ ] Error rate by type
- [ ] Resource usage (memory, CPU)
```

```typescript
// Distributed tracing
- [ ] Add OpenTelemetry support
- [ ] Trace instruction execution
- [ ] Correlation IDs across requests
- [ ] Span context propagation
```

```typescript
// Health checks
- [ ] Deep health check (browser connectivity)
- [ ] Dependency checks
- [ ] Resource availability
- [ ] Degraded state handling
```

**Tasks**:
- [ ] Implement comprehensive metrics
- [ ] Add tracing support (optional)
- [ ] Enhanced health endpoint
- [ ] Monitoring dashboard (Grafana example)

### 4.4 Configuration & Deployment
```typescript
// src/config.ts - comprehensive configuration
- [ ] Environment-based config
- [ ] Config validation
- [ ] Secure defaults
- [ ] Override mechanisms
```

```yaml
# Example config schema
server:
  port: 39400
  host: '127.0.0.1'
  requestTimeout: 120000
  maxRequestSize: 5242880  # 5MB

browser:
  headless: true
  executablePath: ''        # Auto-detect
  args: []

session:
  maxConcurrent: 10
  idleTimeoutMs: 300000     # 5 minutes
  poolSize: 5
  cleanupIntervalMs: 60000

telemetry:
  screenshot:
    enabled: true
    fullPage: true
    quality: 80
    maxSizeBytes: 512000
  dom:
    enabled: true
    maxSizeBytes: 524288
  console:
    enabled: true
    maxEntries: 100
  network:
    enabled: true
    maxEvents: 200

limits:
  maxMemoryMB: 2048
  maxDiskMB: 5120
  maxExecutionTimeMs: 600000

logging:
  level: 'info'
  format: 'json'

metrics:
  enabled: true
  port: 9090
```

**Tasks**:
- [ ] Comprehensive config schema
- [ ] Environment variable mapping
- [ ] Config file support (optional)
- [ ] Validation on startup
- [ ] Config documentation

### 4.5 Testing & Quality
```typescript
// Contract compliance tests
- [ ] Validate all StepOutcome fields
- [ ] Schema version compatibility
- [ ] Failure taxonomy correctness
- [ ] Screenshot format validation
```

```typescript
// Stress testing
- [ ] Concurrent session handling
- [ ] Memory leak detection
- [ ] Long-running session stability
- [ ] Resource exhaustion scenarios
```

```typescript
// Integration with Go API
- [ ] End-to-end workflow tests
- [ ] Contract compatibility tests
- [ ] Error propagation tests
```

**Tasks**:
- [ ] Contract compliance test suite
- [ ] Stress tests
- [ ] Integration tests with Go API
- [ ] Performance regression tests
- [ ] Test coverage report >95%

### 4.6 Documentation
```markdown
# Complete documentation suite
- [ ] docs/architecture.md - Architecture overview with diagrams
- [ ] docs/api.md - Complete API specification
- [ ] docs/handlers.md - Handler implementation guide
- [ ] docs/contracts.md - Go contract compatibility
- [ ] docs/configuration.md - Configuration reference
- [ ] docs/development.md - Development setup and workflow
- [ ] docs/testing.md - Testing guide
- [ ] docs/deployment.md - Production deployment guide
- [ ] docs/monitoring.md - Monitoring and observability
- [ ] docs/troubleshooting.md - Common issues and solutions
```

**Tasks**:
- [ ] Write comprehensive documentation
- [ ] Add architecture diagrams
- [ ] Document all handlers with examples
- [ ] Deployment playbook
- [ ] Monitoring runbook
- [ ] Update main README

### 4.7 Production Deployment
```dockerfile
# Dockerfile for production
- [ ] Multi-stage build
- [ ] Minimal base image
- [ ] Non-root user
- [ ] Health checks
- [ ] Security scanning
```

```yaml
# Docker Compose example
- [ ] Service definition
- [ ] Resource limits
- [ ] Health checks
- [ ] Volume mounts
- [ ] Network configuration
```

```bash
# Deployment scripts
- [ ] Build script
- [ ] Test script
- [ ] Deploy script
- [ ] Rollback script
```

**Tasks**:
- [ ] Production Dockerfile
- [ ] Docker Compose example
- [ ] Kubernetes manifests (optional)
- [ ] Deployment automation
- [ ] Production checklist

### Deliverables
- ✅ Production-grade performance
- ✅ Comprehensive error handling
- ✅ Full observability (logs, metrics, traces)
- ✅ Complete documentation
- ✅ Deployment ready
- ✅ Test coverage >95%

---

## Implementation Checklist

### Phase 1: Foundation ✅
- [ ] TypeScript project setup
- [ ] Type definitions
- [ ] Core infrastructure (logging, metrics)
- [ ] Session management
- [ ] HTTP layer
- [ ] Migrate existing handlers (13 types)
- [ ] Testing infrastructure
- [ ] Basic documentation

### Phase 2: Critical Features ✅
- [ ] Frame-switch (CRITICAL!)
- [ ] Focus/blur
- [ ] Cookie/storage operations
- [ ] Select dropdown
- [ ] Keyboard/shortcuts

### Phase 3: Advanced Features ✅
- [ ] Drag/drop and gestures
- [ ] Multi-tab support
- [ ] Network mocking
- [ ] Device rotation

### Phase 4: Production ✅
- [ ] Performance optimization
- [ ] Enhanced error handling
- [ ] Full observability
- [ ] Configuration management
- [ ] Comprehensive testing
- [ ] Complete documentation
- [ ] Deployment preparation

---

## Success Metrics

### Code Quality
- ✅ 100% TypeScript (no `any` types except where necessary)
- ✅ Test coverage >95%
- ✅ Zero ESLint errors
- ✅ All handlers documented

### Functionality
- ✅ All 28 node types implemented
- ✅ Contract compliance: 100%
- ✅ Frame-switch working (fixes contract violation)
- ✅ Multi-tab support enabled

### Performance
- ✅ Instruction execution <100ms overhead (vs raw Playwright)
- ✅ Screenshot capture <500ms
- ✅ Session startup <2s
- ✅ Support 10+ concurrent sessions

### Reliability
- ✅ Zero memory leaks
- ✅ Graceful degradation under load
- ✅ Proper error propagation
- ✅ Resource cleanup on all paths

---

## Risk Mitigation

### Risk: Breaking existing functionality during migration
**Mitigation**:
- Comprehensive test suite before migration
- Side-by-side comparison tests
- Gradual rollout with feature flag

### Risk: Performance regression
**Mitigation**:
- Performance benchmarks before/after
- Load testing
- Profiling and optimization
- Session pooling

### Risk: Contract incompatibility
**Mitigation**:
- Strict type checking against Go contracts
- Contract compliance tests
- Integration tests with Go API
- Schema validation

### Risk: Incomplete testing
**Mitigation**:
- Test-first development
- Coverage requirements (>95%)
- Integration tests
- Contract tests

---

## Next Steps

1. **Review and approve this plan**
2. **Set up Phase 1 project structure**
3. **Begin TypeScript migration**
4. **Implement frame-switch handler (highest priority)**
5. **Iterate through phases**

---

## Timeline Summary

| Phase | Duration | Key Deliverables |
|-------|----------|------------------|
| Phase 1: Foundation | Week 1 | TypeScript migration, testing, session mgmt |
| Phase 2: Critical Features | Week 2 | Frame-switch, focus, cookies, select, keyboard |
| Phase 3: Advanced Features | Week 3 | Drag/drop, multi-tab, network mock, rotation |
| Phase 4: Production | Week 4 | Optimization, docs, deployment |

**Total: 4 weeks to completion**

---

## References

- Go Contracts: `api/automation/contracts/`
- Current Driver: `playwright-driver/src/server.ts` (TypeScript v2.0)
- Handler Docs: `docs/nodes/*.md`
- Automation README: `api/automation/README.md`
- Engine README: `api/automation/engine/README.md`
