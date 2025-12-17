import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getDownloadParams } from '../types';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError, validateTimeout, logger, scopedLog, LogContext } from '../utils';
import * as fs from 'fs/promises';
import * as path from 'path';

/**
 * Track in-flight downloads to prevent duplicate download operations.
 * Key: composite of sessionId + selector/url
 * Value: Promise that resolves to the handler result
 *
 * Idempotency guarantee:
 * - Concurrent download requests for the same resource will await the first
 * - Prevents duplicate file downloads from retries or concurrent calls
 */
const pendingDownloads: Map<string, Promise<HandlerResult>> = new Map();

/**
 * Cache completed download results for idempotent retries.
 * Key: composite of sessionId + selector/url
 * Value: { result, timestamp } for successful downloads
 *
 * Idempotency guarantee:
 * - If a download was successful, subsequent requests return the cached result
 * - Prevents re-downloading the same file on retry
 * - Cache entries expire after DOWNLOAD_CACHE_TTL_MS
 */
interface CachedDownloadResult {
  result: HandlerResult;
  timestamp: number;
}
const completedDownloads: Map<string, CachedDownloadResult> = new Map();

/**
 * TTL for cached download results (5 minutes).
 * Matches instruction idempotency cache TTL.
 */
const DOWNLOAD_CACHE_TTL_MS = 300_000;

/**
 * Clean up expired download cache entries.
 */
function cleanupDownloadCache(): void {
  const now = Date.now();
  for (const [key, value] of completedDownloads.entries()) {
    if (now - value.timestamp > DOWNLOAD_CACHE_TTL_MS) {
      completedDownloads.delete(key);
    }
  }
}

// Run cache cleanup every 2 minutes
setInterval(cleanupDownloadCache, 120_000).unref();

/**
 * Clear download cache for a session.
 * Exported for use by session reset operations.
 */
export function clearSessionDownloadCache(sessionId: string): void {
  for (const key of completedDownloads.keys()) {
    if (key.startsWith(`${sessionId}:`)) {
      completedDownloads.delete(key);
    }
  }
}

/**
 * Download handler
 *
 * Handles file download operations
 *
 * Idempotency behavior:
 * - If the same download (by selector/URL) is already in progress, waits for it
 * - Download path includes timestamp to avoid overwriting files
 * - Safe to retry; concurrent requests get the same result
 */
export class DownloadHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['download'];
  }

  async execute(
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, sessionId } = context;

    try {
      // Get typed params from instruction.action (required after migration)
      const typedParams = instruction.action ? getDownloadParams(instruction.action) : undefined;
      const params = this.requireTypedParams(typedParams, 'download', instruction.nodeId);

      // Hardened: Validate timeout bounds
      const timeout = validateTimeout(params.timeoutMs, DEFAULT_TIMEOUT_MS, 'download');
      const selector = params.selector;
      const targetURL = params.url;

      if (!selector && !targetURL) {
        return {
          success: false,
          error: {
            message: 'download instruction requires selector or url parameter',
            code: 'MISSING_PARAM',
            kind: 'orchestration',
            retryable: false,
          },
        };
      }

      // Generate idempotency key from session + target
      const downloadKey = `${sessionId}:${selector || ''}:${targetURL || ''}`;

      // Idempotency: Check for completed download with same key (retry case)
      const cachedResult = completedDownloads.get(downloadKey);
      if (cachedResult && Date.now() - cachedResult.timestamp < DOWNLOAD_CACHE_TTL_MS) {
        logger.info(scopedLog(LogContext.INSTRUCTION, 'returning cached download result'), {
          sessionId,
          selector,
          url: targetURL,
          cacheAgeMs: Date.now() - cachedResult.timestamp,
          hint: 'Download was previously successful; returning cached result (idempotent)',
        });
        return cachedResult.result;
      }

      // Idempotency: Check for in-flight download with same key
      const pendingDownload = pendingDownloads.get(downloadKey);
      if (pendingDownload) {
        logger.debug(scopedLog(LogContext.INSTRUCTION, 'download already in progress, waiting'), {
          sessionId,
          selector,
          url: targetURL,
          hint: 'Concurrent download request detected; waiting for in-flight download',
        });
        return pendingDownload;
      }

      // Create the download promise and track it
      const downloadPromise = this.executeDownload(page, selector, targetURL, timeout, sessionId, downloadKey);
      pendingDownloads.set(downloadKey, downloadPromise);

      try {
        return await downloadPromise;
      } finally {
        // Clean up in-flight tracking
        pendingDownloads.delete(downloadKey);
      }
    } catch (error) {
      logger.error('Download failed', {
        error: error instanceof Error ? error.message : String(error),
      });

      const driverError = normalizeError(error);

      return {
        success: false,
        error: {
          message: driverError.message,
          code: driverError.code,
          kind: driverError.kind,
          retryable: driverError.retryable,
        },
      };
    }
  }

  /**
   * Internal download execution logic.
   * Separated from execute() to enable idempotency tracking.
   *
   * @param downloadKey - Cache key for storing successful result
   */
  private async executeDownload(
    page: import('playwright').Page,
    selector: string | undefined,
    targetURL: string | undefined,
    timeout: number,
    _sessionId: string,
    downloadKey: string
  ): Promise<HandlerResult> {
    try {
      logger.debug('Starting download', {
        selector,
        url: targetURL,
        timeout,
      });

      let download;

      if (selector) {
        // Click selector to trigger download
        // Use Promise.race to fail fast if click fails before download starts
        const downloadPromise = page.waitForEvent('download', { timeout });
        const clickPromise = page.click(selector, { timeout });

        // Wait for click first - if it fails, we shouldn't wait for download
        try {
          await clickPromise;
        } catch (clickError) {
          // Click failed - don't wait for download event that will never come
          throw clickError;
        }

        // Click succeeded, now wait for download
        download = await downloadPromise;
      } else if (targetURL) {
        // Navigate to URL to trigger download
        // Similar pattern: start download listener, then navigate
        const downloadPromise = page.waitForEvent('download', { timeout });
        const gotoPromise = page.goto(targetURL, { timeout });

        try {
          await gotoPromise;
        } catch (gotoError) {
          // Navigation failed
          throw gotoError;
        }

        download = await downloadPromise;
      }

      if (!download) {
        return {
          success: false,
          error: {
            message: 'Download event not triggered',
            code: 'DOWNLOAD_FAILED',
            kind: 'engine',
            retryable: true,
          },
        };
      }

      // Hardened: Verify /tmp is writable before saving
      const downloadDir = '/tmp';
      const savePath = path.join(downloadDir, `download-${Date.now()}-${download.suggestedFilename()}`);

      try {
        await fs.access(downloadDir, fs.constants.W_OK);
      } catch {
        return {
          success: false,
          error: {
            message: `Download directory not writable: ${downloadDir}`,
            code: 'FILESYSTEM_ERROR',
            kind: 'infra',
            retryable: false,
          },
        };
      }

      await download.saveAs(savePath);

      logger.info('Download successful', {
        filename: download.suggestedFilename(),
        savePath,
      });

      const result: HandlerResult = {
        success: true,
        extracted_data: {
          download_path: savePath,
          filename: download.suggestedFilename(),
          url: download.url(),
        },
      };

      // Cache successful download for idempotent retries
      completedDownloads.set(downloadKey, {
        result,
        timestamp: Date.now(),
      });

      return result;
    } catch (error) {
      logger.error('Download failed', {
        error: error instanceof Error ? error.message : String(error),
      });

      const driverError = normalizeError(error);

      return {
        success: false,
        error: {
          message: driverError.message,
          code: driverError.code,
          kind: driverError.kind,
          retryable: driverError.retryable,
        },
      };
    }
  }
}
