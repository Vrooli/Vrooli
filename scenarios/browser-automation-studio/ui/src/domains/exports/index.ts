/**
 * Exports domain - handles replay generation, export rendering, and media output
 *
 * Structure:
 * - api/: API clients with testable seams (workflow, download)
 * - presentation/: Pure formatting utilities and UI configuration
 * - hooks/: React hooks for export flows
 * - replay/: ReplayPresentation + ReplayPlayer shell, theming (backgrounds, cursors, chrome)
 * - store.ts: Export state management
 */

// Store
export { useExportStore } from './store';
export type { Export, CreateExportInput, UpdateExportInput, ExportState } from './store';

// API clients (with testable seams)
export {
  defaultWorkflowApiClient,
  createMockWorkflowApiClient,
  defaultDownloadClient,
  createMockDownloadClient,
  type WorkflowApiClient,
  type WorkflowInfo,
  type DownloadClient,
  type DownloadResult,
} from './api';

// Presentation utilities
export {
  formatFileSize,
  formatDuration,
  formatExecutionDuration,
  formatCapturedLabel,
  formatSeconds,
  EXPORT_STATUS_CONFIG,
  EXECUTION_STATUS_CONFIG,
  getExportStatusConfig,
  getExecutionStatusConfig,
  FORMAT_CONFIG,
  getFormatConfig,
} from './presentation';

// Main components
export { default as ReplayPlayer } from "./replay/ReplayPlayer";
export { ExportSuccessPanel } from "./ExportSuccessPanel";
export { ExportDetailsModal } from "./ExportDetailsModal";
export { ExportsTab } from "./ExportsTab";

// Replay types, constants, and utilities
export * from "./replay";
