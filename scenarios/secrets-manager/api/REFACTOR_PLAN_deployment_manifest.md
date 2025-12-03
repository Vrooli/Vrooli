# Refactoring Plan: deployment_manifest.go

## Overview

This plan refactors `deployment_manifest.go` (882 lines) into a well-organized, testable, and maintainable structure following established patterns in the secrets-manager codebase.

## Current State Analysis

### Problems Identified

| Problem | Lines | Impact |
|---------|-------|--------|
| **God function** | 67-317 | `generateDeploymentManifest` is 250+ lines doing SQL queries, validation, transformation, resource resolution, summary computation, and telemetry |
| **Mixed concerns** | 319-363 | `buildFallbackManifest` duplicates summary construction logic |
| **Inline types** | 19-65 | Analyzer types are embedded rather than organized |
| **Global db dependency** | 98 | Direct use of global `db` variable makes testing impossible |
| **Path resolution sprawl** | 722-802 | 80 lines of path-climbing logic duplicated across codebase |
| **Utility functions** | 804-882 | Generic utilities mixed with domain logic |
| **No interfaces** | Throughout | No abstraction for external service calls or database |
| **No tests** | N/A | Zero test coverage for critical deployment manifest generation |

### File Dependency Map

```
deployment_handlers.go
        │
        ▼
deployment_manifest.go ──────► scenario-dependency-analyzer (HTTP)
        │
        ▼
      types.go (DeploymentManifest, BundleSecretPlan, etc.)
```

## Target Architecture

```
deployment_manifest_builder.go   # ManifestBuilder struct + Build() method
deployment_manifest_fetcher.go   # SecretFetcher, AnalyzerClient interfaces + implementations
deployment_manifest_summary.go   # Summary computation logic
deployment_manifest_bundle.go    # Bundle secret plan derivation
deployment_manifest_resolver.go  # Resource and path resolution
deployment_manifest_test.go      # Comprehensive tests with mocks
analyzer_types.go                # Types for analyzer report structures
```

## Detailed Refactoring Steps

### Phase 1: Extract Types (Low Risk)

**Goal:** Move analyzer-specific types to dedicated file

**File: `analyzer_types.go`** (new)
```go
package main

import "time"

// Analyzer report types - structures received from scenario-dependency-analyzer service
type analyzerDeploymentReport struct {
    Scenario      string                       `json:"scenario"`
    ReportVersion int                          `json:"report_version"`
    GeneratedAt   time.Time                    `json:"generated_at"`
    Dependencies  []analyzerDependency         `json:"dependencies"`
    Aggregates    map[string]analyzerAggregate `json:"aggregates"`
}

type analyzerDependency struct { ... }
type analyzerRequirement struct { ... }
type analyzerTierSupport struct { ... }
type analyzerAggregate struct { ... }
type analyzerRequirementTotals struct { ... }
```

**Lines to move:** 19-65

---

### Phase 2: Define Interfaces (Foundation for Testing)

**Goal:** Create interfaces for external dependencies

**File: `deployment_manifest_fetcher.go`** (new)

```go
package main

import "context"

// SecretStore abstracts database access for deployment secrets
type SecretStore interface {
    // FetchSecrets retrieves secrets for the given resources and tier
    FetchSecrets(ctx context.Context, tier string, resources []string, includeOptional bool) ([]DeploymentSecretEntry, error)
    // PersistManifest stores manifest telemetry
    PersistManifest(ctx context.Context, scenario, tier string, manifest *DeploymentManifest) error
}

// AnalyzerClient abstracts calls to scenario-dependency-analyzer
type AnalyzerClient interface {
    // FetchDeploymentReport retrieves the analyzer report for a scenario
    FetchDeploymentReport(ctx context.Context, scenario string) (*analyzerDeploymentReport, error)
}

// ResourceResolver abstracts resource discovery from service.json and analyzer
type ResourceResolver interface {
    // ResolveResources determines the effective resource list for a scenario
    ResolveResources(ctx context.Context, scenario string, requestedResources []string) []string
}
```

---

### Phase 3: Implement Database Abstraction

**Goal:** Replace global `db` access with injectable SecretStore

**File: `deployment_manifest_fetcher.go`** (continued)

```go
// PostgresSecretStore implements SecretStore using PostgreSQL
type PostgresSecretStore struct {
    db     *sql.DB
    logger *Logger
}

func NewPostgresSecretStore(db *sql.DB, logger *Logger) *PostgresSecretStore {
    return &PostgresSecretStore{db: db, logger: logger}
}

func (s *PostgresSecretStore) FetchSecrets(ctx context.Context, tier string, resources []string, includeOptional bool) ([]DeploymentSecretEntry, error) {
    // Move SQL query from lines 102-146 of deployment_manifest.go
    // Move row scanning from lines 157-216
}

func (s *PostgresSecretStore) PersistManifest(ctx context.Context, scenario, tier string, manifest *DeploymentManifest) error {
    // Move telemetry insert from lines 308-314
}
```

**Lines to extract:** 102-216, 308-314

---

### Phase 4: Implement Analyzer Client

**Goal:** Extract HTTP client logic for analyzer service

**File: `deployment_manifest_fetcher.go`** (continued)

```go
// HTTPAnalyzerClient implements AnalyzerClient using HTTP
type HTTPAnalyzerClient struct {
    logger      *Logger
    portFn      func(ctx context.Context) (string, error) // Injected for testing
    reportCache map[string]*analyzerReportCacheEntry
    cacheMu     sync.RWMutex
}

type analyzerReportCacheEntry struct {
    report    *analyzerDeploymentReport
    fetchedAt time.Time
}

func NewHTTPAnalyzerClient(logger *Logger) *HTTPAnalyzerClient {
    return &HTTPAnalyzerClient{
        logger:      logger,
        portFn:      discoverAnalyzerPort, // Default implementation
        reportCache: make(map[string]*analyzerReportCacheEntry),
    }
}

func (c *HTTPAnalyzerClient) FetchDeploymentReport(ctx context.Context, scenario string) (*analyzerDeploymentReport, error) {
    // Move logic from fetchAnalyzerReportViaService (lines 579-609)
    // Move discoverAnalyzerPort (lines 611-623)
    // Add caching with staleness check (from ensureAnalyzerDeploymentReport lines 544-577)
}
```

**Lines to extract:** 492-623

---

### Phase 5: Implement Resource Resolver

**Goal:** Extract resource resolution logic

**File: `deployment_manifest_resolver.go`** (new)

```go
// DefaultResourceResolver implements ResourceResolver
type DefaultResourceResolver struct {
    analyzerClient AnalyzerClient
    logger         *Logger
}

func NewResourceResolver(analyzer AnalyzerClient, logger *Logger) *DefaultResourceResolver {
    return &DefaultResourceResolver{analyzerClient: analyzer, logger: logger}
}

func (r *DefaultResourceResolver) ResolveResources(ctx context.Context, scenario string, requestedResources []string) []string {
    // Move resolveScenarioResources logic (lines 722-802)
    // Move analyzer resource extraction (lines 625-636)
    // Move resource merging logic (lines 638-657)
}

// resolveScenarioPath finds the service.json for a scenario
func (r *DefaultResourceResolver) resolveScenarioPath(scenario string) string {
    // Extract path resolution from resolveScenarioResources
}
```

**Lines to extract:** 625-657, 722-802

---

### Phase 6: Extract Summary Builder

**Goal:** Isolate summary computation logic

**File: `deployment_manifest_summary.go`** (new)

```go
// SummaryBuilder computes deployment summary statistics
type SummaryBuilder struct{}

func NewSummaryBuilder() *SummaryBuilder {
    return &SummaryBuilder{}
}

// BuildSummary computes the DeploymentSummary from secret entries
func (b *SummaryBuilder) BuildSummary(
    entries []DeploymentSecretEntry,
    scenarioResources, analyzerResources []string,
    scenario string,
) DeploymentSummary {
    // Move classification/strategy counting (lines 235-265)
    // Move blocking secrets extraction (lines 267-272)
    // Move scope readiness calculation (lines 285-288)
}

// computeBlockingDetails creates detailed blocking secret information
func (b *SummaryBuilder) computeBlockingDetails(
    entry DeploymentSecretEntry,
    scenarioResources, analyzerResources []string,
    scenario string,
) BlockingSecretDetail {
    // Extract from lines 249-263
}
```

**Lines to extract:** 235-288

---

### Phase 7: Extract Bundle Plan Builder

**Goal:** Isolate bundle secret plan derivation

**File: `deployment_manifest_bundle.go`** (new)

```go
// BundlePlanBuilder derives bundle secret plans from deployment entries
type BundlePlanBuilder struct{}

func NewBundlePlanBuilder() *BundlePlanBuilder {
    return &BundlePlanBuilder{}
}

// DeriveBundlePlans creates BundleSecretPlan slice from entries
func (b *BundlePlanBuilder) DeriveBundlePlans(entries []DeploymentSecretEntry) []BundleSecretPlan {
    // Move deriveBundleSecretPlans (lines 365-373)
}

// bundleSecretFromEntry converts a single entry to a bundle plan
func (b *BundlePlanBuilder) bundleSecretFromEntry(entry DeploymentSecretEntry) *BundleSecretPlan {
    // Move bundleSecretFromEntry (lines 375-408)
}

// deriveSecretClass determines the bundle secret class
func (b *BundlePlanBuilder) deriveSecretClass(entry DeploymentSecretEntry) string {
    // Move deriveSecretClass (lines 410-431)
}

// deriveSecretID generates a stable secret identifier
func (b *BundlePlanBuilder) deriveSecretID(entry DeploymentSecretEntry) string {
    // Move deriveSecretID (lines 433-449)
}

// deriveBundleTarget determines secret injection target
func (b *BundlePlanBuilder) deriveBundleTarget(entry DeploymentSecretEntry) BundleSecretTarget {
    // Move deriveBundleTarget (lines 451-471)
}

// derivePrompt builds prompt metadata for user_prompt secrets
func (b *BundlePlanBuilder) derivePrompt(entry DeploymentSecretEntry) *PromptMetadata {
    // Move derivePrompt (lines 473-490)
}
```

**Lines to extract:** 365-490

---

### Phase 8: Create ManifestBuilder (Main Orchestrator)

**Goal:** Create the main builder following OrientationBuilder pattern

**File: `deployment_manifest_builder.go`** (new)

```go
// ManifestBuilder orchestrates deployment manifest generation
type ManifestBuilder struct {
    secretStore      SecretStore
    analyzerClient   AnalyzerClient
    resourceResolver ResourceResolver
    summaryBuilder   *SummaryBuilder
    bundlePlanBuilder *BundlePlanBuilder
    logger           *Logger
}

// ManifestBuilderConfig configures ManifestBuilder construction
type ManifestBuilderConfig struct {
    DB     *sql.DB
    Logger *Logger
}

// NewManifestBuilder creates a production ManifestBuilder
func NewManifestBuilder(cfg ManifestBuilderConfig) *ManifestBuilder {
    secretStore := NewPostgresSecretStore(cfg.DB, cfg.Logger)
    analyzerClient := NewHTTPAnalyzerClient(cfg.Logger)
    resourceResolver := NewResourceResolver(analyzerClient, cfg.Logger)

    return &ManifestBuilder{
        secretStore:       secretStore,
        analyzerClient:    analyzerClient,
        resourceResolver:  resourceResolver,
        summaryBuilder:    NewSummaryBuilder(),
        bundlePlanBuilder: NewBundlePlanBuilder(),
        logger:            cfg.Logger,
    }
}

// Build generates a deployment manifest for the given request
func (b *ManifestBuilder) Build(ctx context.Context, req DeploymentManifestRequest) (*DeploymentManifest, error) {
    scenario := strings.TrimSpace(req.Scenario)
    tier := strings.TrimSpace(strings.ToLower(req.Tier))
    if scenario == "" || tier == "" {
        return nil, fmt.Errorf("scenario and tier are required")
    }

    // 1. Resolve effective resources
    effectiveResources := b.resourceResolver.ResolveResources(ctx, scenario, req.Resources)

    // 2. Handle no-database fallback
    if b.secretStore == nil {
        return b.buildFallbackManifest(scenario, tier, effectiveResources), nil
    }

    // 3. Fetch secrets from database
    entries, err := b.secretStore.FetchSecrets(ctx, tier, effectiveResources, req.IncludeOptional)
    if err != nil {
        return nil, err
    }
    if len(entries) == 0 {
        return nil, fmt.Errorf("no secrets discovered for manifest request")
    }

    // 4. Get analyzer insights
    analyzerReport, _ := b.analyzerClient.FetchDeploymentReport(ctx, scenario)

    // 5. Build summary
    summary := b.summaryBuilder.BuildSummary(entries, /* ... */)

    // 6. Derive bundle plans
    bundlePlans := b.bundlePlanBuilder.DeriveBundlePlans(entries)

    // 7. Assemble manifest
    manifest := &DeploymentManifest{
        Scenario:      scenario,
        Tier:          tier,
        GeneratedAt:   time.Now(),
        Resources:     effectiveResources,
        Secrets:       entries,
        BundleSecrets: bundlePlans,
        Summary:       summary,
    }

    // 8. Add analyzer insights if available
    if analyzerReport != nil {
        manifest.Dependencies = convertAnalyzerDependencies(analyzerReport)
        manifest.TierAggregates = convertAnalyzerAggregates(analyzerReport)
        manifest.AnalyzerGeneratedAt = &analyzerReport.GeneratedAt
    }

    // 9. Persist telemetry
    _ = b.secretStore.PersistManifest(ctx, scenario, tier, manifest)

    return manifest, nil
}

// buildFallbackManifest creates a manifest when database is unavailable
func (b *ManifestBuilder) buildFallbackManifest(scenario, tier string, resources []string) *DeploymentManifest {
    // Move buildFallbackManifest logic (lines 319-363)
}
```

---

### Phase 9: Extract Utilities to Shared Location

**Goal:** Move generic utilities to shared package

**File: `string_utils.go`** (new, or add to existing utils file)

```go
// dedupeStrings returns unique strings sorted alphabetically
func dedupeStrings(values []string) []string { /* lines 804-825 */ }

// intersectStrings returns strings present in both slices
func intersectStrings(a, b []string) []string { /* lines 827-851 */ }

// containsString checks if target exists in list
func containsString(list []string, target string) bool { /* lines 853-860 */ }

// decodeJSONMap safely decodes JSON to map[string]interface{}
func decodeJSONMap(payload []byte) map[string]interface{} { /* lines 862-871 */ }

// decodeStringMap safely decodes JSON to map[string]string
func decodeStringMap(payload []byte) map[string]string { /* lines 873-882 */ }
```

**Lines to extract:** 804-882

---

### Phase 10: Update Handlers

**Goal:** Wire up new ManifestBuilder

**File: `deployment_handlers.go`** (modify)

```go
type DeploymentHandlers struct {
    builder *ManifestBuilder
}

func NewDeploymentHandlers(builder *ManifestBuilder) *DeploymentHandlers {
    return &DeploymentHandlers{builder: builder}
}

func (h *DeploymentHandlers) DeploymentSecrets(w http.ResponseWriter, r *http.Request) {
    // ... validation ...

    manifest, err := h.builder.Build(r.Context(), req)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to generate manifest: %v", err), http.StatusBadRequest)
        return
    }

    writeJSON(w, http.StatusOK, manifest)
}
```

**File: `server.go`** (modify)

```go
func newAPIServer(db *sql.DB, logger *Logger) *APIServer {
    manifestBuilder := NewManifestBuilder(ManifestBuilderConfig{
        DB:     db,
        Logger: logger,
    })

    return &APIServer{
        db: db,
        handlers: handlerSet{
            // ...
            deployment: NewDeploymentHandlers(manifestBuilder),
            // ...
        },
    }
}
```

---

### Phase 11: Add Comprehensive Tests

**Goal:** Create test suite with mocks

**File: `deployment_manifest_test.go`** (new)

```go
package main

import (
    "context"
    "testing"
)

// Mock implementations
type mockSecretStore struct {
    secrets []DeploymentSecretEntry
    err     error
}

func (m *mockSecretStore) FetchSecrets(ctx context.Context, tier string, resources []string, includeOptional bool) ([]DeploymentSecretEntry, error) {
    return m.secrets, m.err
}

func (m *mockSecretStore) PersistManifest(ctx context.Context, scenario, tier string, manifest *DeploymentManifest) error {
    return nil
}

type mockAnalyzerClient struct {
    report *analyzerDeploymentReport
    err    error
}

func (m *mockAnalyzerClient) FetchDeploymentReport(ctx context.Context, scenario string) (*analyzerDeploymentReport, error) {
    return m.report, m.err
}

// Tests
func TestManifestBuilder_Build_Success(t *testing.T) { /* ... */ }
func TestManifestBuilder_Build_NoSecrets(t *testing.T) { /* ... */ }
func TestManifestBuilder_Build_FallbackNoDatabase(t *testing.T) { /* ... */ }
func TestManifestBuilder_Build_AnalyzerUnavailable(t *testing.T) { /* ... */ }
func TestSummaryBuilder_BuildSummary(t *testing.T) { /* ... */ }
func TestBundlePlanBuilder_DeriveSecretClass(t *testing.T) { /* ... */ }
func TestResourceResolver_ResolveResources(t *testing.T) { /* ... */ }
```

---

## Migration Checklist

### Pre-Migration
- [ ] Ensure all current behavior is documented
- [ ] Create backup of deployment_manifest.go
- [ ] Verify existing integration tests pass (if any)

### Phase Execution Order
1. [ ] **Phase 1**: Extract analyzer types → `analyzer_types.go`
2. [ ] **Phase 9**: Extract utilities → `string_utils.go`
3. [ ] **Phase 2**: Define interfaces → `deployment_manifest_fetcher.go`
4. [ ] **Phase 3**: Implement PostgresSecretStore
5. [ ] **Phase 4**: Implement HTTPAnalyzerClient
6. [ ] **Phase 5**: Implement ResourceResolver → `deployment_manifest_resolver.go`
7. [ ] **Phase 6**: Extract SummaryBuilder → `deployment_manifest_summary.go`
8. [ ] **Phase 7**: Extract BundlePlanBuilder → `deployment_manifest_bundle.go`
9. [ ] **Phase 8**: Create ManifestBuilder → `deployment_manifest_builder.go`
10. [ ] **Phase 10**: Update handlers and server wiring
11. [ ] **Phase 11**: Add comprehensive tests

### Post-Migration
- [ ] Run `gofumpt -w .` to format all new files
- [ ] Run `golangci-lint run` to check for issues
- [ ] Verify API responses are identical (no breaking changes)
- [ ] Delete original `deployment_manifest.go`
- [ ] Update any documentation referencing the old structure

## File Summary

| File | Lines | Purpose |
|------|-------|---------|
| `analyzer_types.go` | ~50 | Analyzer report type definitions |
| `deployment_manifest_builder.go` | ~120 | Main ManifestBuilder orchestrator |
| `deployment_manifest_fetcher.go` | ~180 | Interfaces + SecretStore + AnalyzerClient |
| `deployment_manifest_resolver.go` | ~100 | Resource resolution logic |
| `deployment_manifest_summary.go` | ~80 | Summary computation |
| `deployment_manifest_bundle.go` | ~130 | Bundle plan derivation |
| `deployment_manifest_test.go` | ~200 | Test suite with mocks |
| `string_utils.go` | ~80 | Shared utility functions |
| **Total** | ~940 | vs. original 882 (slightly larger but much more maintainable) |

## Risk Assessment

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| API response changes | Low | Test response equality before/after |
| Analyzer integration breaks | Medium | Keep fallback behavior, add retries |
| Performance regression | Low | Same SQL queries, just reorganized |
| Missing edge cases | Medium | Comprehensive test coverage |

## Success Criteria

1. All API responses identical to current implementation
2. Test coverage > 80% for new code
3. No global state dependencies (except logger)
4. Clear separation of concerns
5. Each file < 200 lines
6. golangci-lint passes with no new warnings
