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
**Purpose**: Pure functions for form validation and config building
**Functions**:
```typescript
export function validateGeneratorInputs(options: ValidateGeneratorInputsOptions): string | null
export function buildDesktopConfig(options: BuildDesktopConfigOptions): DesktopConfig
export function resolveEndpoints(input: EndpointResolutionInput): EndpointResolution
export function getSelectedPlatforms(platforms: PlatformSelection): string[]
```
**Status**: ✅ Implemented

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
| `validateGeneratorInputs()` | Pure function, direct unit tests |
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
