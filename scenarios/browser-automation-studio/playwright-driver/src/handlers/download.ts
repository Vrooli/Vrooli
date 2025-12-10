import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import { DownloadParamsSchema } from '../types/instruction';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError, validateTimeout, validateParams, logger, scopedLog, LogContext } from '../utils';
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
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, sessionId } = context;

    try {
      // Hardened: Validate params object exists
      const rawParams = validateParams(instruction.params, 'download');
      const params = DownloadParamsSchema.parse(rawParams);

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
      const downloadPromise = this.executeDownload(page, selector, targetURL, timeout, sessionId);
      pendingDownloads.set(downloadKey, downloadPromise);

      try {
        return await downloadPromise;
      } finally {
        // Clean up tracking
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
   */
  private async executeDownload(
    page: import('playwright').Page,
    selector: string | undefined,
    targetURL: string | undefined,
    timeout: number,
    _sessionId: string
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
