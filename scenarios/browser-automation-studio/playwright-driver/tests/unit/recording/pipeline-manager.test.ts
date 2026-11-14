import type { BrowserContext, Page } from 'rebrowser-playwright';
import {
  RecordingPipelineManager,
  type PipelineManagerOptions,
} from '../../../src/recording/orchestration/pipeline-manager';
import type { RecordingContextInitializer } from '../../../src/recording/io/context-initializer';
import { waitForScriptReady } from '../../../src/recording/validation/verification';

jest.mock('../../../src/recording/validation/verification', () => ({
  waitForScriptReady: jest.fn(),
}));

const waitForScriptReadyMock = waitForScriptReady as jest.MockedFunction<typeof waitForScriptReady>;

const verification = {
  loaded: true,
  ready: true,
  inMainContext: true,
  handlersCount: 7,
  version: 'test',
};

function createInitializer(eventRouteActive: { value: boolean }): RecordingContextInitializer {
  return {
    isInitialized: jest.fn().mockReturnValue(true),
    hasEventRoute: jest.fn().mockImplementation(() => eventRouteActive.value),
    setEventHandler: jest.fn(),
    clearEventHandler: jest.fn(),
    setupPageEventRoute: jest.fn().mockImplementation(() => {
      eventRouteActive.value = true;
      return Promise.resolve();
    }),
  } as unknown as RecordingContextInitializer;
}

function createManager(eventRouteActive: boolean): RecordingPipelineManager {
  const page = { url: jest.fn().mockReturnValue('https://example.com') } as unknown as Page;
  const context = { pages: jest.fn().mockReturnValue([]) } as unknown as BrowserContext;
  return new RecordingPipelineManager(
    page,
    context,
    createInitializer({ value: eventRouteActive }),
    { sessionId: 'pipeline-verification-test' } as PipelineManagerOptions
  );
}

describe('RecordingPipelineManager verification', () => {
  beforeEach(() => {
    waitForScriptReadyMock.mockResolvedValue(verification);
  });

  it('does not admit ready when the page event route is absent', async () => {
    const manager = createManager(false);

    await manager.initialize();
    const result = await manager.verifyPipeline({ retries: 0 });

    expect(result.eventRouteActive).toBe(false);
    expect(manager.getPhase()).toBe('error');
    expect(manager.getError()?.code).toBe('EVENT_ROUTE_FAILED');
  });

  it('admits ready when the page event route is registered', async () => {
    const manager = createManager(true);

    await manager.initialize();
    const result = await manager.verifyPipeline({ retries: 0 });

    expect(result.eventRouteActive).toBe(true);
    expect(manager.getPhase()).toBe('ready');
  });

  it('re-verifies the route after recovering from route failure', async () => {
    const routeState = { value: false };
    const page = { url: jest.fn().mockReturnValue('https://example.com') } as unknown as Page;
    const context = { pages: jest.fn().mockReturnValue([]) } as unknown as BrowserContext;
    const initializer = createInitializer(routeState);
    const manager = new RecordingPipelineManager(
      page,
      context,
      initializer,
      { sessionId: 'pipeline-recovery-test' } as PipelineManagerOptions
    );

    await manager.initialize();
    await manager.verifyPipeline({ retries: 0 });
    expect(manager.getPhase()).toBe('error');

    expect(await manager.attemptRecovery()).toBe(true);
    expect(manager.getPhase()).toBe('ready');
    expect(manager.getVerification()?.eventRouteActive).toBe(true);
  });

  it('detaches page ownership when activation fails but retains stop ownership', async () => {
    const page = {
      url: jest.fn().mockReturnValue('about:blank'),
      frames: jest.fn().mockReturnValue([]),
      evaluate: jest.fn()
        .mockRejectedValueOnce(new Error('activation failed'))
        .mockResolvedValue(undefined),
      on: jest.fn(),
      off: jest.fn(),
    } as unknown as Page;
    const context = {
      pages: jest.fn().mockReturnValue([page]),
      on: jest.fn(),
      off: jest.fn(),
    } as unknown as BrowserContext;
    const manager = new RecordingPipelineManager(
      page,
      context,
      createInitializer({ value: true }),
      { sessionId: 'pipeline-start-failure-test' } as PipelineManagerOptions
    );

    await manager.initialize();
    await manager.verifyPipeline({ retries: 0 });

    await expect(manager.startRecording({
      autoVerify: false,
      onEntry: jest.fn(),
    })).rejects.toThrow('activation failed');

    expect(manager.getPhase()).toBe('error');
    expect(manager.isRecording()).toBe(true);
    expect(context.off).toHaveBeenCalledWith('page', expect.any(Function));
    expect(page.off).toHaveBeenCalledWith('framenavigated', expect.any(Function));
    expect(page.off).toHaveBeenCalledWith('close', expect.any(Function));
    await expect(manager.stopRecording()).resolves.toMatchObject({ actionCount: 0 });
    expect(manager.isRecording()).toBe(false);
  });
});
