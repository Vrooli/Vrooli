import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import { UploadFileParamsSchema } from '../types/instruction';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError } from '../utils';

/**
 * Upload file handler
 *
 * Handles file upload operations
 */
export class UploadHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['uploadfile', 'upload'];
  }

  async execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    try {
      // Validate parameters
      const params = UploadFileParamsSchema.parse(instruction.params);

      if (!params.selector) {
        return {
          success: false,
          error: {
            message: 'uploadfile instruction missing selector parameter',
            code: 'MISSING_PARAM',
            kind: 'orchestration',
            retryable: false,
          },
        };
      }

      const filePath = params.filePath || params.file_path || params.path;
      if (!filePath) {
        return {
          success: false,
          error: {
            message: 'uploadfile instruction missing filePath parameter',
            code: 'MISSING_PARAM',
            kind: 'orchestration',
            retryable: false,
          },
        };
      }

      const timeout = params.timeoutMs || DEFAULT_TIMEOUT_MS;

      logger.debug('Uploading file', {
        selector: params.selector,
        filePath,
        timeout,
      });

      // Set input files
      await page.setInputFiles(params.selector, filePath, { timeout });

      logger.info('File upload successful', {
        selector: params.selector,
        filePath,
      });

      return {
        success: true,
      };
    } catch (error) {
      logger.error('Upload failed', {
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
