/**
 * Export domain module - unified exports for replay export functionality.
 *
 * This module follows screaming architecture principles:
 * - Clear domain boundaries (config, transformations, api, hooks)
 * - Pure functions for testability (transformations/, config/)
 * - Clean seams for dependency injection (api/)
 * - Focused React hooks (hooks/)
 *
 * Directory structure:
 * - config/         - Types, constants, and presentation config (single source of truth)
 * - transformations/ - Pure functions for dimension scaling and spec transformation
 * - api/            - API client with testable seam
 * - hooks/          - React hooks for state management
 */

// Config (types, constants, presentation - single source of truth)
export {
  // Types
  type BinaryExportFormat,
  type DataExportFormat,
  type ExportDimensionPreset,
  type ExportDimensionPresetOption,
  type ExportDimensions,
  type ExportFormat,
  type ExportFormatDisplayConfig,
  type ExportFormatOption,
  type ExportRenderSource,
  type ExportRenderSourceOption,
  type ExportStatus,
  type ExportStatusConfig,
  // Constants
  DEFAULT_EXPORT_DIMENSIONS,
  DIMENSION_PRESET_CONFIG,
  EXPORT_EXTENSIONS,
  EXPORT_FORMAT_OPTIONS,
  EXPORT_RENDER_SOURCE_OPTIONS,
  // Format/status display config (re-exported from presentation module)
  EXPORT_STATUS_CONFIG,
  FORMAT_CONFIG,
  // Helper functions
  buildDimensionPresetOptions,
  buildExportFileName,
  coerceMetricNumber,
  formatCapturedLabel,
  generateDefaultFileStem,
  getExportStatusConfig,
  getFormatConfig,
  getFormatExtension,
  isBinaryFormat,
  sanitizeFileStem,
} from "./config";

// Transformations (pure functions)
export {
  // Types
  type Dimensions,
  type DimensionPreset,
  type DimensionPresetId,
  type ScaleFactors,
  type ScaleMovieSpecOptions,
  // Constants
  DEFAULT_DIMENSIONS,
  PRESET_DIMENSIONS,
  // Functions
  calculateScaleFactors,
  extractCanvasDimensions,
  isScalingNeeded,
  parseDimensionInput,
  resolveDimensionPreset,
  scaleDimensions,
  scaleFrameRect,
  scaleFrames,
  scaleFrameViewport,
  scaleMovieSpec,
  scalePresentation,
} from "./transformations";

// API (with testing seams)
export {
  // Types
  type BuildExportRequestOptions,
  type ExecuteBinaryExportOptions,
  type ExecuteBinaryExportResult,
  type ExportApiClient,
  type ExportOverrides,
  type ExportRequestPayload,
  type ExportResult,
  type FileSystemFileHandle,
  type FileSystemWritableFileStream,
  type RecordedVideo,
  type RecordedVideoStatus,
  type RenderSource,
  type SaveFilePickerOptions,
  // Pure functions
  buildExportOverrides,
  buildExportRequest,
  createJsonBlob,
  // API functions
  createMockExportApiClient,
  defaultExportApiClient,
  executeBinaryExport,
  saveWithNativeFilePicker,
  supportsFileSystemAccess,
  triggerBlobDownload,
} from "./api";

// Hooks (React state management)
export {
  // useExportDialogState
  buildFileName,
  useExportDialogState,
  type ExportDialogFormState,
  type UseExportDialogStateOptions,
  // useRecordedVideoStatus
  useRecordedVideoStatus,
  type UseRecordedVideoStatusOptions,
  type UseRecordedVideoStatusResult,
} from "./hooks";

// Context (for ExportDialog state management)
export {
  // Types
  type BuildExportDialogContextOptions,
  type ExportDialogActions,
  type ExportDialogContextValue,
  type ExportDimensionState,
  type ExportFileState,
  type ExportFormatState,
  type ExportMetricsState,
  type ExportPreviewState,
  type ExportProgressState,
  type ExportRenderSourceState,
  // Context & Provider
  ExportDialogProvider,
  // Hooks
  useExportDialogActions,
  useExportDialogContext,
  useExportDimensionState,
  useExportFileState,
  useExportFormatState,
  useExportMetricsState,
  useExportPreviewState,
  useExportProgressState,
  useExportRenderSourceState,
  // Builder
  buildExportDialogContextValue,
} from "./context";

// Components (React UI components)
export { ExportDialog, ExportDialogDefault } from "./components";
