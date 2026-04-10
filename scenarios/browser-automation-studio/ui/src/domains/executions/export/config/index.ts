/**
 * Export configuration module - types, constants, and presentation config.
 *
 * This is the SINGLE SOURCE OF TRUTH for export-related configuration.
 * All other modules should import from here rather than defining their own.
 *
 * @module export/config
 */

// Types
export type {
  BinaryExportFormat,
  DataExportFormat,
  ExportDimensionPreset,
  ExportDimensionPresetOption,
  ExportDimensions,
  ExportFormat,
  ExportFormatDisplayConfig,
  ExportFormatOption,
  ExportRenderSource,
  ExportRenderSourceOption,
  ExportStatus,
  ExportStatusConfig,
  ExportStylization,
  ExportStylizationOption,
} from "./types";

// Constants
export {
  // File extensions
  EXPORT_EXTENSIONS,
  // Dimension presets
  DEFAULT_EXPORT_DIMENSIONS,
  DIMENSION_PRESET_CONFIG,
  // Format options (for ExportDialog)
  EXPORT_FORMAT_OPTIONS,
  EXPORT_RENDER_SOURCE_OPTIONS,
  // Stylization options
  EXPORT_STYLIZATION_OPTIONS,
  // Helper functions
  buildDimensionPresetOptions,
  buildExportFileName,
  coerceMetricNumber,
  generateDefaultFileStem,
  getFormatExtension,
  isBinaryFormat,
  sanitizeFileStem,
} from "./constants";

// Re-export presentation utilities from the canonical presentation module.
// These provide format/status display config and formatting utilities.
export {
  formatCapturedLabel,
  FORMAT_CONFIG,
  getFormatConfig,
  EXPORT_STATUS_CONFIG,
  getExportStatusConfig,
} from "@/domains/exports/presentation";
