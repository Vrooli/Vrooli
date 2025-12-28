/**
 * Session AI Navigate Route Handler
 *
 * POST /session/:id/ai-navigate - Starts AI-driven browser navigation.
 *
 * This handler initiates a vision-based AI agent that navigates the browser
 * to achieve the user's goal. The navigation runs asynchronously, emitting
 * step events to the callback URL as it progresses.
 *
 * REQUEST:
 * {
 *   prompt: string;           // User's goal (e.g., "Order chicken from the menu")
 *   model: string;            // Model ID (e.g., "qwen3-vl-30b")
 *   api_key: string;          // API key for the model provider
 *   max_steps?: number;       // Maximum steps (default: 20)
 *   callback_url: string;     // URL to POST step events to
 * }
 *
 * RESPONSE (immediate):
 * {
 *   navigation_id: string;    // Correlation ID for this navigation session
 *   status: "started";        // Always "started" on success
 * }
 *
 * STEP EVENTS (POSTed to callback_url):
 * See NavigationStep type for full schema.
 *
 * ABORT:
 * POST /session/:id/ai-navigate/abort to abort in-progress navigation.
 */

import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { Config } from '../config';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { logger } from '../utils';
import { v4 as uuidv4 } from 'uuid';

// AI modules
import {
  createVisionAgent,
  type NavigationConfig,
  type NavigationStep,
  type VisionAgentDeps,
} from '../ai/vision-agent';
import { createVisionClient, isModelSupported, getSupportedModelIds } from '../ai/vision-client/factory';
import { createScreenshotCapture, createElementAnnotator } from '../ai/screenshot';
import { createActionExecutor } from '../ai/action';
import { createCallbackEmitter, emitNavigationComplete, type NavigationCompleteEvent } from '../ai/emitter';
import { BEHAVIOR_SETTINGS_KEY } from '../session/context-builder';
import type { BehaviorSettings } from '../types/browser-profile';

/**
 * Request body for AI navigation.
 */
interface AINavigateRequest {
  prompt: string;
  model: string;
  api_key: string;
  max_steps?: number;
  callback_url: string;
}

/**
 * Response for AI navigation start.
 */
interface AINavigateResponse {
  navigation_id: string;
  status: 'started';
  model: string;
  max_steps: number;
}

/**
 * Track active navigations per session (to support abort).
 */
const activeNavigations = new Map<string, { agent: ReturnType<typeof createVisionAgent>; navigationId: string }>();

/**
 * Handle POST /session/:id/ai-navigate
 *
 * Starts AI-driven navigation for the session.
 */
export async function handleSessionAINavigate(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  _config: Config
): Promise<void> {
  // Validate session exists
  const session = sessionManager.getSession(sessionId);
  if (!session) {
    sendError(res, new Error(`Session not found: ${sessionId}`), '/session/:id/ai-navigate');
    return;
  }

  // Check session phase
  if (session.phase !== 'ready' && session.phase !== 'recording') {
    sendError(
      res,
      new Error(`Session is in ${session.phase} phase. AI navigation requires 'ready' or 'recording' phase.`),
      '/session/:id/ai-navigate'
    );
    return;
  }

  // Check if navigation already in progress
  const existing = activeNavigations.get(sessionId);
  if (existing) {
    sendJson(res, 409, {
      error: 'conflict',
      message: 'AI navigation already in progress for this session',
      navigation_id: existing.navigationId,
    });
    return;
  }

  // Parse request body
  let rawBody: Record<string, unknown>;
  try {
    rawBody = await parseJsonBody(req, _config);
  } catch (err) {
    sendError(res, err as Error, '/session/:id/ai-navigate');
    return;
  }
  const body = rawBody as unknown as AINavigateRequest;

  // Validate required fields
  if (!body.prompt || typeof body.prompt !== 'string') {
    sendJson(res, 400, { error: 'bad_request', message: 'prompt is required and must be a string' });
    return;
  }
  if (!body.model || typeof body.model !== 'string') {
    sendJson(res, 400, { error: 'bad_request', message: 'model is required and must be a string' });
    return;
  }
  if (!body.api_key || typeof body.api_key !== 'string') {
    sendJson(res, 400, { error: 'bad_request', message: 'api_key is required and must be a string' });
    return;
  }
  if (!body.callback_url || typeof body.callback_url !== 'string') {
    sendJson(res, 400, { error: 'bad_request', message: 'callback_url is required and must be a string' });
    return;
  }

  // Validate model is supported
  if (!isModelSupported(body.model)) {
    const supported = getSupportedModelIds();
    sendJson(res, 400, {
      error: 'bad_request',
      message: `Model '${body.model}' is not supported`,
      supported_models: supported,
    });
    return;
  }

  // Validate max_steps
  const maxSteps = body.max_steps ?? 20;
  if (maxSteps < 1 || maxSteps > 100) {
    sendJson(res, 400, { error: 'bad_request', message: 'max_steps must be between 1 and 100' });
    return;
  }

  // Generate navigation ID
  const navigationId = `nav_${uuidv4().replace(/-/g, '').slice(0, 12)}`;

  // Create logger wrapper for the agent
  const agentLogger = {
    debug: (msg: string, meta?: Record<string, unknown>) => logger.debug(msg, meta),
    info: (msg: string, meta?: Record<string, unknown>) => logger.info(msg, meta),
    warn: (msg: string, meta?: Record<string, unknown>) => logger.warn(msg, meta),
    error: (msg: string, meta?: Record<string, unknown>) => logger.error(msg, meta),
  };

  // Create dependencies
  let visionClient;
  try {
    visionClient = createVisionClient({
      modelId: body.model,
      apiKey: body.api_key,
      timeoutMs: 60000, // 60s timeout for vision API calls
      maxRetries: 2,
    });
  } catch (err) {
    sendJson(res, 400, {
      error: 'bad_request',
      message: `Failed to create vision client: ${err instanceof Error ? err.message : String(err)}`,
    });
    return;
  }

  const screenshotCapture = createScreenshotCapture();
  const annotator = createElementAnnotator();

  // Get behavior settings from the session's browser context for human-like typing
  const behaviorSettings = session.context
    ? (session.context as any)[BEHAVIOR_SETTINGS_KEY] as BehaviorSettings | undefined
    : undefined;

  const actionExecutor = createActionExecutor({
    behaviorSettings,
  });
  const stepEmitter = createCallbackEmitter();

  const deps: VisionAgentDeps = {
    visionClient,
    screenshotCapture,
    annotator,
    actionExecutor,
    stepEmitter,
    logger: agentLogger,
  };

  // Create vision agent
  const agent = createVisionAgent(deps);

  // Store in active navigations
  activeNavigations.set(sessionId, { agent, navigationId });

  // Navigation config
  const navConfig: NavigationConfig = {
    prompt: body.prompt,
    page: session.page,
    maxSteps,
    model: body.model,
    apiKey: body.api_key,
    callbackUrl: body.callback_url,
    navigationId,
    onStep: async (step: NavigationStep) => {
      // Additional step logging
      logger.debug('Navigation step completed', {
        navigationId,
        stepNumber: step.stepNumber,
        action: step.action.type,
        goalAchieved: step.goalAchieved,
      });
    },
  };

  // Send immediate response (navigation runs in background)
  const response: AINavigateResponse = {
    navigation_id: navigationId,
    status: 'started',
    model: body.model,
    max_steps: maxSteps,
  };

  sendJson(res, 202, response);

  // Start navigation in background
  logger.info('Starting AI navigation', {
    sessionId,
    navigationId,
    prompt: body.prompt,
    model: body.model,
    maxSteps,
    callbackUrl: body.callback_url,
  });

  // Run navigation asynchronously
  agent
    .navigate(navConfig)
    .then((result) => {
      logger.info('AI navigation completed', {
        sessionId,
        navigationId,
        status: result.status,
        totalSteps: result.totalSteps,
        totalTokens: result.totalTokens,
        durationMs: result.totalDurationMs,
      });

      // Emit completion event
      const completeEvent: NavigationCompleteEvent = {
        navigationId: result.navigationId,
        status: result.status,
        totalSteps: result.totalSteps,
        totalTokens: result.totalTokens,
        totalDurationMs: result.totalDurationMs,
        finalUrl: result.finalUrl,
        error: result.error,
        summary: result.summary,
      };
      emitNavigationComplete(body.callback_url, completeEvent).catch((err) => {
        logger.warn('Failed to emit navigation complete', {
          navigationId,
          error: err instanceof Error ? err.message : String(err),
        });
      });
    })
    .catch((err) => {
      logger.error('AI navigation failed unexpectedly', {
        sessionId,
        navigationId,
        error: err instanceof Error ? err.message : String(err),
      });

      // Emit failure event
      const failureEvent: NavigationCompleteEvent = {
        navigationId,
        status: 'failed',
        totalSteps: 0,
        totalTokens: 0,
        totalDurationMs: 0,
        finalUrl: session.page.url(),
        error: err instanceof Error ? err.message : String(err),
      };
      emitNavigationComplete(body.callback_url, failureEvent).catch(() => {});
    })
    .finally(() => {
      // Clean up
      activeNavigations.delete(sessionId);
    });
}

/**
 * Handle POST /session/:id/ai-navigate/abort
 *
 * Aborts in-progress AI navigation for the session.
 */
export async function handleSessionAINavigateAbort(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  _config: Config
): Promise<void> {
  // Validate session exists
  const session = sessionManager.getSession(sessionId);
  if (!session) {
    sendError(res, new Error(`Session not found: ${sessionId}`), '/session/:id/ai-navigate/abort');
    return;
  }

  // Check for active navigation
  const active = activeNavigations.get(sessionId);
  if (!active) {
    sendJson(res, 404, {
      error: 'not_found',
      message: 'No AI navigation in progress for this session',
    });
    return;
  }

  // Abort navigation
  active.agent.abort();

  logger.info('AI navigation abort requested', {
    sessionId,
    navigationId: active.navigationId,
  });

  sendJson(res, 200, {
    status: 'aborting',
    navigation_id: active.navigationId,
    message: 'Abort signal sent. Navigation will stop after current step completes.',
  });
}

/**
 * Handle POST /session/:id/ai-navigate/resume
 *
 * Resumes AI navigation after human intervention is complete.
 */
export async function handleSessionAINavigateResume(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  _config: Config
): Promise<void> {
  // Validate session exists
  const session = sessionManager.getSession(sessionId);
  if (!session) {
    sendError(res, new Error(`Session not found: ${sessionId}`), '/session/:id/ai-navigate/resume');
    return;
  }

  // Check for active navigation
  const active = activeNavigations.get(sessionId);
  if (!active) {
    sendJson(res, 404, {
      error: 'not_found',
      message: 'No AI navigation in progress for this session',
    });
    return;
  }

  // Check if agent is paused
  if (!active.agent.isPaused()) {
    sendJson(res, 409, {
      error: 'conflict',
      message: 'Navigation is not paused. Cannot resume.',
      navigation_id: active.navigationId,
    });
    return;
  }

  // Resume navigation
  active.agent.resume();

  logger.info('AI navigation resumed after human intervention', {
    sessionId,
    navigationId: active.navigationId,
  });

  sendJson(res, 200, {
    status: 'resumed',
    navigation_id: active.navigationId,
    message: 'Navigation resumed. Will continue from where it paused.',
  });
}

/**
 * Handle GET /session/:id/ai-navigate/status
 *
 * Returns status of AI navigation for the session.
 */
export async function handleSessionAINavigateStatus(
  _req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  _config: Config
): Promise<void> {
  // Validate session exists
  const session = sessionManager.getSession(sessionId);
  if (!session) {
    sendError(res, new Error(`Session not found: ${sessionId}`), '/session/:id/ai-navigate/status');
    return;
  }

  // Check for active navigation
  const active = activeNavigations.get(sessionId);
  if (!active) {
    sendJson(res, 200, {
      status: 'idle',
      message: 'No AI navigation in progress',
    });
    return;
  }

  const isPaused = active.agent.isPaused();
  sendJson(res, 200, {
    status: isPaused ? 'awaiting_human' : 'navigating',
    navigation_id: active.navigationId,
    is_navigating: active.agent.isNavigating(),
    is_paused: isPaused,
  });
}

/**
 * Handle GET /ai/models
 *
 * Returns list of supported vision models.
 */
export async function handleListAIModels(
  _req: IncomingMessage,
  res: ServerResponse
): Promise<void> {
  const models = getSupportedModelIds();
  sendJson(res, 200, {
    models,
    count: models.length,
  });
}
