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
 *
 * RESPONSIBILITY ZONES
 * ====================
 *
 * This handler orchestrates multiple concerns. Each zone is marked inline:
 *
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ ZONE                │ RESPONSIBILITY              │ COULD EXTRACT TO    │
 * ├─────────────────────┼─────────────────────────────┼─────────────────────┤
 * │ [PRESENTATION]      │ HTTP parsing, response      │ (stays in route)    │
 * │ [INFRASTRUCTURE]    │ Idempotency cache lookups   │ infra/              │
 * │ [COORDINATION]      │ Session phase management    │ execution/          │
 * │ [DOMAIN:VALIDATION] │ Instruction structure check │ execution/          │
 * │ [DOMAIN:EXECUTION]  │ Handler dispatch            │ execution/          │
 * │ [CROSS-CUTTING]     │ Telemetry collection        │ telemetry/          │
 * │ [DOMAIN:OUTCOME]    │ StepOutcome building        │ outcome/            │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * The handleSessionRun function is intentionally comprehensive - it shows
 * the full pipeline in one place. P1.1 refactoring extracts the core execution
 * logic into InstructionExecutor while keeping the route handler as thin
 * coordination layer.
 *
 * @see execution/instruction-executor.ts - Extracted execution pipeline (P1.1)
 */

import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import type { SessionState } from '../types/session';
import type { HandlerRegistry } from '../handlers';
import type { Config } from '../config';
import type { Metrics } from '../utils/metrics';
import type { ExecutedInstructionRecord } from '../types/session';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { getIdempotencyCache } from '../infra';
import {
  executeInstruction,
  validateInstruction,
  createInstructionKey,
  type ExecutionContext,
} from '../execution';
import { logger, scopedLog, LogContext } from '../utils';
import { MAX_EXECUTED_INSTRUCTIONS_PER_SESSION } from '../constants';
import winston from 'winston';

// =============================================================================
// Constants
// =============================================================================

const IDEMPOTENCY_KEY_HEADER = 'x-idempotency-key';

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
  if (session.executedInstructions.size >= MAX_EXECUTED_INSTRUCTIONS_PER_SESSION) {
    const firstKey = session.executedInstructions.keys().next().value;
    if (firstKey) {
      session.executedInstructions.delete(firstKey);
      logger.debug(scopedLog(LogContext.INSTRUCTION, 'evicted old instruction from tracking'), {
        sessionId: session.id,
        evictedKey: firstKey,
        maxTracked: MAX_EXECUTED_INSTRUCTIONS_PER_SESSION,
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
  const idempotencyKey = req.headers[IDEMPOTENCY_KEY_HEADER] as string | undefined;

  // ─────────────────────────────────────────────────────────────────────────
  // [INFRASTRUCTURE] Idempotency cache fast path
  // ─────────────────────────────────────────────────────────────────────────
  const idempotencyCache = getIdempotencyCache();
  const cachedResponse = idempotencyKey ? idempotencyCache.lookup(idempotencyKey, sessionId) : null;
  if (cachedResponse) {
    sendJson(res, 200, cachedResponse);
    return;
  }

  let enteredExecutingPhase = false;

  try {
    const session = sessionManager.getSession(sessionId);

    // ─────────────────────────────────────────────────────────────────────────
    // [COORDINATION] Session phase guard - prevent concurrent execution
    // ─────────────────────────────────────────────────────────────────────────
    if (session.phase === 'executing') {
      logger.warn(scopedLog(LogContext.INSTRUCTION, 'concurrent execution rejected'), { sessionId, phase: session.phase });
      sendJson(res, 409, {
        error: { code: 'SESSION_BUSY', message: 'Session is already executing an instruction', kind: 'orchestration', retryable: true },
      });
      return;
    }

    sessionManager.setSessionPhase(sessionId, 'executing');
    enteredExecutingPhase = true;

    // ─────────────────────────────────────────────────────────────────────────
    // [PRESENTATION] Parse request body
    // [DOMAIN:VALIDATION] Validate instruction structure (delegated to executor)
    // ─────────────────────────────────────────────────────────────────────────
    const body = await parseJsonBody(req, config);
    const rawInstruction = (body as Record<string, unknown>).instruction;
    const validationResult = validateInstruction(rawInstruction);
    if (!validationResult.valid) {
      sessionManager.setSessionPhase(sessionId, 'ready');
      sendJson(res, 400, {
        error: { code: validationResult.error.code, message: validationResult.error.message, kind: 'orchestration', retryable: false },
      });
      return;
    }

    const instruction = validationResult.instruction;
    const instructionKey = createInstructionKey(instruction);

    // ─────────────────────────────────────────────────────────────────────────
    // [INFRASTRUCTURE] Replay detection via session-level instruction cache
    // ─────────────────────────────────────────────────────────────────────────
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

    // ─────────────────────────────────────────────────────────────────────────
    // [DOMAIN:EXECUTION] Delegate to InstructionExecutor service
    // This handles: handler dispatch, telemetry, outcome building
    // ─────────────────────────────────────────────────────────────────────────
    const executionContext: ExecutionContext = {
      page: session.page,
      browserContext: session.context,
      config,
      logger: appLogger,
      metrics: appMetrics,
      sessionId,
    };

    const executionResult = await executeInstruction(instruction, executionContext, handlerRegistry);
    sessionManager.incrementInstructionCount(sessionId);

    const { driverOutcome, success } = executionResult;
    const completedAt = new Date();

    // ─────────────────────────────────────────────────────────────────────────
    // [INFRASTRUCTURE] Cache result for replay detection and idempotency
    // [COORDINATION] Reset session phase
    // [PRESENTATION] Send response
    // ─────────────────────────────────────────────────────────────────────────
    recordExecutedInstruction(session, instructionKey, driverOutcome, success, completedAt);
    sessionManager.setSessionPhase(sessionId, session.recordingController?.isRecording() ? 'recording' : 'ready');
    if (idempotencyKey) {
      idempotencyCache.store(idempotencyKey, sessionId, instructionKey, driverOutcome);
    }

    sendJson(res, 200, driverOutcome);
  } catch (error) {
    // ─────────────────────────────────────────────────────────────────────────
    // [COORDINATION] Error recovery - reset phase
    // [PRESENTATION] Error response
    // ─────────────────────────────────────────────────────────────────────────
    if (enteredExecutingPhase) {
      try { sessionManager.setSessionPhase(sessionId, 'ready'); } catch { /* Session may be closed */ }
    }
    sendError(res, error as Error, `/session/${sessionId}/run`);
    appMetrics.instructionErrors.inc({ type: 'unknown', error_kind: 'engine' });
  }
}

