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
import { authorizeInstruction } from '../session/permission-policy';
import { getInputParams } from '../proto';
import { getFrameSwitchParams } from '../proto';
import { getDocument } from '../handlers/base';

const BAS_REFERENCE_SELECTOR = /^@bas-ref:(e[0-9]{1,90})$/;

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
  let settleInstruction: (() => void) | undefined;
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
    if (session.extensionOperationInFlight) {
      sendJson(res, 409, { error: { code: 'SESSION_BUSY', message: 'Session cannot execute an instruction while an extension operation is in flight', kind: 'orchestration', retryable: false } });
      return;
    }
    const action = validation.instruction.action as unknown as { params?: { value?: { selector?: string } } };
    const requestedSelector = action?.params?.value?.selector;
    if (typeof requestedSelector === 'string' && requestedSelector.startsWith('@bas-ref:')) {
      const match = BAS_REFERENCE_SELECTOR.exec(requestedSelector);
      const reference = match?.[1];
      const target = reference ? session.accessibilityReferenceTargets?.get(reference) : undefined;
      const currentPageId = session.pageBindings.getId(session.page);
      if (!target || target.expiresAt <= Date.now() || target.pageId !== currentPageId || target.url !== session.page.url()) {
        sendJson(res, 409, { error: { code: 'ACCESSIBILITY_REFERENCE_STALE', message: 'Accessibility reference is unknown, expired, or belongs to another page', kind: 'orchestration', retryable: true } });
        return;
      }
      // Rewrite in the driver only; the public task contract contains the ref,
      // while the selector remains private to this exact leased session.
      action.params!.value!.selector = target.selector;
    }
    const typedInput = validation.instruction.action ? getInputParams(validation.instruction.action) : undefined;
    if (session.spec?.permission_profile && typedInput) {
      if (typedInput.isSensitive || await isSensitiveInputTarget(session.page, typedInput.selector)) {
        sendJson(res, 403, { error: { code: 'BROWSER_PERMISSION_DENIED', message: 'raw sensitive input is unsupported; use the credential authority reference flow', kind: 'auth', retryable: false } });
        return;
      }
    }
    const frameSwitch = validation.instruction.action
      ? getFrameSwitchParams(validation.instruction.action)
      : undefined;
    let permissionURL = session.page.url();
    const selectedFrames = session.frameStack ?? [];
    if (selectedFrames.length && !getNavigateTarget(validation.instruction) && frameSwitch?.action !== 'exit') {
      try {
        const selected = getDocument({
          page: session.page,
          frameStack: selectedFrames,
        });
        permissionURL = selected === session.page
          ? session.page.url()
          : (frameSwitch?.action === 'parent'
              ? selectedFrames.at(-2)?.url() ?? session.page.url()
              : selected.url());
      } catch (error) {
        sendJson(res, 403, { error: { code: 'BROWSER_PERMISSION_DENIED', message: error instanceof Error ? error.message : 'Selected frame is stale', kind: 'auth', retryable: false } });
        return;
      }
    }
    const permission = authorizeInstruction(
      session.spec?.permission_profile,
      validation.instruction,
      permissionURL,
      Date.now(),
      session.createdAt.getTime(),
      session.instructionCount,
    );
    if (!permission.allowed) {
      sendJson(res, 403, { error: { code: 'BROWSER_PERMISSION_DENIED', message: permission.reason, kind: 'auth', retryable: false } });
      return;
    }
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
    if (session.extensionOperationInFlight || !sessionManager.canAcceptInstructions(sessionId) || !sessionManager.setSessionPhase(sessionId, 'executing')) {
      sendJson(res, 409, { error: { code: 'SESSION_BUSY', message: `Session cannot execute an instruction while ${session.phase}`, kind: 'orchestration', retryable: false } });
      return;
    }
    session.instructionInFlight = true;
    session.instructionSettlement = new Promise<void>((resolve) => { settleInstruction = resolve; });
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
      },
      frameStack: session.frameStack ?? [],
      permissionProfile: session.spec?.permission_profile,
      tabStack: session.context.pages(),
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
      if (session.instructionInterrupted) {
        throw new Error('Session close interrupted the browser operation before its outcome was retained');
      }
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
      executingSession.instructionSettlement = undefined;
      settleInstruction?.();
      if (executingSession.phase === 'executing') {
        sessionManager.setSessionPhase(sessionId, executingSession.pipelineManager?.isRecording() ? 'recording' : 'ready');
      }
    }
  }
}

function getNavigateTarget(instruction: Parameters<typeof authorizeInstruction>[1]): string | undefined {
  const action = instruction.action;
  if (!action) return undefined;
  const params = action.params;
  if (params?.case !== 'navigate') return undefined;
  return params.value.url;
}

async function isSensitiveInputTarget(page: SessionState['page'], selector: string): Promise<boolean> {
  if (!selector.trim()) return true;
  return page.locator(selector).first().evaluate((element) => {
    if (!(element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement)) return false;
    const type = element instanceof HTMLInputElement ? element.type.toLowerCase() : '';
    const sensitiveAutocomplete = new Set(['current-password', 'new-password', 'one-time-code', 'cc-name', 'cc-number', 'cc-exp', 'cc-exp-month', 'cc-exp-year', 'cc-csc']);
    const autocomplete = (element.getAttribute('autocomplete') ?? '').toLowerCase().split(/\s+/);
    return type === 'password' || type === 'hidden' || autocomplete.some((token) => sensitiveAutocomplete.has(token));
  });
}
