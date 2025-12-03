# Requirements Sync Native Go Implementation Plan

## Overview

This document outlines the plan to implement requirements synchronization natively in test-genie's Go API, replacing the current Node.js script delegation (`scripts/requirements/`). The design follows **screaming architecture** principles where package names reveal domain intent, with clear **boundaries of responsibility** and **seams for testing**.

## Current State

```
orchestrator/requirements/
├── requirements_syncer.go      # Shells out to Node.js
└── requirements_syncer_test.go
```

The current `nodeRequirementsSyncer` builds JSON payloads and invokes:
```
node scripts/requirements/report.js --scenario <name> --mode sync
```

## Target State

A fully native Go implementation within `internal/requirements/` that:
1. Discovers and parses requirement files
2. Loads test evidence from phase results
3. Enriches requirements with live test status
4. Syncs requirement files and writes metadata
5. Generates reports in multiple formats

---

## Package Structure (Screaming Architecture)

```
internal/
├── requirements/                    # Domain: Requirements Management
│   ├── requirement.go              # Core domain types (Requirement, Validation)
│   ├── module.go                   # RequirementModule (parsed file)
│   ├── evidence.go                 # EvidenceRecord, EvidenceMap
│   ├── status.go                   # Status enums and derivation rules
│   │
│   ├── discovery/                  # Subdomain: File Discovery
│   │   ├── discovery.go            # RequirementDiscoverer interface + impl
│   │   ├── scanner.go              # Directory walking, import resolution
│   │   └── discovery_test.go
│   │
│   ├── parsing/                    # Subdomain: JSON Parsing
│   │   ├── parser.go               # RequirementParser interface + impl
│   │   ├── normalizer.go           # Field normalization (validation→validations)
│   │   ├── schema.go               # Schema validation (optional)
│   │   └── parsing_test.go
│   │
│   ├── evidence/                   # Subdomain: Test Evidence Loading
│   │   ├── loader.go               # EvidenceLoader interface + impl
│   │   ├── phase_results.go        # Parse coverage/phase-results/*.json
│   │   ├── vitest.go               # Parse ui/coverage/vitest-requirements.json
│   │   ├── manual.go               # Parse coverage/manual-validations/log.jsonl
│   │   └── evidence_test.go
│   │
│   ├── enrichment/                 # Subdomain: Status Enrichment
│   │   ├── enricher.go             # RequirementEnricher interface + impl
│   │   ├── validation_matcher.go   # Match validations to evidence
│   │   ├── status_aggregator.go    # Aggregate validation→requirement status
│   │   ├── hierarchy_resolver.go   # Parent/child rollup with cycle detection
│   │   └── enrichment_test.go
│   │
│   ├── sync/                       # Subdomain: Synchronization
│   │   ├── syncer.go               # RequirementSyncer interface + impl
│   │   ├── status_updater.go       # Update requirement/validation statuses
│   │   ├── validation_discovery.go # Auto-add missing vitest validations
│   │   ├── orphan_detector.go      # Detect/remove orphaned validations
│   │   ├── metadata_writer.go      # Write coverage/sync/*.json
│   │   ├── file_writer.go          # Write updated requirement files
│   │   └── sync_test.go
│   │
│   ├── snapshot/                   # Subdomain: Snapshot Generation
│   │   ├── builder.go              # SnapshotBuilder interface + impl
│   │   ├── operational_targets.go  # Derive OT rollups from requirements
│   │   └── snapshot_test.go
│   │
│   ├── reporting/                  # Subdomain: Report Generation
│   │   ├── reporter.go             # Reporter interface + impl
│   │   ├── json_renderer.go        # JSON output format
│   │   ├── markdown_renderer.go    # Markdown table format
│   │   ├── trace_renderer.go       # Full traceability format
│   │   └── reporting_test.go
│   │
│   ├── validation/                 # Subdomain: Structural Validation
│   │   ├── validator.go            # StructureValidator interface + impl
│   │   ├── rules.go                # Validation rules (duplicates, refs, etc.)
│   │   └── validation_test.go
│   │
│   └── service.go                  # RequirementsService - orchestrates all subdomains
│
├── orchestrator/
│   ├── requirements/
│   │   ├── requirements_syncer.go  # Updated: delegates to requirements.Service
│   │   └── requirements_syncer_test.go
│   └── ...
```

---

## Domain Types

### Core Types (`requirements/requirement.go`)

```go
package requirements

import "time"

// Requirement represents a single requirement with its validations
type Requirement struct {
    ID          string            `json:"id"`
    Title       string            `json:"title"`
    Status      DeclaredStatus    `json:"status"`
    PRDRef      string            `json:"prd_ref,omitempty"`
    Category    string            `json:"category,omitempty"`
    Criticality Criticality       `json:"criticality,omitempty"`
    Description string            `json:"description,omitempty"`
    Tags        []string          `json:"tags,omitempty"`
    Children    []string          `json:"children,omitempty"`
    DependsOn   []string          `json:"depends_on,omitempty"`
    Blocks      []string          `json:"blocks,omitempty"`
    Validations []Validation      `json:"validation,omitempty"`

    // Enriched fields (not persisted to requirement files)
    LiveStatus       LiveStatus        `json:"-"`
    AggregatedStatus AggregatedStatus  `json:"-"`
    SourceFile       string            `json:"-"`
}

// Validation represents a test/automation reference for a requirement
type Validation struct {
    Type       ValidationType   `json:"type"`
    Ref        string           `json:"ref,omitempty"`
    WorkflowID string           `json:"workflow_id,omitempty"`
    Phase      string           `json:"phase,omitempty"`
    Status     ValidationStatus `json:"status"`
    Notes      string           `json:"notes,omitempty"`
    Scenario   string           `json:"scenario,omitempty"`
    Folder     string           `json:"folder,omitempty"`
    Metadata   map[string]any   `json:"metadata,omitempty"`

    // Enriched fields (not persisted)
    LiveStatus  LiveStatus     `json:"-"`
    LiveDetails *LiveDetails   `json:"-"`
}

// LiveDetails contains test execution metadata
type LiveDetails struct {
    Timestamp       time.Time `json:"timestamp,omitempty"`
    DurationSeconds float64   `json:"duration_seconds,omitempty"`
    Evidence        string    `json:"evidence,omitempty"`
    SourcePath      string    `json:"source_path,omitempty"`
}
```

### Status Types (`requirements/status.go`)

```go
package requirements

// DeclaredStatus is user-specified requirement status
type DeclaredStatus string

const (
    StatusPending        DeclaredStatus = "pending"
    StatusPlanned        DeclaredStatus = "planned"
    StatusInProgress     DeclaredStatus = "in_progress"
    StatusComplete       DeclaredStatus = "complete"
    StatusNotImplemented DeclaredStatus = "not_implemented"
)

// LiveStatus is derived from test evidence
type LiveStatus string

const (
    LivePassed  LiveStatus = "passed"
    LiveFailed  LiveStatus = "failed"
    LiveSkipped LiveStatus = "skipped"
    LiveNotRun  LiveStatus = "not_run"
    LiveUnknown LiveStatus = "unknown"
)

// ValidationStatus is derived from live test results
type ValidationStatus string

const (
    ValStatusNotImplemented ValidationStatus = "not_implemented"
    ValStatusPlanned        ValidationStatus = "planned"
    ValStatusImplemented    ValidationStatus = "implemented"
    ValStatusFailing        ValidationStatus = "failing"
)

// Criticality levels from PRD
type Criticality string

const (
    CriticalityP0 Criticality = "P0"
    CriticalityP1 Criticality = "P1"
    CriticalityP2 Criticality = "P2"
)

// ValidationType categorizes validation sources
type ValidationType string

const (
    ValTypeTest       ValidationType = "test"
    ValTypeAutomation ValidationType = "automation"
    ValTypeManual     ValidationType = "manual"
    ValTypeLighthouse ValidationType = "lighthouse"
)

// Status derivation functions
func DeriveValidationStatus(live LiveStatus) ValidationStatus { ... }
func DeriveRequirementStatus(current DeclaredStatus, validations []Validation) DeclaredStatus { ... }
func DeriveLiveRollup(statuses []LiveStatus) LiveStatus { ... }
func DeriveDeclaredRollup(statuses []DeclaredStatus) DeclaredStatus { ... }
func NormalizeLiveStatus(s string) LiveStatus { ... }
func NormalizeDeclaredStatus(s string) DeclaredStatus { ... }
```

### Module Types (`requirements/module.go`)

```go
package requirements

// RequirementModule represents a parsed requirement file
type RequirementModule struct {
    Metadata     ModuleMetadata  `json:"_metadata,omitempty"`
    Imports      []string        `json:"imports,omitempty"`
    Requirements []Requirement   `json:"requirements"`

    // Tracking fields
    FilePath     string `json:"-"`
    RelativePath string `json:"-"`
    IsIndex      bool   `json:"-"`
}

type ModuleMetadata struct {
    Module          string `json:"module,omitempty"`
    Description     string `json:"description,omitempty"`
    LastValidatedAt string `json:"last_validated_at,omitempty"`
    AutoSyncEnabled *bool  `json:"auto_sync_enabled,omitempty"`
    SchemaVersion   string `json:"schema_version,omitempty"`
}
```

### Evidence Types (`requirements/evidence.go`)

```go
package requirements

import "time"

// EvidenceRecord represents test execution evidence
type EvidenceRecord struct {
    RequirementID   string
    Status          LiveStatus
    Phase           string
    Evidence        string
    UpdatedAt       time.Time
    DurationSeconds float64
    SourcePath      string
    Metadata        map[string]any
}

// EvidenceMap indexes evidence by requirement ID
type EvidenceMap map[string][]EvidenceRecord

// ManualValidation from log.jsonl
type ManualValidation struct {
    RequirementID string    `json:"requirement_id"`
    Status        string    `json:"status"`
    ValidatedAt   time.Time `json:"validated_at"`
    ExpiresAt     time.Time `json:"expires_at,omitempty"`
    ValidatedBy   string    `json:"validated_by,omitempty"`
    ArtifactPath  string    `json:"artifact_path,omitempty"`
    Notes         string    `json:"notes,omitempty"`
}
```

---

## Interface Definitions (Seams for Testing)

### Discovery (`requirements/discovery/discovery.go`)

```go
package discovery

import "context"

// DiscoveredFile represents a found requirement file
type DiscoveredFile struct {
    AbsolutePath string
    RelativePath string
    IsIndex      bool
}

// RequirementDiscoverer finds requirement files in a scenario
type RequirementDiscoverer interface {
    // Discover finds all requirement files, respecting index.json imports
    Discover(ctx context.Context, scenarioRoot string) ([]DiscoveredFile, error)
}

// FileSystem abstracts file operations for testing
type FileSystem interface {
    ReadDir(path string) ([]os.DirEntry, error)
    ReadFile(path string) ([]byte, error)
    Stat(path string) (os.FileInfo, error)
    Walk(root string, fn filepath.WalkFunc) error
}
```

### Parsing (`requirements/parsing/parser.go`)

```go
package parsing

import (
    "context"
    "github.com/vrooli/test-genie/internal/requirements"
)

// RequirementParser parses requirement JSON files
type RequirementParser interface {
    // Parse reads and normalizes a requirement file
    Parse(ctx context.Context, filePath string) (*requirements.RequirementModule, error)

    // ParseAll parses multiple files and builds an index
    ParseAll(ctx context.Context, files []discovery.DiscoveredFile) (ModuleIndex, error)
}

// ModuleIndex provides fast lookup by requirement ID
type ModuleIndex struct {
    Modules      []*requirements.RequirementModule
    ByID         map[string]*requirements.Requirement
    ByFile       map[string]*requirements.RequirementModule
    ParentIndex  map[string]string // childID -> parentID
}
```

### Evidence (`requirements/evidence/loader.go`)

```go
package evidence

import (
    "context"
    "github.com/vrooli/test-genie/internal/requirements"
)

// EvidenceLoader loads test evidence from various sources
type EvidenceLoader interface {
    // LoadAll loads evidence from all sources
    LoadAll(ctx context.Context, scenarioRoot string) (*EvidenceBundle, error)
}

// EvidenceBundle contains all loaded evidence
type EvidenceBundle struct {
    PhaseResults      requirements.EvidenceMap
    VitestEvidence    map[string][]VitestFile
    ManualValidations *ManualManifest
}

// VitestFile represents a discovered vitest test file
type VitestFile struct {
    FilePath      string
    RequirementID string
    TestNames     []string
}

// ManualManifest contains parsed manual validations
type ManualManifest struct {
    ManifestPath string
    Entries      []requirements.ManualValidation
    ByRequirement map[string]requirements.ManualValidation
}
```

### Enrichment (`requirements/enrichment/enricher.go`)

```go
package enrichment

import (
    "context"
    "github.com/vrooli/test-genie/internal/requirements"
    "github.com/vrooli/test-genie/internal/requirements/evidence"
    "github.com/vrooli/test-genie/internal/requirements/parsing"
)

// RequirementEnricher enriches requirements with live test status
type RequirementEnricher interface {
    // Enrich attaches live status to requirements and validations
    Enrich(ctx context.Context, index *parsing.ModuleIndex, evidence *evidence.EvidenceBundle) error

    // ComputeSummary calculates aggregate statistics
    ComputeSummary(modules []*requirements.RequirementModule) Summary
}

// Summary contains aggregate statistics
type Summary struct {
    Total           int
    ByDeclaredStatus map[requirements.DeclaredStatus]int
    ByLiveStatus     map[requirements.LiveStatus]int
    ByCriticality    map[requirements.Criticality]int
    CriticalityGap   int // P0/P1 not complete
}
```

### Sync (`requirements/sync/syncer.go`)

```go
package sync

import (
    "context"
    "github.com/vrooli/test-genie/internal/requirements"
    "github.com/vrooli/test-genie/internal/requirements/evidence"
    "github.com/vrooli/test-genie/internal/requirements/parsing"
)

// SyncOptions configures sync behavior
type SyncOptions struct {
    PruneOrphans     bool     // Remove validations for deleted test files
    AllowPartial     bool     // Allow sync without full test suite
    TestCommands     []string // Commands that were executed
    DryRun           bool     // Preview changes without writing
}

// SyncResult contains sync operation results
type SyncResult struct {
    FilesUpdated      int
    ValidationsAdded  int
    ValidationsRemoved int
    StatusesChanged   int
    Errors            []error
}

// RequirementSyncer synchronizes requirement files with test results
type RequirementSyncer interface {
    // Sync updates requirement files based on test evidence
    Sync(ctx context.Context, index *parsing.ModuleIndex, evidence *evidence.EvidenceBundle, opts SyncOptions) (*SyncResult, error)
}

// FileWriter abstracts file writing for testing
type FileWriter interface {
    WriteJSON(path string, data any) error
    WriteJSONL(path string, records []any) error
}
```

### Reporting (`requirements/reporting/reporter.go`)

```go
package reporting

import (
    "context"
    "io"
    "github.com/vrooli/test-genie/internal/requirements"
    "github.com/vrooli/test-genie/internal/requirements/enrichment"
    "github.com/vrooli/test-genie/internal/requirements/parsing"
)

// OutputFormat specifies report output format
type OutputFormat string

const (
    FormatJSON     OutputFormat = "json"
    FormatMarkdown OutputFormat = "markdown"
    FormatTrace    OutputFormat = "trace"
)

// ReportOptions configures report generation
type ReportOptions struct {
    Format         OutputFormat
    IncludePending bool
    Phase          string // For phase-inspect mode
}

// Reporter generates requirement reports
type Reporter interface {
    // Generate writes a report to the provided writer
    Generate(ctx context.Context, index *parsing.ModuleIndex, summary enrichment.Summary, opts ReportOptions, w io.Writer) error
}
```

---

## Service Layer (`requirements/service.go`)

```go
package requirements

import (
    "context"
    "github.com/vrooli/test-genie/internal/requirements/discovery"
    "github.com/vrooli/test-genie/internal/requirements/enrichment"
    "github.com/vrooli/test-genie/internal/requirements/evidence"
    "github.com/vrooli/test-genie/internal/requirements/parsing"
    "github.com/vrooli/test-genie/internal/requirements/reporting"
    "github.com/vrooli/test-genie/internal/requirements/sync"
    "github.com/vrooli/test-genie/internal/requirements/validation"
)

// Service orchestrates requirement operations
type Service struct {
    discoverer RequirementDiscoverer
    parser     RequirementParser
    loader     EvidenceLoader
    enricher   RequirementEnricher
    syncer     RequirementSyncer
    reporter   Reporter
    validator  StructureValidator
}

// NewService creates a Service with production dependencies
func NewService(scenarioRoot string) *Service { ... }

// SyncInput matches the existing orchestrator interface
type SyncInput struct {
    ScenarioName     string
    ScenarioDir      string
    PhaseDefinitions []phases.Definition
    PhaseResults     []phases.ExecutionResult
    CommandHistory   []string
}

// Sync performs full requirements synchronization
func (s *Service) Sync(ctx context.Context, input SyncInput) error {
    // 1. Discover requirement files
    files, err := s.discoverer.Discover(ctx, input.ScenarioDir)
    if err != nil {
        return fmt.Errorf("discovery: %w", err)
    }

    // 2. Parse all requirement files
    index, err := s.parser.ParseAll(ctx, files)
    if err != nil {
        return fmt.Errorf("parsing: %w", err)
    }

    // 3. Load test evidence
    evidence, err := s.loader.LoadAll(ctx, input.ScenarioDir)
    if err != nil {
        return fmt.Errorf("loading evidence: %w", err)
    }

    // 4. Enrich requirements with live status
    if err := s.enricher.Enrich(ctx, index, evidence); err != nil {
        return fmt.Errorf("enrichment: %w", err)
    }

    // 5. Sync files
    opts := sync.SyncOptions{
        TestCommands: input.CommandHistory,
    }
    result, err := s.syncer.Sync(ctx, index, evidence, opts)
    if err != nil {
        return fmt.Errorf("sync: %w", err)
    }

    log.Printf("Sync complete: %d files updated, %d validations added",
        result.FilesUpdated, result.ValidationsAdded)
    return nil
}

// Report generates a requirements report
func (s *Service) Report(ctx context.Context, scenarioDir string, opts reporting.ReportOptions, w io.Writer) error { ... }

// Validate checks requirement structure
func (s *Service) Validate(ctx context.Context, scenarioDir string) ([]validation.Issue, error) { ... }
```

---

## Integration with Orchestrator

Update `orchestrator/requirements/requirements_syncer.go`:

```go
package requirements

import (
    "context"
    reqsvc "github.com/vrooli/test-genie/internal/requirements"
)

// Syncer interface remains unchanged for backward compatibility
type Syncer interface {
    Sync(ctx context.Context, input SyncInput) error
}

// nativeRequirementsSyncer uses the native Go implementation
type nativeRequirementsSyncer struct {
    service *reqsvc.Service
}

// NewSyncer creates a native Go requirements syncer
func NewSyncer(projectRoot string) Syncer {
    return &nativeRequirementsSyncer{
        service: reqsvc.NewService(projectRoot),
    }
}

func (s *nativeRequirementsSyncer) Sync(ctx context.Context, input SyncInput) error {
    return s.service.Sync(ctx, reqsvc.SyncInput{
        ScenarioName:     input.ScenarioName,
        ScenarioDir:      input.ScenarioDir,
        PhaseDefinitions: input.PhaseDefinitions,
        PhaseResults:     input.PhaseResults,
        CommandHistory:   input.CommandHistory,
    })
}
```

---

## Implementation Phases

### Phase 1: Core Domain Types & Discovery
**Files:** `requirement.go`, `module.go`, `evidence.go`, `status.go`, `discovery/*.go`

- Define all domain types with JSON tags
- Implement status normalization and derivation functions
- Build file discovery with import resolution
- Unit tests for all status derivation rules

**Estimated complexity:** ~800 lines Go

### Phase 2: Parsing & Evidence Loading
**Files:** `parsing/*.go`, `evidence/*.go`

- JSON parsing with field normalization
- Phase results loader (`coverage/phase-results/*.json`)
- Vitest evidence loader (`ui/coverage/vitest-requirements.json`)
- Manual manifest loader (`coverage/manual-validations/log.jsonl`)
- Build requirement index with parent/child mapping

**Estimated complexity:** ~1,000 lines Go

### Phase 3: Enrichment & Status Aggregation
**Files:** `enrichment/*.go`

- Match validations to evidence records
- Attach live status to validations
- Aggregate validation status to requirement level
- Parent/child status rollup with cycle detection
- Summary statistics computation

**Estimated complexity:** ~600 lines Go

### Phase 4: Sync Implementation
**Files:** `sync/*.go`

- Status update logic
- Orphaned validation detection
- Missing vitest validation discovery
- Sync metadata file writing (`coverage/sync/*.json`)
- Requirement file writing (only when changed)

**Estimated complexity:** ~1,200 lines Go

### Phase 5: Reporting & Snapshot
**Files:** `reporting/*.go`, `snapshot/*.go`

- JSON, Markdown, Trace renderers
- Operational target derivation
- Snapshot builder (`coverage/requirements-sync/latest.json`)

**Estimated complexity:** ~600 lines Go

### Phase 6: Validation & Integration
**Files:** `validation/*.go`, `service.go`, update orchestrator

- Structure validation rules
- Service orchestration layer
- Integration with existing orchestrator
- End-to-end tests

**Estimated complexity:** ~500 lines Go

---

## Testing Strategy

### Unit Tests (Per Package)

Each package has `*_test.go` with:
- Table-driven tests for all public functions
- Edge cases for status derivation
- Error path coverage

### Interface Mocks

Create test doubles for all interfaces:

```go
// In discovery/discovery_test.go
type mockFileSystem struct {
    files map[string][]byte
    dirs  map[string][]os.DirEntry
}

func (m *mockFileSystem) ReadFile(path string) ([]byte, error) {
    if data, ok := m.files[path]; ok {
        return data, nil
    }
    return nil, os.ErrNotExist
}
```

### Integration Tests

Located in `internal/requirements/integration_test.go`:
- Full pipeline from discovery → sync
- Uses temporary directories with fixture files
- Validates file output matches expected

### Fixtures

```
internal/requirements/testdata/
├── scenarios/
│   └── test-scenario/
│       ├── requirements/
│       │   ├── index.json
│       │   └── 01-module/
│       │       └── module.json
│       └── coverage/
│           ├── phase-results/
│           │   └── unit.json
│           └── sync/
└── expected/
    └── sync-output/
```

---

## File I/O Boundaries

All file operations go through interfaces:

```go
// In internal/requirements/io.go

// Reader abstracts file reading
type Reader interface {
    ReadFile(path string) ([]byte, error)
    ReadDir(path string) ([]os.DirEntry, error)
    Stat(path string) (os.FileInfo, error)
    Exists(path string) bool
}

// Writer abstracts file writing
type Writer interface {
    WriteFile(path string, data []byte, perm os.FileMode) error
    MkdirAll(path string, perm os.FileMode) error
}

// osReader implements Reader using os package
type osReader struct{}

// osWriter implements Writer using os package
type osWriter struct{}
```

This allows tests to inject mock file systems without touching disk.

---

## Error Handling

```go
package requirements

import "errors"

var (
    ErrNoRequirementsDir  = errors.New("requirements directory not found")
    ErrInvalidJSON        = errors.New("invalid JSON in requirement file")
    ErrCycleDetected      = errors.New("cycle detected in requirement hierarchy")
    ErrDuplicateID        = errors.New("duplicate requirement ID")
    ErrMissingReference   = errors.New("validation references non-existent file")
)

// ParseError wraps parsing errors with file context
type ParseError struct {
    FilePath string
    Line     int
    Err      error
}

func (e *ParseError) Error() string {
    return fmt.Sprintf("%s:%d: %v", e.FilePath, e.Line, e.Err)
}

func (e *ParseError) Unwrap() error {
    return e.Err
}
```

---

## Migration Path

1. **Parallel implementation**: Build native Go alongside existing Node.js
2. **Feature flag**: Add `REQUIREMENTS_SYNC_NATIVE=true` env var to toggle
3. **Validation**: Run both and compare outputs
4. **Cutover**: Make native the default, keep Node.js as fallback
5. **Removal**: Delete Node.js delegation code after stability period

---

## Success Criteria

- [ ] All Node.js functionality replicated in Go
- [ ] All existing tests pass
- [ ] New unit tests achieve >80% coverage
- [ ] Integration tests cover full sync pipeline
- [ ] No external process spawning for sync
- [ ] Sync completes in <2 seconds for typical scenarios
- [ ] Clear error messages for all failure modes

---

## Appendix: Key Algorithms to Port

### Status Priority (for evidence selection)
```go
var statusPriority = map[LiveStatus]int{
    LiveFailed:  4,
    LiveSkipped: 3,
    LivePassed:  2,
    LiveNotRun:  1,
    LiveUnknown: 0,
}
```

### Cycle Detection (for parent/child rollup)
```go
func (r *hierarchyResolver) resolveWithCycleDetection(
    id string,
    visited map[string]bool,
    resolving map[string]bool,
) error {
    if resolving[id] {
        return ErrCycleDetected
    }
    if visited[id] {
        return nil
    }
    resolving[id] = true
    // ... resolve children
    delete(resolving, id)
    visited[id] = true
    return nil
}
```

### Validation Source Detection
```go
func detectValidationSource(v Validation) ValidationSource {
    switch {
    case strings.HasSuffix(v.Ref, ".test.ts"),
         strings.HasSuffix(v.Ref, ".test.tsx"):
        return ValidationSource{Kind: "phase", Phase: "unit"}
    case strings.HasSuffix(v.Ref, "_test.go"):
        return ValidationSource{Kind: "phase", Phase: "unit"}
    case strings.Contains(v.Ref, "test/playbooks/"):
        return ValidationSource{Kind: "automation", Name: extractWorkflowSlug(v.Ref)}
    case v.Type == ValTypeManual:
        return ValidationSource{Kind: "manual"}
    default:
        return ValidationSource{Kind: "unknown"}
    }
}
```
