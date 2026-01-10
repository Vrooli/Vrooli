/**
 * Export configuration constants - all export-related configuration values.
 *
 * Consolidated from:
 * - domains/executions/viewer/exportConfig.ts
 * - domains/exports/presentation/formatConfig.ts
 * - domains/executions/export/hooks/useExportDialogState.ts
 */

import {
  Clapperboard,
  Film,
  FileJson,
  Image,
  Monitor,
  Palette,
  Video,
} from "lucide-react";
import type {
  ExportDimensions,
  ExportDimensionPreset,
  ExportFormat,
  ExportFormatOption,
  ExportRenderSourceOption,
  ExportStylizationOption,
} from "./types";

// =============================================================================
// File Extensions
// =============================================================================

/**
 * File extension for each export format.
 */
export const EXPORT_EXTENSIONS: Record<ExportFormat, string> = {
  mp4: "mp4",
  gif: "gif",
  json: "json",
  html: "html",
};

// =============================================================================
// Dimension Presets
// =============================================================================

/**
 * Default export dimensions (720p).
 */
export const DEFAULT_EXPORT_DIMENSIONS: ExportDimensions = {
  width: 1280,
  height: 720,
};

/**
 * Dimension preset configurations.
 */
export const DIMENSION_PRESET_CONFIG: Record<
  "1080p" | "720p",
  ExportDimensions & { label: string }
> = {
  "1080p": { width: 1920, height: 1080, label: "1080p (Full HD)" },
  "720p": { width: 1280, height: 720, label: "720p (HD)" },
};

// =============================================================================
// Export Format Options (for ExportDialog)
// =============================================================================

/**
 * Export format options for the export dialog selector.
 * Uses Clapperboard/Film icons for video formats.
 */
export const EXPORT_FORMAT_OPTIONS: ExportFormatOption[] = [
  {
    id: "mp4",
    label: "MP4 Video",
    description: "1080p marketing reel (server render)",
    icon: Clapperboard,
    badge: "Default",
  },
  {
    id: "gif",
    label: "Animated GIF",
    description: "Looped shareable highlight",
    icon: Film,
    badge: "Guide",
  },
  {
    id: "json",
    label: "JSON Package",
    description: "Raw replay bundle for tooling",
    icon: FileJson,
    badge: "Data",
  },
];

/**
 * Render source options for the export dialog.
 */
export const EXPORT_RENDER_SOURCE_OPTIONS: ExportRenderSourceOption[] = [
  {
    id: "auto",
    label: "Auto",
    description: "Prefer recording when available, fall back to slideshow.",
    icon: Clapperboard,
  },
  {
    id: "recorded_video",
    label: "Recording",
    description: "Export the native screen recording captured during execution.",
    icon: Video,
  },
  {
    id: "replay_frames",
    label: "Slideshow",
    description: "Create a stylized video from captured screenshots with cursor animation.",
    icon: Image,
  },
];

// =============================================================================
// Stylization Options (for ExportDialog)
// =============================================================================

/**
 * Stylization options for the export dialog.
 * Controls whether visual enhancements are applied during export.
 */
export const EXPORT_STYLIZATION_OPTIONS: ExportStylizationOption[] = [
  {
    id: "stylized",
    label: "Stylized",
    description: "Apply browser frame, background, and cursor overlay.",
    icon: Palette,
  },
  {
    id: "raw",
    label: "Raw",
    description: "Export content exactly as captured.",
    icon: Monitor,
  },
];

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Checks if a format is a binary format requiring server-side rendering.
 */
export function isBinaryFormat(format: ExportFormat): boolean {
  return format === "mp4" || format === "gif";
}

/**
 * Gets the file extension for a format.
 */
export function getFormatExtension(format: ExportFormat): string {
  return EXPORT_EXTENSIONS[format];
}

/**
 * Builds a complete file name from stem and format.
 */
export function buildExportFileName(stem: string, format: ExportFormat): string {
  return `${stem}.${EXPORT_EXTENSIONS[format]}`;
}

/**
 * Sanitizes a file stem to be safe for file systems.
 * Replaces non-alphanumeric characters (except - and _) with dashes.
 */
export function sanitizeFileStem(value: string, fallback: string): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return fallback;
  }
  const sanitized = trimmed.replace(/[^a-zA-Z0-9-_]/g, "-");
  return sanitized || fallback;
}

/**
 * Generates a default file stem for an export.
 */
export function generateDefaultFileStem(executionId: string): string {
  return `browser-automation-replay-${executionId.slice(0, 8)}`;
}

/**
 * Builds dimension preset options array including the spec dimensions.
 */
export function buildDimensionPresetOptions(
  specDimensions: ExportDimensions,
  customDimensions: ExportDimensions,
): Array<{
  id: ExportDimensionPreset;
  label: string;
  width: number;
  height: number;
  description?: string;
}> {
  return [
    {
      id: "spec" as const,
      label: `Scenario (${specDimensions.width}×${specDimensions.height})`,
      width: specDimensions.width,
      height: specDimensions.height,
      description: "Match recorded canvas",
    },
    {
      id: "1080p" as const,
      label: DIMENSION_PRESET_CONFIG["1080p"].label,
      width: DIMENSION_PRESET_CONFIG["1080p"].width,
      height: DIMENSION_PRESET_CONFIG["1080p"].height,
      description: "Great for polished exports",
    },
    {
      id: "720p" as const,
      label: DIMENSION_PRESET_CONFIG["720p"].label,
      width: DIMENSION_PRESET_CONFIG["720p"].width,
      height: DIMENSION_PRESET_CONFIG["720p"].height,
      description: "Smaller file size",
    },
    {
      id: "custom" as const,
      label: "Custom size",
      width: customDimensions.width,
      height: customDimensions.height,
      description: "Set explicit pixel dimensions",
    },
  ];
}

// formatCapturedLabel is re-exported from @/domains/exports/presentation
// to maintain a single source of truth for formatting utilities

/**
 * Coerces a metric value to a number, returning 0 for invalid values.
 */
export function coerceMetricNumber(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string") {
    const parsed = Number.parseFloat(value);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  return 0;
}
