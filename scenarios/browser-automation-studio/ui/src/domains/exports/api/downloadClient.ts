/**
 * Download Client
 *
 * Provides a testable seam for file operations.
 * For Electron apps, supports revealing files in the system file manager
 * instead of browser downloads (since files are already on the local filesystem).
 */

import { getConfig } from "@/config";

// =============================================================================
// Types
// =============================================================================

export interface DownloadResult {
  success: boolean;
  error?: string;
}

export interface DownloadClient {
  /**
   * Downloads a file from a URL.
   * @deprecated For Electron apps, use revealFile or openFolder instead.
   */
  downloadFromUrl(url: string, fileName: string): Promise<DownloadResult>;

  /**
   * Downloads a blob as a file.
   * @deprecated For Electron apps, use revealFile or openFolder instead.
   */
  downloadBlob(blob: Blob, fileName: string): DownloadResult;

  /**
   * Copies text to clipboard.
   */
  copyToClipboard(text: string): Promise<DownloadResult>;

  /**
   * Reveals a file in the system file manager (e.g., "Show in Finder" on macOS).
   * This opens the containing folder and selects/highlights the file.
   */
  revealFile(exportId: string): Promise<DownloadResult>;

  /**
   * Opens the containing folder in the system file manager.
   * Unlike revealFile, this just opens the folder without selecting the file.
   */
  openFolder(exportId: string): Promise<DownloadResult>;
}

// =============================================================================
// Default Implementation
// =============================================================================

async function downloadFromUrl(
  url: string,
  fileName: string,
): Promise<DownloadResult> {
  try {
    const response = await fetch(url);
    if (!response.ok) {
      return { success: false, error: 'Download failed' };
    }

    const blob = await response.blob();
    return downloadBlob(blob, fileName);
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Download failed';
    return { success: false, error: message };
  }
}

function downloadBlob(blob: Blob, fileName: string): DownloadResult {
  try {
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = fileName;
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
    URL.revokeObjectURL(url);
    return { success: true };
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Download failed';
    return { success: false, error: message };
  }
}

async function copyToClipboard(text: string): Promise<DownloadResult> {
  try {
    await navigator.clipboard.writeText(text);
    return { success: true };
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Copy failed';
    return { success: false, error: message };
  }
}

/**
 * Reveals a file in the system file manager via the backend API.
 */
async function revealFile(exportId: string): Promise<DownloadResult> {
  try {
    const { API_URL } = await getConfig();
    const response = await fetch(`${API_URL}/exports/${exportId}/reveal`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });

    if (!response.ok) {
      const text = await response.text();
      return { success: false, error: text || 'Failed to reveal file' };
    }

    return { success: true };
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Failed to reveal file';
    return { success: false, error: message };
  }
}

/**
 * Opens the containing folder in the system file manager via the backend API.
 */
async function openFolder(exportId: string): Promise<DownloadResult> {
  try {
    const { API_URL } = await getConfig();
    const response = await fetch(`${API_URL}/exports/${exportId}/open-folder`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });

    if (!response.ok) {
      const text = await response.text();
      return { success: false, error: text || 'Failed to open folder' };
    }

    return { success: true };
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Failed to open folder';
    return { success: false, error: message };
  }
}

/**
 * Default download client implementation.
 */
export const defaultDownloadClient: DownloadClient = {
  downloadFromUrl,
  downloadBlob,
  copyToClipboard,
  revealFile,
  openFolder,
};

// =============================================================================
// Testing Utilities
// =============================================================================

/**
 * Creates a mock download client for testing.
 */
export function createMockDownloadClient(
  overrides: Partial<DownloadClient> = {},
): DownloadClient {
  return {
    downloadFromUrl: overrides.downloadFromUrl ?? (async () => ({ success: true })),
    downloadBlob: overrides.downloadBlob ?? (() => ({ success: true })),
    copyToClipboard: overrides.copyToClipboard ?? (async () => ({ success: true })),
    revealFile: overrides.revealFile ?? (async () => ({ success: true })),
    openFolder: overrides.openFolder ?? (async () => ({ success: true })),
  };
}
