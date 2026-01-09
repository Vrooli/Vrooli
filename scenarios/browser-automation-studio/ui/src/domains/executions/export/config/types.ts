/**
 * Export configuration types - the canonical type definitions for export functionality.
 *
 * This is the SINGLE SOURCE OF TRUTH for export-related types.
 * All other modules should import from here rather than defining their own.
 */

import type { LucideIcon } from "lucide-react";

// =============================================================================
// Export Format Types
// =============================================================================

/**
 * Supported export formats.
 * - mp4: Video file for marketing/sharing
 * - gif: Animated image for quick demos
 * - json: Raw replay data for programmatic use
 * - html: Self-contained HTML bundle with embedded player
 */
export type ExportFormat = "mp4" | "gif" | "json" | "html";

/**
 * Binary export formats that require server-side rendering.
 */
export type BinaryExportFormat = "mp4" | "gif";

/**
 * Data export formats that can be generated client-side.
 */
export type DataExportFormat = "json" | "html";

// =============================================================================
// Dimension Types
// =============================================================================

/**
 * Dimension preset identifiers.
 * - spec: Match the original recording dimensions
 * - 1080p: Full HD (1920x1080)
 * - 720p: HD (1280x720)
 * - custom: User-specified dimensions
 */
export type ExportDimensionPreset = "spec" | "1080p" | "720p" | "custom";

/**
 * Width and height pair.
 */
export interface ExportDimensions {
  width: number;
  height: number;
}

// =============================================================================
// Render Source Types
// =============================================================================

/**
 * Source for video rendering.
 * - auto: Automatically choose the best source
 * - recorded_video: Use native Playwright recording
 * - replay_frames: Use stylized replay frames
 */
export type ExportRenderSource = "auto" | "recorded_video" | "replay_frames";

// =============================================================================
// Display Configuration Types
// =============================================================================

/**
 * Configuration for displaying an export format option.
 */
export interface ExportFormatOption {
  id: ExportFormat;
  label: string;
  description: string;
  icon: LucideIcon;
  badge?: string;
  disabled?: boolean;
}

/**
 * Configuration for displaying a render source option.
 */
export interface ExportRenderSourceOption {
  id: ExportRenderSource;
  label: string;
  description: string;
  icon: LucideIcon;
}

/**
 * Configuration for displaying a dimension preset option.
 */
export interface ExportDimensionPresetOption {
  id: ExportDimensionPreset;
  label: string;
  width?: number;
  height?: number;
  description?: string;
}

/**
 * UI styling configuration for an export format.
 */
export interface ExportFormatDisplayConfig {
  icon: LucideIcon;
  color: string;
  bgColor: string;
  label: string;
}

// =============================================================================
// Export Status Types
// =============================================================================

/**
 * Status of an export record.
 */
export type ExportStatus = "pending" | "processing" | "completed" | "failed";

/**
 * UI styling configuration for an export status.
 */
export interface ExportStatusConfig {
  icon: LucideIcon;
  color: string;
  bgColor: string;
  label: string;
}
