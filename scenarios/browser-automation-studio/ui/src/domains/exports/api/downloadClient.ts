/**
 * Download Client
 *
 * Provides a testable seam for download operations.
 * Isolates browser-specific download behavior for easier testing.
 */

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
   */
  downloadFromUrl(url: string, fileName: string): Promise<DownloadResult>;

  /**
   * Downloads a blob as a file.
   */
  downloadBlob(blob: Blob, fileName: string): DownloadResult;

  /**
   * Copies text to clipboard.
   */
  copyToClipboard(text: string): Promise<DownloadResult>;
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
 * Default download client implementation.
 */
export const defaultDownloadClient: DownloadClient = {
  downloadFromUrl,
  downloadBlob,
  copyToClipboard,
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
  };
}
