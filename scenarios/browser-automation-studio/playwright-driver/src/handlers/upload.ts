import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getUploadFileParams } from '../types';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError, validateTimeout, logger, scopedLog, LogContext } from '../utils';
import { uploadTracker } from '../infra';
import { constants as fsConstants } from 'fs';
import { access } from 'fs/promises';

/**
 * Upload file handler
 *
 * Handles file upload operations
 *
 * Idempotency behavior:
 * - Setting the same file to the same input is inherently idempotent (browser behavior)
 * - Concurrent uploads for the same selector/file await the first operation
 * - This prevents race conditions when requests are retried
 *
 * ARCHITECTURE:
 * Uses the operation tracker pattern from infra/ to handle:
 * - In-flight deduplication (prevents concurrent uploads of same file)
 * - Session cleanup (tracker clears state on session close/reset)
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

      // Generate operation key from selector + file path
      const fileKey = Array.isArray(filePath) ? filePath.sort().join(':') : filePath;
      const operationKey = `${params.selector}:${fileKey}`;

      // Idempotency: Check for in-flight upload with same parameters
      const inFlight = uploadTracker.getInFlight(sessionId, operationKey);
      if (inFlight) {
        logger.debug(scopedLog(LogContext.INSTRUCTION, 'upload already in progress, waiting'), {
          sessionId,
          selector: params.selector,
          hint: 'Concurrent upload request detected; waiting for in-flight operation',
        });
        return inFlight as Promise<HandlerResult>;
      }

      // Create the upload promise and track it
      const uploadPromise = this.executeUpload(page, params.selector, filePath, timeout, sessionId);
      const cleanup = uploadTracker.trackInFlight(sessionId, operationKey, uploadPromise);

      try {
        return await uploadPromise;
      } finally {
        cleanup();
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
    page: import('rebrowser-playwright').Page,
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
