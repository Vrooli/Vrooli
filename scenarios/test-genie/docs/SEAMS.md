# Test-Genie Seam Documentation

This document catalogs the architectural **seams** in the test-genie codebase, specifically for the **agents** and **containment** subsystems. Seams are intentional variation points that enable testing, extension, and controlled change.

## What is a Seam?

A seam is a place where you can alter behavior without editing the code at that location. In Go, seams are typically:
- **Interfaces** that can be satisfied by different implementations
- **Function parameters** that accept interfaces instead of concrete types
- **Functional options** that configure behavior at construction time

## Package: `internal/agents`

### Seam: `AgentRepository` (model.go:186-266)

**Purpose:** Abstracts agent persistence from business logic.

**Interface:**
```go
type AgentRepository interface {
    Create(ctx context.Context, input CreateAgentInput) (*SpawnedAgent, error)
    Get(ctx context.Context, id string) (*SpawnedAgent, error)
    ListActive(ctx context.Context) ([]*SpawnedAgent, error)
    ListAll(ctx context.Context, limit int) ([]*SpawnedAgent, error)
    // ... other methods
}
```

**Production Implementation:** `PostgresAgentRepository`
**Test Implementation:** In-memory or mock repository

**Usage:**
```go
repo := agents.NewPostgresAgentRepository(db)
svc := agents.NewAgentService(repo, opts...)
```

**Why this seam exists:**
- Enables testing without a real database
- Allows swapping to different storage backends (Redis, file-based, etc.)
- Isolates database concerns from agent lifecycle logic

---

### Seam: `ProcessChecker` (service.go:22-25)

**Purpose:** Abstracts OS process checking for agent lifecycle management.

**Interface:**
```go
type ProcessChecker interface {
    ProcessExists(pid int) bool
}
```

**Production Implementation:** `OSProcessChecker`
**Test Implementation:** `MockProcessChecker`

**Usage:**
```go
svc := agents.NewAgentService(repo,
    agents.WithProcessChecker(&MockProcessChecker{exists: true}),
)
```

**Why this seam exists:**
- Enables testing orphan detection without real processes
- Allows simulating various process states (running, dead, zombie)

---

### Seam: `EnvironmentProvider` (service.go:27-30)

**Purpose:** Abstracts environment variable access for hostname/PID tracking.

**Interface:**
```go
type EnvironmentProvider interface {
    Hostname() string
}
```

**Production Implementation:** `OSEnvironmentProvider`
**Test Implementation:** `MockEnvironmentProvider`

**Why this seam exists:**
- Enables consistent test output regardless of host machine
- Allows testing multi-node scenarios with different hostnames

---

### Seam: `TimeProvider` (service.go:32-35)

**Purpose:** Abstracts time for expiration and lock management testing.

**Interface:**
```go
type TimeProvider interface {
    Now() time.Time
}
```

**Production Implementation:** `RealTimeProvider`
**Test Implementation:** `MockTimeProvider`

**Why this seam exists:**
- Enables testing time-sensitive lock expiration logic
- Allows simulating lock renewal failures without real timeouts

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

## Package: `internal/security`

### Seam: `AllowedCommandsProvider` (bash_validator.go:25-28)

**Purpose:** Abstracts the source of allowed bash command prefixes.

**Interface:**
```go
type AllowedCommandsProvider interface {
    GetAllowedCommands() []AllowedBashCommand
}
```

**Production Implementation:** `DefaultAllowedCommandsProvider`

**Usage:**
```go
validator := security.NewBashCommandValidator(
    security.WithAllowedCommandsProvider(customProvider),
)
```

**Why this seam exists:**
- Enables scenario-specific command allowlists
- Allows testing with minimal or extended command sets
- Supports configuration-driven security policies

---

### Seam: `BlockedPatternsProvider` (bash_validator.go:33-35)

**Purpose:** Abstracts the source of blocked prompt patterns.

**Interface:**
```go
type BlockedPatternsProvider interface {
    GetBlockedPatterns() []BlockedPattern
}
```

**Why this seam exists:**
- Enables testing pattern matching without default patterns
- Allows scenario-specific blocking rules

---

## Package: `internal/app/httpserver`

### Seam: Server Dependencies (server.go:40-51)

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
    AgentService        *agents.AgentService
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

Used in `AgentService`, `DockerProvider`, and `BashCommandValidator`:

```go
// Construction with options
svc := agents.NewAgentService(repo,
    agents.WithProcessChecker(checker),
    agents.WithTimeProvider(timer),
)
```

### 2. Interface Injection Pattern

Used in `Server.Dependencies`:

```go
deps := httpserver.Dependencies{
    ContainmentSelector: mockSelector,
    AgentService:        mockAgentService,
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
| `AgentRepository` | Yes | Inline in tests | `PostgresAgentRepository` | Yes |
| `ProcessChecker` | Yes | `MockProcessChecker` | `OSProcessChecker` | Yes |
| `EnvironmentProvider` | Yes | `MockEnvironmentProvider` | `OSEnvironmentProvider` | Yes |
| `TimeProvider` | Yes | `MockTimeProvider` | `RealTimeProvider` | Yes |
| `Provider` | Yes | Via seams | `DockerProvider`, `FallbackProvider` | Yes |
| `ProviderSelector` | Yes | Via seams | `Manager` | Yes |
| `CommandLookup` | Yes | `MockCommandLookup` | `OSCommandLookup` | Yes |
| `CommandRunner` | Yes | `MockCommandRunner` | `OSCommandRunner` | Yes |
| `AllowedCommandsProvider` | Via handler tests | Custom providers | `DefaultAllowedCommandsProvider` | Yes |
| `BlockedPatternsProvider` | Via handler tests | Custom providers | `DefaultBlockedPatternsProvider` | Yes |
