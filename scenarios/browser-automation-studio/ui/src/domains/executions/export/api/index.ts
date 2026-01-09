/**
 * Export API module - provides clean seams for export-related API operations.
 */

export {
  // Types
  type ExportApiClient,
  type ExportOverrides,
  type ExportRequestPayload,
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
  type ExecuteBinaryExportOptions,
  type ExecuteBinaryExportResult,
  type ExportFormat,
  type FileSystemFileHandle,
  type FileSystemWritableFileStream,
  type RenderSource,
  type SaveFilePickerOptions,
  // Pure functions
  buildExportOverrides,
  buildExportRequest,
  createJsonBlob,
  // API functions
  executeBinaryExport,
  saveWithNativeFilePicker,
  supportsFileSystemAccess,
  triggerBlobDownload,
} from "./executeExport";
