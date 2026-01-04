import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getDownloadParams } from '../types';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError, validateTimeout, logger, scopedLog, LogContext } from '../utils';
import { downloadTracker } from '../infra';
import * as fs from 'fs/promises';
import * as path from 'path';

/**
 * Clear download cache for a session.
 *
 * @deprecated This function is kept for backward compatibility.
 * The download tracker now handles cleanup via the session cleanup registry.
 */
export function clearSessionDownloadCache(sessionId: string): void {
  // Delegate to the tracker - it's already registered with the cleanup registry
  downloadTracker.clearSession(sessionId);
}

/**
 * Download handler
 *
 * Handles file download operations
 *
 * Idempotency behavior:
 * - If the same download (by selector/URL) is already in progress, waits for it
 * - Successful downloads are cached for idempotent retries (5 min TTL)
 * - Download path includes timestamp to avoid overwriting files
 * - Safe to retry; concurrent requests get the same result
 *
 * ARCHITECTURE:
 * Uses the operation tracker pattern from infra/ to handle:
 * - In-flight deduplication (prevents concurrent downloads of same resource)
 * - Result caching (returns cached result on retry)
 * - Session cleanup (tracker clears state on session close/reset)
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

      // Generate operation key from selector/url (session is handled by tracker)
      const operationKey = `${selector || ''}:${targetURL || ''}`;

      // Idempotency: Check for cached result from previous successful download
      const cachedResult = downloadTracker.getCached(sessionId, operationKey);
      if (cachedResult) {
        logger.info(scopedLog(LogContext.INSTRUCTION, 'returning cached download result'), {
          sessionId,
          selector,
          url: targetURL,
          hint: 'Download was previously successful; returning cached result (idempotent)',
        });
        return cachedResult as HandlerResult;
      }

      // Idempotency: Check for in-flight download with same key
      const inFlight = downloadTracker.getInFlight(sessionId, operationKey);
      if (inFlight) {
        logger.debug(scopedLog(LogContext.INSTRUCTION, 'download already in progress, waiting'), {
          sessionId,
          selector,
          url: targetURL,
          hint: 'Concurrent download request detected; waiting for in-flight download',
        });
        return inFlight as Promise<HandlerResult>;
      }

      // Create the download promise and track it
      const downloadPromise = this.executeDownload(page, selector, targetURL, timeout, sessionId, operationKey);
      const cleanup = downloadTracker.trackInFlight(sessionId, operationKey, downloadPromise);

      try {
        const result = await downloadPromise;
        // Cache successful result
        if (result.success) {
          downloadTracker.cacheResult(sessionId, operationKey, result);
        }
        return result;
      } finally {
        cleanup();
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
   */
  private async executeDownload(
    page: import('rebrowser-playwright').Page,
    selector: string | undefined,
    targetURL: string | undefined,
    timeout: number,
    _sessionId: string,
    _operationKey: string
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

      return {
        success: true,
        extracted_data: {
          download_path: savePath,
          filename: download.suggestedFilename(),
          url: download.url(),
        },
      };
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
