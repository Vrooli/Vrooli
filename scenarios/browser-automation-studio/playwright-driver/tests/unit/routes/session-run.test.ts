import { createMockHttpRequest, createMockHttpResponse, createTestConfig, createTypedInstruction } from '../../helpers';
import { handleSessionRun } from '../../../src/routes/session-run';
import type { SessionManager } from '../../../src/session';
import type { HandlerRegistry } from '../../../src/handlers';
import type { Metrics } from '../../../src/utils/metrics';
import { createNoOpLogger } from '../../../src/utils';

jest.mock('../../../src/infra', () => ({
  getIdempotencyCache: jest.fn(),
}));

jest.mock('../../../src/execution', () => ({
  executeInstruction: jest.fn(),
  validateInstruction: jest.fn(),
  createInstructionKey: jest.fn(),
}));

import { getIdempotencyCache } from '../../../src/infra';
import {
  executeInstruction,
  validateInstruction,
  createInstructionKey,
} from '../../../src/execution';

const mockGetIdempotencyCache = getIdempotencyCache as jest.MockedFunction<
  typeof getIdempotencyCache
>;
const mockExecuteInstruction = executeInstruction as jest.MockedFunction<typeof executeInstruction>;
const mockValidateInstruction = validateInstruction as jest.MockedFunction<
  typeof validateInstruction
>;
const mockCreateInstructionKey = createInstructionKey as jest.MockedFunction<
  typeof createInstructionKey
>;

describe('handleSessionRun', () => {
  const config = createTestConfig();
  const logger = createNoOpLogger();
  const metrics = {
    instructionErrors: { inc: jest.fn() },
  } as unknown as Metrics;

  const handlerRegistry = {} as HandlerRegistry;

  const buildSessionManager = (session: {
    id: string;
    phase: 'ready' | 'executing' | 'recording';
    executedInstructions?: Map<string, unknown>;
    pipelineManager?: { isRecording: jest.Mock<boolean, []> };
  }) =>
    ({
      getSession: jest.fn().mockReturnValue(session),
      setSessionPhase: jest.fn((sessionId: string, phase: string) => {
        if (sessionId === session.id) {
          session.phase = phase as 'ready' | 'executing' | 'recording';
        }
      }),
      incrementInstructionCount: jest.fn(),
      getInstrumentation: jest.fn().mockReturnValue({}),
    }) as unknown as SessionManager;

  const buildSession = (
    overrides?: Partial<{
      phase: 'ready' | 'executing' | 'recording';
      executedInstructions: Map<string, unknown>;
      pipelineManager: { isRecording: jest.Mock<boolean, []> };
    }>
  ) => ({
    id: 'session-1',
    phase: 'ready' as const,
    page: {},
    context: {},
    executedInstructions: new Map<string, unknown>(),
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
  });

  it('returns cached idempotency response without executing', async () => {
    const session = buildSession();
    const sessionManager = buildSessionManager(session);

    const cached = { success: true, cached: true };
    const idempotencyCache = {
      lookup: jest.fn().mockReturnValue(cached),
      store: jest.fn(),
    };
    mockGetIdempotencyCache.mockReturnValue(idempotencyCache as never);

    const req = createMockHttpRequest({
      headers: { 'x-idempotency-key': 'idem-1' },
    });
    const res = createMockHttpResponse();

    await handleSessionRun(
      req,
      res,
      session.id,
      sessionManager,
      handlerRegistry,
      config,
      logger,
      metrics
    );

    expect(idempotencyCache.lookup).toHaveBeenCalledWith('idem-1', session.id);
    expect(mockExecuteInstruction).not.toHaveBeenCalled();
    expect(res.statusCode).toBe(200);
    expect(res.getJSON()).toEqual(cached);
  });

  it('rejects concurrent execution with 409', async () => {
    const session = buildSession({ phase: 'executing' });
    const sessionManager = buildSessionManager(session);
    const idempotencyCache = {
      lookup: jest.fn().mockReturnValue(null),
      store: jest.fn(),
    };
    mockGetIdempotencyCache.mockReturnValue(idempotencyCache as never);

    const req = createMockHttpRequest({
      body: { instruction: { type: 'click' } },
    });
    const res = createMockHttpResponse();

    await handleSessionRun(
      req,
      res,
      session.id,
      sessionManager,
      handlerRegistry,
      config,
      logger,
      metrics
    );

    expect(res.statusCode).toBe(409);
    expect(res.getJSON().error.code).toBe('SESSION_BUSY');
  });

  it('returns 400 when instruction validation fails and resets phase', async () => {
    const session = buildSession();
    const sessionManager = buildSessionManager(session);
    const idempotencyCache = {
      lookup: jest.fn().mockReturnValue(null),
      store: jest.fn(),
    };
    mockGetIdempotencyCache.mockReturnValue(idempotencyCache as never);
    mockValidateInstruction.mockReturnValue({
      valid: false,
      error: { code: 'INVALID_INSTRUCTION', message: 'Bad instruction' },
    } as never);

    const req = createMockHttpRequest({
      body: { instruction: { type: 'click' } },
    });
    const res = createMockHttpResponse();

    await handleSessionRun(
      req,
      res,
      session.id,
      sessionManager,
      handlerRegistry,
      config,
      logger,
      metrics
    );

    expect(sessionManager.setSessionPhase).toHaveBeenCalledWith(session.id, 'ready');
    expect(res.statusCode).toBe(400);
    expect(res.getJSON().error.code).toBe('INVALID_INSTRUCTION');
  });

  it('returns cached replay result and restores recording phase', async () => {
    const cachedOutcome = { success: true, replay: true };
    const instructionKey = 'instruction-key';
    const executedInstructions = new Map<string, unknown>([
      [
        instructionKey,
        {
          key: instructionKey,
          executedAt: new Date(),
          success: true,
          cachedOutcome,
        },
      ],
    ]);

    const session = buildSession({
      executedInstructions,
      pipelineManager: { isRecording: jest.fn().mockReturnValue(true) },
    });
    const sessionManager = buildSessionManager(session);

    const idempotencyCache = {
      lookup: jest.fn().mockReturnValue(null),
      store: jest.fn(),
    };
    mockGetIdempotencyCache.mockReturnValue(idempotencyCache as never);
    mockValidateInstruction.mockReturnValue({
      valid: true,
      instruction: createTypedInstruction('click', {}, { index: 1, nodeId: 'node-1' }),
    } as never);
    mockCreateInstructionKey.mockReturnValue(instructionKey);

    const req = createMockHttpRequest({
      headers: { 'x-idempotency-key': 'idem-2' },
      body: { instruction: { type: 'click' } },
    });
    const res = createMockHttpResponse();

    await handleSessionRun(
      req,
      res,
      session.id,
      sessionManager,
      handlerRegistry,
      config,
      logger,
      metrics
    );

    expect(sessionManager.setSessionPhase).toHaveBeenCalledWith(session.id, 'recording');
    expect(idempotencyCache.store).toHaveBeenCalledWith(
      'idem-2',
      session.id,
      instructionKey,
      cachedOutcome
    );
    expect(res.statusCode).toBe(200);
    expect(res.getJSON()).toEqual(cachedOutcome);
  });

  it('executes instruction, caches outcome, and returns response', async () => {
    const session = buildSession();
    const sessionManager = buildSessionManager(session);
    const idempotencyCache = {
      lookup: jest.fn().mockReturnValue(null),
      store: jest.fn(),
    };
    mockGetIdempotencyCache.mockReturnValue(idempotencyCache as never);

    const instruction = { type: 'click', index: 2, nodeId: 'node-2' };
    mockValidateInstruction.mockReturnValue({
      valid: true,
      instruction,
    } as never);
    mockCreateInstructionKey.mockReturnValue('key-2');
    mockExecuteInstruction.mockResolvedValue({
      driverOutcome: { success: true, ok: true },
      success: true,
    } as never);

    const req = createMockHttpRequest({
      headers: { 'x-idempotency-key': 'idem-3' },
      body: { instruction },
    });
    const res = createMockHttpResponse();

    await handleSessionRun(
      req,
      res,
      session.id,
      sessionManager,
      handlerRegistry,
      config,
      logger,
      metrics
    );

    expect(mockExecuteInstruction).toHaveBeenCalled();
    expect(session.executedInstructions?.has('key-2')).toBe(true);
    expect(idempotencyCache.store).toHaveBeenCalledWith('idem-3', session.id, 'key-2', {
      success: true,
      ok: true,
      audio_strategy: 'synthetic_sink',
      host_audio_outcome: 'no_device',
      host_audio_reason: 'test host has no output',
    });
    expect(res.statusCode).toBe(200);
    expect(res.getJSON()).toEqual({
      success: true,
      ok: true,
      audio_strategy: 'synthetic_sink',
      host_audio_outcome: 'no_device',
      host_audio_reason: 'test host has no output',
    });
  });

  it('handles execution errors and increments metrics', async () => {
    const session = buildSession();
    const sessionManager = buildSessionManager(session);
    const idempotencyCache = {
      lookup: jest.fn().mockReturnValue(null),
      store: jest.fn(),
    };
    mockGetIdempotencyCache.mockReturnValue(idempotencyCache as never);
    mockValidateInstruction.mockReturnValue({
      valid: true,
      instruction: { type: 'click', index: 3, nodeId: 'node-3' },
    } as never);
    mockCreateInstructionKey.mockReturnValue('key-3');
    mockExecuteInstruction.mockRejectedValue(new Error('boom'));

    const req = createMockHttpRequest({
      body: { instruction: { type: 'click' } },
    });
    const res = createMockHttpResponse();

    await handleSessionRun(
      req,
      res,
      session.id,
      sessionManager,
      handlerRegistry,
      config,
      logger,
      metrics
    );

    expect(sessionManager.setSessionPhase).toHaveBeenCalledWith(session.id, 'ready');
    expect(metrics.instructionErrors.inc).toHaveBeenCalledWith({
      type: 'unknown',
      error_kind: 'engine',
    });
    expect(res.statusCode).toBe(500);
    expect(res.getJSON().error.code).toBe('INTERNAL_ERROR');
  });
});
