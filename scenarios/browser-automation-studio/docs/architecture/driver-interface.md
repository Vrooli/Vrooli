# Driver Interface Architecture

_Last reviewed: 2026-01-30_

## Overview

The browser-automation-studio implements a **pluggable driver architecture** that separates browser drivers (HTTP-based communication layer) from navigators (AI-powered vision navigation). The system uses a layered design with clear separation of concerns.

## Architecture Layers

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           API Request Layer                                  │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │              VisionNavigationHandler                                 │   │
│  │  - Extracts X-Client-Source header (ui/cli/api)                     │   │
│  │  - Validates entitlements via CreditPolicy                          │   │
│  │  - Delegates to NavigatorRegistry                                   │   │
│  └───────────────────────────────┬─────────────────────────────────────┘   │
└──────────────────────────────────┼──────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         NavigatorRegistry                                    │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  - Register(nav VisionNavigator)                                     │   │
│  │  - SelectNavigator(ctx, source, preferredType) -> VisionNavigator   │   │
│  │  - ListNavigators(ctx, source) -> []NavigatorInfo                   │   │
│  └───────────────────────────────┬─────────────────────────────────────┘   │
│                                  │                                          │
│     Selection Priority Order: [Playwright, ClaudeCode, ...]                │
└──────────────────────────────────┼──────────────────────────────────────────┘
                                   │
              ┌────────────────────┴────────────────────┐
              │                                         │
              ▼                                         ▼
┌─────────────────────────────────┐   ┌─────────────────────────────────┐
│  PlaywrightVisionNavigator      │   │  ClaudeCodeVisionNavigator      │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │   │  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│  Status: Available              │   │  Status: Stub (future)          │
│                                 │   │                                 │
│  CreditPolicy:                  │   │  CreditPolicy:                  │
│  - 2 credits/step               │   │  - 0 credits (local)            │
│  - Bypass: BYOK, Openrouter     │   │  - No bypass needed             │
│                                 │   │                                 │
│  ClientSourcePolicy:            │   │  ClientSourcePolicy:            │
│  - All sources (ui/cli/api)     │   │  - CLI only                     │
│                                 │   │                                 │
│  Transport: HTTP                │   │  Transport: CLI subprocess      │
└────────────────┬────────────────┘   └─────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Driver Layer                                       │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    driver.Client (HTTP)                              │   │
│  │                                                                      │   │
│  │  Timeouts:                   Circuit Breaker:                       │   │
│  │  - Recording: 30s            - Configurable thresholds              │   │
│  │  - Execution: 5min           - Built-in resilience                  │   │
│  │                                                                      │   │
│  │  Interface Methods:                                                  │   │
│  │  ┌──────────────────┬──────────────────┬──────────────────────┐    │   │
│  │  │ Session          │ Navigation       │ Recording            │    │   │
│  │  ├──────────────────┼──────────────────┼──────────────────────┤    │   │
│  │  │ CreateSession    │ Navigate         │ StopRecording        │    │   │
│  │  │ CloseSession     │ Reload           │ GetRecordingStatus   │    │   │
│  │  │ ResetSession     │ GetNavState      │ GetRecordedActions   │    │   │
│  │  └──────────────────┴──────────────────┴──────────────────────┘    │   │
│  │  ┌──────────────────┬──────────────────┐                           │   │
│  │  │ Execution        │ Visual           │                           │   │
│  │  ├──────────────────┼──────────────────┤                           │   │
│  │  │ RunInstruction   │ CaptureScreenshot│                           │   │
│  │  │                  │ GetFrame         │                           │   │
│  │  └──────────────────┴──────────────────┘                           │   │
│  └───────────────────────────────┬─────────────────────────────────────┘   │
└──────────────────────────────────┼──────────────────────────────────────────┘
                                   │ HTTP
                                   ▼
                    ┌──────────────────────────────┐
                    │    playwright-driver         │
                    │    (external service)        │
                    └──────────────────────────────┘
```

## Interface Definitions

### Target-owned application attachment

Cross-platform validation uses one `AppTarget` descriptor. Its `target_kind`
selects an admitted-URL policy (`electron` or `android-webview`); BAS attaches
to the target-owned renderer and never launches the application or opens a
debugging port. The executor calls the resolver seam for every scenario
navigation, so adding a WebView kind does not create a parallel target field.
The attach also requires the matching immutable `ValidationContext` and
isolation lease.

### VisionNavigator Interface

Each navigator implements the `VisionNavigator` interface:

```go
type VisionNavigator interface {
    // Core navigation
    Navigate(ctx context.Context, req NavigationRequest) (NavigationHandle, error)

    // Policies (strategy pattern)
    CreditPolicy() CreditPolicy
    ClientSourcePolicy() ClientSourcePolicy

    // Introspection
    Type() NavigatorType
    IsAvailable(ctx context.Context) bool
    Description() string
    UnavailableReason(ctx context.Context) string
}
```

### NavigationHandle Interface

```go
type NavigationHandle interface {
    ID() string
    SessionID() string
    Status() NavigationStatus
    Wait(ctx context.Context) error
    Abort(ctx context.Context) error
    Resume(ctx context.Context) error
}
```

### ClientInterface (Driver Client)

```go
type ClientInterface interface {
    // Recording operations
    StopRecording(ctx context.Context, sessionID string) (*StopRecordingResponse, error)
    GetRecordingStatus(ctx context.Context, sessionID string) (*RecordingStatusResponse, error)
    GetRecordedActions(ctx context.Context, sessionID string, clear bool) (*GetActionsResponse, error)

    // Navigation operations
    Navigate(ctx context.Context, sessionID string, req *NavigateRequest) (*NavigateResponse, error)
    Reload(ctx context.Context, sessionID string, req *ReloadRequest) (*ReloadResponse, error)
    GetNavigationState(ctx context.Context, sessionID string) (*NavigationStateResponse, error)

    // Session operations
    CreateSession(ctx context.Context, req *CreateSessionRequest) (*CreateSessionResponse, error)
    CloseSession(ctx context.Context, sessionID string) (*CloseSessionResponse, error)
    ResetSession(ctx context.Context, sessionID string) error

    // Execution operations
    RunInstruction(ctx context.Context, sessionID string, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error)

    // Visual operations
    CaptureScreenshot(ctx context.Context, sessionID string, req *CaptureScreenshotRequest) (*CaptureScreenshotResponse, error)
    GetFrame(ctx context.Context, sessionID, queryParams string) (*GetFrameResponse, error)
}
```

## Policy Architecture

### Credit Policy

```
┌─────────────────────────────────────────────────────────────────┐
│                        CreditPolicy                              │
├─────────────────────────────────────────────────────────────────┤
│  RequiresCredits:  bool                                         │
│  OperationType:    credits.OperationType                        │
│  PerStepCharging:  bool                                         │
│  CreditsPerStep:   int                                          │
│  BypassConditions: []BypassCondition                            │
│                                                                  │
│  Bypass Types:                                                   │
│  ┌──────────────────────┐                                       │
│  │ - BypassBYOK         │ <- User brings their own API key      │
│  │ - BypassOpenrouter   │ <- Using openrouter resource          │
│  │ - BypassLocalExec    │ <- Local-only execution               │
│  └──────────────────────┘                                       │
└─────────────────────────────────────────────────────────────────┘
```

### Client Source Policy

```
┌─────────────────────────────────────────────────────────────────┐
│                    ClientSourcePolicy                            │
├─────────────────────────────────────────────────────────────────┤
│  AllowedSources: []ClientSource                                 │
│                                                                  │
│  ┌────────────┬────────────┬────────────┐                       │
│  │     UI     │    CLI     │    API     │                       │
│  │  (browser) │ (terminal) │  (http)    │                       │
│  └────────────┴────────────┴────────────┘                       │
│                                                                  │
│  Predefined Policies:                                            │
│  - AllSourcesPolicy()  -> ui, cli, api                          │
│  - CLIOnlyPolicy()     -> cli only                              │
└─────────────────────────────────────────────────────────────────┘
```

## Two Pathways to Browser Automation

The system provides two distinct approaches to browser automation that share the same underlying driver layer:

```
┌────────────────────────────────────────────────────────────────────────────┐
│                        TWO PATHWAYS TO AUTOMATION                          │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  ┌──────────────────────────────┐    ┌──────────────────────────────┐    │
│  │    MANUAL RECORDING          │    │      AI NAVIGATION           │    │
│  │    (User-driven)             │    │    (Goal-driven)             │    │
│  ├──────────────────────────────┤    ├──────────────────────────────┤    │
│  │                              │    │                              │    │
│  │  User -> Browser -> Playwright│   │  Goal -> VisionNavigator     │    │
│  │        |                     │    │         |                    │    │
│  │  Native event capture        │    │  Screenshot + AI analysis   │    │
│  │        |                     │    │         |                    │    │
│  │  Callback POST to API        │    │  Determine next action      │    │
│  │        |                     │    │         |                    │    │
│  │  Store in TimelineService    │    │  Execute via driver         │    │
│  │        |                     │    │         |                    │    │
│  │  Broadcast via WebSocket     │    │  Loop until goal reached    │    │
│  │        |                     │    │                              │    │
│  │  Generate workflow           │    │                              │    │
│  │                              │    │                              │    │
│  │  Entry point:                │    │  Entry point:                │    │
│  │  POST /recordings/live/...   │    │  VisionNavigationHandler    │    │
│  │                              │    │         |                    │    │
│  │  Uses:                       │    │  NavigatorRegistry          │    │
│  │  - Record Mode Handler       │    │         |                    │    │
│  │  - Live Capture Service      │    │  PlaywrightVisionNavigator  │    │
│  │  - WebSocket Hub             │    │  or ClaudeCodeNavigator     │    │
│  │  - driver.Client             │    │         |                    │    │
│  │                              │    │  driver.Client               │    │
│  │                              │    │                              │    │
│  └──────────────┬───────────────┘    └──────────────┬───────────────┘    │
│                 │                                    │                    │
│                 └────────────────┬───────────────────┘                    │
│                                  │                                        │
│                                  ▼                                        │
│                    ┌─────────────────────────────┐                       │
│                    │      SHARED LAYER           │                       │
│                    │                             │                       │
│                    │  - driver.Client (HTTP)     │                       │
│                    │  - playwright-driver        │                       │
│                    │  - Session management       │                       │
│                    │  - Page tracking            │                       │
│                    └─────────────────────────────┘                       │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘
```

**Key Insight**: Manual recording uses **callbacks** where the playwright-driver POSTs events back to the API, while AI navigation uses **direct commands** where the API tells the driver what to do. Both share the same driver layer underneath.

## Navigator Registry & Selection

The NavigatorRegistry implements the registry pattern for navigator discovery and selection:

```go
type NavigatorRegistry struct {
    mu         sync.RWMutex
    navigators map[NavigatorType]VisionNavigator
    order      []NavigatorType  // Selection priority
}

// Key Methods:
Register(nav VisionNavigator)                           // Add navigator
Get(navType NavigatorType) (VisionNavigator, error)    // Get by type
SelectNavigator(ctx context.Context, source ClientSource,
                preferredType NavigatorType) (VisionNavigator, error)
ListNavigators(ctx context.Context, source ClientSource) []NavigatorInfo
```

### Selection Logic

1. If preferred type specified: validate availability and source policy
2. Otherwise: auto-select first available navigator that allows the client source
3. Returns specific errors for not found, unavailable, or disallowed cases

### Error Types

| Error | Description |
|-------|-------------|
| `ErrNavigatorNotFound` | Navigator type not registered |
| `ErrNavigatorNotAvailable` | Registered but not reachable |
| `ErrNavigatorNotAllowed` | Source not allowed by policy |
| `ErrNoNavigatorsAvailable` | No suitable options found |

## Data Flow

```
API Request
    |
    v
VisionNavigationHandler
    |
    v
ClientSourcePolicy Check -> ClientSource extraction
    |
    v
NavigatorRegistry.SelectNavigator()
    |
    v
┌─────────────────────────────────┐
│ Available Navigators            │
├─────────────────────────────────┤
│ PlaywrightVisionNavigator       │ (http -> playwright-driver)
│ ClaudeCodeVisionNavigator       │ (cli -> claude --chrome)
└─────────────────────────────────┘
    |
    v
CreditPolicy validation
    |
    v
Navigator.Navigate(request)
    |
    v
Response -> SessionID, NavigationHandle
```

## Architectural Patterns

| Pattern | Implementation | Purpose |
|---------|----------------|---------|
| **Strategy** | VisionNavigator interface | Pluggable navigation implementations |
| **Registry** | NavigatorRegistry | Navigator management and selection |
| **Policy** | CreditPolicy, ClientSourcePolicy | Encapsulate access rules |
| **Dependency Injection** | Options pattern | Configurable components |
| **Circuit Breaker** | driver.Client | Resilience for driver failures |
| **Factory** | Engine factories | Create drivers per execution context |
| **Adapter** | Session wrapper | Mode-aware behavior |

## Key Files

| Layer | File | Purpose |
|-------|------|---------|
| **Interface** | [CODE: api/services/vision/navigator.go] | VisionNavigator interface |
| **Registry** | [CODE: api/services/vision/registry.go] | Navigator selection logic |
| **Impl** | [CODE: api/services/vision/playwright_navigator.go] | Playwright implementation |
| **Impl** | [CODE: api/services/vision/claudecode_navigator.go] | Claude Code stub |
| **Policy** | [CODE: api/services/vision/policy.go] | Credit & source policies |
| **Driver** | [CODE: api/automation/driver/client.go] | HTTP client to playwright-driver |
| **Handler** | [CODE: api/handlers/ai/vision_navigation.go] | HTTP handler integration |
| **Session** | [CODE: api/automation/session/session.go] | Session wrapper around driver |

## Related Documentation

- [DOC: docs/architecture/recording.md] - Manual recording architecture
- [DOC: docs/architecture/ai-navigation.md] - AI navigation architecture
- [DOC: docs/architecture/execution.md] - Workflow execution architecture
- [DOC: docs/SYSTEM_ARCHITECTURE.md] - Complete system overview
