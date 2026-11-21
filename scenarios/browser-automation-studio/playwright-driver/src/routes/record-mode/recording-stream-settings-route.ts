/**
 * Recording stream settings route.
 *
 * Stream configuration belongs with the recording route surface, separately
 * from diagnostics and pipeline testing endpoints.
 */

import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../../session';
import type { Config } from '../../config';
import { parseJsonBody, sendJson, sendError } from '../../middleware';
import { logger, scopedLog, LogContext } from '../../utils';
import { updateFrameStreamSettings, getFrameStreamSettings } from '../../frame-streaming';
import type { StreamSettingsRequest, StreamSettingsResponse } from './types';

/**
 * Update stream settings endpoint
 *
 * POST /session/:id/record/stream-settings
 *
 * Updates stream settings for an active session. Quality and FPS can be
 * updated immediately. Scale changes require session restart.
 */
export async function handleStreamSettings(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    sessionManager.getSession(sessionId);

    const body = await parseJsonBody(req, config);
    const request = body as unknown as StreamSettingsRequest;
    const invalidNumber = ['quality', 'fps'].some((key) =>
      body[key] !== undefined && (typeof body[key] !== 'number' || !Number.isFinite(body[key])));
    if (invalidNumber || (request.quality !== undefined && !Number.isInteger(request.quality)) ||
        (request.perfMode !== undefined && typeof request.perfMode !== 'boolean') ||
        (request.scale !== undefined && request.scale !== 'css' && request.scale !== 'device')) {
      sendJson(res, 400, { error: 'INVALID_STREAM_SETTINGS', message: 'Quality must be an integer, FPS a finite number, perfMode a boolean, and scale css or device.' });
      return;
    }

    const currentSettings = getFrameStreamSettings(sessionId);
    if (!currentSettings) {
      logger.info(scopedLog(LogContext.RECORDING, 'stream settings update - no active stream'), {
        sessionId,
        requestedQuality: request.quality,
        requestedFps: request.fps,
      });

      const response: StreamSettingsResponse = {
        session_id: sessionId,
        quality: request.quality ?? 55,
        fps: request.fps ?? 30,
        current_fps: 0,
        scale: request.scale ?? 'css',
        is_streaming: false,
        updated: false,
        perf_mode: request.perfMode ?? false,
      };
      sendJson(res, 200, response);
      return;
    }

    let scaleWarning: string | undefined;
    if (request.scale !== undefined && request.scale !== currentSettings.scale) {
      scaleWarning = `Scale cannot be changed mid-session. Current: ${currentSettings.scale}, Requested: ${request.scale}. Restart session to change scale.`;
      logger.info(scopedLog(LogContext.RECORDING, 'stream settings - scale change rejected'), {
        sessionId,
        currentScale: currentSettings.scale,
        requestedScale: request.scale,
      });
    }

    const updated = await updateFrameStreamSettings(sessionId, {
      quality: request.quality,
      fps: request.fps,
      perfMode: request.perfMode,
    });
    const newSettings = getFrameStreamSettings(sessionId);
    const response: StreamSettingsResponse = {
      session_id: sessionId,
      quality: newSettings?.quality ?? currentSettings.quality,
      fps: newSettings?.fps ?? currentSettings.fps,
      current_fps: newSettings?.currentFps ?? currentSettings.currentFps,
      scale: newSettings?.scale ?? currentSettings.scale,
      is_streaming: newSettings?.isStreaming ?? currentSettings.isStreaming,
      updated,
      scale_warning: scaleWarning,
      perf_mode: newSettings?.perfMode ?? currentSettings.perfMode,
    };
    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/record/stream-settings`);
  }
}
