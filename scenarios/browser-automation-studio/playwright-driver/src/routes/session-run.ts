/**
 * Session Run Route Handler
 *
 * POST /session/:id/run - Executes a single browser automation instruction.
 *
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ REQUEST FLOW:                                                           │
 * │                                                                         │
 * │  Request ──▶ Idempotency Check ──▶ Session Lock ──▶ Validate           │
 * │                    │                    │              │                │
 * │                    │ (cached?)          │ (busy?)      │                │
 * │                    ▼                    ▼              ▼                │
 * │               Return cached        409 Conflict   400 Bad Request      │
 * │                                                                         │
 * │  ──▶ Replay Check ──▶ Execute ──▶ Collect Telemetry ──▶ Build Outcome  │
 * │           │              │                                   │          │
 * │           │ (seen?)      │                                   │          │
 * │           ▼              │                                   ▼          │
 * │      Return cached  ◀───┴───────────────────────────▶  Cache & Return   │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * CONCURRENCY: One instruction per session at a time (returns 409 if busy)
 * IDEMPOTENCY: x-idempotency-key header enables safe retries
 */

import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { SessionState } from '../types/session';
import type { HandlerRegistry } from '../handlers';
import type { Config } from '../config';
import type { Metrics } from '../utils/metrics';
import type { ExecutedInstructionRecord } from '../types/session';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { TelemetryOrchestrator } from '../telemetry';
import { buildStepOutcome, toDriverOutcome } from '../outcome';
import {
  CompiledInstructionSchema,
  toHandlerInstruction,
  parseProtoLenient,
  type HandlerInstruction,
} from '../proto';
import { getIdempotencyCache } from '../infra';
import { logger, scopedLog, LogContext } from '../utils';
import winston from 'winston';

// =============================================================================
// Constants
// =============================================================================

const IDEMPOTENCY_KEY_HEADER = 'x-idempotency-key';
const MAX_EXECUTED_INSTRUCTIONS = 1000;

// =============================================================================
// Idempotency Cache (delegated to infra/idempotency-cache.ts)
// =============================================================================

/**
 * Clear idempotency cache entries for a session.
 * Called when a session is closed to prevent stale cache entries.
 */
export function clearSessionIdempotencyCache(sessionId: string): void {
  getIdempotencyCache().clearSession(sessionId);
}

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Create a unique key for an instruction based on nodeId and index.
 * Used to track which instructions have been executed in a session.
 */
function createInstructionKey(instruction: HandlerInstruction): string {
  return `${instruction.nodeId}:${instruction.index}`;
}

/**
 * Validate that a raw instruction object has all required fields.
 * Accepts both wire format (snake_case: node_id) and proto format (camelCase: nodeId).
 *
 * @returns Error message if invalid, null if valid
 */
function validateInstructionStructure(rawInstruction: unknown): string | null {
  if (!rawInstruction || typeof rawInstruction !== 'object') {
    return 'Missing or invalid instruction: must be an object';
  }
  const inst = rawInstruction as Record<string, unknown>;
  if (typeof inst.index !== 'number') return 'Missing or invalid instruction.index: must be a number';
  // Accept both node_id (wire format) and nodeId (proto format)
  const nodeId = inst.node_id ?? inst.nodeId;
  if (!nodeId || typeof nodeId !== 'string') return 'Missing or invalid instruction.node_id: must be a non-empty string';
  if (!inst.type || typeof inst.type !== 'string') return 'Missing or invalid instruction.type: must be a non-empty string';
  if (!inst.params || typeof inst.params !== 'object') return 'Missing or invalid instruction.params: must be an object';
  return null;
}

/** Record an executed instruction in the session's tracking map. */
function recordExecutedInstruction(
  session: SessionState,
  instructionKey: string,
  driverOutcome: unknown,
  success: boolean,
  completedAt: Date
): void {
  if (!session.executedInstructions) return;

  // Enforce max size by evicting oldest
  if (session.executedInstructions.size >= MAX_EXECUTED_INSTRUCTIONS) {
    const firstKey = session.executedInstructions.keys().next().value;
    if (firstKey) {
      session.executedInstructions.delete(firstKey);
      logger.debug(scopedLog(LogContext.INSTRUCTION, 'evicted old instruction from tracking'), {
        sessionId: session.id,
        evictedKey: firstKey,
        maxTracked: MAX_EXECUTED_INSTRUCTIONS,
      });
    }
  }

  session.executedInstructions.set(instructionKey, {
    key: instructionKey,
    executedAt: completedAt,
    success,
    cachedOutcome: driverOutcome,
  } as ExecutedInstructionRecord);
}

/**
 * Run instruction endpoint
 *
 * POST /session/:id/run
 *
 * Executes a single instruction in the browser session.
 *
 * Flow: ready -> executing -> ready
 * Concurrency: One instruction per session (returns 409 if busy)
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
  const idempotencyKey = req.headers[IDEMPOTENCY_KEY_HEADER] as string | undefined;

  // Fast path: Return cached result if available
  const idempotencyCache = getIdempotencyCache();
  const cachedResponse = idempotencyKey ? idempotencyCache.lookup(idempotencyKey, sessionId) : null;
  if (cachedResponse) {
    sendJson(res, 200, cachedResponse);
    return;
  }

  let enteredExecutingPhase = false;

  try {
    const session = sessionManager.getSession(sessionId);

    // Guard: Prevent concurrent execution
    if (session.phase === 'executing') {
      logger.warn(scopedLog(LogContext.INSTRUCTION, 'concurrent execution rejected'), { sessionId, phase: session.phase });
      sendJson(res, 409, {
        error: { code: 'SESSION_BUSY', message: 'Session is already executing an instruction', kind: 'orchestration', retryable: true },
      });
      return;
    }

    sessionManager.setSessionPhase(sessionId, 'executing');
    enteredExecutingPhase = true;

    // Parse and validate instruction
    const body = await parseJsonBody(req, config);
    const rawInstruction = (body as Record<string, unknown>).instruction;
    const validationError = validateInstructionStructure(rawInstruction);
    if (validationError) {
      sessionManager.setSessionPhase(sessionId, 'ready');
      sendJson(res, 400, {
        error: { code: 'INVALID_INSTRUCTION', message: validationError, kind: 'orchestration', retryable: false },
      });
      return;
    }

    // Parse with proto schema and convert to handler-friendly format
    const protoInstruction = parseProtoLenient(CompiledInstructionSchema, rawInstruction);
    const instruction = toHandlerInstruction(protoInstruction);
    const instructionKey = createInstructionKey(instruction);

    // Check session-level instruction cache (replay detection)
    const previousExecution = session.executedInstructions?.get(instructionKey);
    if (previousExecution?.cachedOutcome) {
      logger.info(scopedLog(LogContext.INSTRUCTION, 'returning cached replay result'), {
        sessionId, type: instruction.type, stepIndex: instruction.index, nodeId: instruction.nodeId,
      });
      sessionManager.setSessionPhase(sessionId, session.recordingController?.isRecording() ? 'recording' : 'ready');
      if (idempotencyKey) {
        idempotencyCache.store(idempotencyKey, sessionId, instructionKey, previousExecution.cachedOutcome);
      }
      sendJson(res, 200, previousExecution.cachedOutcome);
      return;
    }

    if (previousExecution) {
      logger.info(scopedLog(LogContext.INSTRUCTION, 'replay detected (no cached outcome)'), {
        sessionId, type: instruction.type, stepIndex: instruction.index,
      });
    }

    logger.info(scopedLog(LogContext.INSTRUCTION, 'executing'), {
      sessionId, type: instruction.type, stepIndex: instruction.index, nodeId: instruction.nodeId,
      instructionCount: session.instructionCount, isReplay: !!previousExecution,
      selector: instruction.params.selector,
      url: instruction.params.url,
    });

    // Execute instruction with telemetry orchestration
    const handler = handlerRegistry.getHandler(instruction);
    const telemetryOrchestrator = new TelemetryOrchestrator(session.page, config);
    telemetryOrchestrator.start();

    let result: HandlerResult;
    let instructionDuration: number;
    try {
      const instructionStart = Date.now();
      const handlerContext: HandlerContext = {
        page: session.page, context: session.context, config, logger: appLogger, metrics: appMetrics, sessionId,
      };
      result = await handler.execute(instruction, handlerContext);
      instructionDuration = Date.now() - instructionStart;
    } finally {
      // Telemetry orchestrator disposed after collecting
    }

    sessionManager.incrementInstructionCount(sessionId);
    recordMetrics(appMetrics, instruction.type, result, instructionDuration);

    // Collect telemetry via orchestrator (consolidates screenshot, DOM, console, network)
    const telemetry = await telemetryOrchestrator.collectForStep(result);
    telemetryOrchestrator.dispose();

    // Build and convert outcome
    const completedAt = new Date();
    const outcome = buildStepOutcome({
      instruction, result, startedAt, completedAt, finalUrl: session.page.url(),
      screenshot: telemetry.screenshot,
      domSnapshot: telemetry.domSnapshot,
      consoleLogs: telemetry.consoleLogs,
      networkEvents: telemetry.networkEvents,
    });

    logger.info(scopedLog(LogContext.INSTRUCTION, result.success ? 'completed' : 'failed'), {
      sessionId, type: instruction.type, stepIndex: instruction.index, success: result.success,
      durationMs: outcome.durationMs, finalUrl: session.page.url(), instructionCount: session.instructionCount,
      ...(result.error && { errorCode: result.error.code, errorKind: result.error.kind, errorMessage: result.error.message }),
    });

    const driverOutcome = toDriverOutcome(outcome, telemetry.screenshot, telemetry.domSnapshot);

    // Record for replay detection and cache
    recordExecutedInstruction(session, instructionKey, driverOutcome, result.success, completedAt);
    sessionManager.setSessionPhase(sessionId, session.recordingController?.isRecording() ? 'recording' : 'ready');
    if (idempotencyKey) {
      idempotencyCache.store(idempotencyKey, sessionId, instructionKey, driverOutcome);
    }

    sendJson(res, 200, driverOutcome);
  } catch (error) {
    if (enteredExecutingPhase) {
      try { sessionManager.setSessionPhase(sessionId, 'ready'); } catch { /* Session may be closed */ }
    }
    sendError(res, error as Error, `/session/${sessionId}/run`);
    appMetrics.instructionErrors.inc({ type: 'unknown', error_kind: 'engine' });
  }
}

// =============================================================================
// Execution Helpers
// =============================================================================

import type { HandlerContext } from '../handlers/base';
import type { HandlerResult } from '../outcome';

function recordMetrics(
  appMetrics: Metrics,
  instructionType: string,
  result: HandlerResult,
  instructionDuration: number
): void {
  appMetrics.instructionDuration.observe({ type: instructionType, success: String(result.success) }, instructionDuration);
  if (!result.success) {
    appMetrics.instructionErrors.inc({ type: instructionType, error_kind: result.error?.kind || 'unknown' });
  }
}
