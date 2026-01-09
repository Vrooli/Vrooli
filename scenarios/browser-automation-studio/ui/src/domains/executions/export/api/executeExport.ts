/**
 * Export execution utilities - handles the actual export API call and file download.
 *
 * This module separates the export execution concern from UI state management,
 * making the logic testable and reusable.
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
}

export interface ExportRequestPayload {
  format: ExportFormat;
  file_name: string;
  render_source: RenderSource;
  movie_spec?: ReplayMovieSpec;
  overrides?: ExportOverrides;
}

export interface ExecuteBinaryExportOptions {
  executionId: string;
  payload: ExportRequestPayload;
  onProgress?: (message: string) => void;
}

export interface ExecuteBinaryExportResult {
  blob: Blob;
  fileName: string;
  contentType: string;
}

// =============================================================================
// Pure Functions
// =============================================================================

/**
 * Builds an export request payload from the given options.
 * This is a pure function that can be easily tested.
 */
export function buildExportRequest(options: BuildExportRequestOptions): ExportRequestPayload {
  const { format, fileName, renderSource, movieSpec, overrides } = options;

  const hasExportFrames =
    movieSpec && Array.isArray(movieSpec.frames) && movieSpec.frames.length > 0;

  const payload: ExportRequestPayload = {
    format,
    file_name: fileName,
    render_source: renderSource,
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
// API Functions
// =============================================================================

/**
 * Executes a binary export (MP4/GIF) and returns the blob.
 */
export async function executeBinaryExport(
  options: ExecuteBinaryExportOptions,
): Promise<ExecuteBinaryExportResult> {
  const { executionId, payload } = options;
  const { API_URL } = await getConfig();

  const acceptHeader = payload.format === "gif" ? "image/gif" : "video/mp4";

  const response = await fetch(`${API_URL}/executions/${executionId}/export`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Accept: acceptHeader,
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Export request failed (${response.status})`);
  }

  const blob = await response.blob();
  const contentType = response.headers.get("Content-Type") ?? acceptHeader;

  return {
    blob,
    fileName: payload.file_name,
    contentType,
  };
}

/**
 * Triggers a browser download for a blob.
 */
export function triggerBlobDownload(blob: Blob, fileName: string): void {
  const url = URL.createObjectURL(blob);
  try {
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = fileName;
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
  } finally {
    URL.revokeObjectURL(url);
  }
}

/**
 * Creates a JSON blob from a movie spec.
 */
export function createJsonBlob(spec: ReplayMovieSpec): Blob {
  return new Blob([JSON.stringify(spec, null, 2)], {
    type: "application/json",
  });
}

// =============================================================================
// File System Access API Types
// =============================================================================

export type SaveFilePickerOptions = {
  suggestedName?: string;
  types?: Array<{
    description?: string;
    accept: Record<string, string[]>;
  }>;
};

export type FileSystemWritableFileStream = {
  write: (data: BlobPart) => Promise<void>;
  close: () => Promise<void>;
};

export type FileSystemFileHandle = {
  createWritable: () => Promise<FileSystemWritableFileStream>;
};

/**
 * Saves a blob using the native file picker.
 * Throws if the user cancels or the browser doesn't support the API.
 */
export async function saveWithNativeFilePicker(
  blob: Blob,
  suggestedName: string,
  description: string,
  accept: Record<string, string[]>,
): Promise<void> {
  const picker = await (
    window as typeof window & {
      showSaveFilePicker?: (options?: SaveFilePickerOptions) => Promise<FileSystemFileHandle>;
    }
  ).showSaveFilePicker?.({
    suggestedName,
    types: [{ description, accept }],
  });

  if (!picker) {
    throw new Error("Unable to open save dialog");
  }

  const writable = await picker.createWritable();
  await writable.write(blob);
  await writable.close();
}

/**
 * Checks if the browser supports the File System Access API.
 */
export function supportsFileSystemAccess(): boolean {
  return (
    typeof window !== "undefined" &&
    typeof (window as typeof window & { showSaveFilePicker?: unknown }).showSaveFilePicker ===
      "function"
  );
}
