import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import { isOperational } from '../session';
import type { Config } from '../config';
import type { StartSessionRequest, StartSessionResponse, SessionSpec } from '../types';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { InvalidInstructionError, ResourceLimitError, PlaywrightDriverError, SessionNotFoundError, logger } from '../utils';
import { startFrameStreaming } from '../frame-streaming';
import type { FaultController } from '../fault-control';

/**
 * Start session endpoint
 *
 * POST /session/start
 *
 * Response includes:
 * - session_id: Unique session identifier
 * - phase: Current session phase ('ready' for new sessions)
 * - created_at: ISO 8601 timestamp
 * - reused: Whether an existing session was reused (only true for reuse_mode != 'fresh')
 */
export async function handleSessionStart(
  req: IncomingMessage,
  res: ServerResponse,
  sessionManager: SessionManager,
  config: Config,
  faultController?: FaultController
): Promise<void> {
  try {
    const body = await parseJsonBody(req, config);
    const request = body as unknown as StartSessionRequest;
    const options = request.session_options;

    // Validate required fields
    if (!request.execution_id || typeof request.execution_id !== 'string') {
      throw new InvalidInstructionError(
        'Missing or invalid execution_id: must be a non-empty string',
        {
          field: 'execution_id',
          received: typeof request.execution_id,
        }
      );
    }
    if (!request.workflow_id || typeof request.workflow_id !== 'string') {
      throw new InvalidInstructionError(
        'Missing or invalid workflow_id: must be a non-empty string',
        {
          field: 'workflow_id',
          received: typeof request.workflow_id,
        }
      );
    }
    if (!options || typeof options !== 'object' || Array.isArray(options)) {
      throw new InvalidInstructionError('Missing or invalid session_options: must be an object');
    }
    if (!options.viewport || typeof options.viewport !== 'object') {
      throw new InvalidInstructionError(
        'Missing or invalid session_options.viewport: must be an object with width and height',
        {
          field: 'session_options.viewport',
          received: typeof options.viewport,
        }
      );
    }
    if (typeof options.viewport.width !== 'number' || options.viewport.width <= 0) {
      throw new InvalidInstructionError('Invalid session_options.viewport.width: must be a positive number', {
        field: 'session_options.viewport.width',
        received: options.viewport.width,
      });
    }
    if (typeof options.viewport.height !== 'number' || options.viewport.height <= 0) {
      throw new InvalidInstructionError('Invalid session_options.viewport.height: must be a positive number', {
        field: 'session_options.viewport.height',
        received: options.viewport.height,
      });
    }

    // Validate reuse_mode if provided
    const validReuseModes = ['fresh', 'clean', 'reuse'];
    if (typeof options.reuse_mode !== 'string' || !validReuseModes.includes(options.reuse_mode)) {
      throw new InvalidInstructionError(
        `Invalid reuse_mode: must be one of ${validReuseModes.join(', ')}`,
        {
          field: 'session_options.reuse_mode',
          received: options.reuse_mode,
          valid: validReuseModes,
        }
      );
    }

    if (options.frame_scale !== 'css' && options.frame_scale !== 'device') {
      throw new InvalidInstructionError('Invalid session_options.frame_scale: must be css or device', {
        field: 'session_options.frame_scale',
        received: options.frame_scale,
      });
    }

    if (
      requiresArtifactRoot(options.required_capabilities) &&
      !options.artifact_paths?.root?.trim()
    ) {
      throw new InvalidInstructionError(
        'session_options.artifact_paths.root is required when recording video/trace/HAR artifacts',
        {
          field: 'session_options.artifact_paths.root',
          required_for: options.required_capabilities,
        }
      );
    }

    const spec: SessionSpec = {
      execution_id: request.execution_id,
      workflow_id: request.workflow_id,
      ...{ ...options, browser_profile: interactiveProfileDefault(options.labels, options.browser_profile) },
      reuse_mode: options.reuse_mode as 'fresh' | 'clean' | 'reuse',
    };

    const drillToken = typeof req.headers['x-playwright-drill-token'] === 'string' ? req.headers['x-playwright-drill-token'] : undefined;
    if (faultController?.consume(drillToken, 'driver_unavailable')) {
      throw new PlaywrightDriverError('controlled driver-unavailable drill outcome', 'DRILL_DRIVER_UNAVAILABLE');
    }
    if (faultController && faultController.capacityReserved(drillToken) > 0 && faultController.consume(drillToken, 'capacity_lease')) {
      throw new ResourceLimitError('controlled capacity lease rejected session admission', { drill: true });
    }
    // Start session - returns session info including whether it was reused and actual viewport
    const { sessionId, leaseId, reused, createdAt, actualViewport } =
      await sessionManager.startSession(spec);

    if (faultController?.consume(drillToken, 'fail_after_session_registration')) {
      await sessionManager.forceCloseSession(sessionId);
      throw new PlaywrightDriverError('controlled failure after session registration; session was reconciled', 'DRILL_SESSION_REGISTRATION_FAILURE');
    }

    const frameStreaming = options.frame_streaming;
    if (frameStreaming?.url) {
      // Readiness may finish after release, reuse or close. Every page lookup
      // belongs to this immutable lease, including lookups by the live stream.
      const executionId = spec.execution_id;
      const provider = {
        getSession: (id: string) => {
          const session = sessionManager.getSessionForLease(id, executionId, leaseId);
          if (!isOperational(session.phase)) throw new SessionNotFoundError(id);
          return session;
        },
      };
      void sessionManager.waitForPipelineReady(sessionId, 5000).then((ready) => {
        if (!ready) return;
        const session = provider.getSession(sessionId);
        startFrameStreaming(sessionId, provider, {
          streamUrl: frameStreaming.url,
          streamKind: 'execution',
          quality: frameStreaming.quality,
          fps: frameStreaming.fps,
          scale: session.spec.frame_scale!,
        });
      }).catch((error: unknown) => {
        logger.debug('Deferred frame preview did not start', { sessionId, error: String(error) });
      });
    }

    const current = sessionManager.getSessionForLease(sessionId, spec.execution_id, leaseId);
    const activePageId = current.pageBindings.getId(current.page);
    if (!isOperational(current.phase) || !activePageId) throw new SessionNotFoundError(sessionId);

    const response: StartSessionResponse = {
      session_id: sessionId,
      last_instruction_sequence: current.lastInstructionSequence,
      active_page_id: activePageId,
      lease_id: leaseId,
      phase: 'ready',
      created_at: createdAt.toISOString(),
      reused: reused || undefined, // Only include if true
      actual_viewport: actualViewport, // Report actual Playwright viewport
      audio_device_evidence: current.audioDeviceEvidence,
    };

    sendJson(res, 200, response);
  } catch (error) {
    sendError(res, error as Error, '/session/start');
  }
}

export const interactiveProfileDefault = (labels?: SessionSpec['labels'], profile?: SessionSpec['browser_profile']): SessionSpec['browser_profile'] => profile ?? (labels?.mode === 'recording' ? { preset: 'stealth' } : undefined);

function requiresArtifactRoot(
  capabilities?: StartSessionRequest['session_options']['required_capabilities']
): boolean {
  if (!capabilities) {
    return false;
  }
  return Boolean(
    capabilities.video ||
    capabilities.har ||
    capabilities.tracing ||
    capabilities.performance_trace ||
    capabilities.accessibility
  );
}
