/**
 * Browser API Seams
 *
 * This module provides seams for browser-specific APIs, enabling:
 * 1. Easy mocking in tests without complex global stubbing
 * 2. Clear separation between domain logic and browser infrastructure
 * 3. Centralized error handling for browser API failures
 *
 * Components should import these functions instead of calling browser APIs directly.
 */

// ============================================================================
// Types
// ============================================================================

export interface ClipboardWriteResult {
  success: boolean;
  error?: string;
}

export interface FileReadResult {
  success: boolean;
  content?: string;
  error?: string;
}

export interface DownloadTriggerOptions {
  url: string;
  newWindow?: boolean;
}

// ============================================================================
// Clipboard Operations
// ============================================================================

/**
 * Write text to the system clipboard.
 * Returns a result object instead of throwing on failure.
 */
export async function writeToClipboard(text: string): Promise<ClipboardWriteResult> {
  try {
    await navigator.clipboard.writeText(text);
    return { success: true };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : "Failed to copy to clipboard"
    };
  }
}

// ============================================================================
// File Operations
// ============================================================================

/**
 * Read text content from a File object.
 * Returns a result object instead of throwing on failure.
 */
export async function readFileAsText(file: File): Promise<FileReadResult> {
  try {
    const content = await file.text();
    return { success: true, content };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : "Failed to read file"
    };
  }
}

// ============================================================================
// Download Operations
// ============================================================================

/**
 * Trigger a download by opening a URL.
 * For browser-native download behavior.
 */
export function triggerDownload(options: DownloadTriggerOptions): void {
  const { url, newWindow = true } = options;
  if (newWindow) {
    window.open(url, "_blank");
  } else {
    window.location.href = url;
  }
}

/**
 * Create a download link and trigger it programmatically.
 * Use this for blob/data URL downloads where you want the "save as" dialog.
 */
export function triggerBlobDownload(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

// ============================================================================
// Testing Utilities
// ============================================================================

/**
 * Default implementations for testing.
 * Tests can override these to control browser behavior.
 */
export const browserMocks = {
  /**
   * Create a mock clipboard that tracks writes.
   */
  createClipboardMock: () => {
    const writes: string[] = [];
    return {
      writeText: async (text: string) => {
        writes.push(text);
      },
      getWrites: () => [...writes],
      clear: () => {
        writes.length = 0;
      }
    };
  },

  /**
   * Create a mock file reader that returns controlled content.
   */
  createFileReaderMock: (content: string) => ({
    text: async () => content
  })
};
