/**
 * Export API client - provides a clean seam for export-related API operations.
 *
 * This module isolates all export API calls behind a testable interface.
 * Tests can provide mock implementations to verify behavior without network calls.
 */

import type { ReplayMovieSpec } from "@/types/export";
import { getConfig } from "@/config";

// =============================================================================
// Types
// =============================================================================

export interface RecordedVideo {
  id: string;
  url?: string;
  contentType?: string;
  sizeBytes?: number;
}

export interface RecordedVideoStatus {
  available: boolean;
  count: number;
  videos: RecordedVideo[];
}

export interface ExportRequestPayload {
  format: "mp4" | "gif" | "json" | "html";
  file_name: string;
  render_source: "auto" | "recorded_video" | "replay_frames";
  movie_spec?: ReplayMovieSpec;
  overrides?: ExportOverrides;
}

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

export interface ExportResult {
  blob: Blob;
  contentType: string;
  fileName: string;
}

// =============================================================================
// API Client Interface (Seam for testing)
// =============================================================================

export interface ExportApiClient {
  /**
   * Fetches the recorded video status for an execution.
   */
  fetchRecordedVideoStatus(
    executionId: string,
    signal?: AbortSignal,
  ): Promise<RecordedVideoStatus>;

  /**
   * Executes an export request and returns the resulting blob.
   */
  executeExport(
    executionId: string,
    payload: ExportRequestPayload,
    signal?: AbortSignal,
  ): Promise<ExportResult>;
}

// =============================================================================
// Default Implementation
// =============================================================================

async function fetchRecordedVideoStatus(
  executionId: string,
  signal?: AbortSignal,
): Promise<RecordedVideoStatus> {
  const { API_URL } = await getConfig();
  const response = await fetch(`${API_URL}/executions/${executionId}/recorded-videos`, {
    signal,
  });

  if (!response.ok) {
    throw new Error(`Recorded videos unavailable (${response.status})`);
  }

  const payload = (await response.json()) as { videos?: unknown };
  const videos = Array.isArray(payload?.videos)
    ? (payload.videos as RecordedVideo[])
    : [];

  return {
    available: videos.length > 0,
    count: videos.length,
    videos,
  };
}

async function executeExport(
  executionId: string,
  payload: ExportRequestPayload,
  signal?: AbortSignal,
): Promise<ExportResult> {
  const { API_URL } = await getConfig();
  const acceptHeader = payload.format === "gif" ? "image/gif" : "video/mp4";

  const response = await fetch(`${API_URL}/executions/${executionId}/export`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Accept: acceptHeader,
    },
    body: JSON.stringify(payload),
    signal,
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Export request failed (${response.status})`);
  }

  const blob = await response.blob();
  const contentType = response.headers.get("Content-Type") ?? "application/octet-stream";

  return {
    blob,
    contentType,
    fileName: payload.file_name,
  };
}

/**
 * Default export API client implementation.
 */
export const defaultExportApiClient: ExportApiClient = {
  fetchRecordedVideoStatus,
  executeExport,
};

// =============================================================================
// Testing Utilities
// =============================================================================

/**
 * Creates a mock export API client for testing.
 */
export function createMockExportApiClient(
  overrides: Partial<ExportApiClient> = {},
): ExportApiClient {
  return {
    fetchRecordedVideoStatus: overrides.fetchRecordedVideoStatus ?? (async () => ({
      available: false,
      count: 0,
      videos: [],
    })),
    executeExport: overrides.executeExport ?? (async () => ({
      blob: new Blob(),
      contentType: "application/octet-stream",
      fileName: "test.mp4",
    })),
  };
}
