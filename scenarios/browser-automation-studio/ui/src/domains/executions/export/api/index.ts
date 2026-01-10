/**
 * Export API module - provides clean seams for export-related API operations.
 */

export {
  // Types
  type ExportApiClient,
  type ExportResult,
  type RecordedVideo,
  type RecordedVideoStatus,
  // Implementations
  createMockExportApiClient,
  defaultExportApiClient,
} from "./exportClient";

export {
  // Types
  type BuildExportRequestOptions,
  type ExportFormat,
  type ExportProgress,
  type ExportRequestPayload,
  type ExportStatusResponse,
  type RenderSource,
  type ServerExportResponse,
  // Pure functions
  buildExportOverrides,
  buildExportRequest,
  // API functions
  executeServerExport,
  getExportStatus,
  getLastOutputDir,
  saveOutputDir,
} from "./executeExport";
