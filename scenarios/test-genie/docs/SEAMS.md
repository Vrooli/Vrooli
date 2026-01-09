# Test-Genie Seam Documentation

This document catalogs the architectural **seams** in the test-genie codebase. Seams are intentional variation points that enable testing, extension, and controlled change.

## What is a Seam?

A seam is a place where you can alter behavior without editing the code at that location. In Go, seams are typically:
- **Interfaces** that can be satisfied by different implementations
- **Function parameters** that accept interfaces instead of concrete types
- **Functional options** that configure behavior at construction time

## Agent State Management

> **NOTE:** Agent state (tasks, runs, profiles) is now managed by the **agent-manager** service.
> Test-genie uses the `agentmanager` package to communicate with agent-manager via HTTP.
> The seams documented below are for **containment** (OS-level isolation) which test-genie
> still manages locally as a reference for agent-manager's actual execution environment.

---

## Package: `internal/containment`

### Seam: `Provider` (provider.go:78-95)

**Purpose:** Abstracts the execution environment for agent sandboxing.

**Interface:**
```go
type Provider interface {
    Type() ContainmentType
    IsAvailable(ctx context.Context) bool
    PrepareCommand(ctx context.Context, config ExecutionConfig) (*exec.Cmd, error)
    Info() ProviderInfo
}
```

**Implementations:**
- `DockerProvider` - Docker container isolation (security level 7)
- `FallbackProvider` - No isolation (security level 0)

**Why this seam exists:**
- Enables testing agent spawning without Docker
- Allows adding new containment strategies (bubblewrap, nsjail, etc.)
- Graceful degradation when Docker unavailable

---

### Seam: `ProviderSelector` (provider.go:116-127)

**Purpose:** Abstracts the selection logic for choosing a containment provider.

**Interface:**
```go
type ProviderSelector interface {
    SelectProvider(ctx context.Context) Provider
    GetStatus(ctx context.Context) ContainmentStatus
    ListProviders(ctx context.Context) []ProviderInfo
}
```

**Production Implementation:** `Manager`
**Test Implementation:** Mock selector returning specific provider

**Usage in Server:**
```go
type Dependencies struct {
    ContainmentSelector containment.ProviderSelector // Optional: defaults to DefaultManager()
}
```

**Why this seam exists:**
- Enables HTTP handler testing without real containment
- Allows testing specific provider selection scenarios
- Injectable via `Dependencies` struct

---

### Seam: `CommandLookup` (docker.go:24-26)

**Purpose:** Abstracts `exec.LookPath` for testing Docker availability.

**Interface:**
```go
type CommandLookup interface {
    LookPath(file string) (string, error)
}
```

**Production Implementation:** `OSCommandLookup`
**Test Implementation:** `MockCommandLookup`

**Usage:**
```go
provider := containment.NewDockerProvider(
    containment.WithCommandLookup(&MockCommandLookup{available: map[string]bool{"docker": false}}),
)
```

**Why this seam exists:**
- Enables testing Docker availability detection without Docker
- Allows simulating "Docker not installed" scenarios

---

### Seam: `CommandRunner` (docker.go:30-32)

**Purpose:** Abstracts command execution for availability checking.

**Interface:**
```go
type CommandRunner interface {
    Run(ctx context.Context, name string, args ...string) error
}
```

**Production Implementation:** `OSCommandRunner`
**Test Implementation:** `MockCommandRunner`

**Why this seam exists:**
- Enables testing "Docker installed but daemon not running" scenarios
- Avoids real Docker invocation in tests

---

## Package: `internal/app/httpserver`

### Seam: Server Dependencies (server.go)

**Purpose:** Dependency injection for all HTTP handler services.

**Structure:**
```go
type Dependencies struct {
    DB                  *sql.DB
    SuiteQueue          suiteRequestQueue
    Executions          execution.ExecutionHistory
    ExecutionSvc        suiteExecutor
    Scenarios           scenarioDirectory
    PhaseCatalog        phaseCatalog
    AgentService        *agentmanager.AgentService  // Agent-manager integration
    ContainmentSelector containment.ProviderSelector
    Logger              Logger
}
```

**Why this seam exists:**
- All handler dependencies are injectable interfaces
- Enables comprehensive HTTP integration testing
- Supports different configurations for dev/test/prod

---

## Seam Usage Patterns

### 1. Functional Options Pattern

Used in `DockerProvider`:

```go
// Construction with options
provider := containment.NewDockerProvider(
    containment.WithCommandLookup(mockLookup),
    containment.WithContainmentConfig(cfg),
)
```

### 2. Interface Injection Pattern

Used in `Server.Dependencies`:

```go
deps := httpserver.Dependencies{
    ContainmentSelector: mockSelector,
    AgentService:        mockAgentService,  // Uses agentmanager.AgentService
}
```

### 3. Default Fallback Pattern

Optional dependencies default to production implementations:

```go
containmentSel := deps.ContainmentSelector
if containmentSel == nil {
    containmentSel = containment.DefaultManager()
}
```

---

## Testing Guidelines

1. **Always use seams in tests** - Never instantiate production dependencies directly in tests
2. **Mock one layer at a time** - Test handlers with mock services, services with mock repositories
3. **Test edge cases at seams** - Seams are where behavior varies, so test the variations
4. **Add seams proactively** - If you find yourself wanting to test something but can't, add a seam

---

## Adding New Seams

When adding a new seam:

1. **Define an interface** in the same package as the consumer
2. **Create production implementation** with `OS`/`Default`/`Real` prefix
3. **Use functional options** for construction-time injection
4. **Document the seam** in this file
5. **Add tests** that exercise the seam with mock implementations

---

## Seam Health Indicators

| Seam | Has Tests | Mock Impl | Production Impl | Documentation |
|------|-----------|-----------|-----------------|---------------|
| `Provider` | Yes | Via seams | `DockerProvider`, `FallbackProvider` | Yes |
| `ProviderSelector` | Yes | Via seams | `Manager` | Yes |
| `CommandLookup` | Yes | `MockCommandLookup` | `OSCommandLookup` | Yes |
| `CommandRunner` | Yes | `MockCommandRunner` | `OSCommandRunner` | Yes |

> **Note:** Agent-related seams (AgentRepository, ProcessChecker, EnvironmentProvider, TimeProvider,
> AllowedCommandsProvider, BlockedPatternsProvider) have been removed. Agent lifecycle is now
> managed by the agent-manager service.
