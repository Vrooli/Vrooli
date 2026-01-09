# Testing Seams Documentation

This document describes the architectural seams in the browser-automation-studio UI codebase. Seams are boundaries where behavior can be substituted for testing, enabling isolation of components and easy verification of behavior.

## Export Domain (`src/domains/executions/export/`)

The export domain has been refactored to follow screaming architecture principles with clear testing seams.

### Directory Structure

```
export/
├── index.ts                    # Public API - re-exports all modules
├── config/                     # Types, constants, presentation config (single source of truth)
│   ├── index.ts
│   ├── types.ts                # Canonical type definitions
│   ├── constants.ts            # Format options, dimension presets, status config
│   └── constants.test.ts       # 42 unit tests
├── context/                    # React context for ExportDialog state
│   ├── index.ts
│   └── ExportDialogContext.tsx # Provider, hooks, builder
├── components/                 # React UI components
│   ├── index.ts
│   └── ExportDialog.tsx        # Context-based export dialog
├── transformations/            # Pure functions (no side effects)
│   ├── index.ts
│   ├── scaleDimensions.ts      # Dimension math utilities
│   ├── scaleDimensions.test.ts # 20 unit tests
│   ├── scaleMovieSpec.ts       # Movie spec transformations
│   └── scaleMovieSpec.test.ts  # 17 unit tests
├── api/                        # API client with injection seam
│   ├── index.ts
│   ├── exportClient.ts         # Interface + default impl
│   └── executeExport.ts        # Export execution utilities
└── hooks/                      # React hooks
    ├── index.ts
    ├── useExportDialogState.ts # Form state management
    └── useRecordedVideoStatus.ts # Video availability check
```

### Seam 0: Configuration Module (Single Source of Truth)

**Location:** `config/` for types and constants, `@/domains/exports/presentation/` for format/status display config

**Purpose:** Export configuration is split into two concerns:
1. **Types and dialog constants** (`config/types.ts`, `config/constants.ts`): Export format types, dimension presets, format options for the dialog
2. **Display configuration** (`@/domains/exports/presentation/`): Format and status styling used by `ExportsTab.tsx`

The `config/index.ts` re-exports from both locations, providing a unified API.

**Key exports:**
```typescript
import {
  // Types (from config/types.ts)
  ExportFormat,
  ExportRenderSource,
  ExportDimensionPreset,
  ExportDimensions,
  ExportFormatOption,
  ExportStatus,
  ExportFormatDisplayConfig,
  ExportStatusConfig,

  // Constants (from config/constants.ts)
  EXPORT_EXTENSIONS,
  DEFAULT_EXPORT_DIMENSIONS,
  DIMENSION_PRESET_CONFIG,
  EXPORT_FORMAT_OPTIONS,
  EXPORT_RENDER_SOURCE_OPTIONS,

  // Format/status display config (from @/domains/exports/presentation/)
  FORMAT_CONFIG,
  EXPORT_STATUS_CONFIG,

  // Helper functions
  isBinaryFormat,
  getFormatConfig,
  getExportStatusConfig,
  buildExportFileName,
  sanitizeFileStem,
  generateDefaultFileStem,
  buildDimensionPresetOptions,
} from '@/domains/executions/export';
```

**How to test:**
```typescript
// Direct unit testing - no mocks needed
expect(isBinaryFormat('mp4')).toBe(true);
expect(getFormatConfig('gif').label).toBe('Animated GIF');
expect(sanitizeFileStem('my file!', 'default')).toBe('my-file-');
```

**59 unit tests** verify all helper functions and configuration values (42 in config + 17 in presentation).

### Seam 0.5: ExportDialog Context

**Location:** `context/`

**Purpose:** Eliminates prop drilling in ExportDialog by providing state via React context. The context groups state into logical domains:
- `formatState` - Export format selection
- `dimensionState` - Output dimensions
- `fileState` - File naming and destination
- `renderSourceState` - Video render source selection
- `previewState` - Preview rendering state
- `progressState` - Export progress and loading
- `metricsState` - Frame count, duration, etc.
- `actions` - Dialog open/close/confirm

**Provider:**
```typescript
import {
  ExportDialogProvider,
  buildExportDialogContextValue,
} from '@/domains/executions/export';

// Build context value from hook results
const contextValue = buildExportDialogContextValue({
  dialogTitleId: 'export-title',
  dialogDescriptionId: 'export-desc',
  exportFormat: 'mp4',
  setExportFormat: (f) => setFormat(f),
  // ... other props from useExecutionExport
});

// Wrap ExportDialog in provider
<ExportDialogProvider value={contextValue}>
  <ExportDialog isOpen={isOpen} />
</ExportDialogProvider>
```

**Granular hooks for testing:**
```typescript
import {
  useExportFormatState,
  useExportDimensionState,
  useExportFileState,
  useExportPreviewState,
  useExportProgressState,
  useExportMetricsState,
  useExportDialogActions,
} from '@/domains/executions/export';

// Each hook returns only the state it needs
const { format, setFormat, isBinaryExport } = useExportFormatState();
const { isExporting, statusMessage } = useExportProgressState();
```

**Benefits:**
- ExportDialog props reduced from 55+ to just 1 (`isOpen`)
- Sub-components can access only the state they need
- Easy to test individual sections by mocking context
- Clear boundaries between state domains

### Seam 1: Pure Transformations

**Location:** `transformations/`

**Purpose:** All dimension scaling and movie spec transformation logic is extracted into pure functions with no side effects.

**How to test:**
```typescript
import { scaleMovieSpec, resolveDimensionPreset } from '@/domains/executions/export';

// Direct unit testing - no mocks needed
const result = scaleMovieSpec(spec, { targetDimensions: { width: 1920, height: 1080 } });
expect(result.presentation.canvas).toEqual({ width: 1920, height: 1080 });
```

**Key functions:**
- `calculateScaleFactors(source, target)` - Calculate X/Y scale ratios
- `scaleDimensions(dims, factors, fallback)` - Scale width/height
- `scaleFrameRect(rect, factors, fallback)` - Scale position + dimensions
- `resolveDimensionPreset(preset, spec, custom)` - Resolve preset to dimensions
- `scaleMovieSpec(spec, options)` - Transform entire movie spec
- `extractCanvasDimensions(presentation)` - Extract dimensions from presentation

### Seam 2: Export API Client

**Location:** `api/exportClient.ts`

**Purpose:** All API calls are behind an interface that can be replaced for testing.

**Interface:**
```typescript
export interface ExportApiClient {
  fetchRecordedVideoStatus(executionId: string, signal?: AbortSignal): Promise<RecordedVideoStatus>;
  executeExport(executionId: string, payload: ExportRequestPayload, signal?: AbortSignal): Promise<ExportResult>;
}
```

**How to test:**
```typescript
import { createMockExportApiClient } from '@/domains/executions/export';

// Create a mock client
const mockClient = createMockExportApiClient({
  fetchRecordedVideoStatus: async () => ({
    available: true,
    count: 2,
    videos: [{ id: 'v1' }, { id: 'v2' }],
  }),
});

// Inject into hook
const { available, count } = useRecordedVideoStatus({
  executionId: 'test-123',
  apiClient: mockClient,
});
```

### Seam 3: Recorded Video Status Hook

**Location:** `hooks/useRecordedVideoStatus.ts`

**Purpose:** Isolates the concern of checking recorded video availability.

**Injection point:** `apiClient` parameter

**How to test:**
```typescript
import { useRecordedVideoStatus, createMockExportApiClient } from '@/domains/executions/export';

// In tests, inject a mock client
renderHook(() => useRecordedVideoStatus({
  executionId: 'exec-123',
  apiClient: createMockExportApiClient({
    fetchRecordedVideoStatus: async () => ({ available: true, count: 1, videos: [] }),
  }),
}));
```

### Seam 4: Export Execution Utilities

**Location:** `api/executeExport.ts`

**Purpose:** Isolated utilities for export request building and file handling.

**Pure functions (no mocks needed):**
```typescript
import { buildExportRequest, buildExportOverrides, createJsonBlob } from '@/domains/executions/export';

// Test request building
const payload = buildExportRequest({
  format: 'mp4',
  fileName: 'test.mp4',
  renderSource: 'auto',
  movieSpec: null,
  overrides: buildExportOverrides({ chromeTheme: 'dark' }),
});

// Test blob creation
const blob = createJsonBlob(movieSpec);
expect(blob.type).toBe('application/json');
```

**Side-effect utilities (test via integration):**
- `executeBinaryExport()` - Makes API call
- `triggerBlobDownload()` - Manipulates DOM
- `saveWithNativeFilePicker()` - Uses File System Access API

## Boundaries of Responsibility

### config/
- **Owns:** Type definitions, constants, display configuration
- **Does NOT:** Perform side effects, make API calls, manage React state

### context/
- **Owns:** React context for ExportDialog state, provider/consumer pattern
- **Does NOT:** Perform business logic, make API calls, define types (uses config/)

### components/
- **Owns:** UI rendering, user interaction handling
- **Does NOT:** Manage orchestration state (uses context/), define types (uses config/)

### transformations/
- **Owns:** Mathematical transformations, dimension calculations
- **Does NOT:** Make API calls, manage state, handle DOM

### api/
- **Owns:** API communication, request/response handling
- **Does NOT:** Manage React state, perform transformations

### hooks/
- **Owns:** React state management, effect orchestration
- **Does NOT:** Perform complex transformations (delegates to transformations/), make direct API calls (uses api/)

## Testing Strategy

### Unit Tests (Pure Functions)
- All functions in `transformations/` have 100% test coverage
- Test files co-located: `*.test.ts` next to source
- Run with: `pnpm exec vitest run --project export-domain`

### Integration Tests (Hooks with Mocks)
- Use `createMockExportApiClient()` to mock API layer
- Test state transitions and effect behavior
- Verify correct delegation to transformation functions

### E2E Tests
- Test full export flow through UI
- Verify file download behavior
- Test with real API (or mocked at network level)

## Migration Notes

### Phase 1: Hook Extraction (Complete)

The `useExecutionExport` hook was refactored to use extracted modules:

**Before (1003 lines):**
- Inline dimension scaling logic (~100 lines)
- Inline movie spec transformation (~100 lines)
- Inline recorded video status check (~45 lines)
- Inline API calls (~50 lines)

**After (793 lines):**
- Uses `scaleMovieSpec()` from transformations/
- Uses `useRecordedVideoStatus()` hook
- Uses `buildExportRequest()`, `executeBinaryExport()`, etc. from api/
- Uses `resolveDimensionPreset()` for dimension calculations

**Line reduction:** ~210 lines (21% smaller)

### Phase 2: Configuration Consolidation (Complete)

Export configuration was consolidated with clear separation of concerns:

**Before:**
- `viewer/exportConfig.ts` - types, format options, dimension presets
- `exports/presentation/formatConfig.ts` - format display config (duplicate)
- `config/constants.ts` - FORMAT_DISPLAY_CONFIG (duplicate)
- `hooks/useExportDialogState.ts` - duplicate types, sanitize function
- `api/executeExport.ts` - duplicate format type

**After:**
- Types in `config/types.ts` (canonical type definitions)
- Dialog constants in `config/constants.ts` (format options, dimension presets, helpers)
- Format/status display config in `@/domains/exports/presentation/` (used by ExportsTab)
- `config/index.ts` re-exports from both locations for unified API
- Single `ExportFormat` type used everywhere
- 59 unit tests verify configuration (42 config + 17 presentation)

### Phase 3: Context-Based ExportDialog (Complete)

ExportDialog was refactored to use React context instead of prop drilling:

**Before:**
- 55+ props passed to ExportDialog
- Hard to test individual sections
- All state coupled together

**After:**
- ExportDialog accepts only `isOpen` prop
- State provided via ExportDialogContext
- 7 granular context hooks for isolated testing
- Sub-components access only what they need

**Migration path:**
```typescript
// Old approach (55+ props)
<ExportDialog
  isOpen={isOpen}
  onClose={onClose}
  exportFormat={format}
  setExportFormat={setFormat}
  // ... 50+ more props
/>

// New approach (context-based)
const contextValue = buildExportDialogContextValue({
  dialogTitleId: 'title-id',
  ...exportController.exportDialogProps,
  onClose: handleClose,
  onConfirm: handleConfirm,
});

<ExportDialogProvider value={contextValue}>
  <ExportDialog isOpen={isOpen} />
</ExportDialogProvider>
```

**Testability:** Significantly improved - core logic is now independently testable

### Phase 4: Deprecated Code Cleanup (Complete)

All consumers were migrated to use the context-based ExportDialog:

**Deleted files:**
- `viewer/ExportDialog.tsx` (581 lines) - Old prop-drilling implementation
- `viewer/exportConfig.ts` (158 lines) - Deprecated, replaced by `export/config/`

**Updated files:**
- `viewer/index.ts` - Removed re-export of deprecated exportConfig
- `InlineExecutionViewer.tsx` - Now uses ExportDialogProvider pattern
- `viewer/useReplayCustomization.tsx` - Now imports from `@/domains/executions/export`

**Line savings:** ~739 lines of deprecated code removed

**Migration example (InlineExecutionViewer):**
```typescript
// Before: Direct import of deprecated ExportDialog
import ExportDialog from './viewer/ExportDialog';

<ExportDialog
  isOpen={exportController.isExportDialogOpen}
  onClose={exportController.closeExportDialog}
  onConfirm={exportController.confirmExport}
  dialogTitleId="..."
  {...exportController.exportDialogProps}
/>

// After: Context-based approach
import {
  ExportDialog,
  ExportDialogProvider,
  buildExportDialogContextValue,
} from '@/domains/executions/export';

<ExportDialogProvider
  value={buildExportDialogContextValue({
    dialogTitleId: '...',
    onClose: exportController.closeExportDialog,
    onConfirm: exportController.confirmExport,
    ...exportController.exportDialogProps,
  })}
>
  <ExportDialog isOpen={exportController.isExportDialogOpen} />
</ExportDialogProvider>
```

---

## Exports Domain (`src/domains/exports/`)

The exports domain has been refactored to consolidate UI presentation utilities and provide testable seams for API and download operations.

### Directory Structure

```
exports/
├── index.ts                    # Public API - re-exports all modules
├── store.ts                    # Zustand store for export CRUD
├── api/                        # API clients with testable seams
│   ├── index.ts
│   ├── workflowClient.ts       # Workflow info fetching
│   └── downloadClient.ts       # Download and clipboard operations
├── presentation/               # Pure UI utilities (no side effects)
│   ├── index.ts
│   ├── formatters.ts           # formatFileSize, formatDuration, etc.
│   ├── statusConfig.ts         # Export/execution status styling
│   └── formatConfig.ts         # Export format styling
├── hooks/
│   ├── useNewExportFlow.ts     # Export flow orchestration
│   └── useExecutionFilters.ts  # Execution filtering logic
├── ExportsTab.tsx              # Main exports listing UI
├── SelectExecutionDialog.tsx   # Execution picker modal
├── InlineExportDialog.tsx      # Export dialog wrapper
├── ExportSuccessPanel.tsx      # Success confirmation UI
└── replay/                     # Replay presentation components
```

### Seam 5: Workflow API Client

**Location:** `api/workflowClient.ts`

**Purpose:** Isolates workflow API calls for the export flow, making it testable without network calls.

**Interface:**
```typescript
export interface WorkflowApiClient {
  fetchWorkflow(workflowId: string, signal?: AbortSignal): Promise<WorkflowInfo>;
}
```

**How to test:**
```typescript
import { createMockWorkflowApiClient } from '@/domains/exports';

// Create a mock client
const mockClient = createMockWorkflowApiClient({
  fetchWorkflow: async (id) => ({
    id,
    name: 'Test Workflow',
    projectId: 'project-123',
  }),
});

// Inject into hook
const result = useNewExportFlow({
  onViewExecution: jest.fn(),
  workflowApiClient: mockClient,
});
```

### Seam 6: Download Client

**Location:** `api/downloadClient.ts`

**Purpose:** Isolates browser download and clipboard operations for testing.

**Interface:**
```typescript
export interface DownloadClient {
  downloadFromUrl(url: string, fileName: string): Promise<DownloadResult>;
  downloadBlob(blob: Blob, fileName: string): DownloadResult;
  copyToClipboard(text: string): Promise<DownloadResult>;
}
```

**How to test:**
```typescript
import { createMockDownloadClient } from '@/domains/exports';

// Create a mock client
const mockClient = createMockDownloadClient({
  downloadFromUrl: async () => ({ success: true }),
  copyToClipboard: async () => ({ success: true }),
});

// Inject into component
<ExportsTab
  onViewExecution={mockFn}
  downloadClient={mockClient}
/>
```

### Seam 7: Presentation Utilities

**Location:** `presentation/`

**Purpose:** All formatting and configuration logic is extracted into pure functions for easy testing. The presentation module uses canonical types from `@/domains/executions/export/config/types.ts` for consistency.

**Type relationship:**
- `ExportFormatDisplayConfig` - defined in `config/types.ts`, used by `FORMAT_CONFIG`
- `ExportStatusConfig` - defined in `config/types.ts`, used by `EXPORT_STATUS_CONFIG`
- `ExportFormat`, `ExportStatus` - defined in `config/types.ts`, used for Record keys

**Pure functions (no mocks needed):**
```typescript
import {
  formatFileSize,
  formatDuration,
  formatExecutionDuration,
  FORMAT_CONFIG,
  getFormatConfig,
  EXPORT_STATUS_CONFIG,
  getExportStatusConfig,
  getExecutionStatusConfig,
} from '@/domains/exports';

// Direct unit testing
expect(formatFileSize(1024)).toBe('1.0 KB');
expect(formatDuration(90000)).toBe('1:30');
expect(getFormatConfig('mp4').label).toBe('MP4 Video');
expect(getExportStatusConfig('completed').label).toBe('Ready');
```

### Boundaries of Responsibility

#### api/
- **Owns:** Network requests, clipboard access, file downloads
- **Does NOT:** Format data, manage React state, render UI

#### presentation/
- **Owns:** Formatting logic, UI configuration constants
- **Does NOT:** Make API calls, manage state, perform side effects

#### hooks/
- **Owns:** React state management, flow orchestration
- **Does NOT:** Perform complex formatting (delegates to presentation/), make direct API calls (uses api/)

#### Components (ExportsTab, SelectExecutionDialog, etc.)
- **Owns:** UI rendering, user interaction handling
- **Does NOT:** Make direct API calls (uses injected clients), perform complex formatting (uses presentation/)

### Testing Strategy

#### Unit Tests (Pure Functions)
- All functions in `presentation/` can be tested without mocks
- Test files co-located with source when needed

#### Integration Tests (Components with Mocks)
- Use `createMockWorkflowApiClient()` for workflow operations
- Use `createMockDownloadClient()` for download/clipboard operations
- Verify correct delegation to presentation utilities

#### E2E Tests
- Test full export creation flow through UI
- Verify download behavior with real API

### Migration Notes

The exports domain components have been refactored to use consolidated utilities:

**Before:**
- `ExportsTab.tsx`: Inline `formatConfig`, `statusConfig`, `formatFileSize`, `formatDuration` (65 lines of config)
- `SelectExecutionDialog.tsx`: Separate inline `statusConfig`, `formatDuration` (30 lines of config)
- `useNewExportFlow.ts`: Direct `fetch()` call for workflow data

**After:**
- All formatting and config moved to `presentation/` module
- All API calls behind testable seams in `api/` module
- Components import and use the consolidated utilities
- Duplicate code eliminated

**Improvements:**
- **DRY:** Single source of truth for formatting and config
- **Testability:** All side effects behind injectable seams
- **Clarity:** Clear separation between presentation, API, and React concerns

---

## Export Page Utilities (`src/export/`)

The standalone export page (used in iframes and as a standalone viewer) has been refactored to extract reusable utilities.

### Directory Structure

```
export/
├── index.ts              # Public API - re-exports all modules
├── types.ts              # Type definitions (FrameTimeline, ExportMetadata, etc.)
├── bootstrap.ts          # Bootstrap utilities for composer initialization
├── timeline.ts           # Pure functions for timeline computation
├── frameMapping.ts       # Pure functions for converting spec data to replay frames
├── status.ts             # Status normalization utilities
├── ReplayExportPage.tsx  # Main component (reduced from 1574 to 1089 lines)
├── composerEntry.tsx     # Composer entry point
├── composer.html         # Composer HTML template
└── exportBootstrap.tsx   # Bootstrap entry point
```

### Seam 8: Bootstrap Utilities

**Location:** `bootstrap.ts`

**Purpose:** Provides pure functions for initializing the export composer and decoding payloads.

**Pure functions:**
```typescript
import {
  addPadding,
  decodeExportPayload,
  decodeJsonSpec,
  ensureBasExportBootstrap,
  getBootstrapPayload,
  resolveBootstrapSpec,
} from '@/export';

// Decode a base64 movie spec
const spec = decodeExportPayload(base64String);

// Get bootstrap payload from window object
const payload = getBootstrapPayload();

// Resolve spec from bootstrap
const { spec, executionId, apiBase } = resolveBootstrapSpec();
```

### Seam 9: Timeline Utilities

**Location:** `timeline.ts`

**Purpose:** Pure functions for computing timeline data from movie specs.

**Pure functions:**
```typescript
import {
  buildTimeline,
  computeTotalDuration,
  findFrameForTime,
  clampProgress,
  DEFAULT_FRAME_DURATION_MS,
} from '@/export';

// Build timeline from frames
const timeline = buildTimeline(movieSpec.frames);

// Compute total duration
const duration = computeTotalDuration(movieSpec.summary, timeline);

// Find frame for a given time
const { index, progress } = findFrameForTime(currentMs, timeline);
```

### Seam 10: Frame Mapping Utilities

**Location:** `frameMapping.ts`

**Purpose:** Pure functions for converting movie spec data to replay frames.

**Pure functions:**
```typescript
import {
  toReplayFrame,
  resolveAssetUrl,
  mapWatermarkSettings,
  mapIntroCardSettings,
  mapOutroCardSettings,
  isWatermarkPosition,
} from '@/export';

// Convert movie frame to replay frame
const replayFrame = toReplayFrame(movieFrame, index, assetMap);

// Resolve asset URL
const url = resolveAssetUrl(asset);

// Map settings from spec
const watermark = mapWatermarkSettings(spec.presentation?.watermark);
const intro = mapIntroCardSettings(spec.presentation?.intro_card);
const outro = mapOutroCardSettings(spec.presentation?.outro_card);
```

### Boundaries of Responsibility

#### types.ts
- **Owns:** Type definitions for timeline, metadata, payloads, window augmentation
- **Does NOT:** Contain any logic or side effects

#### bootstrap.ts
- **Owns:** Payload decoding, bootstrap initialization
- **Does NOT:** Manage React state, perform API calls

#### timeline.ts
- **Owns:** Timeline computation, frame finding, progress clamping
- **Does NOT:** Decode payloads, manage state

#### frameMapping.ts
- **Owns:** Converting spec data to replay frames, settings mapping
- **Does NOT:** Compute timeline, manage state

#### status.ts
- **Owns:** Status normalization and message generation
- **Does NOT:** Make API calls, manage state

### Testing Strategy

#### Unit Tests (Pure Functions)
- All functions in `bootstrap.ts`, `timeline.ts`, `frameMapping.ts`, `status.ts` are pure and can be tested without mocks
- Direct unit testing with expected inputs/outputs

#### Integration Tests
- Test `ReplayExportPage` component with mocked bootstrap payloads
- Verify correct delegation to utility functions

### Migration Notes

**Before:**
- `ReplayExportPage.tsx`: 1574 lines with inline utilities, types, and constants

**After:**
- `ReplayExportPage.tsx`: 1089 lines (31% reduction)
- Utilities extracted to dedicated modules with clear boundaries
- Types consolidated in `types.ts`
- Pure functions in `timeline.ts`, `frameMapping.ts`, `bootstrap.ts`, `status.ts`

**Improvements:**
- **Focused component:** Main component now focuses on rendering and state management
- **Testability:** All utilities are pure functions that can be tested independently
- **Reusability:** Utilities can be used by other components needing timeline/frame logic
- **Type safety:** Types are defined once and imported where needed

---

## Consolidated Utilities

### Status Functions (`exports/presentation/statusConfig.ts`)

The following functions have been consolidated from deprecated locations:

- `mapExportPreviewStatus()` - Maps proto enum to status label
- `normalizePreviewStatus()` - Normalizes status string to lowercase
- `describePreviewStatusMessage()` - Generates human-readable status messages
- `ExportPreviewStatusLabel` type - Status label union type

**Migration:** Import from `@/domains/exports/presentation` instead of deprecated `@/domains/executions/utils/exportHelpers`.

### URL Utilities (`utils/executionTypeMappers.ts`)

The `stripApiSuffix()` function has been consolidated into the existing URL utilities module:

```typescript
import { stripApiSuffix } from '@/utils/executionTypeMappers';

// Strip /api/v1 suffix from URL
const baseUrl = stripApiSuffix(apiUrl);
```

### Formatting Utilities

`formatCapturedLabel()` is now the canonical source in `@/domains/exports/presentation/formatters.ts`. The `@/domains/executions/export/config` module re-exports it for backwards compatibility.
