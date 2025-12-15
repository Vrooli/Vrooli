/**
 * Recording Validation Handlers
 *
 * Handles selector validation and replay preview operations.
 * These are used to test recorded actions before generating workflows.
 */

import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../../session';
import type { Config } from '../../config';
import { parseJsonBody, sendJson, sendError } from '../../middleware';
import { logger, scopedLog, LogContext } from '../../utils';
import { createRecordModeController } from '../../recording/controller';
import type {
  ValidateSelectorRequest,
  ValidateSelectorResponse,
  ReplayPreviewRequest,
  ReplayPreviewResponse,
} from './types';

/**
 * Validate selector endpoint
 *
 * POST /session/:id/record/validate-selector
 *
 * Validates a selector on the current page.
 */
export async function handleValidateSelector(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config);
    const request = body as unknown as ValidateSelectorRequest;

    if (!request.selector) {
      sendJson(res, 400, {
        error: 'MISSING_SELECTOR',
        message: 'selector field is required',
      });
      return;
    }

    // Create controller if needed for validation
    if (!session.recordingController) {
      session.recordingController = createRecordModeController(session.page, sessionId);
    }

    const validation = await session.recordingController.validateSelector(request.selector);

    const response: ValidateSelectorResponse = {
      valid: validation.valid,
      match_count: validation.matchCount,
      selector: validation.selector,
      error: validation.error,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/validate-selector`);
  }
}

/**
 * Replay preview endpoint
 *
 * POST /session/:id/record/replay-preview
 *
 * Replays recorded actions to test if they work before generating a workflow.
 */
export async function handleReplayPreview(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const session = sessionManager.getSession(sessionId);
    const body = await parseJsonBody(req, config);
    const request = body as unknown as ReplayPreviewRequest;

    if (!request.entries || !Array.isArray(request.entries) || request.entries.length === 0) {
      sendJson(res, 400, {
        error: 'MISSING_ENTRIES',
        message: 'entries field is required and must be a non-empty array',
      });
      return;
    }

    // Create controller if needed
    if (!session.recordingController) {
      session.recordingController = createRecordModeController(session.page, sessionId);
    }

    logger.info(scopedLog(LogContext.RECORDING, 'replay started'), {
      sessionId,
      entryCount: request.entries.length,
      limit: request.limit,
      stopOnFailure: request.stop_on_failure ?? true,
    });

    // Execute replay
    const result = await session.recordingController.replayPreview({
      entries: request.entries,
      limit: request.limit,
      stopOnFailure: request.stop_on_failure ?? true,
      actionTimeout: request.action_timeout ?? config.execution.replayActionTimeoutMs,
    });

    logger.info(scopedLog(LogContext.RECORDING, result.success ? 'replay completed' : 'replay failed'), {
      sessionId,
      success: result.success,
      passed: result.passedActions,
      failed: result.failedActions,
      durationMs: result.totalDurationMs,
      stoppedEarly: result.stoppedEarly,
    });

    // Convert to API response format (snake_case)
    const response: ReplayPreviewResponse = {
      success: result.success,
      total_actions: result.totalActions,
      passed_actions: result.passedActions,
      failed_actions: result.failedActions,
      results: result.results.map((r) => ({
        entry_id: r.entryId,
        sequence_num: r.sequenceNum,
        action_type: r.actionType,
        success: r.success,
        duration_ms: r.durationMs,
        error: r.error ? {
          message: r.error.message,
          code: r.error.code,
          match_count: r.error.matchCount,
          selector: r.error.selector,
        } : undefined,
        screenshot_on_error: r.screenshotOnError,
      })),
      total_duration_ms: result.totalDurationMs,
      stopped_early: result.stoppedEarly,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/replay-preview`);
  }
}
