import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import { DownloadParamsSchema } from '../types/instruction';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError } from '../utils';

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
      // Validate parameters
      const params = DownloadParamsSchema.parse(instruction.params);

      const timeout = params.timeoutMs || DEFAULT_TIMEOUT_MS;
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

      // Save download to default temp path
      const savePath = `/tmp/download-${Date.now()}-${download.suggestedFilename()}`;
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
