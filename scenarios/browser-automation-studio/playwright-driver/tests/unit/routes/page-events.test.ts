import * as pageEvents from '../../../src/routes/record-mode/page-events';
import type { Config } from '../../../src/config';
import type { SessionManager } from '../../../src/session';

const { sendPageEvent, setupPageLifecycleListeners, pageEventCircuitBreaker } = pageEvents;

jest.mock('../../../src/routes/record-mode/recording-pages', () => ({
  captureThumbnail: jest.fn().mockResolvedValue('thumb'),
  emitHistoryCallback: jest.fn().mockResolvedValue(undefined),
}));

describe('page event routes', () => {
  const config: Config = {
    history: {
      callbackUrl: 'http://callback',
      thumbnailEnabled: true,
      thumbnailQuality: 60,
    },
  } as Config;

  beforeEach(() => {
    global.fetch = jest.fn().mockResolvedValue({ ok: true, status: 200, statusText: 'OK' }) as typeof fetch;
    if (!global.crypto) {
      (global as typeof globalThis).crypto = { randomUUID: jest.fn().mockReturnValue('page-1') } as Crypto;
    } else if (!('randomUUID' in global.crypto)) {
      global.crypto.randomUUID = jest.fn().mockReturnValue('page-1');
    }
  });

  afterEach(() => {
    pageEventCircuitBreaker.cleanup('session-1');
    jest.clearAllMocks();
  });

  it('sends page event via callback when circuit is closed', async () => {
    const event = {
      sessionId: 'session-1',
      driverPageId: 'page-1',
      vrooliPageId: '',
      eventType: 'created',
      url: 'https://example.com',
      title: 'Example',
      timestamp: new Date().toISOString(),
    };

    await sendPageEvent('session-1', 'http://callback', event);

    expect(global.fetch).toHaveBeenCalledWith('http://callback', expect.objectContaining({
      method: 'POST',
    }));
  });

  it('skips sending when circuit is open and half-open not allowed', async () => {
    for (let i = 0; i < 5; i += 1) {
      pageEventCircuitBreaker.recordFailure('session-1');
    }

    await sendPageEvent('session-1', 'http://callback', {
      sessionId: 'session-1',
      driverPageId: 'page-2',
      vrooliPageId: '',
      eventType: 'created',
      url: 'https://example.com',
      title: 'Example',
      timestamp: new Date().toISOString(),
    });

    expect(global.fetch).not.toHaveBeenCalled();
  });

  it('sets up listeners for new pages and emits events', async () => {
    const pageHandlers: Record<string, (arg?: unknown) => Promise<void> | void> = {};
    const mainFrame = {};
    const newPage = {
      opener: jest.fn().mockResolvedValue(null),
      waitForLoadState: jest.fn().mockResolvedValue(undefined),
      url: jest.fn().mockReturnValue('https://example.com'),
      title: jest.fn().mockResolvedValue('Example'),
      mainFrame: jest.fn().mockReturnValue(mainFrame),
      on: jest.fn((event: string, handler: (arg?: unknown) => void) => {
        pageHandlers[event] = handler;
      }),
    } as unknown as Parameters<SessionManager['getSession']>[0]['page'];

    const contextHandlers: Record<string, (page: typeof newPage) => Promise<void>> = {};
    const session = {
      context: {
        on: jest.fn((event: string, handler: (page: typeof newPage) => Promise<void>) => {
          contextHandlers[event] = handler;
        }),
        off: jest.fn(),
      },
      pages: [],
      pageIdMap: new Map(),
      pageToIdMap: new Map(),
    } as unknown as ReturnType<SessionManager['getSession']>;

    const cleanup = setupPageLifecycleListeners('session-1', session, 'http://callback', config);

    await contextHandlers.page?.(newPage);

    expect(session.pages).toHaveLength(1);
    const createdPayload = (global.fetch as jest.Mock).mock.calls[0]?.[1]?.body as string;
    expect(createdPayload).toContain('\"eventType\":\"created\"');

    await pageHandlers.framenavigated?.(mainFrame);
    const navigatedPayload = (global.fetch as jest.Mock).mock.calls[1]?.[1]?.body as string;
    expect(navigatedPayload).toContain('\"eventType\":\"navigated\"');

    await pageHandlers.close?.();
    const closedPayload = (global.fetch as jest.Mock).mock.calls[2]?.[1]?.body as string;
    expect(closedPayload).toContain('\"eventType\":\"closed\"');

    cleanup();
    expect(session.context.off).toHaveBeenCalled();
  });
});
