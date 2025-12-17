import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getUploadFileParams } from '../types';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError, validateTimeout, logger, scopedLog, LogContext } from '../utils';
import { constants as fsConstants } from 'fs';
import { access } from 'fs/promises';

/**
 * Track in-flight uploads to prevent duplicate concurrent upload operations.
 * Key: composite of sessionId + selector + filePath
 * Value: Promise that resolves to the handler result
 *
 * Idempotency guarantee:
 * - Concurrent upload requests for the same selector/file will await the first
 * - Prevents race conditions when retries or concurrent calls happen
 */
const pendingUploads: Map<string, Promise<HandlerResult>> = new Map();

/**
 * Generate upload operation key for idempotency tracking.
 */
function generateUploadKey(sessionId: string, selector: string, filePath: string | string[]): string {
  const fileKey = Array.isArray(filePath) ? filePath.sort().join(':') : filePath;
  return `${sessionId}:${selector}:${fileKey}`;
}

/**
 * Upload file handler
 *
 * Handles file upload operations
 *
 * Idempotency behavior:
 * - Setting the same file to the same input is inherently idempotent (browser behavior)
 * - Concurrent uploads for the same selector/file await the first operation
 * - This prevents race conditions when requests are retried
 */
export class UploadHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['uploadfile', 'upload'];
  }

  async execute(
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, sessionId } = context;

    try {
      // Get typed params from instruction.action (required after migration)
      const typedParams = instruction.action ? getUploadFileParams(instruction.action) : undefined;
      const params = this.requireTypedParams(typedParams, 'uploadfile', instruction.nodeId);

      if (!params.selector) {
        return this.missingParamError('uploadfile', 'selector');
      }

      // Use filePaths from typed params
      const filePaths = params.filePaths;
      if (!filePaths || filePaths.length === 0) {
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

      // Hardened: Pre-validate first file exists and is readable
      const fileToCheck = filePaths[0];
      if (fileToCheck) {
        try {
          await access(fileToCheck, fsConstants.R_OK);
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

      // Determine the actual file path(s) to upload - single file or array
      const filePath = filePaths.length === 1 ? filePaths[0] : filePaths;

      // Generate idempotency key for this upload operation
      const uploadKey = generateUploadKey(sessionId, params.selector, filePath);

      // Idempotency: Check for in-flight upload with same parameters
      const pendingUpload = pendingUploads.get(uploadKey);
      if (pendingUpload) {
        logger.debug(scopedLog(LogContext.INSTRUCTION, 'upload already in progress, waiting'), {
          sessionId,
          selector: params.selector,
          hint: 'Concurrent upload request detected; waiting for in-flight operation',
        });
        return pendingUpload;
      }

      // Create the upload promise and track it
      const uploadPromise = this.executeUpload(page, params.selector, filePath, timeout, sessionId);
      pendingUploads.set(uploadKey, uploadPromise);

      try {
        return await uploadPromise;
      } finally {
        // Clean up in-flight tracking
        pendingUploads.delete(uploadKey);
      }
    } catch (error) {
      logger.error(scopedLog(LogContext.INSTRUCTION, 'upload failed'), {
        sessionId,
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
   * Internal upload execution logic.
   * Separated from execute() to enable idempotency tracking.
   */
  private async executeUpload(
    page: import('playwright').Page,
    selector: string,
    filePath: string | string[],
    timeout: number,
    sessionId: string
  ): Promise<HandlerResult> {
    logger.debug(scopedLog(LogContext.INSTRUCTION, 'uploading file'), {
      sessionId,
      selector,
      filePath,
      timeout,
    });

    // Set input files
    await page.setInputFiles(selector, filePath, { timeout });

    logger.info(scopedLog(LogContext.INSTRUCTION, 'file upload successful'), {
      sessionId,
      selector,
      filePath,
    });

    return {
      success: true,
    };
  }
}
