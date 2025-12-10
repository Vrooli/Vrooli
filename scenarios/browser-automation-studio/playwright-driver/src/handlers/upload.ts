import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import { UploadFileParamsSchema } from '../types/instruction';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError, validateTimeout, validateParams } from '../utils';
import * as fs from 'fs/promises';

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
      // Hardened: Validate params object exists
      const rawParams = validateParams(instruction.params, 'uploadfile');
      const params = UploadFileParamsSchema.parse(rawParams);

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

      // Hardened: Pre-validate file exists and is readable
      // Note: filePath may be a string or string[] - only validate single files
      const fileToCheck = Array.isArray(filePath) ? filePath[0] : filePath;
      if (fileToCheck) {
        try {
          await fs.access(fileToCheck, fs.constants.R_OK);
        } catch {
          return {
            success: false,
            error: {
              message: `File not accessible: ${fileToCheck}`,
              code: 'FILE_NOT_FOUND',
              kind: 'orchestration',
              retryable: false,
            },
          };
        }
      }

      // Hardened: Validate timeout bounds
      const timeout = validateTimeout(params.timeoutMs, DEFAULT_TIMEOUT_MS, 'uploadfile');

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
