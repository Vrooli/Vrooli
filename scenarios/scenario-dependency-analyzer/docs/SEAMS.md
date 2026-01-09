# Seams Documentation

This document describes the seams (deliberate boundaries for behavior substitution) in the scenario-dependency-analyzer codebase. Seams enable testability, modularity, and safe evolution.

## Overview

A **seam** is a place where behavior can vary or be substituted without invasive changes. Good seams make code:
- Easier to test (substitute behavior in tests)
- Safer to change (isolate integrations)
- More modular (clear boundaries between concerns)

## Seam Categories

### 1. Service Interfaces (`internal/app/services/services.go`)

**Status: Strong seam, well-respected**

The services package defines interfaces for all major operations:

| Interface | Purpose | Status |
|-----------|---------|--------|
| `AnalysisService` | Scenario analysis operations | Well-defined |
| `ScanService` | Scan/apply workflows | Well-defined |
| `GraphService` | Dependency graph generation | Well-defined |
| `OptimizationService` | Optimization recommendations | Well-defined |
| `ScenarioService` | Catalog and detail operations | Well-defined |
| `DependencyService` | Stored dependency access and impact analysis | Newly added to keep handlers off store/detector |
| `DeploymentService` | Deployment report access | Well-defined |
| `ProposalService` | Proposed scenario analysis | Well-defined |

Handlers now route dependency health, catalog retrieval, and impact analysis through `DependencyService`, avoiding direct calls to the store or detector in presentation code.
Optimization now consumes the detector through constructor injection instead of calling global bridge helpers.
Optimization workflows also use the injected workspace/dependency services and store for persistence instead of global config/analyzer lookups, keeping orchestration logic inside the service layer and integrations behind service boundaries.

**Usage:**
```go
// Handler accesses services through interfaces
svc := h.analysisService()
result, err := svc.AnalyzeScenario(name)
```

**Testing:**
```go
// Tests can substitute mock implementations
h.services.Graph = mockGraphService{}
```

### 2. External Dependency Seams (`internal/seams/`)

**Status: Extended (second pass)**

The seams package provides interfaces for external dependencies that vary:

| Interface | Purpose | Real Implementation | Test Double |
|-----------|---------|---------------------|-------------|
| `Clock` | Time operations | `RealClock` | `TestClock` |
| `IDGenerator` | UUID generation | `UUIDGenerator` | `SequentialIDGenerator` |
| `FileSystem` | Filesystem operations | `OSFileSystem` | `MemoryFileSystem` |
| `DeploymentReporter` | Deployment analysis | (future) | (future) |
| `DependencyStore` | Persistence operations | (via store.Store) | `MockStore` |
| `DependencyDetector` | Dependency detection | (via detection.Detector) | `MockDetector` |

**Usage:**
```go
// Production code uses default seams
deps := seams.Default

// Test code injects controlled behavior
deps := seams.NewTestDependencies()
result := extractDeclaredResourcesWithSeams(name, cfg, deps)

// Testing with mock store/detector
deps := seams.NewTestDependencies()
mockStore := deps.Store.(*seams.MockStore)
mockStore.SetMetrics(map[string]interface{}{"scenarios_found": 5})
```

**Functions with seam support:**
- `extractDeclaredResourcesWithSeams()` - controlled time/ID generation
- `convertDeclaredScenariosToDependenciesWithSeams()` - controlled time/ID generation
- `Analyzer.generateGraphWithSeams()` - graph building with controlled time/ID generation and explicit store/catalog seams

### 3. Detection Engine (`internal/detection/`)

**Status: Interface defined, integration in progress**

The `Detector` struct coordinates dependency detection:

```go
type Detector struct {
    cfg             appconfig.Config
    catalog         *catalogManager
    resourceScanner *resourceScanner
    scenarioScanner *scenarioScanner
}
```

**Current seams:**
- Public methods: `ScanResources()`, `ScanScenarioDependencies()`, `ScanSharedWorkflows()`
- Catalog queries: `KnownScenario()`, `KnownResource()`
- `seams.DependencyDetector` interface defined (second pass)
- `seams.MockDetector` test double available

**Usage for testing:**
```go
deps := seams.NewTestDependencies()
mockDetector := deps.Detector.(*seams.MockDetector)
mockDetector.AddScenario("test-scenario")
mockDetector.SetDetectedResources([]types.ScenarioDependency{...})
```

**Migration status:** The interface is defined but the concrete Detector doesn't implement it yet due to type constraints (interface methods use `interface{}` while concrete methods use domain types). Future work should create adapter wrappers.

### 4. Store (`internal/store/`)

**Status: Interface defined, integration in progress**

The `Store` struct handles all database persistence:

```go
type Store struct {
    db *sql.DB
}
```

**Current seams:**
- All public methods are well-defined operations
- Store can be nil (graceful degradation)
- `seams.DependencyStore` interface defined (second pass)
- `seams.MockStore` test double available

**Usage for testing:**
```go
deps := seams.NewTestDependencies()
mockStore := deps.Store.(*seams.MockStore)
mockStore.SetDependencies("my-scenario", map[string]interface{}{
    "resources": []interface{}{...},
})
```

**Migration status:** Similar to DependencyDetector, the interface uses `interface{}` for type independence. Future work should create typed adapter wrappers or update consuming code to work through the seams.

### 5. Deployment Package (`internal/deployment/`)

**Status: Weak seam, direct calls**

The deployment package is called directly from multiple locations:

```go
// Direct calls bypass any abstraction
deployment.BuildReport(scenarioName, scenarioPath, scenariosDir, cfg)
deployment.PersistReport(scenarioPath, report)
deployment.LoadReport(scenarioPath)
```

**Locations of direct calls:**
- `analysis.go:87-91` - AnalyzeScenario
- `service_registry.go:257,276-280` - deploymentService

**Future improvement:** Create `DeploymentReporter` interface and inject it.

## Known Issues and Technical Debt

### Global State

The following global state exists in the codebase:

| Variable | Location | Purpose | Impact |
|----------|----------|---------|--------|
| `db *sql.DB` | `runtime.go:17` | Fallback database connection | Used for health checks when no runtime is available |
| `defaultRuntime *Runtime` | `runtime.go:19` | Runtime singleton | Accessed via `currentRuntime()` |
| `fallbackAnalyzer *Analyzer` | `runtime_helpers.go:11` | Lazy analyzer | Created on first access |
| `fallbackDetector *Detector` | `detection_bridge.go:13` | Lazy detector | Created on first access |

**Mitigation:**
- Tests can set these globals (e.g., `db = testDB`)
- Prefer using Runtime/Analyzer instances over globals

### Bridge Files

Bridge files provide package-level functions that access global state:

| File | Functions | Issue |
|------|-----------|-------|
| `store_bridge.go` | `loadStoredDependencies()`, `listScenarioNames()`, etc. | Bypass service layer |
| `detection_bridge.go` | `isKnownScenario()`, `scanForResourceUsage()`, etc. | Bypass service layer |

**Usage pattern:**
```go
// Current: Direct global access
stored, _ := loadStoredDependencies(name)

// Preferred: Through service layer
svc := h.scenarioService()
detail, _ := svc.GetScenarioDetail(name)
```

Dependency-centric HTTP handlers now rely on `DependencyService` rather than touching the store or detector directly, reducing new bridge usage in presentation code.

### Configuration Loading

`appconfig.Load()` is called repeatedly throughout the code instead of being injected once.

**Locations:**
- `api/main.go:25`
- `runtime_helpers.go:17`
- `scenario_workspace.go:22`
- `config_sync.go:20`
- `config_sync.go:156`
- `config_sync.go:305`

**Future improvement:** Pass config via dependency injection.

## Testing Recommendations

### Using Seams for Deterministic Tests

```go
func TestExtractDeclaredResources(t *testing.T) {
    // Create test dependencies with controlled time/IDs
    deps := seams.NewTestDependencies()

    cfg := &types.ServiceConfig{
        Resources: map[string]types.Resource{
            "postgres": {Type: "database", Required: true},
        },
    }

    result := extractDeclaredResourcesWithSeams("test-scenario", cfg, deps)

    // Assertions are now deterministic
    assert.Equal(t, "test-1", result[0].ID) // Predictable ID
    assert.Equal(t, time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), result[0].DiscoveredAt)
}
```

### Testing Services

```go
func TestAnalysisHandler(t *testing.T) {
    // Create handler with mock services
    h := &handler{
        services: services.Registry{
            Analysis: &mockAnalysisService{
                result: &types.DependencyAnalysisResponse{...},
            },
        },
    }

    // Test handler behavior
    router := setupRouter(h)
    rec := makeRequest(t, router, "GET", "/api/v1/analyze/test")

    assert.Equal(t, http.StatusOK, rec.Code)
}
```

## Analyzer Seams Integration

**Status: Seams field added (second pass)**

The Analyzer now supports seam injection via functional options:

```go
// Default construction (uses seams.Default)
analyzer := NewAnalyzer(cfg, db)

// With custom seams for testing
testSeams := seams.NewTestDependencies()
analyzer := NewAnalyzer(cfg, db, WithSeams(testSeams))

// Access seams from analyzer
deps := analyzer.Seams()
now := deps.Clock.Now()
```

The `Analyzer` struct includes:
- `seams *seams.Dependencies` field
- `WithSeams()` functional option
- `Seams()` accessor method

## Migration Path

### Phase 1 (Complete)
- [x] Create seams package with Clock, IDGenerator, FileSystem interfaces
- [x] Add `*WithSeams` variants for key functions
- [x] Document existing seams and issues

### Phase 2 (Complete - Second Pass)
- [x] Define `DependencyStore` interface in seams package
- [x] Define `DependencyDetector` interface in seams package
- [x] Add `MockStore` and `MockDetector` test doubles
- [x] Add seams field to Analyzer struct
- [x] Update Analyzer constructor to accept seams via `WithSeams()` option
- [x] Document seam contracts and usage patterns

### Phase 3 (Future)
- [ ] Create adapter wrappers to connect concrete Store/Detector to seam interfaces
- [ ] Implement `DeploymentReporter` real implementation
- [ ] Reduce global state usage in bridge files
- [ ] Remove bridge files in favor of service layer
- [ ] Inject configuration consistently through seams

## Seam Design Principles

1. **Prefer interfaces over concrete types** at package boundaries
2. **Make seams practical for testing** - easy to substitute behavior
3. **Keep seams focused** - each seam should represent one variation point
4. **Document seam contracts** - callers should understand what varies
5. **Incremental improvement** - add seams where most valuable first
