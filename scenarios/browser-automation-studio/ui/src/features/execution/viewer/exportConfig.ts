/**
 * Export configuration constants for execution replay exports
 */
import {
  Clapperboard,
  Film,
  FileJson,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";

// =============================================================================
// Export Format Types
// =============================================================================

export type ExportFormat = "json" | "mp4" | "gif";

export type ExportDimensionPreset = "spec" | "1080p" | "720p" | "custom";

// =============================================================================
// File Extensions
// =============================================================================

export const EXPORT_EXTENSIONS: Record<ExportFormat, string> = {
  json: "json",
  mp4: "mp4",
  gif: "gif",
};

// =============================================================================
// Dimension Presets
// =============================================================================

export const DIMENSION_PRESET_CONFIG: Record<
  "1080p" | "720p",
  { width: number; height: number; label: string }
> = {
  "1080p": { width: 1920, height: 1080, label: "1080p (Full HD)" },
  "720p": { width: 1280, height: 720, label: "720p (HD)" },
};

export const DEFAULT_EXPORT_WIDTH = 1280;
export const DEFAULT_EXPORT_HEIGHT = 720;

// =============================================================================
// Export Format Options
// =============================================================================

export interface ExportFormatOption {
  id: ExportFormat;
  label: string;
  description: string;
  icon: LucideIcon;
  badge?: string;
  disabled?: boolean;
}

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

// =============================================================================
// Helper Functions
// =============================================================================

export const sanitizeFileStem = (value: string, fallback: string): string => {
  const trimmed = value.trim();
  if (!trimmed) {
    return fallback;
  }
  const sanitized = trimmed.replace(/[^a-zA-Z0-9-_]/g, "-");
  if (!sanitized) {
    return fallback;
  }
  return sanitized;
};

export const coerceMetricNumber = (value: unknown): number => {
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
};

export const formatCapturedLabel = (count: number, noun: string): string => {
  const rounded = Math.round(count);
  const suffix = rounded === 1 ? "" : "s";
  return `${rounded} ${noun}${suffix}`;
};
