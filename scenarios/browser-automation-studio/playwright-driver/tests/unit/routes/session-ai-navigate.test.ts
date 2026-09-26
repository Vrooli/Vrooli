import { createMockHttpRequest, createMockHttpResponse, createTestConfig, createMockPage } from '../../helpers';
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

const mockCreateVisionAgent = createVisionAgent as jest.MockedFunction<typeof createVisionAgent>;

describe('AI navigation lifecycle ownership', () => {
  it('registers one session cleanup owner and joins navigation settlement', async () => {
    let settleNavigation!: (result: never) => void;
    const navigation = new Promise<never>((resolve) => { settleNavigation = resolve; });
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
    const sessionManager = { getSession: jest.fn().mockReturnValue(session) } as never;
    const response = createMockHttpResponse();

    await handleSessionAINavigate(
      createMockHttpRequest({
        method: 'POST',
        body: { prompt: 'fixture task', model: 'fixture-model', callback_url: 'http://callback.test' },
      }),
      response,
      session.id,
      sessionManager,
      createTestConfig(),
    );

    expect(response.statusCode).toBe(202);
    expect(session.aiNavigationCleanup).toBeDefined();
    const cleanup = session.aiNavigationCleanup;
    if (!cleanup) throw new Error('AI navigation cleanup was not registered');
    const waiting = cleanup();
    await Promise.resolve();
    expect(agent.abort).toHaveBeenCalledTimes(1);

    settleNavigation({
      navigationId: 'fixture-navigation', status: 'aborted', totalSteps: 0,
      totalTokens: 0, totalDurationMs: 0, finalUrl: '',
    });
    await waiting;
    expect(session.aiNavigationCleanup).toBeUndefined();
  });
});
