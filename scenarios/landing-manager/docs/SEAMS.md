# Seams Documentation

> **Last Updated**: 2025-11-29
> **Purpose**: Document deliberate boundaries (seams) where behavior can vary or be substituted without invasive changes

## Overview

A **seam** is a deliberate boundary in the code where behavior can vary or be substituted without invasive changes. Seams enable:
- **Testability**: Replace external dependencies with mocks in tests
- **Flexibility**: Swap implementations without widespread code changes
- **Isolation**: Contain side effects behind explicit boundaries

This document catalogs the seams in landing-manager, their purposes, and how to use them.

---

## 1. Command Executor Seam

**Location**: `api/util/command.go`

**Purpose**: Isolate external CLI command execution (vrooli CLI, kill, etc.) behind an interface, enabling tests to substitute mock responses instead of shelling out to actual commands.

### Interface

```go
type CommandExecutor interface {
    Execute(name string, args ...string) CommandResult
    ExecuteWithContext(ctx context.Context, name string, args ...string) CommandResult
}

type CommandResult struct {
    Output   string
    ExitCode int
    Err      error
}
```

### Implementations

| Implementation | Purpose | Usage |
|---------------|---------|-------|
| `RealCommandExecutor` | Executes actual OS commands via `os/exec` | Production |
| `MockCommandExecutor` | Returns configurable responses | Testing |

### Consumers

1. **PreviewService** (`api/services/preview_service.go`)
   - Uses executor to call `vrooli scenario port` for dynamic port discovery
   - Constructor seam: `NewPreviewServiceWithExecutor(executor)`

2. **Handler (lifecycle operations)** (`api/handlers/lifecycle.go`)
   - Uses executor for all lifecycle commands: start, stop, restart, status, logs
   - Uses executor for process checks (`kill -0 <pid>`)
   - Constructor seam: `NewHandlerWithExecutor(..., cmdExecutor)`

### Testing Pattern

**Test Helper** (`lifecycle_handlers_success_test.go`):
```go
// setupHandlerWithMockExecutor creates a handler with a mocked CLI executor.
// This allows tests to verify behavior without shelling out to real CLI commands.
func setupHandlerWithMockExecutor(t *testing.T) (*handlers.Handler, *util.MockCommandExecutor) {
    db := setupTestDB(t)
    t.Cleanup(func() { db.Close() })

    mockExec := util.NewMockCommandExecutor()
    registry := services.NewTemplateRegistry()
    generator := services.NewScenarioGenerator(registry)
    personaService := services.NewPersonaService(registry.GetTemplatesDir())
    previewService := services.NewPreviewService()
    analyticsService := services.NewAnalyticsService()

    h := handlers.NewHandlerWithExecutor(db, registry, generator, personaService, previewService, analyticsService, mockExec)
    return h, mockExec
}
```

**Lifecycle Handler Test Example**:
```go
func TestHandleScenarioStart_SuccessWithStagingArea(t *testing.T) {
    tmpRoot := t.TempDir()
    os.Setenv("VROOLI_ROOT", tmpRoot)
    defer os.Unsetenv("VROOLI_ROOT")

    // Create a staging area scenario
    stagingPath := filepath.Join(tmpRoot, "scenarios", "landing-manager", "generated", "test-staging-start")
    os.MkdirAll(stagingPath, 0755)

    h, mockExec := setupHandlerWithMockExecutor(t)
    mockExec.SetResponse("vrooli", "Scenario started successfully", 0, nil)

    // Test the handler - CLI calls are mocked
    router := mux.NewRouter()
    router.HandleFunc("/api/v1/lifecycle/{scenario_id}/start", h.HandleScenarioStart)
    // ... make request and verify response

    // Verify CLI was called with correct arguments
    call := mockExec.Calls[0]
    // Staging scenarios include --path flag
    foundPathFlag := slices.Contains(call.Args, "--path")
    assert.True(t, foundPathFlag)
}
```

**Process Check Test Example** (staging area with `kill -0`):
```go
func TestHandleScenarioStatus_StagingAreaWithProcessCheck(t *testing.T) {
    tmpRoot := t.TempDir()
    os.Setenv("VROOLI_ROOT", tmpRoot)
    defer os.Unsetenv("VROOLI_ROOT")

    // Create staging scenario
    stagingPath := filepath.Join(tmpRoot, "scenarios", "landing-manager", "generated", "test-status")
    os.MkdirAll(stagingPath, 0755)

    // Create PID file to trigger process check
    processDir := filepath.Join(os.Getenv("HOME"), ".vrooli", "processes", "scenarios", "test-status")
    os.MkdirAll(processDir, 0755)
    os.WriteFile(filepath.Join(processDir, "api.pid"), []byte("12345"), 0644)
    t.Cleanup(func() { os.RemoveAll(processDir) })

    h, mockExec := setupHandlerWithMockExecutor(t)
    mockExec.SetResponse("kill", "", 0, nil) // Process exists

    // ... test returns running=true because kill -0 succeeded via seam
}
```

**PreviewService Test Example**:
```go
func TestPreviewLinks_Success(t *testing.T) {
    // Create mock executor with configured responses
    mockExec := util.NewMockCommandExecutor()
    mockExec.SetResponse("vrooli", "38000", 0, nil) // Mock UI_PORT response

    // Inject via constructor seam
    ps := services.NewPreviewServiceWithExecutor(mockExec)

    // Test behavior
    result, err := ps.GetPreviewLinks("test-scenario")

    // Verify mock was called correctly
    if len(mockExec.Calls) != 2 { // UI_PORT and API_PORT calls
        t.Error("Expected 2 command calls")
    }
}
```

---

## 2. HTTP Client Seam

**Location**: `api/handlers/handlers.go`

**Purpose**: Isolate external HTTP requests (to issue tracker, etc.) behind an injectable http.Client, enabling tests to substitute mock HTTP servers.

### Interface

Uses standard `*http.Client` with injectable timeout.

### Consumers

1. **Handler.postJSON** (`api/handlers/customize.go`)
   - Uses HTTPClient for issue tracker API calls
   - Constructor seam: `NewHandlerWithHTTPClient(..., httpClient)`

### Testing Pattern

```go
func TestCustomize_IssueTrackerIntegration(t *testing.T) {
    // Create mock HTTP server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
    }))
    defer server.Close()

    // Create handler with custom client
    client := &http.Client{Timeout: 1 * time.Second}
    h := handlers.NewHandlerWithHTTPClient(db, registry, generator, personaSvc, previewSvc, analyticsSvc, client)

    // Set environment to point to mock server
    os.Setenv("APP_ISSUE_TRACKER_API_BASE", server.URL)

    // Test customize endpoint...
}
```

---

## 3. Service Layer Seams

**Location**: `api/handlers/handlers.go`, `api/services/*`

**Purpose**: Separate business logic from HTTP handling via dependency injection, enabling unit testing of each layer independently.

### Service Interfaces (Implicit)

The Handler struct accepts service implementations via constructor:

```go
type Handler struct {
    DB               *sql.DB
    Registry         *services.TemplateRegistry
    Generator        *services.ScenarioGenerator
    PersonaService   *services.PersonaService
    PreviewService   *services.PreviewService
    AnalyticsService *services.AnalyticsService
    HTTPClient       *http.Client
    CmdExecutor      util.CommandExecutor
}
```

### Services and Their Test Seams

| Service | Test Constructor | Environment Override |
|---------|-----------------|---------------------|
| `TemplateRegistry` | `NewTemplateRegistryWithDir(dir)` | `TEMPLATES_DIR` |
| `ScenarioGenerator` | Uses injected registry | `GEN_OUTPUT_DIR`, `TEMPLATE_PAYLOAD_DIR` |
| `PreviewService` | `NewPreviewServiceWithExecutor(exec)` | `GEN_OUTPUT_DIR` |
| `AnalyticsService` | `NewAnalyticsService()` | `ANALYTICS_DATA_DIR` |
| `PersonaService` | Takes templates dir in constructor | - |

---

## 4. Validation Seam

**Location**: `api/validation/validation.go`

**Purpose**: Centralize input validation patterns in one module, ensuring consistent validation across all handlers and enabling easy adjustment of validation rules.

### Validation Functions

```go
func ValidateTemplateID(id string) error
func ValidateScenarioName(name string) error
func ValidateScenarioSlug(slug string) error
func ValidateScenarioID(scenarioID string) error
func ValidatePersonaID(id string) error
func ValidateBrief(brief string) error
func ValidateTailParam(tail string) (string, bool)
```

### Patterns

Compiled regex patterns available for direct use:
- `TemplateIDPattern`
- `ScenarioNamePattern`
- `ScenarioSlugPattern`
- `ScenarioIDPattern`
- `PersonaIDPattern`

---

## 5. Environment/Configuration Seams

**Location**: `api/util/scenario.go`, service constructors

**Purpose**: Allow tests to override paths and configuration via environment variables, avoiding hardcoded paths that break in test environments.

### Environment Variables

| Variable | Purpose | Used By |
|----------|---------|---------|
| `VROOLI_ROOT` | Override Vrooli installation root | `util.GetVrooliRoot()` |
| `GEN_OUTPUT_DIR` | Override generated scenarios output path | `util.GenerationRoot()` |
| `TEMPLATES_DIR` | Override templates directory | `TemplateRegistry` |
| `TEMPLATE_PAYLOAD_DIR` | Override template payload source | `ScenarioGenerator` |
| `ANALYTICS_DATA_DIR` | Override analytics data storage | `AnalyticsService` |
| `APP_ISSUE_TRACKER_API_BASE` | Issue tracker API endpoint | `handlers.resolveIssueTrackerBase()` |
| `APP_ISSUE_TRACKER_API_PORT` | Issue tracker API port | `handlers.resolveIssueTrackerBase()` |

---

## 6. Logging Seam

**Location**: `api/util/logging.go`

**Purpose**: Centralize structured logging, making it easy to change logging format or destination without touching handlers.

### Functions

```go
func LogStructured(msg string, fields map[string]interface{})
func LogStructuredError(msg string, fields map[string]interface{})
```

### Usage in Handlers

```go
h.Log("scenario_started", map[string]interface{}{"scenario_id": scenarioID})
```

---

## Seam Usage Guidelines

### When to Use Seams

1. **External Commands**: Always use `CommandExecutor` instead of direct `exec.Command`
2. **HTTP Calls**: Always use injected `HTTPClient` instead of `http.DefaultClient`
3. **File Paths**: Check for environment variable overrides in test environments
4. **Validation**: Use centralized validation functions from `validation` package

### When Adding New Seams

1. Create an interface if behavior needs to vary
2. Provide at least two implementations (real + mock)
3. Add constructor that accepts the seam dependency
4. Document in this file
5. Add tests that exercise both implementations

### Anti-Patterns to Avoid

1. **Direct exec.Command**: Always go through `CommandExecutor` for testability
2. **Hardcoded paths**: Use environment variable overrides for test isolation
3. **Global state**: Prefer dependency injection over package-level variables
4. **Leaking implementation**: Keep HTTP/CLI details in service layer, not handlers

---

## Future Seam Opportunities

1. **Database Interface**: Currently uses `*sql.DB` directly; could benefit from repository interface for pure unit tests
2. **File System Seam**: `os.Stat`, `os.ReadDir`, etc. could be abstracted for tests that don't need real filesystem
3. **Time Seam**: `time.Now()` calls could use injectable clock for deterministic tests
