import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { HandlerRegistry } from '../handlers';
import type { Config } from '../config';
import type { Metrics } from '../utils/metrics';
import type { CompiledInstruction } from '../types';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { captureScreenshot, captureDOMSnapshot, ConsoleLogCollector, NetworkCollector } from '../telemetry';
import { buildStepOutcome, toDriverOutcome } from '../outcome';
import { logger, scopedLog, LogContext } from '../utils';
import winston from 'winston';

/**
 * Run instruction endpoint
 *
 * POST /session/:id/run
 *
 * Executes a single instruction in the browser session.
 * Updates session phase during execution for observability.
 *
 * Signal flow:
 * 1. instruction: executing - start of execution with key params
 * 2. instruction: completed - outcome with success/failure details
 *
 * Phase transitions:
 * - ready -> executing -> ready (success)
 * - ready -> executing -> ready (failure, instruction failed but session ok)
 */
export async function handleSessionRun(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  handlerRegistry: HandlerRegistry,
  config: Config,
  appLogger: winston.Logger,
  appMetrics: Metrics
): Promise<void> {
  const startedAt = new Date();

  // Track whether we've entered the executing phase (for cleanup on error)
  let enteredExecutingPhase = false;

  try {
    // Get session and update phase to executing
    const session = sessionManager.getSession(sessionId);
    sessionManager.setSessionPhase(sessionId, 'executing');
    enteredExecutingPhase = true;

    // Parse instruction
    const body = await parseJsonBody(req, config);
    const rawInstruction = (body as Record<string, unknown>).instruction;

    // Validate instruction structure before proceeding
    if (!rawInstruction || typeof rawInstruction !== 'object') {
      sessionManager.setSessionPhase(sessionId, 'ready');
      sendJson(res, 400, {
        error: {
          code: 'INVALID_INSTRUCTION',
          message: 'Missing or invalid instruction: must be an object',
          kind: 'orchestration',
          retryable: false,
          hint: 'Ensure the request body contains an "instruction" object with type, index, node_id, and params',
        },
      });
      return;
    }

    const instruction = rawInstruction as CompiledInstruction;

    // Validate required instruction fields
    if (typeof instruction.index !== 'number') {
      sessionManager.setSessionPhase(sessionId, 'ready');
      sendJson(res, 400, {
        error: {
          code: 'INVALID_INSTRUCTION',
          message: 'Missing or invalid instruction.index: must be a number',
          kind: 'orchestration',
          retryable: false,
        },
      });
      return;
    }
    if (!instruction.node_id || typeof instruction.node_id !== 'string') {
      sessionManager.setSessionPhase(sessionId, 'ready');
      sendJson(res, 400, {
        error: {
          code: 'INVALID_INSTRUCTION',
          message: 'Missing or invalid instruction.node_id: must be a non-empty string',
          kind: 'orchestration',
          retryable: false,
        },
      });
      return;
    }
    if (!instruction.type || typeof instruction.type !== 'string') {
      sessionManager.setSessionPhase(sessionId, 'ready');
      sendJson(res, 400, {
        error: {
          code: 'INVALID_INSTRUCTION',
          message: 'Missing or invalid instruction.type: must be a non-empty string',
          kind: 'orchestration',
          retryable: false,
        },
      });
      return;
    }
    if (!instruction.params || typeof instruction.params !== 'object') {
      sessionManager.setSessionPhase(sessionId, 'ready');
      sendJson(res, 400, {
        error: {
          code: 'INVALID_INSTRUCTION',
          message: 'Missing or invalid instruction.params: must be an object',
          kind: 'orchestration',
          retryable: false,
        },
      });
      return;
    }

    // Log instruction start with context for debugging
    // Use consistent LogContext prefix for easy filtering
    logger.info(scopedLog(LogContext.INSTRUCTION, 'executing'), {
      sessionId,
      type: instruction.type,
      stepIndex: instruction.index,
      nodeId: instruction.node_id,
      instructionCount: session.instructionCount,
      // Include key params for debugging without exposing sensitive data
      selector: (instruction.params as Record<string, unknown>).selector,
      url: (instruction.params as Record<string, unknown>).url,
    });

    // Get handler
    const handler = handlerRegistry.getHandler(instruction);

    // Setup telemetry collectors
    const consoleCollector = new ConsoleLogCollector(session.page, config.telemetry.console.maxEntries);
    const networkCollector = new NetworkCollector(session.page, config.telemetry.network.maxEvents);

    // Execute handler
    const instructionStart = Date.now();
    const result = await handler.execute(instruction, {
      page: session.page,
      context: session.context,
      config,
      logger: appLogger,
      metrics: appMetrics,
      sessionId,
    });
    const instructionDuration = Date.now() - instructionStart;

    // Increment instruction count
    sessionManager.incrementInstructionCount(sessionId);

    // Record metrics
    appMetrics.instructionDuration.observe(
      { type: instruction.type, success: String(result.success) },
      instructionDuration
    );

    if (!result.success) {
      appMetrics.instructionErrors.inc({
        type: instruction.type,
        error_kind: result.error?.kind || 'unknown',
      });
    }

    // Capture telemetry (if not already captured by handler)
    const screenshot = result.screenshot || (config.telemetry.screenshot.enabled
      ? await captureScreenshot(session.page, config)
      : undefined);

    const domSnapshot = result.domSnapshot || (config.telemetry.dom.enabled
      ? await captureDOMSnapshot(session.page, config)
      : undefined);

    const consoleLogs = result.consoleLogs || (config.telemetry.console.enabled
      ? consoleCollector.getAndClear()
      : undefined);

    const networkEvents = result.networkEvents || (config.telemetry.network.enabled
      ? networkCollector.getAndClear()
      : undefined);

    // Build step outcome using domain logic
    const completedAt = new Date();
    const outcome = buildStepOutcome({
      instruction,
      result,
      startedAt,
      completedAt,
      finalUrl: session.page.url(),
      screenshot: screenshot as Parameters<typeof buildStepOutcome>[0]['screenshot'],
      domSnapshot,
      consoleLogs,
      networkEvents,
    });

    // Log completion with outcome summary
    // This is the key signal for understanding instruction execution flow
    logger.info(scopedLog(LogContext.INSTRUCTION, result.success ? 'completed' : 'failed'), {
      sessionId,
      type: instruction.type,
      stepIndex: instruction.index,
      success: result.success,
      durationMs: outcome.duration_ms,
      finalUrl: session.page.url(),
      instructionCount: session.instructionCount,
      // Include failure details for debugging (if any)
      ...(result.error && {
        errorCode: result.error.code,
        errorKind: result.error.kind,
        errorMessage: result.error.message,
        retryable: result.error.retryable,
      }),
    });

    // Return session to ready state (or recording if recording was active)
    const nextPhase = session.recordingController?.isRecording() ? 'recording' : 'ready';
    sessionManager.setSessionPhase(sessionId, nextPhase);

    // Convert to driver wire format (flat fields for screenshot/dom)
    const screenshotData = screenshot as { base64?: string; media_type?: string; width?: number; height?: number } | undefined;
    const driverOutcome = toDriverOutcome(outcome, screenshotData, domSnapshot);

    sendJson(res, 200, driverOutcome);
  } catch (error) {
    // Return session to ready state on error (only if we entered executing phase)
    // This prevents trying to set phase on a session that doesn't exist (SessionNotFoundError)
    if (enteredExecutingPhase) {
      try {
        sessionManager.setSessionPhase(sessionId, 'ready');
      } catch {
        // Session may have been closed during execution - ignore
        logger.debug(scopedLog(LogContext.SESSION, 'could not reset phase on error'), {
          sessionId,
          hint: 'Session may have been closed during execution',
        });
      }
    }

    sendError(res, error as Error, `/session/${sessionId}/run`);

    // Record error metric
    appMetrics.instructionErrors.inc({
      type: 'unknown',
      error_kind: 'engine',
    });
  }
}
