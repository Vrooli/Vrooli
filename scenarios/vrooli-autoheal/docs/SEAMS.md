# Seams and Responsibility Boundaries

This document describes the architectural seams and responsibility boundaries in the vrooli-autoheal scenario. Understanding these boundaries helps maintainers know where to add or modify behavior.

## Overview

vrooli-autoheal follows a layered architecture with clear responsibility separation:

```
┌─────────────────────────────────────────────────────────────────┐
│                     Entry / Presentation                        │
│  main.go (server lifecycle) │ handlers/*.go (HTTP) │ UI (React)│
├─────────────────────────────────────────────────────────────────┤
│                     Orchestration / Wiring                      │
│              bootstrap/checks.go (check registration)           │
├─────────────────────────────────────────────────────────────────┤
│                        Domain Rules                             │
│  checks/types.go (interfaces, status logic) │ checks/registry.go│
├─────────────────────────────────────────────────────────────────┤
│                   Domain Implementations                        │
│     checks/infra/*.go (infrastructure) │ checks/vrooli/*.go     │
├─────────────────────────────────────────────────────────────────┤
│                   Integrations / Infrastructure                 │
│  persistence/store.go │ platform/platform.go │ config/config.go │
└─────────────────────────────────────────────────────────────────┘
```

---

## API Responsibility Zones

### 1. Entry / Presentation Layer

**Location:** `api/main.go`, `api/internal/handlers/`

**Responsibility:**
- HTTP server lifecycle (startup, graceful shutdown)
- Route registration and middleware
- Request/response serialization (JSON encoding)
- Lifecycle enforcement (reject direct execution)

**Boundaries:**
- Does NOT contain business logic or domain rules
- Does NOT directly access storage (delegates to Store)
- Does NOT create health check instances (delegates to bootstrap)

**Where to add:**
- New HTTP endpoints → `handlers/handlers.go`
- New middleware → `main.go:setupRouter()`
- Server configuration → `main.go:startServer()`

### 2. Orchestration / Wiring Layer

**Location:** `api/internal/bootstrap/`

**Responsibility:**
- Deciding which checks get registered at startup
- Providing operational defaults (target hosts, domains)
- Wiring platform capabilities to checks that need them

**Boundaries:**
- Does NOT implement check logic (delegates to check packages)
- Does NOT handle HTTP (that's the handlers' job)
- Owns operational constants (DefaultNetworkTarget, DefaultDNSDomain)

**Where to add:**
- New default checks → `bootstrap/checks.go:RegisterDefaultChecks()`
- Operational defaults → constants in `bootstrap/checks.go`

### 3. Domain Rules Layer

**Location:** `api/internal/checks/types.go`, `api/internal/checks/registry.go`

**Responsibility:**
- Core types: `Status`, `Result`, `Check` interface, `Summary`
- Pure domain logic: `WorstStatus()`, `AggregateStatus()`, `ComputeSummary()`
- Registry behavior: registration, interval filtering, platform filtering

**Boundaries:**
- Contains NO I/O operations
- Does NOT depend on persistence or HTTP
- Platform capabilities are injected, not detected internally

**Where to add:**
- New status types or result fields → `types.go`
- Status calculation logic → `types.go`
- Registry behavior changes → `registry.go`

### 4. Domain Implementations (Health Checks)

**Location:** `api/internal/checks/infra/`, `api/internal/checks/vrooli/`

**Responsibility:**
- Implementing specific health checks (network, DNS, Docker, etc.)
- Defining check metadata (ID, description, interval, platform filter)
- Running the actual check and returning results

**Boundaries:**
- Each check is self-contained with a single responsibility
- Checks receive dependencies via constructor injection (platform caps)
- Checks do NOT embed operational defaults (those come from bootstrap)

**Where to add:**
- New infrastructure checks → `checks/infra/`
- New Vrooli-specific checks → `checks/vrooli/`
- Platform-specific behavior → inject `*platform.Capabilities`

### 5. Integrations / Infrastructure Layer

**Location:** `api/internal/persistence/`, `api/internal/platform/`, `api/internal/config/`

**Responsibility:**
- `persistence/store.go`: Database CRUD for health results
- `platform/platform.go`: OS/capability detection (cached)
- `config/config.go`: Environment variable loading

**Boundaries:**
- These are adapters to external systems
- Domain logic does NOT live here
- Platform detection is cached; detection logic is isolated

**Where to add:**
- Database operations → `persistence/store.go`
- New platform capabilities → `platform/platform.go`
- Environment configuration → `config/config.go`

---

## UI Responsibility Zones

### 1. Entry / Presentation

**Location:** `ui/src/main.tsx`, `ui/src/App.tsx`, `ui/src/components/`

**Responsibility:**
- React component rendering
- User interaction handling
- Visual state (loading, error, success)

**Boundaries:**
- Does NOT contain business logic
- Data fetching delegated to api client

### 2. API Integration

**Location:** `ui/src/lib/api.ts`

**Responsibility:**
- HTTP client wrapper
- Type definitions matching API responses
- Request/response handling

**Boundaries:**
- Does NOT render UI
- Does NOT contain React hooks (just async functions)

### 3. Cross-Cutting

**Location:** `ui/src/consts/selectors.ts`

**Responsibility:**
- Test selector constants for UI testing

---

## Key Design Decisions

### Dependency Injection for Platform Capabilities

**Before (leaked responsibility):**
```go
func (c *CloudflaredCheck) Run(ctx context.Context) checks.Result {
    caps := platform.Detect() // Hidden dependency!
    ...
}
```

**After (injected dependency):**
```go
func NewCloudflaredCheck(caps *platform.Capabilities) *CloudflaredCheck {
    return &CloudflaredCheck{caps: caps}
}
```

This change:
- Makes dependencies explicit and testable
- Allows testing with mock platform capabilities
- Removes hidden coupling to global state

### Operational Defaults in Bootstrap Layer

**Before (embedded defaults):**
```go
func NewNetworkCheck(target string) *NetworkCheck {
    if target == "" {
        target = "8.8.8.8:53" // Embedded default
    }
    ...
}
```

**After (explicit in bootstrap):**
```go
// In bootstrap/checks.go
const DefaultNetworkTarget = "8.8.8.8:53"

func RegisterDefaultChecks(registry *checks.Registry, caps *platform.Capabilities) {
    registry.Register(infra.NewNetworkCheck(DefaultNetworkTarget))
}
```

This change:
- Keeps check implementations pure
- Centralizes operational configuration
- Makes defaults visible and changeable in one place

---

## Adding New Features

### Adding a New Health Check

1. Create check in `api/internal/checks/infra/` or `checks/vrooli/`
2. Implement the `Check` interface
3. If platform-dependent, accept `*platform.Capabilities` in constructor
4. Register in `bootstrap/checks.go:RegisterDefaultChecks()`
5. Add tests in the same package

### Adding a New API Endpoint

1. Add handler method in `handlers/handlers.go`
2. Register route in `main.go:setupRouter()`
3. Add corresponding client function in `ui/src/lib/api.ts` if needed

### Adding Platform Detection

1. Add field to `platform.Capabilities` struct
2. Add detection function in `platform/platform.go`
3. Call detection in `detect()` function
4. Update tests in `platform_test.go`

---

## Testing Boundaries

| Layer | Test Location | What to Test |
|-------|---------------|--------------|
| Handlers | `handlers/` (future) | HTTP status codes, JSON structure |
| Registry | `checks/registry_test.go` | Registration, filtering, execution |
| Types | `checks/types_test.go` | Status aggregation, summary computation |
| Checks | `checks/infra/*_test.go` | Check interface compliance, result structure |
| Platform | `platform/platform_test.go` | Detection accuracy, caching |
| UI | `ui/src/*.test.tsx` | Component rendering, user interaction |
