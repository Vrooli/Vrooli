/**
 * Export execution utilities - handles the actual export API call.
 *
 * This module separates the export execution concern from UI state management,
 * making the logic testable and reusable.
 *
 * All exports are saved server-side (no browser downloads). Binary exports
 * (mp4/gif) use async rendering with WebSocket progress updates.
 */

import type { ReplayMovieSpec } from "@/types/export";
import { getConfig } from "@/config";

// =============================================================================
// Types
// =============================================================================

export type ExportFormat = "mp4" | "gif" | "json" | "html";
export type RenderSource = "auto" | "recorded_video" | "replay_frames";

export interface ExportOverrides {
  theme_preset?: {
    chrome_theme?: string;
    background_theme?: string;
  };
  cursor_preset?: {
    theme?: string;
    initial_position?: string;
    scale?: number;
    click_animation?: string;
  };
}

export interface BuildExportRequestOptions {
  format: ExportFormat;
  fileName: string;
  renderSource: RenderSource;
  movieSpec: ReplayMovieSpec | null;
  overrides?: ExportOverrides;
  outputDir: string;
}

export interface ExportRequestPayload {
  format: ExportFormat;
  file_name: string;
  render_source: RenderSource;
  output_dir: string;
  movie_spec?: ReplayMovieSpec;
  overrides?: ExportOverrides;
}

/** Response from async export endpoint */
export interface ServerExportResponse {
  export_id: string;
  execution_id: string;
  status: "processing";
  message?: string;
}

/** Export status from status endpoint */
export interface ExportStatusResponse {
  export_id: string;
  execution_id: string;
  status: "pending" | "processing" | "completed" | "failed";
  format: string;
  name: string;
  storage_url?: string;
  file_size_bytes?: number;
  error?: string;
}

/** Export progress from WebSocket */
export interface ExportProgress {
  export_id: string;
  execution_id: string;
  stage: "preparing" | "capturing" | "encoding" | "finalizing" | "completed" | "failed";
  progress_percent: number;
  status: "processing" | "completed" | "failed";
  storage_url?: string;
  file_size_bytes?: number;
  error?: string;
  timestamp?: string;
}

// =============================================================================
// Pure Functions
// =============================================================================

/**
 * Builds an export request payload from the given options.
 * This is a pure function that can be easily tested.
 */
export function buildExportRequest(options: BuildExportRequestOptions): ExportRequestPayload {
  const { format, fileName, renderSource, movieSpec, overrides, outputDir } = options;

  const hasExportFrames =
    movieSpec && Array.isArray(movieSpec.frames) && movieSpec.frames.length > 0;

  const payload: ExportRequestPayload = {
    format,
    file_name: fileName,
    render_source: renderSource,
    output_dir: outputDir,
  };

  if (hasExportFrames && movieSpec) {
    payload.movie_spec = movieSpec;
  } else if (overrides) {
    payload.overrides = overrides;
  }

  return payload;
}

/**
 * Builds export overrides from customization settings.
 */
export function buildExportOverrides(settings: {
  chromeTheme?: string;
  backgroundTheme?: string;
  cursorTheme?: string;
  cursorInitialPosition?: string;
  cursorScale?: number;
  cursorClickAnimation?: string;
}): ExportOverrides {
  return {
    theme_preset: {
      chrome_theme: settings.chromeTheme,
      background_theme: settings.backgroundTheme,
    },
    cursor_preset: {
      theme: settings.cursorTheme,
      initial_position: settings.cursorInitialPosition,
      scale: Number.isFinite(settings.cursorScale) ? settings.cursorScale : 1,
      click_animation: settings.cursorClickAnimation,
    },
  };
}

// =============================================================================
// localStorage Keys
// =============================================================================

const EXPORT_OUTPUT_DIR_KEY = "bas-export-output-dir";
const DEFAULT_OUTPUT_DIR = "data/exports";

/**
 * Gets the last-used export output directory from localStorage.
 */
export function getLastOutputDir(): string {
  if (typeof window === "undefined") return DEFAULT_OUTPUT_DIR;
  return localStorage.getItem(EXPORT_OUTPUT_DIR_KEY) ?? DEFAULT_OUTPUT_DIR;
}

/**
 * Saves the export output directory to localStorage.
 */
export function saveOutputDir(dir: string): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(EXPORT_OUTPUT_DIR_KEY, dir);
}

// =============================================================================
// API Functions
// =============================================================================

/**
 * Executes a server-side export. For binary formats (mp4/gif), this returns
 * immediately with an export ID. Progress is delivered via WebSocket.
 */
export async function executeServerExport(options: {
  executionId: string;
  payload: ExportRequestPayload;
}): Promise<ServerExportResponse> {
  const { executionId, payload } = options;
  const { API_URL } = await getConfig();

  const response = await fetch(`${API_URL}/executions/${executionId}/export`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Export request failed (${response.status})`);
  }

  // Parse JSON response
  const data = await response.json();

  // Save output_dir to localStorage for next time
  if (payload.output_dir) {
    saveOutputDir(payload.output_dir);
  }

  return {
    export_id: data.export_id,
    execution_id: data.execution_id,
    status: "processing",
    message: data.message,
  };
}

/**
 * Gets the current status of an export.
 * Useful for reconnection scenarios when WebSocket connection was lost.
 */
export async function getExportStatus(exportId: string): Promise<ExportStatusResponse> {
  const { API_URL } = await getConfig();

  const response = await fetch(`${API_URL}/exports/${exportId}/status`, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
    },
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Failed to get export status (${response.status})`);
  }

  return response.json();
}
