---
title: "Seams & Architecture"
description: "Testability boundaries, responsibility zones, and substitution points"
category: "technical"
order: 6
audience: ["developers"]
---

# Seams & Architecture

> **Last Updated**: 2025-12-02
> **Purpose**: Document deliberate boundaries (seams) where behavior can vary or be substituted without invasive changes

## Overview

A **seam** is a deliberate boundary in the code where behavior can vary or be substituted without invasive changes. Seams enable:
- **Testability**: Replace external dependencies with mocks in tests
- **Flexibility**: Swap implementations without widespread code changes
- **Isolation**: Contain side effects behind explicit boundaries

This document catalogs the seams in this landing page template, their purposes, and how to use them.

---

## Responsibility Zones

### Entry / Presentation Layer

HTTP handlers in `api/*_handlers.go` (e.g., `account_handlers.go`, `variant_handlers.go`, `metrics_handlers.go`) only:
- Parse and validate transport concerns
- Enforce auth middleware
- Serialize responses

Client utilities live in `ui/src/shared/api/*.ts` and exclusively call REST endpoints.

### Coordination / Domain Layer

Services such as `LandingConfigService`, `VariantService`, `ContentService`, `PlanService`, `MetricsService`, and `DownloadAuthorizer`:
- Work while hiding HTTP/DB details
- Encapsulate business rules (variant selection, pricing assembly, analytics validation, download gating)
- Prevent presentation code from duplicating logic

### Integrations / Infrastructure Layer

Database access stays in services like `download_service.go`, `content_service.go`, `account_service.go`, etc. They are the only layer:
- Issuing SQL or touching storage
- Interacting with Stripe adapters
- Environment/config parsing (`main.go`, `.env` files)

### Cross-Cutting Concerns

- Logging helpers in `logging.go`
- Middleware (session/auth) in `auth.go`
- Helper utilities (`writeJSON`, `resolveUserIdentity`)

These remain thin seams that presentation code reuses without embedding domain rules.

---

## Boundary Clarifications

- `/api/v1/downloads` routes through `DownloadAuthorizer`. The handler no longer reasons about plan bundles or entitlement states; it validates request params and maps domain errors to HTTP status codes.

- Variant HTTP handlers live in `api/variant_handlers.go`, keeping transport validation separate from `VariantService`'s domain logic.

- Download API calls moved to `ui/src/shared/api/downloads.ts`, separating landing config clients from entitlement-aware download logic.

- Metrics ingestion validation moved into `MetricsService.TrackEvent`. The service enforces required fields via `MetricValidationError`.

- Admin portal React routes hand off orchestration to thin controllers under `ui/src/surfaces/admin-portal/controllers`.

---

## Testability Seams

### 1. Command Executor Seam

**Location**: `api/util/command.go`

**Purpose**: Isolate external CLI command execution (vrooli CLI, kill, etc.) behind an interface, enabling tests to substitute mock responses.

**Interface:**
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

**Implementations:**

| Implementation | Purpose | Usage |
|---------------|---------|-------|
| `RealCommandExecutor` | Executes actual OS commands via `os/exec` | Production |
| `MockCommandExecutor` | Returns configurable responses | Testing |

**Testing Pattern:**
```go
func TestPreviewLinks_Success(t *testing.T) {
    mockExec := util.NewMockCommandExecutor()
    mockExec.SetResponse("vrooli", "38000", 0, nil)

    ps := services.NewPreviewServiceWithExecutor(mockExec)
    result, err := ps.GetPreviewLinks("test-scenario")

    // Verify mock was called correctly
    if len(mockExec.Calls) != 2 {
        t.Error("Expected 2 command calls")
    }
}
```

---

### 2. HTTP Client Seam

**Location**: `api/handlers/handlers.go`

**Purpose**: Isolate external HTTP requests behind an injectable http.Client.

**Usage:**
```go
func TestCustomize_Integration(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
    }))
    defer server.Close()

    client := &http.Client{Timeout: 1 * time.Second}
    h := handlers.NewHandlerWithHTTPClient(..., client)
    // Test...
}
```

---

### 3. Service Layer Seams

**Location**: `api/handlers/handlers.go`, `api/services/*`

**Purpose**: Separate business logic from HTTP handling via dependency injection.

**Handler struct:**
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

---

### 4. Environment/Configuration Seams

**Location**: `api/util/scenario.go`, service constructors

**Purpose**: Allow tests to override paths via environment variables.

| Variable | Purpose | Used By |
|----------|---------|---------|
| `VROOLI_ROOT` | Override Vrooli installation root | `util.GetVrooliRoot()` |
| `GEN_OUTPUT_DIR` | Override generated scenarios path | `util.GenerationRoot()` |
| `TEMPLATES_DIR` | Override templates directory | `TemplateRegistry` |
| `ANALYTICS_DATA_DIR` | Override analytics storage | `AnalyticsService` |

---

## How to Extend Safely

### Presentation Changes
New endpoints or UI calls should stick to parsing and delegating to services. If a handler needs business rules, extract them into a service first.

### Domain Updates
Go into the relevant service (`MetricsService`, `DownloadAuthorizer`, etc.). Add dedicated tests so logic can evolve without starting the full stack.

### Integration Updates
Schema tweaks, Stripe wiring, etc. belong in the service touching that system. Keep SQL/SDK usage localized.

### Cross-Cutting Enhancements
Logging, tracing, feature flags should hook through middleware or helper seams.

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

1. **Direct exec.Command**: Always go through `CommandExecutor`
2. **Hardcoded paths**: Use environment variable overrides
3. **Global state**: Prefer dependency injection over package-level variables
4. **Leaking implementation**: Keep HTTP/CLI details in service layer

---

## Future Seam Opportunities

1. **Database Interface**: Could benefit from repository interface for pure unit tests
2. **File System Seam**: `os.Stat`, `os.ReadDir` could be abstracted
3. **Time Seam**: `time.Now()` calls could use injectable clock

---

## See Also

- [Configuration Guide](CONFIGURATION_GUIDE.md) - Environment variables
- [API Reference](api/README.md) - Endpoint documentation
