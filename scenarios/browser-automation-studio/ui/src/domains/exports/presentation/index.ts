/**
 * Export Presentation Module
 *
 * Consolidates UI presentation utilities for the exports domain:
 * - Formatters for displaying file sizes, durations, and counts
 * - Status configuration for consistent status styling
 * - Format configuration for export format display
 *
 * All functions are pure and side-effect-free for easy testing.
 */

// Formatters
export {
  formatFileSize,
  formatDuration,
  formatExecutionDuration,
  formatCapturedLabel,
  formatSeconds,
} from './formatters';

// Status configuration
export {
  EXPORT_STATUS_CONFIG,
  EXECUTION_STATUS_CONFIG,
  getExportStatusConfig,
  getExecutionStatusConfig,
  // Export preview status utilities
  type ExportPreviewStatusLabel,
  mapExportPreviewStatus,
  normalizePreviewStatus,
  describePreviewStatusMessage,
} from './statusConfig';

// Format configuration
export {
  FORMAT_CONFIG,
  getFormatConfig,
} from './formatConfig';
