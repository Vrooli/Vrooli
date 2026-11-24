import { createMockHttpRequest, createMockHttpResponse, createTestConfig, createTypedInstruction, createMockPage, createMockContext, createMockFrame } from '../../helpers';
import { handleSessionRun } from '../../../src/routes/session-run';
import { SessionManager } from '../../../src/session';
import { DriverPageBindings } from '../../../src/session/page-bindings';
import type { SessionPhase, SessionState } from '../../../src/types';
import { PassThrough } from 'node:stream';
import type { IncomingMessage } from 'node:http';
import type { HandlerRegistry } from '../../../src/handlers';
import type { Metrics } from '../../../src/utils/metrics';
import { createNoOpLogger } from '../../../src/utils';

jest.mock('../../../src/execution', () => ({
  executeInstruction: jest.fn(),
  validateInstruction: jest.fn(),
}));

import {
  executeInstruction,
  validateInstruction,
} from '../../../src/execution';

const mockExecuteInstruction = executeInstruction as jest.MockedFunction<typeof executeInstruction>;
const mockValidateInstruction = validateInstruction as jest.MockedFunction<
  typeof validateInstruction
>;
const createLeasedRequest = (options: NonNullable<Parameters<typeof createMockHttpRequest>[0]> = {}) =>
  createMockHttpRequest({ ...options, body: { execution_id: 'exec-1', lease_id: 'lease-1', operation_sequence: 1, invocation_id: 'visit-1', attempt: 1, ...(options.body as Record<string, unknown> | undefined) } });

describe('handleSessionRun', () => {
  const config = createTestConfig();
  const logger = createNoOpLogger();
  const metrics = {
    instructionErrors: { inc: jest.fn() },
  } as unknown as Metrics;

  const handlerRegistry = {} as HandlerRegistry;

  const buildSessionManager = (session: { id: string; phase: SessionPhase }) => {
    const manager = new SessionManager(config);
    Reflect.set(manager, 'sessions', new Map([[session.id, session as SessionState]]));
    jest.spyOn(manager, 'setSessionPhase');
    return manager;
  };

  const buildSession = (
    overrides?: Partial<{
      phase: SessionPhase;
      extensionOperationInFlight: boolean;
      instructionReceipts: Map<number, {fingerprint: string; response: string}>;
      pipelineManager: { isRecording: jest.Mock<boolean, []> };
    }>
  ) => ({
    id: 'session-1',
    ownerExecutionId: 'exec-1',
    leaseId: 'lease-1',
    leaseReleasedAt: undefined as Date | undefined,
    phase: 'ready' as const,
    page: createMockPage(),
    spec: { execution_id: 'exec-1', reuse_mode: 'fresh' as const, browser_backend: 'managed' as const | 'installed' },
    createdAt: new Date(),
    lastUsedAt: new Date(),
    instructionCount: 0,
    context: createMockContext(),
    storageOrigins: new Set<string>(),
    pageBindings: new DriverPageBindings(),
    activeMocks: new Map(),
    instructionReceipts: new Map<number, {fingerprint: string; response: string}>(),
    lastInstructionSequence: 0,
    pipelineManager: { isRecording: jest.fn().mockReturnValue(false) },
    audioStrategy: 'synthetic_sink' as const,
    audioCapability: {
      outcome: 'no_device' as const,
      currentTimeDelta: 0,
      durationMs: 1200,
      reason: 'test host has no output',
    },
    ...overrides,
  });

  beforeEach(() => {
    jest.clearAllMocks();
    mockValidateInstruction.mockReturnValue({
      valid: true,
      instruction: createTypedInstruction('click', {}, { index: 1, nodeId: 'node-1' }),
    } as never);
    mockExecuteInstruction.mockReset().mockResolvedValue({
      driverOutcome: { success: true }, success: true,
    } as never);
  });

  it('resolves a short-lived opaque accessibility reference inside its exact leased session', async () => {
    const session = buildSession();
    session.pageBindings.getId = () => 'page-1';
    const pageId = 'page-1';
    session.accessibilityReferenceTargets = new Map([['e123', {
      selector: 'button[data-testid="continue"]', pageId, url: session.page.url(), expiresAt: Date.now() + 5_000,
    }]]);
    const manager = buildSessionManager(session);
    mockValidateInstruction.mockReturnValue({
      valid: true,
      instruction: createTypedInstruction('click', { selector: '@bas-ref:e123' }),
    } as never);
    const run = request(manager);
    await run.done;
    expect(run.res.statusCode).toBe(200);
    const executed = mockExecuteInstruction.mock.calls[0][0] as unknown as { action: { params: { value: { selector: string } } } };
    expect(executed.action.params.value.selector).toBe('button[data-testid="continue"]');
  });

  it.each(['managed', 'installed'] as const)(
    '%s owner rejects stale opaque refs before browser effects', async (browser_backend) => {
      const targets = [
        ['unknown ref', undefined],
        ['expired ref', { selector: '#unsafe', pageId: 'page-1', url: 'http://localhost', expiresAt: Date.now() - 1 }],
        ['different page', { selector: '#unsafe', pageId: 'other-page', url: 'http://localhost', expiresAt: Date.now() + 5_000 }],
        ['different URL', { selector: '#unsafe', pageId: 'page-1', url: 'https://other.example', expiresAt: Date.now() + 5_000 }],
      ] as const;
      for (const [_name, target] of targets) {
        const session = buildSession();
        session.pageBindings.getId = () => 'page-1';
        session.spec.browser_backend = browser_backend;
        if (target) session.accessibilityReferenceTargets = new Map([['e123', target]]);
        const manager = buildSessionManager(session);
        mockValidateInstruction.mockReturnValue({
          valid: true,
          instruction: createTypedInstruction('click', { selector: '@bas-ref:e123' }),
        } as never);
        const run = request(manager);
        await run.done;
        expect(run.res.statusCode).toBe(409);
        expect(run.res.getJSON().error.code).toBe('ACCESSIBILITY_REFERENCE_STALE');
        expect(mockExecuteInstruction).not.toHaveBeenCalled();
      }
    }
  );

  it.each(['managed', 'installed'] as const)(
    '%s owner applies the same URL and effect policy before browser effects', async (browser_backend) => {
      const deniedCases = [
        { name: 'observe-only mutation', profile: { effect: 'observe', control: false, view: true, url_allow: ['https://safe.test/*'] }, url: 'https://safe.test/start', instruction: createTypedInstruction('click', {}) },
        { name: 'denied navigation origin', profile: { effect: 'act', control: true, view: true, url_allow: ['https://safe.test/*'] }, url: 'https://safe.test/start', instruction: createTypedInstruction('navigate', { url: 'https://blocked.test/private' }) },
        { name: 'redirected current origin', profile: { effect: 'act', control: true, view: true, url_allow: ['https://safe.test/*'] }, url: 'https://blocked.test/redirect', instruction: createTypedInstruction('click', {}) },
      ] as const;
      for (const denied of deniedCases) {
        const session = buildSession();
        session.spec.browser_backend = browser_backend;
        session.spec.permission_profile = {
          effect: denied.profile.effect as 'act' | 'observe', approval: 'none',
          control: denied.profile.control, view: denied.profile.view,
          url_allow: [...denied.profile.url_allow],
        };
        session.page.url = () => denied.url;
        const manager = buildSessionManager(session);
        mockValidateInstruction.mockReturnValue({ valid: true, instruction: denied.instruction } as never);
        const run = request(manager);
        await run.done;
        expect(run.res.statusCode).toBe(403);
        expect(run.res.getJSON().error.code).toBe('BROWSER_PERMISSION_DENIED');
        expect(mockExecuteInstruction).not.toHaveBeenCalled();
      }
    }
  );

  it.each(['managed', 'installed'] as const)(
    '%s owner shares API approval tiers and denies unverifiable receipts before effects', async (browser_backend) => {
      const safeURL = 'https://shop.example.test/app/orders';
      for (const tc of [
        { name: 'safety-pass low-impact click remains supported', approval: 'safety_pass' as const, action: createTypedInstruction('click', {}), status: 200 },
        { name: 'safety-pass high-impact input fails closed', approval: 'safety_pass' as const, action: createTypedInstruction('input', { selector: '#email', value: 'user@example.test' }), status: 403 },
        { name: 'every-action click fails closed', approval: 'every_action' as const, action: createTypedInstruction('click', {}), status: 403 },
      ]) {
        mockExecuteInstruction.mockClear();
        const session = buildSession();
        session.spec.browser_backend = browser_backend;
        session.spec.permission_profile = {
          effect: 'act', approval: tc.approval, control: true, view: true,
          url_allow: ['https://shop.example.test/*'],
        };
        session.page.url = () => safeURL;
        session.page.locator = jest.fn().mockReturnValue({ first: () => ({ evaluate: async () => false }) }) as never;
        const manager = buildSessionManager(session);
        mockValidateInstruction.mockReturnValue({ valid: true, instruction: tc.action } as never);
        const run = request(manager);
        await run.done;
        expect(run.res.statusCode).toBe(tc.status);
        expect(mockExecuteInstruction).toHaveBeenCalledTimes(tc.status === 200 ? 1 : 0);
        if (tc.status === 403) {
          expect(run.res.getJSON().error.code).toBe('BROWSER_PERMISSION_DENIED');
          expect(session.lastInstructionSequence).toBe(0);
        }
      }
    }
  );

  it.each(['managed', 'installed'] as const)(
    '%s owner checks the live selected frame origin before dispatch', async (browser_backend) => {
      const page = createMockPage();
      page.url = jest.fn().mockReturnValue('https://safe.test/top');
      const main = createMockFrame({ page: jest.fn().mockReturnValue(page), isDetached: jest.fn().mockReturnValue(false) });
      page.mainFrame.mockReturnValue(main);
      const child = createMockFrame({
        page: jest.fn().mockReturnValue(page),
        parentFrame: jest.fn().mockReturnValue(main),
        isDetached: jest.fn().mockReturnValue(false),
        url: jest.fn().mockReturnValue('https://foreign.test/frame'),
      });
      const session = buildSession() as SessionState;
      session.page = page;
      session.frameStack = [child as never];
      session.spec.browser_backend = browser_backend;
      session.spec.permission_profile = {
        effect: 'act', approval: 'none', control: true, view: true,
        url_allow: ['https://safe.test/*'],
      };
      const manager = buildSessionManager(session);
      const run = request(manager);
      await run.done;
      expect(run.res.statusCode).toBe(403);
      expect(run.res.getJSON().error.code).toBe('BROWSER_PERMISSION_DENIED');
      expect(mockExecuteInstruction).not.toHaveBeenCalled();

      child.isDetached.mockReturnValue(true);
      session.lastInstructionSequence = 0;
      const stale = request(manager);
      await stale.done;
      expect(stale.res.statusCode).toBe(403);
      expect(mockExecuteInstruction).not.toHaveBeenCalled();

      child.isDetached.mockReturnValue(false);
      child.page.mockReturnValue(createMockPage());
      session.lastInstructionSequence = 0;
      const forged = request(manager);
      await forged.done;
      expect(forged.res.statusCode).toBe(403);
      expect(mockExecuteInstruction).not.toHaveBeenCalled();

      child.page.mockReturnValue(page);
      session.spec.permission_profile.url_allow.push('https://foreign.test/*');
      session.lastInstructionSequence = 0;
      const allowed = request(manager);
      await allowed.done;
      expect(allowed.res.statusCode).toBe(200);
      expect(mockExecuteInstruction).toHaveBeenCalledTimes(1);
    }
  );

  it('returns the exact retained response without repeating effects or audio decoration', async () => {
    const session = buildSession({ pipelineManager: { isRecording: jest.fn().mockReturnValue(true) } });
    const manager = buildSessionManager(session);
    const first = request(manager); await first.done;
    const replay = request(manager); await replay.done;
    expect(first.res.statusCode).toBe(200);
    expect(replay.res.getJSON()).toEqual(first.res.getJSON());
    expect(mockExecuteInstruction).toHaveBeenCalledTimes(1);
    expect(session.phase).toBe('recording');
    expect(first.res.getJSON()).toMatchObject({ success: true, audio_strategy: 'synthetic_sink', host_audio_outcome: 'no_device' });
  });

  it('rejects invalid instructions without changing admission state', async () => {
    const session = buildSession(); const manager = buildSessionManager(session);
    mockValidateInstruction.mockReturnValue({ valid: false, error: { code: 'INVALID_INSTRUCTION', message: 'bad' } });
    const result = request(manager); await result.done;
    expect(result.res.statusCode).toBe(400);
    expect(session.phase).toBe('ready');
    expect(session.lastInstructionSequence).toBe(0);
    expect(manager.setSessionPhase).not.toHaveBeenCalled();
    expect(mockExecuteInstruction).not.toHaveBeenCalled();
  });

  it('retains an uncertain result after a post-admission exception', async () => {
    const session = buildSession(); const manager = buildSessionManager(session);
    mockExecuteInstruction.mockRejectedValueOnce(new Error('effect happened, telemetry failed'));
    const first = request(manager); await first.done;
    const repeat = request(manager); await repeat.done;
    expect(first.res.statusCode).toBe(200);
    expect(first.res.getJSON().failure.code).toBe('INSTRUCTION_OUTCOME_UNCERTAIN');
    expect(first.res.getJSON().failure.retryable).not.toBe(true);
    expect(repeat.res.getJSON()).toEqual(first.res.getJSON());
    expect(mockExecuteInstruction).toHaveBeenCalledTimes(1);
    expect(session.phase).toBe('ready');
  });

  it.each([
    { instruction: { type: 'different' } }, { invocation_id: 'changed' }, { attempt: 2 },
  ])('rejects changed operation payload %j before repeating its effect', async (changed) => {
    const session = buildSession(); const manager = buildSessionManager(session);
    await request(manager).done;
    const repeat = request(manager, {}, changed); await repeat.done;
    expect(repeat.res.statusCode).toBe(409);
    expect(repeat.res.getJSON().error.code).toBe('INSTRUCTION_OPERATION_CONFLICT');
    expect(mockExecuteInstruction).toHaveBeenCalledTimes(1);
  });

  it('recognizes reordered JSON objects as the same payload', async () => {
    const session = buildSession(); const manager = buildSessionManager(session);
    await request(manager, {}, { instruction: { type: 'click', context: { a: 1, b: 2 } } }).done;
    const repeat = request(manager, {}, { instruction: { context: { b: 2, a: 1 }, type: 'click' } }); await repeat.done;
    expect(repeat.res.statusCode).toBe(200);
    expect(mockExecuteInstruction).toHaveBeenCalledTimes(1);
  });

  it('allows a declared retry after a known transient outcome', async () => {
    const session = buildSession(); const manager = buildSessionManager(session);
    mockExecuteInstruction.mockResolvedValueOnce({ driverOutcome: { success: false, failure: { retryable: true } }, success: false } as never);
    const first = request(manager); await first.done;
    const repeat = request(manager); await repeat.done;
    expect(repeat.res.getJSON()).toEqual(first.res.getJSON());
    const retry = request(manager, {}, { operation_sequence: 2, attempt: 2 }); await retry.done;
    expect(retry.res.getJSON().success).toBe(true);
    expect(mockExecuteInstruction).toHaveBeenCalledTimes(2);
    expect(mockExecuteInstruction.mock.calls[1]?.[0].attempt).toBe(2);
  });

  it('does not repeat an evicted operation after more than 1000 effects', async () => {
    const session = buildSession(); const manager = buildSessionManager(session);
    for (let sequence = 1; sequence <= 1001; sequence++) {
      const result = request(manager, {}, { operation_sequence: sequence }); await result.done;
      expect(result.res.statusCode).toBe(200);
    }
    const old = request(manager); await old.done;
    expect(old.res.statusCode).toBe(409);
    expect(old.res.getJSON().error.code).toBe('INSTRUCTION_RECEIPT_UNAVAILABLE');
    const retained = request(manager, {}, { operation_sequence: 1001 }); await retained.done;
    expect(retained.res.statusCode).toBe(200);
    expect(mockExecuteInstruction).toHaveBeenCalledTimes(1001);
    expect(session.instructionReceipts.size).toBe(1000);
  });

  it('reset discards receipts while rejecting old operation numbers', async () => {
    const session = buildSession(); const manager = buildSessionManager(session);
    await request(manager).done;
    await manager.resetSession(session.id);
    const old = request(manager); await old.done;
    expect(old.res.statusCode).toBe(409);
    expect(old.res.getJSON().error.code).toBe('INSTRUCTION_RECEIPT_UNAVAILABLE');
    const next = request(manager, {}, { operation_sequence: 2 }); await next.done;
    expect(next.res.statusCode).toBe(200);
    expect(mockExecuteInstruction).toHaveBeenCalledTimes(2);
  });

  it.each([
    { operation_sequence: undefined }, { operation_sequence: 0 }, { operation_sequence: -1 },
    { operation_sequence: 1.5 }, { operation_sequence: Number.MAX_SAFE_INTEGER + 1 },
    { invocation_id: '' }, { invocation_id: undefined }, { attempt: 0 }, { attempt: 1.5 },
  ])('rejects invalid operation identity %j before admission', async (invalid) => {
    const session = buildSession(); const manager = buildSessionManager(session);
    const result = request(manager, {}, invalid); await result.done;
    expect(result.res.statusCode).toBe(400);
    expect(session.lastInstructionSequence).toBe(0);
    expect(manager.setSessionPhase).not.toHaveBeenCalled();
    expect(mockExecuteInstruction).not.toHaveBeenCalled();
  });

  it('requires a supplied HTTP idempotency key to match the lease operation', async () => {
    const session = buildSession(); const manager = buildSessionManager(session);
    const invalid = request(manager, { 'x-idempotency-key': 'arbitrary-old-key' }); await invalid.done;
    expect(invalid.res.statusCode).toBe(400);
    expect(mockExecuteInstruction).not.toHaveBeenCalled();
    const valid = request(manager, { 'x-idempotency-key': 'lease-1:1' }); await valid.done;
    expect(valid.res.statusCode).toBe(200);
  });

  it('rejects marked raw secret input in permission-scoped sessions before effects', async () => {
    const session = buildSession();
    session.spec.permission_profile = {
      effect: 'act', approval: 'none', control: true, view: true,
      url_allow: ['https://shop.example.test/*'],
    };
    const manager = buildSessionManager(session);
    const instruction = createTypedInstruction('input', { selector: '#password', value: 'raw-secret' });
    if (instruction.action?.params.case === 'input') instruction.action.params.value.isSensitive = true;
    mockValidateInstruction.mockReturnValue({
      valid: true,
      instruction,
    } as never);
    const result = request(manager);
    await result.done;
    expect(result.res.statusCode).toBe(403);
    expect(result.res.getJSON().error.message).toMatch(/raw sensitive input is unsupported/);
    expect(mockExecuteInstruction).not.toHaveBeenCalled();
  });

  const request = (manager: SessionManager, headers = {}, body: Record<string, unknown> = {}) => {
    const res = createMockHttpResponse();
    const done = handleSessionRun(
      createLeasedRequest({ headers, body: { instruction: { type: 'click' }, ...body } }),
      res, 'session-1', manager, handlerRegistry, config, logger, metrics
    );
    return { res, done };
  };

  it.each(['initializing', 'resetting', 'closing', 'executing'] as const)(
    'rejects %s before cached results or browser effects', async (phase) => {
      const session = buildSession({ phase });
      const manager = buildSessionManager(session);
      const run = request(manager, { 'x-idempotency-key': 'lease-1:1' });
      await run.done;
      expect(run.res.statusCode).toBe(409);
      expect(session.phase).toBe(phase);
      expect(mockExecuteInstruction).not.toHaveBeenCalled();
    }
  );

  it('rejects instruction execution while a managed extension operation owns the session', async () => {
    const session = buildSession({ extensionOperationInFlight: true });
    const manager = buildSessionManager(session);
    const run = request(manager, { 'x-idempotency-key': 'lease-1:1' });
    await run.done;
    expect(run.res.statusCode).toBe(409);
    expect(run.res.getJSON().error.code).toBe('SESSION_BUSY');
    expect(mockExecuteInstruction).not.toHaveBeenCalled();
    expect(manager.setSessionPhase).not.toHaveBeenCalled();
  });

  it.each(['closing', 'resetting'] as const)(
    'rechecks admission when %s begins while the request body is arriving', async (phase) => {
      const session = buildSession();
      const manager = buildSessionManager(session);
      const req = new PassThrough() as unknown as IncomingMessage;
      req.headers = {};
      const res = createMockHttpResponse();
      const done = handleSessionRun(req, res, session.id, manager, handlerRegistry, config, logger, metrics);
      manager.setSessionPhase(session.id, phase);
      (req as unknown as PassThrough).end(JSON.stringify({ execution_id: 'exec-1', lease_id: 'lease-1', operation_sequence: 1, invocation_id: 'visit-1', attempt: 1, instruction: { type: 'click' } }));
      await done;
      expect(res.statusCode).toBe(409);
      expect(session.phase).toBe(phase);
      expect(mockExecuteInstruction).not.toHaveBeenCalled();
    }
  );

  it('a start retry cannot unlock a delayed instruction', async () => {
    const session = buildSession();
    const manager = buildSessionManager(session);
    let release!: () => void;
    let started!: () => void;
    const entered = new Promise<void>((resolve) => { started = resolve; });
    mockExecuteInstruction.mockImplementationOnce(async () => {
      started();
      await new Promise<void>((resolve) => { release = resolve; });
      return { success: true, driverOutcome: { success: true } } as never;
    });
    const first = request(manager);
    await entered;
    try {
      await manager.startSession(session.spec as never);
      const second = request(manager);
      await second.done;
      expect(second.res.statusCode).toBe(409);
      expect(mockExecuteInstruction).toHaveBeenCalledTimes(1);
      expect(session.phase).toBe('executing');
    } finally {
      release();
      await first.done;
    }
    expect(session.phase).toBe('ready');
  });

  it('close waits for an admitted instruction before tearing down its session', async () => {
    const session = buildSession();
    const manager = buildSessionManager(session);
    let release!: () => void;
    let entered!: () => void;
    const started = new Promise<void>((resolve) => { entered = resolve; });
    mockExecuteInstruction.mockImplementationOnce(async () => {
      entered();
      await new Promise<void>((resolve) => { release = resolve; });
      return { success: true, driverOutcome: { success: true } } as never;
    });

    const run = request(manager);
    await started;
    const close = manager.closeSession(session.id);
    try {
      expect(session.phase).toBe('closing');
      expect(session.context.close).not.toHaveBeenCalled();
      expect(manager.getSessionCount()).toBe(1);
    } finally {
      release();
      await run.done;
      await close;
    }
    expect(session.context.close).toHaveBeenCalledTimes(1);
    expect(manager.getSessionCount()).toBe(0);
  });

  it.each(['invalid', 'throw', 'success'] as const)(
    'restores recording after %s completion', async (result) => {
      const session = buildSession({ phase: 'recording', pipelineManager: { isRecording: jest.fn().mockReturnValue(true) } });
      const manager = buildSessionManager(session);
      if (result === 'invalid') mockValidateInstruction.mockReturnValue({ valid: false, error: { code: 'INVALID', message: 'invalid' } } as never);
      if (result === 'throw') mockExecuteInstruction.mockRejectedValueOnce(new Error('fault'));
      await request(manager).done;
      expect(session.phase).toBe('recording');
    }
  );

  it.each(['closing', 'resetting'] as const)(
    'does not release a concurrent %s operation after instruction failure', async (phase) => {
      const session = buildSession();
      const manager = buildSessionManager(session);
      mockExecuteInstruction.mockImplementationOnce(async () => {
        manager.setSessionPhase(session.id, phase);
        throw new Error('interrupted');
      });
      await request(manager).done;
      expect(session.phase).toBe(phase);
    }
  );

  it.each(['resetting', 'closing'] as const)(
    'rejects instructions throughout an actual delayed %s operation', async (phase) => {
      const session = buildSession();
      const manager = buildSessionManager(session);
      let release!: () => void;
      let entered!: () => void;
      const started = new Promise<void>((resolve) => { entered = resolve; });
      const delayed = async () => {
        entered();
        await new Promise<void>((resolve) => { release = resolve; });
        return null;
      };
      if (phase === 'resetting') session.page.goto.mockImplementationOnce(delayed as never);
      else session.page.close.mockImplementationOnce(delayed as never);
      const operation = phase === 'resetting' ? manager.resetSession(session.id) : manager.closeSession(session.id);
      await started;
      try {
        const run = request(manager);
        await run.done;
        expect(run.res.statusCode).toBe(409);
        expect(session.phase).toBe(phase);
        expect(mockExecuteInstruction).not.toHaveBeenCalled();
      } finally {
        release();
        await operation;
      }
    }
  );

  it('does not admit another instruction after reset until the previous action settles', async () => {
    const session = buildSession();
    const manager = buildSessionManager(session);
    let release!: () => void;
    let started!: () => void;
    const entered = new Promise<void>((resolve) => { started = resolve; });
    mockExecuteInstruction.mockImplementationOnce(async () => {
      started();
      await new Promise<void>((resolve) => { release = resolve; });
      throw new Error('interrupted action settled');
    });
    const first = request(manager);
    await entered;
    let resetSettled = false;
    const reset = manager.resetSession(session.id).then(() => { resetSettled = true; });
    try {
      await Promise.resolve();
      const second = request(manager);
      await second.done;
      expect(second.res.statusCode).toBe(409);
      expect(mockExecuteInstruction).toHaveBeenCalledTimes(1);
      expect(session.page.goto).not.toHaveBeenCalled();
      expect(resetSettled).toBe(false);
    } finally {
      release();
      await first.done;
      await reset;
    }
    const afterSettlement = request(manager, {}, { operation_sequence: 2 });
    await afterSettlement.done;
    expect(afterSettlement.res.statusCode).toBe(200);
    expect(mockExecuteInstruction).toHaveBeenCalledTimes(2);
  });

  it.each([
    ['missing-owner', {}, false, 400],
    ['missing-lease', { execution_id: 'exec-1' }, false, 400],
    ['stale-owner', { execution_id: 'previous', lease_id: 'lease-1' }, false, 404],
    ['stale-token', { execution_id: 'exec-1', lease_id: 'previous' }, false, 404],
    ['released-lease', { execution_id: 'exec-1', lease_id: 'lease-1' }, true, 404],
  ] as const)('rejects %s before activity, phase, caches or effects', async (_name, credentials, released, status) => {
    const session = buildSession();
    session.lastUsedAt = new Date('2020-01-01');
    if (released) session.leaseReleasedAt = new Date('2020-01-02');
    const manager = buildSessionManager(session);
    const res = createMockHttpResponse();
    await handleSessionRun(createMockHttpRequest({ headers: { 'x-idempotency-key': 'lease-1:1' }, body: { ...credentials, instruction: {} } }),
      res, session.id, manager, handlerRegistry, config, logger, metrics);
    expect(res.statusCode).toBe(status);
    expect(session.lastUsedAt).toEqual(new Date('2020-01-01'));
    expect(session.phase).toBe('ready');
    expect(manager.setSessionPhase).not.toHaveBeenCalled();
    expect(mockExecuteInstruction).not.toHaveBeenCalled();
  });

  it('checks the lease after a delayed request body crosses an ownership handoff', async () => {
    const session = buildSession();
    const manager = buildSessionManager(session);
    const req = new PassThrough() as unknown as IncomingMessage;
    req.headers = {};
    const res = createMockHttpResponse();
    const done = handleSessionRun(req, res, session.id, manager, handlerRegistry, config, logger, metrics);
    session.ownerExecutionId = 'next-execution';
    session.leaseId = 'next-lease';
    (req as unknown as PassThrough).end(JSON.stringify({ execution_id: 'exec-1', lease_id: 'lease-1', operation_sequence: 1, invocation_id: 'visit-1', attempt: 1, instruction: {} }));
    await done;
    expect(res.statusCode).toBe(404);
    expect(session.phase).toBe('ready');
    expect(mockExecuteInstruction).not.toHaveBeenCalled();
  });

});
