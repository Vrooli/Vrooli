# SEAMS · scenario-to-desktop

This document captures the architectural seams, boundaries, and responsibility zones for the scenario-to-desktop codebase. It serves as the source of truth for understanding where behavior can vary, where responsibilities live, and how to test each layer.

## Domain Model

**Purpose**: Transform Vrooli scenarios into cross-platform desktop applications using Electron.

### Key Domain Concepts

| Concept | Description | Primary Owner |
|---------|-------------|---------------|
| **Generation** | Creating desktop wrapper from scenario metadata | `api/handlers_desktop.go`, `ui/hooks/useGeneratorDraft.ts` |
| **Building** | Compiling for platforms (Win/Mac/Linux) | `api/build_operations.go` |
| **Preflight** | Validating bundled deployments before packaging | `api/handlers_preflight.go`, `ui/hooks/usePreflightSession.ts` |
| **Signing** | Code signing for production distribution | `api/signing/`, `ui/hooks/useSigningConfig.ts` |
| **Records** | Persisted history of desktop generations | `api/handlers_records.go` |
| **Templates** | Desktop app configuration templates | `templates/` |
| **Bundling** | Packaging runtime + services for offline use | `api/bundle_packager.go`, `api/build_compiler.go` |

### Core Workflows

1. **Quick Generate**: Scenario → Auto-detect metadata → Generate Electron wrapper
2. **Custom Generate**: User config → Validate → Generate with customizations
3. **Bundled Preflight**: Manifest → Validate → Start runtime → Health check services
4. **Platform Build**: Wrapper → npm install → electron-builder → Artifacts

---

## Responsibility Zones

### API Layer (Go)

| Zone | Files | Responsibility |
|------|-------|----------------|
| **Entry/HTTP** | `handlers_*.go` | Request parsing, validation, response formatting |
| **Orchestration** | `handlers_desktop.go`, `handlers_preflight.go` | Workflow coordination, step sequencing |
| **Domain Logic** | `domain/generator.ts`, `platform.go` | Business rules, config transformation |
| **Infrastructure** | `build_compiler.go`, `cli_staging.go`, `file_ops.go` | External tool invocation, file operations |
| **Storage** | `preflight_store.go`, `handlers_records.go` | State persistence, session management |

### UI Layer (React + TypeScript)

| Zone | Files | Responsibility |
|------|-------|----------------|
| **Entry/Presentation** | `components/*.tsx` | User interaction, visual rendering |
| **Orchestration** | `hooks/use*.ts` | State management, API coordination |
| **Domain Logic** | `domain/deployment.ts`, `domain/generator.ts` | Business rules (validation, config building) |
| **Infrastructure** | `lib/api.ts`, `lib/draftStorage.ts` | API calls, persistence |

---

## Implemented Seams

### API Seams

#### 1. Bundle Packager Seam (`bundlePackager`)
**Location**: `api/bundle_packager.go:34-39`
```go
type bundlePackager struct {
    runtimeResolver       runtimeResolver       // Find runtime source directory
    runtimeBuilder        runtimeBuilder        // Build Go binaries
    serviceBinaryCompiler serviceBinaryCompiler // Compile service binaries
    cliStager             cliStager             // Stage CLI helpers
}
```
**Purpose**: Allows substituting runtime building and file operations in tests.
**Usage**: Tests can inject stubs to avoid invoking Go toolchains.
**Status**: ✅ Implemented

#### 2. Preflight Session Store Seam (`PreflightSessionStore`)
**Location**: `api/preflight_store.go:17-31`
```go
type PreflightSessionStore interface {
    Create(manifest *bundlemanifest.Manifest, bundleRoot string, ttlSeconds int) (*preflightSession, error)
    Get(id string) (*preflightSession, bool)
    Refresh(session *preflightSession, ttlSeconds int)
    Stop(id string) bool
    Cleanup()
}
```
**Purpose**: Abstracts preflight session lifecycle for testability.
**Usage**: Tests can inject mock stores to verify session behavior without runtime.
**Status**: ✅ Implemented

#### 3. Preflight Job Store Seam (`PreflightJobStore`)
**Location**: `api/preflight_store.go:33-53`
```go
type PreflightJobStore interface {
    Create() *preflightJob
    Get(id string) (*preflightJob, bool)
    Update(id string, fn func(job *preflightJob))
    SetStep(id, stepID, state, detail string)
    SetResult(id string, updater func(prev *BundlePreflightResponse) *BundlePreflightResponse)
    Finish(id string, status, errMsg string)
    Cleanup()
}
```
**Purpose**: Abstracts async job management for testability.
**Status**: ✅ Implemented

#### 4. Record Store Seam (`DesktopAppRecordStore`)
**Location**: `api/records.go`
**Purpose**: Abstracts persistence of desktop generation records
**Status**: ✅ Existing, well-defined interface

#### 5. Build Store Seam (`BuildStore`)
**Location**: `api/builds_store.go`
**Purpose**: Manages in-memory build status with optional persistence
**Status**: ✅ Clean interface

#### 6. Pipeline Orchestrator Seam (`Orchestrator`)
**Location**: `api/pipeline/interfaces.go`
**Purpose**: Coordinates execution of multi-stage desktop deployment pipelines. Enables testing and substitution of orchestration logic.
**Interface**:
```go
type Orchestrator interface {
    RunPipeline(ctx context.Context, config *Config) (*Status, error)
    ResumePipeline(ctx context.Context, pipelineID string, config *Config) (*Status, error)
    GetStatus(pipelineID string) (*Status, bool)
    CancelPipeline(pipelineID string) bool
    ListPipelines() []*Status
}
```
**Status**: ✅ Implemented (Jan 2026)

#### 7. Pipeline Stage Seam (`Stage`)
**Location**: `api/pipeline/interfaces.go`
**Purpose**: Abstracts individual pipeline stages (bundle, preflight, generate, build, smoketest, distribution) for independent testing and substitution.
**Interface**:
```go
type Stage interface {
    Name() string
    Execute(ctx context.Context, input *StageInput) *StageResult
    CanSkip(input *StageInput) bool
    Dependencies() []string
}
```
**Implementations**: `BundleStage`, `PreflightStage`, `GenerateStage`, `BuildStage`, `SmokeTestStage`, `DistributionStage`
**Deep Dive**: See [Smoke Test Pipeline](reference/smoke-test-pipeline.md) for detailed SmokeTestStage execution flow
**Status**: ✅ Implemented (Jan 2026)

#### 8. Pipeline Store Seam (`Store`)
**Location**: `api/pipeline/interfaces.go`, `api/pipeline/store.go`, `api/pipeline/store_file.go`
**Purpose**: Abstracts pipeline state persistence with in-memory and file-backed implementations
**Interface**:
```go
type Store interface {
    Save(status *Status)
    Get(pipelineID string) (*Status, bool)
    GetByIdempotencyKey(key string) (*Status, bool)  // Idempotency support (Jan 2026)
    Update(pipelineID string, fn func(status *Status)) bool
    UpdateStage(pipelineID, stageName string, result *StageResult) bool
    Delete(pipelineID string) bool
    List() []*Status
    Cleanup(olderThan int64)
}
```
**Idempotency Support**: The `GetByIdempotencyKey()` method enables safe retries by allowing the orchestrator to check if a pipeline with the same client-provided idempotency key already exists. This ensures "running twice is no worse than running once" - critical for network timeout scenarios where clients may retry requests.
**Status**: ✅ Implemented with MemoryStore and FileStore options

#### 9. Pipeline Supporting Seams
**Location**: `api/pipeline/interfaces.go`
**Purpose**: Additional seams for pipeline infrastructure
- `CancelManager` - Manages cancellation functions for running pipelines
- `IDGenerator` - Generates unique pipeline identifiers
- `Logger` - Structured logging abstraction
- `TimeProvider` - Time abstraction for deterministic testing
- `WebhookNotifier` - Webhook notification abstraction
- `ManifestGenerator` - On-demand bundle manifest generation via deployment-manager
**Status**: ✅ All interfaces implemented

### UI Seams

#### 1. API Client Seam (`lib/api.ts`)
**Location**: `ui/src/lib/api.ts`
**Purpose**: Centralizes all API calls; components never construct URLs directly
**Status**: ✅ Well-established, comprehensive type coverage

#### 2. Draft Storage Seam (`lib/draftStorage.ts`)
**Location**: `ui/src/lib/draftStorage.ts`
**Purpose**: Abstracts localStorage persistence for form drafts
**Status**: ✅ Clean seam, easy to mock in tests

#### 3. Generator Domain Logic Seam (`domain/generator.ts`)
**Location**: `ui/src/domain/generator.ts`
**Purpose**: Pure functions for form validation and config building. This is the canonical location for all generator-related domain logic.
**Functions**:
```typescript
// Legacy validation (deprecated)
export function validateGeneratorInputs(options: ValidateGeneratorInputsOptions): string | null

// Comprehensive validation with field associations
export function validateFormInputs(params: ValidateFormInputsParams): ValidationError[]

// Config building
export function buildDesktopConfig(options: BuildDesktopConfigOptions): DesktopConfig
export function resolveEndpoints(input: EndpointResolutionInput): EndpointResolution
export function getSelectedPlatforms(platforms: PlatformSelection): string[]
export function computeStandardOutputPath(scenarioName: string): string
export function computeStagingPreviewPath(scenarioName: string): string
```
**Types**:
```typescript
export interface ValidationError { id: string; message: string; field?: string }
export interface ValidateFormInputsParams { ... } // Comprehensive validation params
```
**Status**: ✅ Implemented, extended Jan 2026

#### 4. Deployment Decision Seam (`domain/deployment.ts`)
**Location**: `ui/src/domain/deployment.ts`
**Purpose**: Pure function for computing connection decisions from mode/type
**Status**: ✅ Excellent - pure, testable, no side effects

#### 5. Domain Types Seam (`domain/types.ts`)
**Location**: `ui/src/domain/types.ts`
**Purpose**: Centralized domain types used across the application - ensures domain types don't leak into presentation layer
**Types**:
```typescript
// Build artifacts
export interface DesktopBuildArtifact { ... }

// Telemetry
export interface TelemetryEvent { ... }
export type OperatingSystem = "Windows" | "macOS" | "Linux"
export interface TelemetryFilePath { ... }

// Download
export type Platform = "win" | "mac" | "linux"
export interface PlatformDisplayInfo { ... }
export interface PlatformArtifactGroup { ... }
```
**Status**: ✅ Implemented (Jan 2026)

#### 6. Download Domain Logic Seam (`domain/download.ts`)
**Location**: `ui/src/domain/download.ts`
**Purpose**: Pure functions for download-related operations: platform validation, artifact grouping, size formatting, URL building
**Functions**:
```typescript
// Platform handling
export function isValidPlatform(value: string): value is Platform
export function parsePlatform(value: string | undefined): Platform | undefined
export function getPlatformIcon(platform: string): string
export function getPlatformName(platform: string): string

// Artifact organization
export function groupArtifactsByPlatform(artifacts: DesktopBuildArtifact[] | undefined): Map<Platform | "unknown", PlatformArtifactGroup>
export function getSortedPlatformGroups(artifacts: DesktopBuildArtifact[] | undefined): PlatformArtifactGroup[]
export function getAvailablePlatforms(artifacts: DesktopBuildArtifact[] | undefined): Platform[]
export function hasDownloadableArtifacts(artifacts: DesktopBuildArtifact[] | undefined, platform: Platform): boolean

// Size formatting
export function formatBytes(bytes: number | undefined): string
export function computeTotalArtifactSize(artifacts: DesktopBuildArtifact[] | undefined): number

// URL building
export function buildDownloadPath(options: DownloadResolverOptions): string
```
**Status**: ✅ Implemented (Jan 2026)

#### 7. Telemetry Domain Logic Seam (`domain/telemetry.ts`)
**Location**: `ui/src/domain/telemetry.ts`
**Purpose**: Pure functions for telemetry file parsing, validation, and path generation
**Functions**:
```typescript
// Path generation
export function generateTelemetryPaths(appName: string): TelemetryFilePath[]
export function getTelemetryPathForOS(appName: string, os: OperatingSystem): string

// JSONL parsing
export function parseJsonlContent(content: string): TelemetryParseResult
export function processTelemetryContent(content: string): { success: boolean; events?: TelemetryEvent[]; error?: string }

// Event validation
export function validateTelemetryEvents(events: TelemetryEvent[]): TelemetryValidationResult
export function isStandardEvent(eventType: string): eventType is StandardEventType

// Display helpers
export function formatEventPreview(event: TelemetryEvent): string
export function generateExampleEvent(): string
```
**Status**: ✅ Implemented (Jan 2026)

#### 8. Browser API Seam (`lib/browser.ts`)
**Location**: `ui/src/lib/browser.ts`
**Purpose**: Abstracts browser-specific APIs (clipboard, file reading, downloads) for testability
**Functions**:
```typescript
// Clipboard operations
export async function writeToClipboard(text: string): Promise<ClipboardWriteResult>

// File operations
export async function readFileAsText(file: File): Promise<FileReadResult>

// Download operations
export function triggerDownload(options: DownloadTriggerOptions): void
export function triggerBlobDownload(blob: Blob, filename: string): void
```
**Usage**: Components use these instead of calling browser APIs directly, enabling easy mocking in tests
**Status**: ✅ Implemented (Jan 2026)

#### 9. Generator Draft Hook Seam (`hooks/useGeneratorDraft.ts`)
**Location**: `ui/src/hooks/useGeneratorDraft.ts`
**Purpose**: Encapsulates draft persistence logic with debounced saving
**Status**: ✅ Implemented

#### 10. Preflight Session Hook Seam (`hooks/usePreflightSession.ts`)
**Location**: `ui/src/hooks/usePreflightSession.ts`
**Purpose**: Encapsulates preflight polling, session state, and lifecycle
**Status**: ✅ Implemented

#### 11. Signing Config Hook Seam (`hooks/useSigningConfig.ts`)
**Location**: `ui/src/hooks/useSigningConfig.ts`
**Purpose**: Encapsulates signing configuration queries
**Status**: ✅ Implemented

#### 12. Preflight Constants Seam (`lib/preflight-constants.ts`)
**Location**: `ui/src/lib/preflight-constants.ts`
**Purpose**: Centralizes all preflight UI constants - styles, guidance text, coverage config
**Functions**:
```typescript
// Style mappings
export const PREFLIGHT_STEP_STYLES: Record<PreflightStepState, string>
export const PREFLIGHT_CHECK_STYLES: Record<BundlePreflightCheck["status"], string>
// Issue guidance
export const PREFLIGHT_ISSUE_GUIDANCE: Record<string, PreflightIssueGuidance>
// Coverage configuration
export const COVERAGE_ROWS: CoverageRow[]
```
**Status**: ✅ Implemented

#### 13. Preflight Utilities Seam (`lib/preflight-utils.ts`)
**Location**: `ui/src/lib/preflight-utils.ts`
**Purpose**: Pure utility functions for preflight data processing
**Functions**:
```typescript
export function formatDuration(ms: number): string
export function parseTimestamp(value?: string): number | null
export function formatTimestamp(value?: string): string
export function formatBytes(value?: number): string
export function getListenURL(detail?: string): string | null
export function getServiceURL(serviceId: string, ports?: Record<string, Record<string, number>>): ServiceURLResult
export function getManifestHealthConfig(manifest: unknown, serviceId: string): ManifestHealthConfig | null
export function formatPortSummary(ports?: Record<string, Record<string, number>>): string
export function detectLikelyRootMismatch(validationValid, missingAssetsCount, missingBinariesCount, bundleManifestPath): boolean
```
**Status**: ✅ Implemented

#### 14. Preflight Sub-Components Seam (`components/preflight/`)
**Location**: `ui/src/components/preflight/`
**Purpose**: Focused, reusable components for preflight UI
**Components**:
- `PreflightStepHeader` - Step number, title, status badge
- `PreflightCheckList` - Collapsible check results
- `ValidationIssuesPanel` - Detailed validation errors with guidance
- `MissingSecretsForm` - Secret input form
- `RuntimeInfoPanel` - Runtime identity display
- `ServicesReadinessGrid` - Service health grid with health peek
- `DiagnosticsPanels` - Log tails, fingerprints, port summary
- `CoverageMap` - Coverage comparison visualization
**Status**: ✅ Implemented

#### 15. Pipeline Store Seam (`store/pipelineStore.ts`)
**Location**: `ui/src/store/pipelineStore.ts`
**Purpose**: Unified Zustand store for pipeline state management. Centralizes pipeline execution, polling, stage results, and preflight state.
**Functions**:
```typescript
// State
scenarioName, pipelineId, pipelineStatus, runStatus, error
bundleResult, preflightResult, generateResult, buildResult, smokeTestResult, distributionResult
stageLogs, pipelineHistory, preflightSecrets, preflightOverride

// Actions
setScenario(name: string | null)
runStage(stage: PipelineStage, config?: Partial<PipelineConfig>): Promise<string>
runFullPipeline(config?: Partial<PipelineConfig>): Promise<string>
cancelPipeline(): Promise<void>
resumePipeline(pipelineId: string): Promise<string>
runBundleStage, runPreflightStage, runSmokeTestStage // Convenience actions
loadPipelineStatus(pipelineId: string): Promise<void>
startPolling(), stopPolling()

// Selectors
selectIsRunning, selectCurrentStage, selectProgress
selectStageStatus, selectCanResume, selectStoppedAfterStage
selectPreflightValidationOk, selectPreflightReadinessOk
selectMissingSecrets, selectPreflightSecretsOk, selectPreflightOk
```
**Status**: ✅ Implemented (Jan 2026)

#### 16. Pipeline Utils Seam (`lib/pipeline-utils.ts`)
**Location**: `ui/src/lib/pipeline-utils.ts`
**Purpose**: Pure utility functions for pipeline status mapping and structured log parsing. Enables components to display user-friendly status and parse structured log entries without embedding parsing logic.
**Functions**:
```typescript
// Status mapping
export type MappedBuildStatus = "building" | "ready" | "partial" | "failed"
export function mapPipelineStatus(status: string): MappedBuildStatus

// Log severity levels (matching backend LogLevel)
export type LogLevel = "INFO" | "WARN" | "ERROR" | "DEBUG"

// Parsed log entry structure
export interface ParsedLogEntry {
  timestamp: string;
  level: LogLevel;
  message: string;
  raw: string;
}

// Log parsing - parses structured log format "[TIMESTAMP] [LEVEL] message"
export function parseLogEntry(raw: string): ParsedLogEntry
export function parseLogs(logs: string[]): ParsedLogEntry[]

// Log filtering - filter by severity level
export function filterLogsByLevel(logs: ParsedLogEntry[], minLevel: LogLevel): ParsedLogEntry[]

// Display helpers
export function getLogLevelStyle(level: LogLevel): { color: string; bg: string }
export function formatLogTimestamp(timestamp: string): string

// Log analysis
export function getLatestSignificantLog(logs: ParsedLogEntry[]): ParsedLogEntry | null
```
**Status**: ✅ Implemented, enhanced Jan 2026 with structured log parsing

#### 17. Scenario State Hook Seam (`hooks/useScenarioState.ts`)
**Location**: `ui/src/hooks/useScenarioState.ts`
**Purpose**: Server-side scenario state persistence. Replaces localStorage-based draft storage with server-side persistence, with conflict detection and staleness checking.
**Functions**:
```typescript
// State
state, formState, isLoading, isError, error, hasInitiallyLoaded
isSaving, saveError, lastSavedAt
isStale, pendingChanges, validationStatus
serverHash, localHash, timestamps, buildArtifacts, stages

// Actions
updateFormState(updates: Partial<FormState>)
saveStageResult(stage: string, result: unknown, formStateUpdates?, options?)
saveNow(): Promise<void>
clearState(): Promise<void>
refetch()
resolveConflict(resolution: "local" | "server")
checkStaleness(config: InputFingerprint): Promise<void>
```
**Features**:
- Debounced auto-save with conflict detection
- Server hash tracking for optimistic concurrency
- Periodic staleness checking against manifest changes
- Stage result persistence with form state updates
**Status**: ✅ Implemented (Jan 2026)

---

## File Organization

### API Module Structure (After Refactoring)

```
api/
├── main.go                        # Server setup, route registration
├── types.go                       # API types (requests, responses)
├── server.go                      # HTTP server configuration
│
├── handlers_desktop.go            # Desktop generation handlers (838 lines)
├── handlers_preflight.go          # Preflight validation handlers (1,669 lines)
├── handlers_records.go            # Desktop record management
├── handlers_deployment_manager.go # Deployment-manager coordination
│
├── bundle_packager.go             # Core bundling orchestration (433 lines)
├── file_ops.go                    # File copy/directory operations (114 lines)
├── platform.go                    # Platform key parsing/normalization (163 lines)
├── build_compiler.go              # Go/Rust/npm/custom compilation (238 lines)
├── cli_staging.go                 # CLI helper staging (127 lines)
├── bundle_size.go                 # Size calculation/warnings (116 lines)
│
├── preflight_store.go             # Session/Job store interfaces (268 lines)
├── records.go                     # Record persistence
├── builds_store.go                # Build status management
└── signing/                       # Code signing subsystem
```

### UI Module Structure (After Refactoring)

```
ui/src/
├── hooks/
│   ├── index.ts                   # Export barrel
│   ├── useGeneratorDraft.ts       # Draft persistence hook
│   ├── usePreflightSession.ts     # Preflight state/polling hook
│   └── useSigningConfig.ts        # Signing configuration hook
│
├── domain/
│   ├── index.ts                   # Export barrel
│   ├── deployment.ts              # Deployment decision logic
│   ├── deployment.test.ts         # Tests for deployment logic
│   ├── download.ts                # Download/artifact domain logic
│   ├── download.test.ts           # Tests for download logic
│   ├── generator.ts               # Generator validation/config logic
│   ├── telemetry.ts               # Telemetry parsing/validation logic
│   └── telemetry.test.ts          # Tests for telemetry logic
│
├── components/
│   ├── GeneratorForm.tsx          # Main generator form (uses hooks)
│   ├── BundledPreflightSection.tsx # Preflight orchestration (718 lines)
│   ├── preflight/                 # Preflight sub-components
│   │   ├── index.ts               # Export barrel
│   │   ├── PreflightStepHeader.tsx
│   │   ├── PreflightCheckList.tsx
│   │   ├── CoverageBadge.tsx
│   │   ├── CoverageMap.tsx
│   │   ├── ValidationIssuesPanel.tsx
│   │   ├── MissingSecretsForm.tsx
│   │   ├── RuntimeInfoPanel.tsx
│   │   ├── ServicesReadinessGrid.tsx
│   │   └── DiagnosticsPanels.tsx
│   └── ...
│
└── lib/
    ├── api.ts                     # API client
    ├── draftStorage.ts            # localStorage draft management
    ├── preflight-constants.ts     # Preflight styles, guidance, config
    └── preflight-utils.ts         # Pure preflight utility functions
```

---

## Testing Seams

### Unit Testing Points

| Seam | Test Strategy |
|------|---------------|
| `bundlePackager` | Inject stub `runtimeBuilder` to avoid Go builds |
| `PreflightSessionStore` | Mock interface for session lifecycle tests |
| `PreflightJobStore` | Mock interface for async job tests |
| `decideConnection()` | Pure function, direct unit tests |
| `validateGeneratorInputs()` | Pure function, direct unit tests (deprecated) |
| `validateFormInputs()` | Pure function, direct unit tests - canonical validation |
| `buildDesktopConfig()` | Pure function, direct unit tests |
| `RecordStore` | Mock interface for persistence tests |
| `BuildStore` | Mock interface for status tracking tests |
| `formatDuration()` | Pure function, direct unit tests |
| `parseTimestamp()` | Pure function, direct unit tests |
| `getServiceURL()` | Pure function, direct unit tests |
| `getManifestHealthConfig()` | Pure function, direct unit tests |
| `detectLikelyRootMismatch()` | Pure function, direct unit tests |
| `groupArtifactsByPlatform()` | Pure function, direct unit tests |
| `formatBytes()` | Pure function, direct unit tests |
| `buildDownloadPath()` | Pure function, direct unit tests |
| `parseJsonlContent()` | Pure function, direct unit tests |
| `processTelemetryContent()` | Pure function, direct unit tests |
| `generateTelemetryPaths()` | Pure function, direct unit tests |
| `writeToClipboard()` | Mock navigator.clipboard in tests |
| `readFileAsText()` | Mock File.text() in tests |
| `triggerDownload()` | Mock window.open in tests |

### Integration Testing Points

| Seam | Test Strategy |
|------|---------------|
| `lib/api.ts` | MSW (Mock Service Worker) or fetch mocks |
| `draftStorage.ts` | localStorage mock |
| `usePreflightSession` | Mock API responses |
| `useGeneratorDraft` | Mock localStorage |

### E2E Testing Points

| Flow | Entry Point |
|------|-------------|
| Quick Generate | `POST /api/v1/desktop/generate/quick` |
| Preflight Validation | `POST /api/v1/desktop/preflight/start` |
| Platform Build | `POST /api/v1/desktop/build/{scenario}` |

---

## Recent Refactoring Completed

### Download & Telemetry Domain Extraction (Jan 2026)
**Goal**: Apply screaming architecture principles to downloading and telemetry-related code

**Changes**:
1. Created `domain/download.ts` - Pure functions for platform validation, artifact grouping, size formatting, URL building
2. Created `domain/telemetry.ts` - Pure functions for JSONL parsing, event validation, telemetry path generation
3. Created `domain/index.ts` - Export barrel for all domain modules
4. Refactored `DownloadButtons.tsx` - Now uses domain functions instead of inline logic
5. Refactored `TelemetryUploadCard.tsx` - Now uses domain functions instead of embedded parsing
6. Updated `scenario-inventory/utils.ts` - Re-exports from domain for backward compatibility
7. Added comprehensive test suites: `download.test.ts` and `telemetry.test.ts`

**Improvements**:
- Domain logic is now pure, testable, and isolated from presentation
- Clear separation between "what the app does" (domain) and "how it looks" (components)
- Components are now thin wrappers around domain logic
- Consistent patterns across the domain layer

### Browser Seams & Architecture Improvements (Jan 2026)
**Goal**: Apply screaming architecture, boundary enforcement, and seam discovery principles to download/telemetry code

**Changes**:
1. Created `domain/types.ts` - Centralized domain types (DesktopBuildArtifact, TelemetryEvent, Platform, etc.)
2. Created `lib/browser.ts` - Browser API seams for clipboard, file reading, downloads
3. Refactored all download/telemetry components to use browser seams instead of direct browser API calls
4. Updated type imports to flow from domain layer instead of component layer
5. Deprecated `scenario-inventory/utils.ts` - now a thin re-export layer

**Files Updated**:
- `DownloadButtons.tsx` - Uses `triggerDownload()`, `writeToClipboard()` from browser seams
- `TelemetryUploadCard.tsx` - Uses `readFileAsText()`, `writeToClipboard()` from browser seams
- `PlatformChip.tsx` - Uses `triggerDownload()`, `writeToClipboard()`, `getPlatformIcon()`, `getPlatformName()`
- `BuildDesktopButton.tsx` - Uses `writeToClipboard()`, `getPlatformIcon()`, `getPlatformName()`
- `ScenarioCard.tsx` - Uses `triggerDownload()`, `getPlatformIcon()`

**Architecture Improvements**:
- **Testability**: Components can now be unit tested by mocking browser seams
- **Responsibility Separation**: Domain types live in domain layer, not presentation layer
- **Seam Enforcement**: Browser side effects isolated behind explicit seam functions
- **Dependency Direction**: Presentation → Domain → Types (unidirectional)

### Validation Logic Consolidation (Jan 2026)
**Goal**: Enforce boundary-of-responsibility by moving validation logic from presentation to domain layer

**Problem Identified**: Validation logic was duplicated:
- `validateGeneratorInputs()` in `domain/generator.ts` - returned single string error
- `validateFormInputs()` in `components/generator/ValidationErrors.tsx` - returned rich ValidationError[] with field associations

**Changes**:
1. Moved comprehensive `validateFormInputs()` and `ValidationError` type to `domain/generator.ts`
2. Deprecated the simpler `validateGeneratorInputs()` in favor of the richer version
3. Converted `ValidationErrors.tsx` to a pure presentation component that re-exports from domain layer
4. Updated `components/generator/index.ts` to maintain backward compatibility via re-exports

**Files Changed**:
- `domain/generator.ts` - Added `validateFormInputs()`, `ValidationError`, `ValidateFormInputsParams`
- `components/generator/ValidationErrors.tsx` - Now a thin re-export layer, reduced from 181 to 58 lines

**Architecture Improvements**:
- **Responsibility Separation**: Validation rules now live in domain layer, presentation only displays errors
- **Testability**: `validateFormInputs()` is now a pure function that can be unit tested without UI setup
- **Single Source of Truth**: No duplicate validation logic across presentation and domain layers

### BundledPreflightSection.tsx Refactoring (Jan 2026)
**Before**: 1,509 lines with mixed presentation and utility functions
**After**: 718 lines - reduced by 52%

Extracted modules:
- `lib/preflight-constants.ts` (225 lines) - Style constants, guidance text, coverage configuration
- `lib/preflight-utils.ts` (277 lines) - Pure utility functions (formatDuration, getServiceURL, etc.)
- `components/preflight/` directory with focused components:
  - `PreflightStepHeader.tsx` (32 lines) - Step header with status badge
  - `PreflightCheckList.tsx` (55 lines) - Collapsible check list
  - `CoverageBadge.tsx` (19 lines) - Coverage status badge
  - `CoverageMap.tsx` (39 lines) - Coverage comparison visualization
  - `ValidationIssuesPanel.tsx` (120 lines) - Validation error details
  - `MissingSecretsForm.tsx` (67 lines) - Secrets input form
  - `RuntimeInfoPanel.tsx` (85 lines) - Runtime identity display
  - `ServicesReadinessGrid.tsx` (246 lines) - Service health grid with peek
  - `DiagnosticsPanels.tsx` (153 lines) - Logs, fingerprints, ports

---

## Remaining Work

### Further UI Integration
**Current**: Hooks are defined but GeneratorForm.tsx hasn't been updated to use them
**Recommendation**: Update GeneratorForm.tsx to import and use extracted hooks:
```typescript
import { useGeneratorDraft, usePreflightSession, useSigningConfig } from '../hooks';
```

---

## Observations

- **Signing subsystem** (`api/signing/`) is well-architected with clean platform abstractions
- **Runtime package** (`runtime/`) is modular with clear domain boundaries (health, secrets, ports)
- **Deployment domain** (`ui/src/domain/deployment.ts`) exemplifies good seam design - pure functions, no side effects
- **API client** (`lib/api.ts`) provides comprehensive coverage but could benefit from domain-based splitting
- **Bundle packager** now follows single-responsibility principle with focused modules

## Architecture Principles Applied

1. **Screaming Architecture**: File names now clearly express their domain purpose
   - `build_compiler.go` screams "compilation"
   - `platform.go` screams "platform handling"
   - `usePreflightSession.ts` screams "preflight session management"

2. **Boundary of Responsibility**: Each module has a single owner
   - `file_ops.go` owns file operations
   - `preflight_store.go` owns session/job lifecycle
   - `useGeneratorDraft.ts` owns draft persistence

3. **Testing Seams**: Interfaces enable substitution
   - `PreflightSessionStore` can be mocked for session tests
   - `bundlePackager` dependencies can be stubbed
   - Pure functions in `domain/` are directly testable

---

## Recent Seam Discovery & Documentation (Jan 2026)

### Pipeline Architecture Seams
Documented the comprehensive seam architecture in the `pipeline/` package:
- **Orchestrator interface** (§6): Coordinates multi-stage pipeline execution
- **Stage interface** (§7): Abstracts individual pipeline stages for independent testing
- **Store interface** (§8): Persistence abstraction with memory and file-backed options
- **Supporting seams** (§9): CancelManager, IDGenerator, Logger, TimeProvider, WebhookNotifier, ManifestGenerator

### UI State Management Seams
Documented new Zustand-based state management:
- **Pipeline Store** (§15): Unified store for pipeline execution, polling, and stage results
- **Pipeline Utils** (§16): Pure functions for status mapping
- **Scenario State Hook** (§17): Server-side persistence with conflict detection

### Key Findings
1. **API layer** has excellent seam architecture with well-defined interfaces for all major components
2. **Pipeline package** follows the Stage/Orchestrator pattern with clear boundaries
3. **UI layer** has been evolving toward stronger seams with domain extraction and browser API abstraction
4. **Pure function extraction** (domain layer) enables comprehensive unit testing without UI setup

### No Weak Seams Identified
The scenario has mature seam architecture. All identified seams are:
- Well-defined with clear interfaces
- Properly documented
- Used consistently throughout the codebase
- Testable via substitution or pure function testing

---

## Boundary-of-Responsibility Enforcement (Jan 2026)

### Build Domain Extraction
**Goal**: Move build progress calculation and status mapping from presentation to domain layer

**Changes**:
1. Created `domain/build.ts` with pure functions for build progress and status transformation
2. Refactored `BuildStatus.tsx` to use domain functions instead of inline business logic
3. Moved `_extractStageResults()` logic from `pipelineStore.ts` to `domain/build.ts`
4. Consolidated `createErrorInfo()` helper from store to `lib/error-utils.ts`

**New Domain Functions** (`domain/build.ts`):
```typescript
// Build stage definitions (domain knowledge)
export const BUILD_STAGES: BuildStageDefinition[]
export function isStageReached(logs: string[], stage: BuildStageDefinition): boolean
export function calculateBuildProgress(status: BuildStatusType | null | undefined): number
export function getBuildStageStatuses(logs: string[], currentProgress: number): BuildStageStatus[]

// Pipeline status transformation
export function extractStageResults(status: VerbosePipelineStatus): ExtractedStageResults
export function pipelineStatusToBuildStatus(pipelineStatus: VerbosePipelineStatus | null): BuildStatusType | null
```

**New Error Utilities** (`lib/error-utils.ts`):
```typescript
export interface ErrorInfo { message, code?, canRetry, requiresInputFix, recoveryHint? }
export function createErrorInfo(err: unknown): ErrorInfo
```

**Boundary Improvements**:
- **BuildStatus.tsx**: Reduced from 233 to ~170 lines; no longer contains log parsing heuristics or status mapping logic
- **pipelineStore.ts**: Store now delegates data transformation to domain layer; removed duplicate `PipelineErrorInfo` interface in favor of shared `ErrorInfo` type
- **Domain layer**: `domain/build.ts` is pure, testable, and owns all build-related domain knowledge

**Files Changed**:
- `domain/build.ts` - New file with build progress and pipeline transformation logic
- `domain/index.ts` - Added build module export
- `components/BuildStatus.tsx` - Refactored to use domain functions
- `store/pipelineStore.ts` - Uses `extractStageResults()` from domain; uses `createErrorInfo()` from error-utils
- `lib/error-utils.ts` - Added `ErrorInfo` interface and `createErrorInfo()` function

**Architecture Improvements**:
- **Responsibility Separation**: Build stage definitions and progress heuristics now live in domain layer
- **Testability**: `calculateBuildProgress()` and `extractStageResults()` can be unit tested without React/store setup
- **Single Source of Truth**: No duplicate stage definitions across components
- **Reduced Coupling**: Store no longer depends on ApiError class methods directly

---

## Browser Seam Enforcement (Jan 2026)

### Clipboard Seam Unification
**Goal**: Ensure all clipboard operations flow through the `writeToClipboard()` seam in `lib/browser.ts`

**Problem Identified**: The `writeToClipboard()` seam existed but was bypassed in 8+ components that called `navigator.clipboard.writeText()` directly.

**Files Updated**:
- `components/layout/DebugJsonModal.tsx` - Now uses `writeToClipboard()`
- `components/layout/SidebarHeader.tsx` - Now uses `writeToClipboard()`
- `components/docs/DocsPanel.tsx` - Now uses `writeToClipboard()`
- `components/preflight/DiagnosticsPanels.tsx` - Now uses `writeToClipboard()`
- `components/BundledPreflightSection.tsx` - Now uses `writeToClipboard()`
- `components/BundledRuntimeSection.tsx` - Now uses `writeToClipboard()`
- `components/scenario-inventory/GenerateDesktopButton.tsx` - Now uses `writeToClipboard()`

**Benefits**:
- **Centralized Error Handling**: All clipboard operations now use result-based error handling instead of try-catch
- **Testability**: All components can now be tested by mocking the browser seam
- **Consistency**: Single pattern for clipboard operations across the codebase
- **Maintainability**: If clipboard API changes, only one file needs updating

### Blob Download Seam Unification
**Goal**: Ensure all blob download operations flow through the `triggerBlobDownload()` seam in `lib/browser.ts`

**Problem Identified**: Two components reimplemented the blob download pattern (`URL.createObjectURL` + DOM manipulation) instead of using the existing seam.

**Files Updated**:
- `components/BundledPreflightSection.tsx` - Now uses `triggerBlobDownload()`
- `components/BundledRuntimeSection.tsx` - Now uses `triggerBlobDownload()`

**Code Eliminated**:
```typescript
// Before (repeated in multiple components):
const url = URL.createObjectURL(blob);
const link = document.createElement("a");
link.href = url;
link.download = filename;
link.click();
URL.revokeObjectURL(url);

// After:
triggerBlobDownload(blob, filename);
```

**Benefits**:
- **No Memory Leaks**: Centralized cleanup of `URL.revokeObjectURL()`
- **Reduced Code Duplication**: 6 lines → 1 line per call site
- **Testability**: Can mock download behavior in tests

---

## Remaining Seam Opportunities

### API Layer - High Priority

#### 1. CommandRunner Interface for Compiler
**Location**: `api/bundle/compiler.go`
**Current State**: Direct `exec.Command()` calls in `compileGoBinary()`, `compileRustBinary()`, `compileNpmBinary()`, `compileCustomBinary()`
**Recommendation**: Create `CommandRunner` interface:
```go
type CommandRunner interface {
    Run(ctx context.Context, cmd string, args []string, opts *RunOptions) (*RunResult, error)
}

type RunOptions struct {
    Dir string
    Env map[string]string
}

type RunResult struct {
    Output   string
    ExitCode int
}
```
**Benefit**: Enables testing compiler logic without invoking actual Go/Rust/npm toolchains

#### 2. Filesystem Abstraction in Compiler
**Location**: `api/bundle/compiler.go` lines 85, 280, 355, 356
**Current State**: Direct `os.MkdirAll()` calls scattered in compile functions
**Recommendation**: Extend `FileOperations` interface to include `MkdirAll()`
**Benefit**: Test directory creation failures without filesystem side effects

#### 3. System Command Executor for Wine Service
**Location**: `api/system/wine_service.go`
**Current State**: Extensive direct `exec.Command()` calls for Wine installation
**Recommendation**: Create dedicated `WineInstaller` interface wrapping installation logic
**Benefit**: Test Wine installation flows without network access or system modifications

### API Layer - Medium Priority

#### 4. Consistent TimeProvider Usage
**Problem**: Pipeline uses `TimeProvider` interface, but build service uses direct `time.Now()`
**Files**: `api/build/service.go` vs `api/pipeline/orchestrator.go`
**Recommendation**: Inject `TimeProvider` into build service for deterministic testing

#### 5. PathProvider for Vrooli Root Detection
**Problem**: Root detection logic duplicated in 3+ locations
**Files**: `api/shared/path/vrooli.go`, `api/generation/analyzer.go`, `api/pipeline/orchestrator.go`
**Recommendation**: Create singleton `PathProvider` used everywhere

#### 6. EnvironmentReader Consistency
**Problem**: `EnvironmentReader` interface exists in distribution package but `os.Getenv()` used directly elsewhere
**Recommendation**: Inject `EnvironmentReader` consistently across all packages

### UI Layer - Medium Priority

#### 1. Confirmation Dialog Seam
**Current State**: Direct `window.confirm()` calls in 2 components
**Files**: `TelemetryUploadCard.tsx`, `RecordsManager.tsx`
**Recommendation**: Create `useConfirmDialog()` hook or `confirmDialog()` function
**Benefit**: Testable confirmation flows; can replace with custom modal

#### 2. Portal Root Configuration
**Current State**: `createPortal()` always uses `document.body` (13+ locations)
**Recommendation**: Create `<PortalProvider>` or centralized portal root config
**Benefit**: Easier testing; isolated portal management

#### 3. Navigation/URL Seam
**Current State**: Direct `window.location` and `window.history` manipulation
**File**: `components/signing/SigningPage.tsx`
**Recommendation**: Use router hooks or create navigation seam
**Benefit**: Testable URL manipulation; cleaner separation

---

## Idempotency & Replay Safety Seams (Jan 2026)

### Design Philosophy
The scenario-to-desktop pipeline implements idempotency at multiple layers to ensure "running twice is no worse than running once". This is critical for:
- Network timeout scenarios where clients retry requests
- UI double-click protection
- Safe recovery from partial failures
- Predictable behavior under repeated execution

### API Layer Idempotency

#### 1. Pipeline Orchestrator Idempotency
**Location**: `api/pipeline/orchestrator.go:161-173`
**Mechanism**: Client-provided idempotency keys
```go
if config.IdempotencyKey != "" {
    if existing, ok := o.store.GetByIdempotencyKey(config.IdempotencyKey); ok {
        // Return existing pipeline instead of creating new one
        return existing, nil
    }
}
```
**Behavior**: If a pipeline with the same idempotency key exists (running or completed), returns the existing status instead of starting a new pipeline.
**Test Coverage**: `[REQ:IDEM-001]` in `api/pipeline/orchestrator_test.go` (lines 877-1204)

#### 2. Pipeline Store Idempotency Key Lookup
**Location**: `api/pipeline/store.go`
**Interface Method**: `GetByIdempotencyKey(key string) (*Status, bool)`
**Purpose**: O(n) scan of pipeline statuses to find matching idempotency key
**Implementations**: `InMemoryStore`, `FileStore`

#### 3. Status IdempotencyKey Field
**Location**: `api/pipeline/types.go:107-112, 170-172`
```go
type Config struct {
    // IdempotencyKey is an optional client-provided key for request deduplication.
    // If a pipeline with the same idempotency key already exists and is running or completed,
    // the existing pipeline status will be returned instead of starting a new pipeline.
    IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type Status struct {
    // IdempotencyKey is the client-provided key for request deduplication.
    IdempotencyKey string `json:"idempotency_key,omitempty"`
}
```

### UI Layer Idempotency

#### 1. Pipeline Store Double-Submission Guard
**Location**: `ui/src/store/pipelineStore.ts`
**Mechanism**: `isSubmitting` flag + idempotency key generation
```typescript
interface PipelineStoreState {
    isSubmitting: boolean;
    currentIdempotencyKey: string | null;
}

// In runStage():
if (isSubmitting) {
    const existingId = get().pipelineId;
    if (existingId) return existingId; // Idempotent return
    throw new Error("A pipeline request is already in progress");
}
```
**Behavior**: Prevents double-clicks from creating duplicate requests; returns existing pipeline ID if already submitting.

#### 2. Idempotency Key Generation
**Location**: `ui/src/lib/pipeline-utils.ts`
```typescript
export function generateIdempotencyKey(scenarioName: string, stage?: string, sessionId?: string): string
export function generateUniqueIdempotencyKey(scenarioName: string, stage?: string): string
export function getSessionId(): string
export function resetSessionId(): void
```
**Design**:
- Session-scoped: Keys are stable within a page session but unique across sessions
- Stage-aware: Different stages get different keys
- Resettable: `resetSessionId()` allows explicit retries with fresh keys

#### 3. Reset for Retry Action
**Location**: `ui/src/store/pipelineStore.ts`
```typescript
resetForRetry: () => {
    resetSessionId(); // Get fresh idempotency keys
    set({ isSubmitting: false, currentIdempotencyKey: null, error: null, errorInfo: null });
}
```
**Purpose**: Allows explicit user-initiated retries to bypass idempotency deduplication.

#### 4. Selectors for UI Guards
**Location**: `ui/src/store/pipelineStore.ts`
```typescript
export const selectIsSubmitting = (state: PipelineStore) => state.isSubmitting;
export const selectIsBusy = (state: PipelineStore) =>
    state.isSubmitting || state.runStatus === "running" || state.runStatus === "starting";
```
**Usage**: Components use these selectors to disable buttons during submission/execution.

### Architectural Considerations

#### Remaining Idempotency Gaps (for future work)
1. **Stage-level idempotency**: Currently only pipeline-level; stages re-execute on resume
2. **Distribution uploads**: S3 uploads don't check if object already exists
3. **File operations**: Bundle stage overwrites files without checking if identical
4. **Manifest generation**: Always regenerates even if unchanged

#### Future Enhancements
- Add stage-level execution fingerprinting
- Implement artifact hash validation before re-execution
- Add S3 object existence checks before upload
- Cache manifest generation with content hash validation

---

## Desktop App Template Seams (Feb 2026)

### Splash Window Architecture

The splash window in the Electron template has been refactored to follow proper seam architecture and responsibility boundaries.

#### 1. Splash Window Manager Seam (`SplashWindowManager`)
**Location**: `templates/vanilla/splash/manager.ts`
**Purpose**: Manages splash window lifecycle with proper separation of concerns

**Interface**:
```typescript
interface ISplashWindowManager {
    create(): Promise<boolean>;
    updateStatus(status: SplashStatus): void;
    close(): Promise<SplashCloseResult>;
    isVisible(): boolean;
    onEscapePressed(callback: () => void): void;
}
```

**Responsibilities**:
- Window creation and destruction
- Status communication via IPC
- Escape key handling for emergency exit

**NOT responsible for**:
- Application startup logic
- Error dialog display
- Main window management

**Testing Seams**:
- `IWindowFactory`: Mock Electron BrowserWindow creation
- `IPathResolver`: Mock path resolution for different environments
- `IIpcMain`: Mock IPC operations

**Status**: ✅ Implemented

#### 2. Server Readiness Seam (`checkServerReadiness`)
**Location**: `templates/vanilla/splash/server-readiness.ts`
**Purpose**: Validates server readiness with proper status code checking

**Interface**:
```typescript
interface IHttpClient {
    get(url: string, timeoutMs: number): Promise<{ statusCode: number; body?: string }>;
}

interface ITimer {
    now(): number;
    sleep(ms: number): Promise<void>;
}

function checkServerReadiness(
    httpClient: IHttpClient,
    config: ReadinessConfig,
    timer?: ITimer,
    onProgress?: ReadinessProgressCallback
): Promise<ReadinessResult>;
```

**Key Design Decisions**:
- **Only accepts 2xx responses**: Fixes bug where 404 was considered "ready"
- **Configurable acceptable status codes**: Can customize per deployment
- **Optional content validation**: For extra validation beyond status code
- **Progress reporting**: For UI feedback during long waits

**Testing Seams**:
- `IHttpClient`: Mock HTTP requests without network
- `ITimer`: Mock timing for deterministic tests

**Status**: ✅ Implemented

#### 3. Splash IPC Channel Seam
**Location**: `templates/vanilla/splash/types.ts`, `templates/vanilla/splash-preload.ts`
**Purpose**: Secure communication between main process and splash window

**Channels**:
```typescript
const SPLASH_IPC_CHANNELS = {
    STATUS_UPDATE: "splash:status-update",  // Main → Splash
    ESCAPE_PRESSED: "splash:escape-pressed", // Splash → Main
    READY: "splash:ready",                   // Splash → Main
};
```

**API exposed to splash renderer**:
```typescript
interface SplashAPI {
    onStatusUpdate(callback: (status: SplashStatus) => void): void;
    offStatusUpdate(callback: (status: SplashStatus) => void): void;
    notifyEscape(): void;
    notifyReady(): void;
}
```

**Status**: ✅ Implemented

### Bug Fixes Applied

#### 1. Focus Trapping Fix
**Problem**: `alwaysOnTop: true` caused splash window to trap focus, making error dialogs inaccessible
**Solution**: Changed default to `alwaysOnTop: false`
**Files**: `templates/vanilla/splash/types.ts` (DEFAULT_SPLASH_CONFIG)

#### 2. Timer-Based Messages Fix
**Problem**: Splash showed "Loading complete!" based on timer, not actual app state
**Solution**: IPC-based status updates from main process
**Files**: `templates/vanilla/splash.html`, `templates/vanilla/splash-preload.ts`

#### 3. Server Readiness 404 Bug Fix
**Problem**: 404 responses were treated as "server ready"
**Solution**: Only accept 2xx status codes as ready
**Files**: `templates/vanilla/splash/server-readiness.ts`

#### 4. Error Dialog Z-Order Fix
**Problem**: Error dialog appeared behind splash window
**Solution**: Use `destroy()` instead of `close()` with delay before showing dialog
**Files**: `templates/vanilla/splash/manager.ts`, `templates/vanilla/main.ts`

### Runtime Exit Monitoring Seam (Feb 2026)

#### 4. Runtime Exit Tracking
**Location**: `templates/vanilla/main.ts` (lines 100-133)
**Purpose**: Module-level tracking for runtime process exit status, enabling smoke tests to detect runtime crashes after server ready check.

**Interface**:
```typescript
interface RuntimeExitInfo {
    exited: boolean;
    code: number | null;
    signal: NodeJS.Signals | null;
    stderr: string;
    exitedAt: Date | null;
}

function resetRuntimeExitTracking(): void;
function hasRuntimeExitedUnexpectedly(): boolean;
```

**Key Design Decisions**:
- **Module-level state**: Exit info persists across async boundaries
- **Stderr capture**: Always captures stderr for error diagnostics
- **Post-success stability check**: 2-second delay after server ready to catch delayed crashes

**Status**: ✅ Implemented

### Enhanced Splash Error Display (Feb 2026)

#### 5. Splash Error Display with Log Panel
**Location**: `templates/vanilla/splash/manager.ts`, `templates/vanilla/splash.html`, `templates/vanilla/splash-preload.ts`
**Purpose**: Display detailed error information in splash window instead of closing and showing dialog

**Extended SplashAPI Interface**:
```typescript
interface SplashAPI {
    // ... existing methods ...
    onLogAppend(callback: (entry: SplashLogEntry) => void): void;
    onLogClear(callback: () => void): void;
    copyLogs(): void;
    onCopyLogsResult(callback: (result: CopyLogsResult) => void): void;
    retry(): void;
}
```

**Extended ISplashWindowManager Interface**:
```typescript
interface ISplashWindowManager {
    // ... existing methods ...
    appendLog(entry: SplashLogEntry): void;
    clearLogs(): void;
    onCopyLogs(callback: () => string): void;
    onRetry(callback: () => void): void;
}
```

**Extended SplashStatus Error Details**:
```typescript
interface SplashStatus {
    error?: {
        title: string;
        message: string;
        recoverable: boolean;
        logs?: string[];        // Diagnostic logs
        stderr?: string;        // Runtime stderr output
        exitCode?: number | null; // Process exit code
        suggestion?: string;    // User-friendly recovery hint
    };
}
```

**IPC Channels Added**:
```typescript
SPLASH_IPC_CHANNELS = {
    // ... existing channels ...
    LOG_APPEND: "splash:log-append",
    LOG_CLEAR: "splash:log-clear",
    COPY_LOGS: "splash:copy-logs",
    COPY_LOGS_RESULT: "splash:copy-logs-result",
    RETRY: "splash:retry",
};
```

**Status**: ✅ Implemented

#### 6. Clipboard Seam for Splash Manager
**Location**: `templates/vanilla/splash/manager.ts`
**Purpose**: Abstract clipboard operations for testability

**Interface**:
```typescript
interface IClipboard {
    writeText(text: string): void;
}
```

**Usage**: Injected into `SplashWindowManager` via `SplashManagerDeps.clipboard`

**Status**: ✅ Implemented

### Test Coverage

Test files:
- `templates/vanilla/splash/__tests__/server-readiness.test.ts` - Server readiness checking
- `templates/vanilla/splash/__tests__/manager.test.ts` - Splash window manager

Key test scenarios:
- ✅ Splash window creation with correct options
- ✅ Status updates via IPC
- ✅ Window close with proper cleanup
- ✅ Escape key handling
- ✅ Server readiness rejects 404 responses
- ✅ Server readiness accepts 2xx responses
- ✅ Timeout handling with progress reporting
- ✅ Content validation support
- ✅ Log append/clear via IPC (Feb 2026)
- ✅ Copy logs with clipboard integration (Feb 2026)
- ✅ Retry callback handling (Feb 2026)
- ✅ Error status with full details (Feb 2026)

---

## Port Environment Seam (Feb 2026)

### Overview

The Port Environment Seam ensures that all port environment variables from `service.json` are properly injected into bundled Go runtimes at startup. This solves the critical issue where Go binaries calling `requireEnv("API_PORT")` would exit with code 1 because the environment variable was never set.

### Data Flow

```
service.json
    ↓ ports.api.env_var = "API_PORT", ports.api.port = 18700
    ↓ ports.ui.env_var = "UI_PORT", ports.ui.port = 36400

analyzer.go extracts ALL ports dynamically
    ↓ metadata.Ports["api"] = {EnvVar: "API_PORT", Port: 18700}
    ↓ metadata.Ports["ui"] = {EnvVar: "UI_PORT", Port: 36400}

DesktopConfig.Ports populated
    ↓ config.Ports = metadata.Ports

template-generator.ts
    ↓ PORTS_CONFIG = {"api":{"envVar":"API_PORT","port":18700},...}

main.ts at startup
    ↓ const PORTS = {"api":{"envVar":"API_PORT","port":18700},...}

main.ts at spawn (startBundledRuntime)
    ↓ runtimeEnv["API_PORT"] = "18700"
    ↓ runtimeEnv["UI_PORT"] = "36400"
    ↓ runtimeEnv["VROOLI_LIFECYCLE_MANAGED"] = "true"
    ↓ runtimeEnv["VROOLI_DESKTOP_MODE"] = "true"

Go binary receives ALL env vars
    ↓ requireEnv("API_PORT") ✓ returns "18700"
```

### Components

#### 1. ScenarioMetadata.Ports
**Location**: [CODE: api/generation/types.go#PortConfig]
**Purpose**: Stores extracted port configurations from service.json
**Structure**:
```go
type PortConfig struct {
    EnvVar      string `json:"env_var"`      // e.g., "API_PORT"
    Port        int    `json:"port"`         // Default port number
    Description string `json:"description"`  // Human-readable description
}

type ScenarioMetadata struct {
    // ... other fields ...
    Ports map[string]PortConfig `json:"ports,omitempty"`
}
```

#### 2. Dynamic Port Extraction
**Location**: [CODE: api/generation/analyzer.go#readServiceJSON]
**Purpose**: Extracts ALL ports from service.json, not just hardcoded api/ui
**Behavior**:
- Iterates over all keys in `service.json.ports`
- Skips ports without `env_var` defined
- Extracts port number from `port` field or first value in `range`

#### 3. DesktopConfig.Ports
**Location**: [CODE: api/generation/types.go#DesktopConfig]
**Purpose**: Passes port configuration to template generator
**Flow**: `ScenarioMetadata.Ports` → `DesktopConfig.Ports` → template variables

#### 4. PORTS_CONFIG Template Variable
**Location**: [CODE: templates/build-tools/template-generator.ts#buildPortsConfig]
**Purpose**: Serializes port config for injection into main.ts
**Format**: JSON object with camelCase keys for TypeScript compatibility

#### 5. PORTS Constant in main.ts
**Location**: [CODE: templates/vanilla/main.ts:77]
**Purpose**: Holds port configuration at runtime
**Usage**: Iterated at spawn time to set environment variables

#### 6. Environment Injection at Spawn
**Location**: [CODE: templates/vanilla/main.ts:2293]
**Purpose**: Sets all port env vars before spawning bundled runtime
**Implementation**:
```typescript
for (const [portKey, portConfig] of Object.entries(PORTS)) {
    if (portConfig.envVar && portConfig.port) {
        runtimeEnv[portConfig.envVar] = String(portConfig.port);
    }
}
```

### Lifecycle Environment Variables

In addition to port env vars, the seam sets these lifecycle markers:

| Variable | Value | Purpose |
|----------|-------|---------|
| `VROOLI_LIFECYCLE_MANAGED` | `"true"` | Signals Go code it's running under Vrooli lifecycle management |
| `VROOLI_DESKTOP_MODE` | `"true"` | Signals Go code it's running in desktop/bundled mode |
| `VROOLI_API_SKIP_STALE_CHECK` | `"true"` | Skips go.mod staleness checks (bundled binaries can't rebuild) |

### Bundle Schema Extension

The bundle.json schema also supports `env_var` for future runtime auto-injection:

**Locations**:
- [CODE: runtime/manifest/manifest.go#PortRequest] - Runtime schema
- [CODE: ../deployment-manager/api/bundles/manifest.go#RequestedPort] - Bundle generation schema

```go
type PortRequest struct {
    Name           string    `json:"name"`
    EnvVar         string    `json:"env_var,omitempty"` // e.g., "API_PORT"
    Range          PortRange `json:"range"`
    RequiresSocket bool      `json:"requires_socket,omitempty"`
}
```

### Testing

To verify the seam is working:

1. **Build a desktop app**: `cd scenarios/git-control-tower && make desktop-build`
2. **Launch and check logs**: Look for `[Desktop App] Setting API_PORT=18700`
3. **Verify app starts**: Should not exit with code 1 due to missing env vars
4. **Check telemetry**: `tail ~/.config/git-control-tower-desktop/deployment-telemetry.jsonl`
   - Should see `"event":"app_ready"`, not `"event":"bundled_runtime_exit"` with code 1

### Breaking Change Notice

This is a breaking change by design. Desktop apps built with the old system (hardcoded APIPort/UIPort, no env var injection) must be rebuilt with the new pipeline. The old approach was fundamentally broken and could not work.

---

## Architecture Principles Applied

1. **Seam-at-the-Boundary**: Browser APIs (clipboard, downloads, file reading) now have explicit seams
2. **No Bypassing**: Components consistently use seams instead of direct browser calls
3. **Result-Based Error Handling**: Seam functions return result objects rather than throwing
4. **Mock-Friendly**: All seams designed with testing in mind
5. **Incremental Improvement**: Each seam enforcement iteration makes the codebase more testable
6. **Idempotency by Design**: Pipeline operations are safe to retry without creating duplicates
7. **Event-Driven Communication**: Splash window uses IPC for status updates, not timers
8. **Post-Success Validation**: Smoke tests now include stability delay to catch delayed crashes (Feb 2026)
9. **User-Visible Error Diagnostics**: Errors shown in splash with logs, stderr, and exit codes (Feb 2026)
10. **Graceful Degradation**: Error UI allows retry for recoverable errors, quit via ESC for all errors (Feb 2026)
