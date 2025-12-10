import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import { DownloadParamsSchema } from '../types/instruction';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError, validateTimeout, validateParams } from '../utils';
import * as fs from 'fs/promises';
import * as path from 'path';

/**
 * Download handler
 *
 * Handles file download operations
 */
export class DownloadHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['download'];
  }

  async execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

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

      logger.debug('Starting download', {
        selector,
        url: targetURL,
        timeout,
      });

      let download;

      if (selector) {
        // Click selector to trigger download
        [download] = await Promise.all([
          page.waitForEvent('download', { timeout }),
          page.click(selector, { timeout }),
        ]);
      } else if (targetURL) {
        // Navigate to URL to trigger download
        [download] = await Promise.all([
          page.waitForEvent('download', { timeout }),
          page.goto(targetURL, { timeout }),
        ]);
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
