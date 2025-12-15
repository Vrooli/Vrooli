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
import { captureScreenshot, captureDOMSnapshot, ConsoleLogCollector, NetworkCollector } from '../telemetry';
import { buildStepOutcome, toDriverOutcome } from '../outcome';
import {
  CompiledInstructionSchema,
  toHandlerInstruction,
  parseProtoLenient,
  type HandlerInstruction,
} from '../proto';
import { logger, scopedLog, LogContext, isRecentTimestamp } from '../utils';
import winston from 'winston';

// =============================================================================
// Constants
// =============================================================================

const IDEMPOTENCY_KEY_HEADER = 'x-idempotency-key';
const IDEMPOTENCY_CACHE_TTL_MS = 300_000; // 5 minutes
const MAX_IDEMPOTENCY_CACHE_SIZE = 10_000;
const MAX_EXECUTED_INSTRUCTIONS = 1000;

// =============================================================================
// Idempotency Cache
// =============================================================================

interface CachedResult {
  response: unknown;
  timestamp: number;
  sessionId: string;
  instructionKey: string;
}

const idempotencyCache = new Map<string, CachedResult>();

// Cleanup runs every minute
setInterval(cleanupIdempotencyCache, 60_000).unref();

function cleanupIdempotencyCache(): void {
  const now = Date.now();
  let expiredCount = 0;

  for (const [key, value] of idempotencyCache.entries()) {
    if (now - value.timestamp > IDEMPOTENCY_CACHE_TTL_MS) {
      idempotencyCache.delete(key);
      expiredCount++;
    }
  }

  enforceMaxCacheSize();

  if (expiredCount > 0) {
    logger.debug('idempotency cache cleanup', { expiredCount, remainingCount: idempotencyCache.size });
  }
}

function enforceMaxCacheSize(): void {
  if (idempotencyCache.size <= MAX_IDEMPOTENCY_CACHE_SIZE) return;

  const toEvict = idempotencyCache.size - MAX_IDEMPOTENCY_CACHE_SIZE;
  let evicted = 0;
  for (const key of idempotencyCache.keys()) {
    if (evicted >= toEvict) break;
    idempotencyCache.delete(key);
    evicted++;
  }
  logger.warn(scopedLog(LogContext.INSTRUCTION, 'idempotency cache size limit reached'), {
    maxSize: MAX_IDEMPOTENCY_CACHE_SIZE,
    evictedForSize: evicted,
  });
}

export function clearSessionIdempotencyCache(sessionId: string): void {
  let clearedCount = 0;
  for (const [key, value] of idempotencyCache.entries()) {
    if (value.sessionId === sessionId) {
      idempotencyCache.delete(key);
      clearedCount++;
    }
  }
  if (clearedCount > 0) {
    logger.debug(scopedLog(LogContext.SESSION, 'cleared session idempotency cache'), { sessionId, clearedCount });
  }
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
 * Look up a cached response from a previous identical request.
 *
 * The idempotency cache allows clients to safely retry requests - if the same
 * x-idempotency-key header is sent, we return the cached result instead of
 * re-executing the instruction.
 *
 * @returns Cached response if found and valid, null otherwise
 */
function lookupIdempotencyCache(
  idempotencyKey: string | undefined,
  sessionId: string
): unknown | null {
  if (!idempotencyKey) return null;

  const cached = idempotencyCache.get(idempotencyKey);
  if (!cached || !isRecentTimestamp(cached.timestamp, IDEMPOTENCY_CACHE_TTL_MS)) return null;

  // Security: Reject if key was used for a different session
  if (cached.sessionId !== sessionId) {
    logger.warn(scopedLog(LogContext.INSTRUCTION, 'idempotency key reused for different session'), {
      sessionId,
      cachedSessionId: cached.sessionId,
      idempotencyKey,
    });
    return null;
  }

  logger.info(scopedLog(LogContext.INSTRUCTION, 'returning cached idempotent result'), {
    sessionId,
    idempotencyKey,
    instructionKey: cached.instructionKey,
    cacheAgeMs: Date.now() - cached.timestamp,
  });

  return cached.response;
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

/**
 * Store a response in the idempotency cache for future identical requests.
 */
function storeInIdempotencyCache(
  idempotencyKey: string | undefined,
  sessionId: string,
  instructionKey: string,
  response: unknown
): void {
  if (!idempotencyKey) return;
  idempotencyCache.set(idempotencyKey, { response, timestamp: Date.now(), sessionId, instructionKey });
  logger.debug(scopedLog(LogContext.INSTRUCTION, 'cached idempotent result'), {
    sessionId,
    idempotencyKey,
    instructionKey,
    cacheSize: idempotencyCache.size,
  });
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
  const cachedResponse = lookupIdempotencyCache(idempotencyKey, sessionId);
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
      storeInIdempotencyCache(idempotencyKey, sessionId, instructionKey, previousExecution.cachedOutcome);
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

    // Execute instruction with telemetry
    const handler = handlerRegistry.getHandler(instruction);
    const { result, consoleLogs, networkEvents, instructionDuration } = await executeWithTelemetry(
      handler, instruction, session, config, appLogger, appMetrics, sessionId
    );

    sessionManager.incrementInstructionCount(sessionId);
    recordMetrics(appMetrics, instruction.type, result, instructionDuration);

    // Capture remaining telemetry
    const screenshot = result.screenshot || (config.telemetry.screenshot.enabled ? await captureScreenshot(session.page, config) : undefined);
    const domSnapshot = result.domSnapshot || (config.telemetry.dom.enabled ? await captureDOMSnapshot(session.page, config) : undefined);

    // Build and convert outcome
    const completedAt = new Date();
    const outcome = buildStepOutcome({
      instruction, result, startedAt, completedAt, finalUrl: session.page.url(),
      screenshot: screenshot as Parameters<typeof buildStepOutcome>[0]['screenshot'],
      domSnapshot, consoleLogs, networkEvents,
    });

    logger.info(scopedLog(LogContext.INSTRUCTION, result.success ? 'completed' : 'failed'), {
      sessionId, type: instruction.type, stepIndex: instruction.index, success: result.success,
      durationMs: outcome.durationMs, finalUrl: session.page.url(), instructionCount: session.instructionCount,
      ...(result.error && { errorCode: result.error.code, errorKind: result.error.kind, errorMessage: result.error.message }),
    });

    const screenshotData = screenshot as { base64?: string; media_type?: string; width?: number; height?: number } | undefined;
    const driverOutcome = toDriverOutcome(outcome, screenshotData, domSnapshot);

    // Record for replay detection and cache
    recordExecutedInstruction(session, instructionKey, driverOutcome, result.success, completedAt);
    sessionManager.setSessionPhase(sessionId, session.recordingController?.isRecording() ? 'recording' : 'ready');
    storeInIdempotencyCache(idempotencyKey, sessionId, instructionKey, driverOutcome);

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

import type { HandlerContext, InstructionHandler } from '../handlers/base';
import type { HandlerResult, ConsoleLogEntry, NetworkEvent } from '../outcome';

async function executeWithTelemetry(
  handler: InstructionHandler,
  instruction: HandlerInstruction,
  session: SessionState,
  config: Config,
  appLogger: winston.Logger,
  appMetrics: Metrics,
  sessionId: string
): Promise<{
  result: HandlerResult;
  consoleLogs: ConsoleLogEntry[] | undefined;
  networkEvents: NetworkEvent[] | undefined;
  instructionDuration: number;
}> {
  const consoleCollector = new ConsoleLogCollector(session.page, config.telemetry.console.maxEntries);
  const networkCollector = new NetworkCollector(session.page, config.telemetry.network.maxEvents);

  try {
    const instructionStart = Date.now();
    const handlerContext: HandlerContext = {
      page: session.page, context: session.context, config, logger: appLogger, metrics: appMetrics, sessionId,
    };
    const result = await handler.execute(instruction, handlerContext);
    const instructionDuration = Date.now() - instructionStart;

    const consoleLogs = result.consoleLogs || (config.telemetry.console.enabled ? consoleCollector.getAndClear() : undefined);
    const networkEvents = result.networkEvents || (config.telemetry.network.enabled ? networkCollector.getAndClear() : undefined);

    return { result, consoleLogs, networkEvents, instructionDuration };
  } finally {
    consoleCollector.dispose();
    networkCollector.dispose();
  }
}

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
