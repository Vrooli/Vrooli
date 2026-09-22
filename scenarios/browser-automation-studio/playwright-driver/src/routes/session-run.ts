/** Execute one operation owned by the current session lease. */
import type { IncomingMessage, ServerResponse } from 'http';
import { createHash } from 'node:crypto';
import type { SessionManager } from '../session';
import type { SessionState, InstructionReceipt } from '../types/session';
import type { HandlerRegistry } from '../handlers';
import type { Config } from '../config';
import type { Metrics } from '../utils/metrics';
import { parseJsonBody, sendJson, sendError } from '../middleware';
import { executeInstruction, validateInstruction, type ExecutionContext } from '../execution';
import { buildStepOutcome, toDriverOutcome } from '../outcome';
import { InvalidInstructionError } from '../utils';
import { MAX_EXECUTED_INSTRUCTIONS_PER_SESSION } from '../constants';
import type winston from 'winston';

function withAudioCapability(outcome: unknown, session: SessionState): unknown {
  if (!outcome || typeof outcome !== 'object' || Array.isArray(outcome)) return outcome;
  return {
    ...outcome,
    audio_strategy: session.audioStrategy,
    host_audio_outcome: session.audioCapability?.outcome,
    host_audio_reason: session.audioCapability?.reason,
    audio_playback_failure: session.audioPlaybackFailure?.(),
  };
}

// Bind a transport operation to the whole JSON payload, independent of object
// key ordering. Arrays retain their order; aliases are distinct wire payloads.
function fingerprint(payload: unknown): string {
  const canonical = JSON.stringify(payload, (_key, value) =>
    value && typeof value === 'object' && !Array.isArray(value)
      ? Object.fromEntries(Object.keys(value).sort().map(key => [key, value[key]]))
      : value);
  return createHash('sha256').update(canonical).digest('hex');
}

function sendReceipt(res: ServerResponse, receipt: InstructionReceipt): void {
  res.statusCode = 200;
  res.setHeader('Content-Type', 'application/json');
  res.end(receipt.response);
}

/**
 * One in-flight operation per session. Retained receipts replay identical
 * requests; the lease highwater forbids re-executing evicted/reset operations.
 * This is an in-memory lease guarantee, not a process-restart guarantee.
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
  let executingSession: SessionState | undefined;
  try {
    // Reading yields: check ownership and phase after the full body arrives.
    const body = await parseJsonBody(req, config);
    const executionId = typeof body.execution_id === 'string' ? body.execution_id : '';
    const leaseId = typeof body.lease_id === 'string' ? body.lease_id : '';
    if (!executionId.trim() || !leaseId.trim()) {
      throw new InvalidInstructionError('execution_id and lease_id are required to run an instruction');
    }
    const validation = validateInstruction(body.instruction);
    if (!validation.valid) {
      sendJson(res, 400, { error: { ...validation.error, kind: 'orchestration', retryable: false } });
      return;
    }
    const session = sessionManager.getSessionForLease(sessionId, executionId, leaseId);
    const sequence = body.operation_sequence;
    if (typeof sequence !== 'number' || !Number.isSafeInteger(sequence) || sequence <= 0 ||
        typeof body.invocation_id !== 'string' || !body.invocation_id.trim() ||
        typeof body.attempt !== 'number' || !Number.isSafeInteger(body.attempt) || body.attempt <= 0) {
      throw new InvalidInstructionError('positive operation_sequence, invocation_id and positive attempt are required');
    }
    const header = req.headers['x-idempotency-key'];
    if (header !== undefined && header !== `${leaseId}:${sequence}`) {
      throw new InvalidInstructionError('X-Idempotency-Key must match lease_id:operation_sequence');
    }
    if (!sessionManager.canAcceptInstructions(sessionId) || !sessionManager.setSessionPhase(sessionId, 'executing')) {
      sendJson(res, 409, { error: { code: 'SESSION_BUSY', message: `Session cannot execute an instruction while ${session.phase}`, kind: 'orchestration', retryable: false } });
      return;
    }
    session.instructionInFlight = true;
    executingSession = session;
    const requestFingerprint = fingerprint({ invocation_id: body.invocation_id, attempt: body.attempt, instruction: body.instruction });
    const receipts = session.instructionReceipts ??= new Map();
    const previous = receipts.get(sequence);
    if (sequence <= (session.lastInstructionSequence ?? 0)) {
      if (previous?.fingerprint === requestFingerprint) sendReceipt(res, previous);
      else sendJson(res, 409, { error: {
        code: previous ? 'INSTRUCTION_OPERATION_CONFLICT' : 'INSTRUCTION_RECEIPT_UNAVAILABLE',
        message: previous ? 'Operation number is already bound to another payload' : 'Operation is older than the retained receipts; its effect must not be repeated',
        kind: 'orchestration', retryable: false,
      } });
      return;
    }
    session.lastInstructionSequence = sequence;
    session.lastUsedAt = new Date();
    const instruction = { ...validation.instruction, attempt: body.attempt, invocationId: body.invocation_id, operationSequence: sequence };
    const startedAt = new Date();
    const executionContext: ExecutionContext = {
      get page() { return session.page; },
      set page(page) {
        if (page !== session.page) session.frameStack.length = 0;
        session.page = page;
        session.currentPageIndex = session.pages.indexOf(page);
      },
      frameStack: session.frameStack,
      tabStack: session.pages,
      browserContext: session.context,
      config,
      logger: appLogger,
      metrics: appMetrics,
      sessionId,
      electronTarget: session.spec?.app_target,
      interactionState: session.spec?.browser_profile?.interaction_state,
    };
    let response: string;
    let uncertain = false;
    try {
      const result = await executeInstruction(instruction, executionContext, handlerRegistry, sessionManager.getInstrumentation());
      sessionManager.incrementInstructionCount(sessionId);
      // Serialize before retaining: a getter/cycle/decoration failure after a
      // browser effect must also become one stable, nonretryable receipt.
      response = JSON.stringify(withAudioCapability(result.driverOutcome, session));
    } catch (error) {
      uncertain = true;
      response = JSON.stringify(toDriverOutcome(buildStepOutcome({
        instruction, startedAt, completedAt: new Date(), finalUrl: '',
        result: { success: false, error: {
          code: 'INSTRUCTION_OUTCOME_UNCERTAIN', kind: 'infra', retryable: false,
          message: `Instruction may have taken effect: ${error instanceof Error ? error.message : String(error)}`,
        } },
      })));
    }
    const receipt = { fingerprint: requestFingerprint, response };
    receipts.set(sequence, receipt);
    if (receipts.size > MAX_EXECUTED_INSTRUCTIONS_PER_SESSION) {
      receipts.delete(receipts.keys().next().value!);
    }
    if (uncertain) appMetrics.instructionErrors.inc({ type: 'unknown', error_kind: 'infra' });
    sendReceipt(res, receipt);
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/run`);
    appMetrics.instructionErrors.inc({ type: 'unknown', error_kind: 'engine' });
  } finally {
    // Reset/close may have taken ownership while execution was pending.
    if (executingSession) {
      executingSession.instructionInFlight = false;
      if (executingSession.phase === 'executing') {
        sessionManager.setSessionPhase(sessionId, executingSession.pipelineManager?.isRecording() ? 'recording' : 'ready');
      }
    }
  }
}
