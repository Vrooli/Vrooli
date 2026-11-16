import type { BrowserContext, Page } from 'rebrowser-playwright';
import {
  RecordingPipelineManager,
  type PipelineManagerOptions,
} from '../../../src/recording/orchestration/pipeline-manager';
import type { RecordingContextInitializer } from '../../../src/recording/io/context-initializer';
import { waitForScriptReady } from '../../../src/recording/validation/verification';
import { acknowledgeTimelineEntries, getTimelineEntries, removeRecordingBuffer } from '../../../src/recording/io/buffer';
import type { RawBrowserEvent } from '../../../src/recording/types';

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

  it('retains failed delivery ownership so stop can retry and acknowledge before cleanup', async () => {
    const sessionId = 'pipeline-delivery-retry-test';
    let eventHandler: ((event: RawBrowserEvent) => Promise<void>) | undefined;
    const page = {
      url: jest.fn().mockReturnValue('about:blank'),
      frames: jest.fn(),
      mainFrame: jest.fn(),
      evaluate: jest.fn().mockResolvedValue(undefined),
      on: jest.fn(),
      off: jest.fn(),
    } as unknown as Page;
    const frame = {
      waitForLoadState: jest.fn().mockResolvedValue(undefined),
      isDetached: jest.fn().mockReturnValue(false),
      page: jest.fn().mockReturnValue(page),
      url: jest.fn().mockReturnValue('about:blank'),
    };
    (page.frames as jest.Mock).mockReturnValue([frame]);
    (page.mainFrame as jest.Mock).mockReturnValue(frame);
    const context = {
      pages: jest.fn().mockReturnValue([page]),
      on: jest.fn(),
      off: jest.fn(),
    } as unknown as BrowserContext;
    const initializer = {
      isInitialized: jest.fn().mockReturnValue(true),
      hasEventRoute: jest.fn().mockReturnValue(true),
      setEventHandler: jest.fn((handler: (event: RawBrowserEvent) => Promise<void>) => { eventHandler = handler; }),
      clearEventHandler: jest.fn(),
      setupPageEventRoute: jest.fn().mockResolvedValue(undefined),
    } as unknown as RecordingContextInitializer;
    const manager = new RecordingPipelineManager(page, context, initializer, { sessionId } as PipelineManagerOptions);
    const deliver = jest.fn().mockRejectedValueOnce(new Error('temporary API failure')).mockResolvedValue(undefined);

    await manager.initialize();
    await manager.verifyPipeline({ retries: 0 });
    await manager.startRecording({ autoVerify: false, recordingId: 'recording-retry-test', onEntry: deliver });

    const event: RawBrowserEvent = {
      id: 'event-retry-test',
      recordingId: 'recording-retry-test',
      actionType: 'click',
      timestamp: Date.now(),
      selector: { primary: '#submit', candidates: [] },
      elementMeta: { tagName: 'button', isVisible: true, isEnabled: true },
      url: 'about:blank',
    };
    await expect(eventHandler?.(event)).rejects.toThrow('temporary API failure');

    await expect(manager.stopRecording()).resolves.toMatchObject({ actionCount: 1 });

    expect(deliver).toHaveBeenCalledTimes(2);
    expect(manager.isRecording()).toBe(false);
    expect(initializer.clearEventHandler).toHaveBeenCalledTimes(1);
    const entries = getTimelineEntries(sessionId);
    expect(entries).toHaveLength(1);
    acknowledgeTimelineEntries(sessionId, entries.map(({ id }) => id));
    removeRecordingBuffer(sessionId);
  });

  it('waits for every page close acknowledgement before returning a failed multi-page stop', async () => {
    const sessionId = 'pipeline-multipage-close-test';
    let releaseSecondPageStop: (() => void) | undefined;
    let stopSettled = false;
    let eventHandler: ((event: RawBrowserEvent) => Promise<void>) | undefined;
    const createPage = (stop: (call: number) => Promise<void>) => {
      const frame = {
        waitForLoadState: jest.fn().mockResolvedValue(undefined),
        isDetached: jest.fn().mockReturnValue(false),
        page: jest.fn(),
        url: jest.fn().mockReturnValue('about:blank'),
      };
      const evaluate = jest.fn(async (script: string) => {
        if (script.includes('"action":"stop"')) {
          await stop(evaluate.mock.calls.filter(([candidate]) => candidate.includes('"action":"stop"')).length);
        }
      });
      const page = {
        url: jest.fn().mockReturnValue('about:blank'),
        frames: jest.fn().mockReturnValue([frame]),
        mainFrame: jest.fn().mockReturnValue(frame),
        evaluate,
        on: jest.fn(),
        off: jest.fn(),
      } as unknown as Page;
      frame.page.mockReturnValue(page);
      return page;
    };
    const firstPage = createPage(async (call) => {
      if (call === 1) throw new Error('page close delivery failed');
    });
    const secondPage = createPage(async (call) => {
      if (call === 1) await new Promise<void>((resolve) => { releaseSecondPageStop = resolve; });
    });
    const context = {
      pages: jest.fn().mockReturnValue([firstPage, secondPage]),
      on: jest.fn(),
      off: jest.fn(),
    } as unknown as BrowserContext;
    const initializer = {
      isInitialized: jest.fn().mockReturnValue(true),
      hasEventRoute: jest.fn().mockReturnValue(true),
      setEventHandler: jest.fn((handler: (event: RawBrowserEvent) => Promise<void>) => { eventHandler = handler; }),
      clearEventHandler: jest.fn(),
      setupPageEventRoute: jest.fn().mockResolvedValue(undefined),
    } as unknown as RecordingContextInitializer;
    const manager = new RecordingPipelineManager(firstPage, context, initializer, { sessionId } as PipelineManagerOptions);
    const deliver = jest.fn().mockRejectedValueOnce(new Error('recording delivery returned 500')).mockResolvedValue(undefined);

    await manager.initialize();
    await manager.verifyPipeline({ retries: 0 });
    await manager.startRecording({ autoVerify: false, recordingId: 'recording-multipage-close-test', onEntry: deliver });

    await expect(eventHandler?.({
      id: 'event-multipage-close-test',
      recordingId: 'recording-multipage-close-test',
      actionType: 'click',
      timestamp: Date.now(),
      selector: { primary: '#submit', candidates: [] },
      elementMeta: { tagName: 'button', isVisible: true, isEnabled: true },
      url: 'about:blank',
    })).rejects.toThrow('recording delivery returned 500');

    const firstStop = manager.stopRecording();
    firstStop.finally(() => { stopSettled = true; }).catch(() => undefined);
    await Promise.resolve();
    await Promise.resolve();
    expect(releaseSecondPageStop).toBeDefined();
    expect(stopSettled).toBe(false);

    releaseSecondPageStop!();
    await expect(firstStop).rejects.toThrow('page close delivery failed');
    expect(stopSettled).toBe(true);
    expect(deliver).toHaveBeenCalledTimes(1);
    expect(initializer.clearEventHandler).not.toHaveBeenCalled();
    expect(manager.getPhase()).toBe('stopping');
    expect(manager.isRecording()).toBe(true);

    await expect(manager.stopRecording()).resolves.toMatchObject({ recordingId: 'recording-multipage-close-test' });
    expect(deliver).toHaveBeenCalledTimes(2);
    expect(initializer.clearEventHandler).toHaveBeenCalledTimes(1);
    const entries = getTimelineEntries(sessionId);
    expect(entries).toHaveLength(1);
    acknowledgeTimelineEntries(sessionId, entries.map(({ id }) => id));
    removeRecordingBuffer(sessionId);
  });
});
