# UI Smoke Test Go Implementation Plan

**Status**: Completed
**Created**: 2025-12-03
**Completed**: 2025-12-03
**Author**: Claude

---

## Executive Summary

Reimplement the UI smoke test functionality in Go within test-genie, replacing the current bash implementation (`scripts/scenarios/testing/shell/ui-smoke.sh`). This enables:
- Full test coverage of the smoke test logic itself
- Consistent error handling and reporting
- Better integration with test-genie's phase system
- Elimination of shell-out dependencies during test execution

---

## Current State Analysis

### Bash Implementation Overview
The current implementation (~1000 lines in `scripts/scenarios/testing/shell/ui-smoke.sh`) performs:

1. **Preflight Checks**
   - Browserless health/availability
   - Chrome process leak detection
   - Bundle freshness validation
   - iframe-bridge dependency check

2. **Browser Session**
   - Sends JavaScript payload to Browserless `/function` endpoint
   - Loads UI in an iframe with host page wrapper
   - Waits for handshake signals from `@vrooli/iframe-bridge`

3. **Handshake Detection**
   - Checks multiple window properties:
     - `window.__vrooliBridgeChildInstalled`
     - `window.IFRAME_BRIDGE_READY`
     - `window.IframeBridge.ready`
     - `window.iframeBridge.ready`
     - `window.IframeBridge.getState().ready`

4. **Artifact Collection**
   - Screenshots (PNG, base64 decoded)
   - Console logs (JSON)
   - Network failures (JSON)
   - Page errors (JSON)
   - DOM snapshot (HTML)

5. **Result Persistence**
   - Writes `coverage/<scenario>/ui-smoke/latest.json`
   - Structured summary with status, timing, artifacts

### How test-genie Currently Consumes Results
```
phase_structure.go:enforceUISmokeTelemetry()
  → fetchScenarioStatus()
    → vrooli scenario status --json
      → reads coverage/<scenario>/ui-smoke/latest.json
```

test-genie validates cached results but never executes the smoke test.

---

## Proposed Architecture

### Design Principles

1. **Screaming Architecture**: Package names communicate intent
2. **Dependency Inversion**: Core logic depends on interfaces, not implementations
3. **Testing Seams**: Every boundary is mockable
4. **Single Responsibility**: Each component has one reason to change
5. **Fail Fast**: Validate preconditions before expensive operations

### Package Structure

```
api/internal/
├── uismoke/                      # Domain: UI smoke testing
│   ├── doc.go                    # Package documentation
│   ├── config.go                 # Configuration types
│   ├── config_test.go
│   ├── result.go                 # Result/observation types
│   ├── result_test.go
│   │
│   ├── orchestrator/             # Application: coordinates smoke test flow
│   │   ├── doc.go
│   │   ├── orchestrator.go       # Main entry point
│   │   ├── orchestrator_test.go
│   │   ├── options.go            # Functional options pattern
│   │   └── interfaces.go         # Dependencies as interfaces
│   │
│   ├── preflight/                # Domain: precondition validation
│   │   ├── doc.go
│   │   ├── checks.go             # Individual check implementations
│   │   ├── checks_test.go
│   │   ├── bundle.go             # Bundle freshness check
│   │   ├── bundle_test.go
│   │   ├── bridge.go             # iframe-bridge dependency check
│   │   └── bridge_test.go
│   │
│   ├── browser/                  # Infrastructure: Browserless client
│   │   ├── doc.go
│   │   ├── client.go             # HTTP client for Browserless API
│   │   ├── client_test.go
│   │   ├── payload.go            # JavaScript payload generation
│   │   ├── payload_test.go
│   │   ├── response.go           # Response parsing
│   │   └── response_test.go
│   │
│   ├── handshake/                # Domain: bridge handshake detection
│   │   ├── doc.go
│   │   ├── detector.go           # Handshake result interpretation
│   │   ├── detector_test.go
│   │   ├── signals.go            # Known signal definitions
│   │   └── signals_test.go
│   │
│   └── artifacts/                # Infrastructure: artifact persistence
│       ├── doc.go
│       ├── writer.go             # Writes artifacts to filesystem
│       ├── writer_test.go
│       ├── types.go              # Artifact type definitions
│       └── paths.go              # Path resolution logic
```

### Component Responsibilities

#### 1. `uismoke` (Root Package)
**Purpose**: Shared types and configuration
**Depends on**: Nothing (leaf package)

```go
// config.go
type Config struct {
    ScenarioName      string
    ScenarioDir       string
    BrowserlessURL    string
    Timeout           time.Duration
    HandshakeTimeout  time.Duration
    Viewport          Viewport
}

type Viewport struct {
    Width  int
    Height int
}

// result.go
type Result struct {
    Status      Status
    Message     string
    DurationMs  int64
    Handshake   HandshakeResult
    Artifacts   ArtifactPaths
    Bundle      BundleStatus
    Raw         json.RawMessage
}

type Status string
const (
    StatusPassed  Status = "passed"
    StatusFailed  Status = "failed"
    StatusSkipped Status = "skipped"
    StatusBlocked Status = "blocked"
)
```

#### 2. `orchestrator`
**Purpose**: Coordinate the smoke test workflow
**Depends on**: Interfaces only (injected implementations)

```go
// interfaces.go
type PreflightChecker interface {
    CheckBrowserless(ctx context.Context) error
    CheckBundleFreshness(ctx context.Context, scenarioDir string) (*BundleStatus, error)
    CheckIframeBridge(ctx context.Context, scenarioDir string) error
}

type BrowserClient interface {
    ExecuteFunction(ctx context.Context, payload string) (*browser.Response, error)
    Health(ctx context.Context) error
}

type ArtifactWriter interface {
    WriteAll(ctx context.Context, artifacts *RawArtifacts) (*ArtifactPaths, error)
}

type HandshakeDetector interface {
    Evaluate(raw *browser.HandshakeRaw) HandshakeResult
}

// orchestrator.go
type Orchestrator struct {
    config     uismoke.Config
    preflight  PreflightChecker
    browser    BrowserClient
    artifacts  ArtifactWriter
    handshake  HandshakeDetector
    logger     io.Writer
}

func (o *Orchestrator) Run(ctx context.Context) (*uismoke.Result, error) {
    // 1. Preflight checks
    // 2. Execute browser session
    // 3. Evaluate handshake
    // 4. Write artifacts
    // 5. Build result
}
```

#### 3. `preflight`
**Purpose**: Validate preconditions before expensive browser operations
**Depends on**: HTTP client (for browserless), filesystem (for bundle/bridge checks)

```go
// checks.go
type Checker struct {
    browserlessURL string
    httpClient     *http.Client
}

func (c *Checker) CheckBrowserless(ctx context.Context) error
func (c *Checker) CheckBundleFreshness(ctx context.Context, scenarioDir string) (*BundleStatus, error)
func (c *Checker) CheckIframeBridge(ctx context.Context, scenarioDir string) error
```

**Testing seam**: Accept `http.Client` and filesystem interface for mocking.

#### 4. `browser`
**Purpose**: Browserless API communication
**Depends on**: HTTP client only

```go
// client.go
type Client struct {
    baseURL    string
    httpClient *http.Client
    timeout    time.Duration
}

func (c *Client) ExecuteFunction(ctx context.Context, payload string) (*Response, error)
func (c *Client) Health(ctx context.Context) error

// payload.go
func BuildSmokePayload(uiURL string, timeout, handshakeTimeout time.Duration) string

// response.go
type Response struct {
    Success     bool
    Console     []ConsoleEntry
    Network     []NetworkEntry
    PageErrors  []PageError
    Handshake   HandshakeRaw
    Screenshot  string // base64
    HTML        string
    Title       string
    URL         string
    Timings     Timings
    Error       string
}
```

**Testing seam**:
- `httptest.Server` for integration tests
- Interface extraction for unit tests

#### 5. `handshake`
**Purpose**: Interpret handshake results from browser response
**Depends on**: Nothing (pure functions)

```go
// signals.go
var KnownSignals = []Signal{
    {Property: "__vrooliBridgeChildInstalled", Description: "Legacy bridge marker"},
    {Property: "IFRAME_BRIDGE_READY", Description: "Global ready flag"},
    {Property: "IframeBridge.ready", Description: "Bridge instance ready"},
    // ...
}

// detector.go
type Detector struct{}

func (d *Detector) Evaluate(raw *browser.HandshakeRaw) HandshakeResult {
    return HandshakeResult{
        Signaled:   raw.Signaled,
        TimedOut:   raw.TimedOut,
        DurationMs: raw.DurationMs,
        Error:      raw.Error,
    }
}
```

**Testing seam**: Pure functions, no external dependencies.

#### 6. `artifacts`
**Purpose**: Persist test artifacts to filesystem
**Depends on**: Filesystem interface

```go
// writer.go
type Writer struct {
    baseDir string
    fs      FileSystem // interface for testing
}

func (w *Writer) WriteAll(ctx context.Context, raw *RawArtifacts) (*ArtifactPaths, error)

// For testing
type FileSystem interface {
    WriteFile(path string, data []byte, perm os.FileMode) error
    MkdirAll(path string, perm os.FileMode) error
}

// Default implementation uses os package
type OSFileSystem struct{}
```

**Testing seam**: `FileSystem` interface allows in-memory testing.

---

## Integration Points

### Phase Integration

```go
// phases/phase_structure.go (updated)

func runStructurePhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
    // ... existing structure checks ...

    // Section: UI Smoke Test
    observations = append(observations, NewSectionObservation("🌐", "Running UI smoke test..."))

    if smokeResult, failure := runUISmoke(ctx, env, logWriter); failure != nil {
        failure.Observations = append(observations, failure.Observations...)
        return *failure
    } else if smokeResult != nil {
        observations = append(observations, smokeResultToObservations(smokeResult)...)
    }

    // ...
}

func runUISmoke(ctx context.Context, env workspace.Environment, logWriter io.Writer) (*uismoke.Result, *RunReport) {
    cfg := uismoke.Config{
        ScenarioName:     env.ScenarioName,
        ScenarioDir:      env.ScenarioDir,
        BrowserlessURL:   getBrowserlessURL(),
        Timeout:          90 * time.Second,
        HandshakeTimeout: 15 * time.Second,
    }

    orch := orchestrator.New(cfg,
        orchestrator.WithLogger(logWriter),
        orchestrator.WithPreflightChecker(preflight.NewChecker(cfg.BrowserlessURL)),
        orchestrator.WithBrowserClient(browser.NewClient(cfg.BrowserlessURL)),
        orchestrator.WithArtifactWriter(artifacts.NewWriter(env.ScenarioDir)),
        orchestrator.WithHandshakeDetector(handshake.NewDetector()),
    )

    result, err := orch.Run(ctx)
    if err != nil {
        return nil, &RunReport{
            Err:                   err,
            FailureClassification: classifyError(err),
            Remediation:           remediationForError(err),
        }
    }

    return result, nil
}
```

### Configuration Loading

Load UI smoke settings from `.vrooli/testing.json`:

```go
// In orchestrator or config package
func LoadFromTestingJSON(scenarioDir string) (*Config, error) {
    path := filepath.Join(scenarioDir, ".vrooli", "testing.json")
    // Parse and extract structure.ui_smoke settings
}
```

### Backward Compatibility

The result JSON format must match existing schema for:
- `vrooli scenario status --json` consumption
- Existing UI in scenarios that display smoke results

---

## Testing Strategy

### Unit Tests

| Package | Test Focus | Mocking Strategy |
|---------|-----------|------------------|
| `uismoke` | Config validation, result building | None needed (pure types) |
| `orchestrator` | Workflow coordination | Interface mocks for all deps |
| `preflight` | Individual checks | `httptest.Server`, mock filesystem |
| `browser` | HTTP client behavior | `httptest.Server` |
| `handshake` | Signal evaluation | None needed (pure functions) |
| `artifacts` | File writing | In-memory filesystem |

### Integration Tests

```go
// orchestrator/orchestrator_integration_test.go
// +build integration

func TestOrchestrator_RealBrowserless(t *testing.T) {
    if os.Getenv("BROWSERLESS_URL") == "" {
        t.Skip("BROWSERLESS_URL not set")
    }
    // Test against real Browserless instance
}
```

### Test Fixtures

```
api/internal/uismoke/testdata/
├── browser_responses/
│   ├── success_with_handshake.json
│   ├── success_no_handshake.json
│   ├── network_failure.json
│   ├── page_error.json
│   └── timeout.json
├── scenarios/
│   ├── valid_bundle/
│   │   └── ui/dist/index.html
│   ├── stale_bundle/
│   │   └── ui/dist/index.html
│   └── missing_bridge/
│       └── ui/package.json
└── payloads/
    └── expected_smoke_payload.js
```

### Coverage Targets

- `uismoke`: 90%+ (simple types)
- `orchestrator`: 85%+ (main coordination logic)
- `preflight`: 80%+ (external dependencies)
- `browser`: 75%+ (HTTP complexity)
- `handshake`: 95%+ (pure logic)
- `artifacts`: 80%+ (filesystem operations)

---

## Implementation Phases

### Phase 1: Core Types and Interfaces - COMPLETE
- [x] Create `uismoke` package with `Config`, `Result`, `Status`
- [x] Define all interfaces in `orchestrator/interfaces.go`
- [x] Write unit tests for types
- [x] Add package documentation (`doc.go` files)

### Phase 2: Browser Client - COMPLETE
- [x] Implement `browser.Client` with Browserless API
- [x] Port JavaScript payload generation from bash
- [x] Implement response parsing
- [x] Add comprehensive tests with `httptest.Server`
- [x] Test against real Browserless (integration test)

### Phase 3: Preflight Checks - COMPLETE
- [x] Implement browserless health check
- [x] Port bundle freshness logic from bash
- [x] Port iframe-bridge dependency check
- [x] Add unit tests with mocked dependencies

### Phase 4: Handshake Detection - COMPLETE
- [x] Define signal constants
- [x] Implement evaluation logic
- [x] Add exhaustive unit tests

### Phase 5: Artifact Writer - COMPLETE
- [x] Implement filesystem writer
- [x] Add path resolution logic
- [x] Test with in-memory filesystem

### Phase 6: Orchestrator Integration - COMPLETE
- [x] Implement `Orchestrator.Run()` workflow
- [x] Add functional options for configuration
- [x] Wire up all components
- [x] Integration tests

### Phase 7: Phase Integration - COMPLETE
- [x] Update `phase_structure.go` to use new implementation
- [x] Native mode enabled by default (can be disabled via `TEST_GENIE_UI_SMOKE_NATIVE=false`)
- [x] Ensure backward-compatible JSON output

### Phase 8: Documentation and Cleanup - COMPLETE
- [x] Plan document updated
- [x] Bash fallback preserved for rollback if needed

**Total Actual Effort**: Completed in single session

---

## Risk Mitigation

### Risk 1: Browserless API Changes
**Mitigation**: Version-pin Browserless, add API version detection, comprehensive response parsing tests.

### Risk 2: JavaScript Payload Complexity
**Mitigation**: Extract payload as embedded file (`//go:embed`), test against real browser, keep bash version as reference.

### Risk 3: Timing-Sensitive Handshake
**Mitigation**: Configurable timeouts, retry logic, clear timeout error messages.

### Risk 4: Breaking Existing Consumers
**Mitigation**: Feature flag, parallel output to validate JSON compatibility, phased rollout.

---

## Success Criteria

1. All existing `vrooli scenario ui-smoke` functionality works via Go
2. test-genie structure phase executes smoke test directly (no shell-out)
3. 80%+ test coverage across all new packages
4. Documentation updated with Go-specific details
5. No regression in smoke test reliability
6. Clear error messages for all failure modes

---

## Open Questions

1. **Should we keep bash as fallback?** During transition, having both might help catch regressions.

2. **Should preflight checks be configurable?** Some teams might want to skip bundle freshness checks.

3. **Should artifacts be optional?** Screenshots are large; some CI environments might not need them.

4. **Should we support custom handshake signals?** Some scenarios might use non-standard bridge implementations.

---

## References

- Current bash implementation: `scripts/scenarios/testing/shell/ui-smoke.sh`
- Browserless API docs: https://docs.browserless.io/
- iframe-bridge package: `packages/iframe-bridge/`
- test-genie phase system: `api/internal/orchestrator/phases/`
