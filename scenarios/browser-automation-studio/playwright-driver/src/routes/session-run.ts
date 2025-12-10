import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { HandlerRegistry } from '../handlers';
import type { Config } from '../config';
import type { Metrics } from '../utils/metrics';
import type { CompiledInstruction, ExecutedInstructionRecord } from '../types';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { captureScreenshot, captureDOMSnapshot, ConsoleLogCollector, NetworkCollector } from '../telemetry';
import { buildStepOutcome, toDriverOutcome } from '../outcome';
import { logger, scopedLog, LogContext, isRecentTimestamp } from '../utils';
import winston from 'winston';

/**
 * Idempotency key header name.
 * When provided, the driver will cache and return the same result for
 * duplicate requests with the same key (within TTL).
 */
const IDEMPOTENCY_KEY_HEADER = 'x-idempotency-key';

/**
 * Cached results for idempotent requests.
 * Key: idempotency key from header
 * Value: cached response data
 */
interface CachedResult {
  response: unknown;
  timestamp: number;
  sessionId: string;
  instructionKey: string;
}
const idempotencyCache: Map<string, CachedResult> = new Map();

/**
 * TTL for cached idempotent results (5 minutes).
 */
const IDEMPOTENCY_CACHE_TTL_MS = 300_000;

/**
 * Clean up expired entries from the idempotency cache.
 * Called periodically to prevent unbounded memory growth.
 */
function cleanupIdempotencyCache(): void {
  const now = Date.now();
  let expiredCount = 0;

  for (const [key, value] of idempotencyCache.entries()) {
    if (now - value.timestamp > IDEMPOTENCY_CACHE_TTL_MS) {
      idempotencyCache.delete(key);
      expiredCount++;
    }
  }

  if (expiredCount > 0) {
    logger.debug('idempotency cache cleanup', {
      expiredCount,
      remainingCount: idempotencyCache.size,
    });
  }
}

// Run cache cleanup every minute
setInterval(cleanupIdempotencyCache, 60_000).unref();

/**
 * Maximum number of executed instructions to track per session.
 * Prevents unbounded memory growth for long-running sessions.
 */
const MAX_EXECUTED_INSTRUCTIONS = 1000;

/**
 * Generate a unique key for instruction idempotency tracking.
 * Combines node_id and index to create a stable identifier.
 */
function getInstructionKey(instruction: CompiledInstruction): string {
  return `${instruction.node_id}:${instruction.index}`;
}

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
 *
 * Concurrency guard:
 * - Only one instruction can execute at a time per session
 * - Returns 409 Conflict if session is already executing
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

  // Check for idempotency key header for cached result lookup
  const idempotencyKey = req.headers[IDEMPOTENCY_KEY_HEADER] as string | undefined;

  // Idempotency: Check cache for existing result with this key
  if (idempotencyKey) {
    const cached = idempotencyCache.get(idempotencyKey);
    if (cached && isRecentTimestamp(cached.timestamp, IDEMPOTENCY_CACHE_TTL_MS)) {
      // Verify this cached result is for the same session
      if (cached.sessionId === sessionId) {
        logger.info(scopedLog(LogContext.INSTRUCTION, 'returning cached idempotent result'), {
          sessionId,
          idempotencyKey,
          instructionKey: cached.instructionKey,
          cacheAgeMs: Date.now() - cached.timestamp,
        });

        sendJson(res, 200, cached.response);
        return;
      } else {
        // Different session - log warning but proceed with new execution
        logger.warn(scopedLog(LogContext.INSTRUCTION, 'idempotency key reused for different session'), {
          sessionId,
          cachedSessionId: cached.sessionId,
          idempotencyKey,
          hint: 'Idempotency keys should be unique per session. Proceeding with new execution.',
        });
      }
    }
  }

  // Track whether we've entered the executing phase (for cleanup on error)
  let enteredExecutingPhase = false;

  try {
    // Get session and check current phase
    const session = sessionManager.getSession(sessionId);

    // Guard: Prevent concurrent execution on same session
    // This protects against race conditions where two requests arrive simultaneously
    if (session.phase === 'executing') {
      logger.warn(scopedLog(LogContext.INSTRUCTION, 'concurrent execution rejected'), {
        sessionId,
        phase: session.phase,
        hint: 'Another instruction is already executing on this session. Wait for it to complete.',
      });
      sendJson(res, 409, {
        error: {
          code: 'SESSION_BUSY',
          message: 'Session is already executing an instruction',
          kind: 'orchestration',
          retryable: true,
          hint: 'Wait for the current instruction to complete before sending another',
        },
      });
      return;
    }

    // Update phase to executing
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

    // Idempotency: Check if this instruction was already executed
    // This handles replay scenarios where the same instruction is sent again
    const instructionKey = getInstructionKey(instruction);
    const previousExecution = session.executedInstructions?.get(instructionKey);

    if (previousExecution) {
      // Instruction was already executed - check if we should return cached result
      // We log but still re-execute to ensure fresh telemetry data
      // The logging helps detect unintentional retries
      logger.info(scopedLog(LogContext.INSTRUCTION, 'replay detected'), {
        sessionId,
        type: instruction.type,
        stepIndex: instruction.index,
        nodeId: instruction.node_id,
        previousSuccess: previousExecution.success,
        previousExecutedAt: previousExecution.executedAt.toISOString(),
        hint: 'Instruction was previously executed; re-executing for fresh state',
      });
    }

    // Log instruction start with context for debugging
    // Use consistent LogContext prefix for easy filtering
    logger.info(scopedLog(LogContext.INSTRUCTION, 'executing'), {
      sessionId,
      type: instruction.type,
      stepIndex: instruction.index,
      nodeId: instruction.node_id,
      instructionCount: session.instructionCount,
      isReplay: !!previousExecution,
      // Include key params for debugging without exposing sensitive data
      selector: (instruction.params as Record<string, unknown>).selector,
      url: (instruction.params as Record<string, unknown>).url,
    });

    // Get handler
    const handler = handlerRegistry.getHandler(instruction);

    // Setup telemetry collectors
    // Temporal hardening: These collectors attach event listeners to the page.
    // They MUST be disposed after use to prevent memory leaks and stale handlers.
    const consoleCollector = new ConsoleLogCollector(session.page, config.telemetry.console.maxEntries);
    const networkCollector = new NetworkCollector(session.page, config.telemetry.network.maxEvents);

    // Declare variables that need to survive the try/finally block
    let result;
    let instructionDuration: number;
    let consoleLogs: ReturnType<typeof consoleCollector.getAndClear> | undefined;
    let networkEvents: ReturnType<typeof networkCollector.getAndClear> | undefined;

    try {
      // Execute handler
      const instructionStart = Date.now();
      result = await handler.execute(instruction, {
        page: session.page,
        context: session.context,
        config,
        logger: appLogger,
        metrics: appMetrics,
        sessionId,
      });
      instructionDuration = Date.now() - instructionStart;

      // Capture collector data BEFORE disposing (must happen in try block)
      // Handler may have already provided these, so only collect if not present
      consoleLogs = result.consoleLogs || (config.telemetry.console.enabled
        ? consoleCollector.getAndClear()
        : undefined);

      networkEvents = result.networkEvents || (config.telemetry.network.enabled
        ? networkCollector.getAndClear()
        : undefined);
    } finally {
      // Temporal hardening: Always dispose collectors to remove event listeners
      // This prevents memory leaks and stale handlers if execution fails or succeeds
      consoleCollector.dispose();
      networkCollector.dispose();
    }

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

    // Capture remaining telemetry (if not already captured by handler)
    const screenshot = result.screenshot || (config.telemetry.screenshot.enabled
      ? await captureScreenshot(session.page, config)
      : undefined);

    const domSnapshot = result.domSnapshot || (config.telemetry.dom.enabled
      ? await captureDOMSnapshot(session.page, config)
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

    // Idempotency: Record instruction execution for replay detection
    if (session.executedInstructions) {
      // Enforce maximum tracking size to prevent unbounded memory growth
      if (session.executedInstructions.size >= MAX_EXECUTED_INSTRUCTIONS) {
        // Evict oldest entry (first in map iteration order)
        const firstKey = session.executedInstructions.keys().next().value;
        if (firstKey) {
          session.executedInstructions.delete(firstKey);
          logger.debug(scopedLog(LogContext.INSTRUCTION, 'evicted old instruction from tracking'), {
            sessionId,
            evictedKey: firstKey,
            maxTracked: MAX_EXECUTED_INSTRUCTIONS,
          });
        }
      }

      const executionRecord: ExecutedInstructionRecord = {
        key: instructionKey,
        executedAt: completedAt,
        success: result.success,
        // Don't cache full outcome to save memory - can be regenerated if needed
      };
      session.executedInstructions.set(instructionKey, executionRecord);
    }

    // Return session to ready state (or recording if recording was active)
    const nextPhase = session.recordingController?.isRecording() ? 'recording' : 'ready';
    sessionManager.setSessionPhase(sessionId, nextPhase);

    // Convert to driver wire format (flat fields for screenshot/dom)
    const screenshotData = screenshot as { base64?: string; media_type?: string; width?: number; height?: number } | undefined;
    const driverOutcome = toDriverOutcome(outcome, screenshotData, domSnapshot);

    // Cache result if idempotency key was provided
    if (idempotencyKey) {
      idempotencyCache.set(idempotencyKey, {
        response: driverOutcome,
        timestamp: Date.now(),
        sessionId,
        instructionKey,
      });

      logger.debug(scopedLog(LogContext.INSTRUCTION, 'cached idempotent result'), {
        sessionId,
        idempotencyKey,
        instructionKey,
        cacheSize: idempotencyCache.size,
      });
    }

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
