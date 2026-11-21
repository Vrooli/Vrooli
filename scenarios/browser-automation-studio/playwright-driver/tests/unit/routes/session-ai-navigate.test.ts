import {
  createMockHttpRequest,
  createMockHttpResponse,
  createTestConfig,
  createMockPage,
} from '../../helpers';
import type { SessionState } from '../../../src/types/session';

jest.mock('../../../src/ai/vision-agent', () => ({
  createVisionAgent: jest.fn(),
}));
jest.mock('../../../src/ai/vision-client/factory', () => ({
  createVisionClient: jest.fn(() => ({})),
  isModelSupported: jest.fn(() => true),
  getSupportedModelIds: jest.fn(() => ['fixture-model']),
}));
jest.mock('../../../src/ai/screenshot', () => ({
  createScreenshotCapture: jest.fn(() => ({})),
  createElementAnnotator: jest.fn(() => ({})),
}));
jest.mock('../../../src/ai/action', () => ({
  createActionExecutor: jest.fn(() => ({})),
}));
jest.mock('../../../src/ai/emitter', () => ({
  createCallbackEmitter: jest.fn(() => ({})),
  emitNavigationComplete: jest.fn().mockResolvedValue(undefined),
}));

import { createVisionAgent } from '../../../src/ai/vision-agent';
import { handleSessionAINavigate } from '../../../src/routes/session-ai-navigate';

const ownerHeaders = {
  'x-bas-execution-id': 'execution-owner',
  'x-bas-session-lease-id': 'lease-token',
};

const mockCreateVisionAgent = createVisionAgent as jest.MockedFunction<typeof createVisionAgent>;

describe('AI navigation lifecycle ownership', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('rejects account-backed navigation without a signed session header', async () => {
    const session = {
      id: 'session-ai-no-token',
      phase: 'ready',
      page: createMockPage(),
      context: {},
    } as unknown as SessionState;
    const sessionManager = { getSessionForLease: jest.fn().mockReturnValue(session) } as never;
    const response = createMockHttpResponse();

    await handleSessionAINavigate(
      createMockHttpRequest({
        method: 'POST',
        headers: ownerHeaders,
        body: {
          prompt: 'fixture task',
          model: 'fixture-model',
          callback_url: 'http://callback.test',
          root_operation_id: 'd'.repeat(32),
          ai_source: 'vrooli-credits',
        },
      }),
      response,
      session.id,
      sessionManager,
      createTestConfig()
    );

    expect(response.statusCode).toBe(401);
    expect(mockCreateVisionAgent).not.toHaveBeenCalled();
  });

  it('registers one session cleanup owner and joins navigation settlement', async () => {
    let settleNavigation!: (result: never) => void;
    const navigation = new Promise<never>((resolve) => {
      settleNavigation = resolve;
    });
    const agent = {
      navigate: jest.fn().mockReturnValue(navigation),
      abort: jest.fn(),
      isNavigating: jest.fn().mockReturnValue(true),
      resume: jest.fn(),
      isPaused: jest.fn().mockReturnValue(false),
    };
    mockCreateVisionAgent.mockReturnValueOnce(agent as never);
    const session = {
      id: 'session-ai-lifecycle',
      phase: 'ready',
      page: createMockPage(),
      context: {},
    } as unknown as SessionState;
    const sessionManager = { getSessionForLease: jest.fn().mockReturnValue(session) } as never;
    const response = createMockHttpResponse();

    await handleSessionAINavigate(
      createMockHttpRequest({
        method: 'POST',
        headers: ownerHeaders,
        body: {
          prompt: 'fixture task',
          model: 'fixture-model',
          callback_url: 'http://callback.test',
          root_operation_id: 'd'.repeat(32),
          ai_source: 'local',
        },
      }),
      response,
      session.id,
      sessionManager,
      createTestConfig()
    );

    expect(response.statusCode).toBe(202);
    expect(session.aiNavigationCleanup).toBeDefined();
    const cleanup = session.aiNavigationCleanup;
    if (!cleanup) throw new Error('AI navigation cleanup was not registered');
    const waiting = cleanup();
    await Promise.resolve();
    expect(agent.abort).toHaveBeenCalledTimes(1);

    settleNavigation({
      navigationId: 'fixture-navigation',
      status: 'aborted',
      totalSteps: 0,
      totalTokens: 0,
      totalDurationMs: 0,
      finalUrl: '',
    });
    await waiting;
    expect(session.aiNavigationCleanup).toBeUndefined();
  });

  it('denies AI navigation for a missing or stale execution lease before touching the page', async () => {
    const session = {
      id: 'session-ai-foreign-lease',
      phase: 'ready',
      page: createMockPage(),
      context: {},
    } as unknown as SessionState;
    const sessionManager = { getSessionForLease: jest.fn(() => { throw new Error('session not found'); }) } as never;
    const response = createMockHttpResponse();

    await handleSessionAINavigate(
      createMockHttpRequest({
        method: 'POST',
        headers: ownerHeaders,
        body: {
          prompt: 'fixture task',
          model: 'fixture-model',
          callback_url: 'http://callback.test',
          root_operation_id: 'd'.repeat(32),
          ai_source: 'local',
        },
      }),
      response,
      session.id,
      sessionManager,
      createTestConfig()
    );

    expect(response.statusCode).toBe(404);
    expect(mockCreateVisionAgent).not.toHaveBeenCalled();
  });
});
