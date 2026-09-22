import { handleRecordStart } from '../../../src/routes/record-mode/recording-lifecycle';
import { EventEmitter } from 'node:events';
import type { Page } from 'rebrowser-playwright';
import { createMockHttpRequest, createMockHttpResponse, createTestConfig } from '../../helpers';
import { handleRecordNewPage, emitHistoryCallback } from '../../../src/routes/record-mode/recording-pages';
import * as pageEvents from '../../../src/routes/record-mode/page-events';
import type { Config } from '../../../src/config';
import type { SessionManager } from '../../../src/session';
import { installFetchMock } from '../../helpers';

const { sendPageEvent, setupPageLifecycleListeners, pageEventCircuitBreaker } = pageEvents;

jest.mock('../../../src/routes/record-mode/recording-pages', () => ({
  ...jest.requireActual('../../../src/routes/record-mode/recording-pages'),
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
  let fetchMock: ReturnType<typeof installFetchMock>;

  beforeEach(() => {
    fetchMock = installFetchMock();
    fetchMock.mockResolvedValue({ ok: true, status: 200, statusText: 'OK' } as Response);
    if (!global.crypto) {
      (global as { crypto?: Crypto }).crypto = { randomUUID: jest.fn().mockReturnValue('page-1') } as unknown as Crypto;
    } else if (!('randomUUID' in global.crypto)) {
      (global.crypto as { randomUUID: () => string }).randomUUID = jest.fn().mockReturnValue('page-1');
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

    expect(fetchMock).toHaveBeenCalledWith('http://callback', expect.objectContaining({
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

    expect(fetchMock).not.toHaveBeenCalled();
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
      off: jest.fn(),
      isClosed: () => false,
    } as unknown as Page;

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

    const { cleanup, ready } = setupPageLifecycleListeners('session-1', session, 'http://callback', config);
    await ready;

    await contextHandlers.page?.(newPage);

    expect(session.pages).toHaveLength(1);
    const createdPayload = fetchMock.mock.calls[0]?.[1]?.body as string;
    expect(createdPayload).toContain('"eventType":"created"');

    await pageHandlers.framenavigated?.(mainFrame);
    const navigatedPayload = fetchMock.mock.calls[1]?.[1]?.body as string;
    expect(navigatedPayload).toContain('"eventType":"navigated"');

    await pageHandlers.close?.();
    const closedPayload = fetchMock.mock.calls[2]?.[1]?.body as string;
    expect(closedPayload).toContain('"eventType":"closed"');

    cleanup();
    expect(session.context.off).toHaveBeenCalled();
  });
});


describe('recording tab callback ownership [REQ:BAS-RH-J03]', () => {
  const config = createTestConfig({ history: { thumbnailEnabled: false, callbackUrl: '' } });
  let fetchMock: ReturnType<typeof installFetchMock>;
  beforeEach(() => {
    jest.clearAllMocks();
    fetchMock = installFetchMock();
    fetchMock.mockResolvedValue({ ok: true, status: 200, statusText: 'OK' } as Response);
  });
  afterEach(() => pageEventCircuitBreaker.cleanup('owned-tabs'));

  function deferred<T>() {
    let resolve!: (value: T) => void;
    const promise = new Promise<T>((done) => { resolve = done; });
    return { promise, resolve };
  }

  function fixture() {
    const context = new EventEmitter();
    const events = new EventEmitter();
    const frame = {};
    const page = Object.assign(events, {
      opener: jest.fn().mockResolvedValue(null),
      waitForLoadState: jest.fn().mockResolvedValue(undefined),
      goto: jest.fn().mockResolvedValue(undefined),
      url: () => 'https://fixture.invalid/first',
      title: jest.fn().mockResolvedValue('First page'),
      mainFrame: () => frame,
      isClosed: () => false,
    }) as unknown as Page & EventEmitter;
    const session = {
      id: 'owned-tabs', phase: 'recording', ownerExecutionId: 'owner', leaseId: 'lease',
      context, pages: [], pageIdMap: new Map(), pageToIdMap: new WeakMap(),
      page, currentPageIndex: 0, frameStack: [],
    } as unknown as ReturnType<SessionManager['getSession']>;
    const setup = () => setupPageLifecycleListeners('owned-tabs', session, 'http://callback', config);
    const open = () => (context.listeners('page')[0] as (page: Page) => Promise<void>)(page);
    return { context, page, frame, session, setup, open };
  }

  it('removes every page listener and makes cleanup repeatable', async () => {
    const f = fixture(); const { cleanup } = f.setup();
    try {
      await f.open();
      expect(fetchMock).toHaveBeenCalledTimes(1);
      cleanup(); cleanup();
      expect(f.context.listenerCount('page')).toBe(0);
      expect(f.page.listenerCount('framenavigated')).toBe(0);
      expect(f.page.listenerCount('close')).toBe(0);
      f.page.emit('framenavigated', f.frame);
      f.page.emit('close');
      expect(fetchMock).toHaveBeenCalledTimes(1);
    } finally { cleanup(); }
  });

  it('cannot attach or publish a pending popup after cleanup', async () => {
    const f = fixture(); const opener = deferred<Page | null>();
    jest.mocked(f.page.opener).mockReturnValue(opener.promise);
    const { cleanup } = f.setup(); const opening = f.open();
    cleanup(); opener.resolve(null); await opening;
    expect(f.page.listenerCount('framenavigated')).toBe(0);
    expect(f.page.listenerCount('close')).toBe(0);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('cannot publish a pending navigation title after cleanup', async () => {
    const f = fixture(); const { cleanup } = f.setup();
    await f.open();
    const title = deferred<string>();
    jest.mocked(f.page.title).mockReturnValue(title.promise);
    const navigation = (f.page.listeners('framenavigated')[0] as (frame: unknown) => Promise<void>)(f.frame);
    cleanup(); title.resolve('Late title'); await navigation;
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(emitHistoryCallback).not.toHaveBeenCalled();
  });

  it('observes popup lookup rejection without an unhandled event-listener failure', async () => {
    const f = fixture(); const { cleanup } = f.setup();
    jest.mocked(f.page.opener).mockRejectedValue(new Error('Page closed while opening'));
    try { await expect(f.open()).resolves.toBeUndefined(); }
    finally { cleanup(); }
  });

  it('binds initial-page navigation to that page after active-tab selection changes', async () => {
    const f = fixture(); const other = fixture();
    f.session.pages.push(f.page, other.page);
    f.session.pageIdMap.set('first-id', f.page); f.session.pageToIdMap.set(f.page, 'first-id');
    f.session.pageIdMap.set('second-id', other.page); f.session.pageToIdMap.set(other.page, 'second-id');
    const { cleanup, ready } = f.setup(); await ready; fetchMock.mockClear(); f.session.page = other.page;
    try {
      const listeners = f.page.listeners('framenavigated');
      expect(listeners).toHaveLength(1);
      await (listeners[0] as (frame: unknown) => Promise<void>)(f.frame);
      const payload = JSON.parse(fetchMock.mock.calls[0][1]!.body as string);
      expect(payload).toMatchObject({ driverPageId: 'first-id', eventType: 'navigated', url: 'https://fixture.invalid/first', title: 'First page' });
      expect(other.page.title).not.toHaveBeenCalled();
    } finally { cleanup(); }
  });

  it('cannot send the initial page callback when recording stops during its title lookup', async () => {
    const f = fixture();
    f.session.pages.push(f.page);
    f.session.pageIdMap.set('first-id', f.page); f.session.pageToIdMap.set(f.page, 'first-id');
    let capturing = false;
    let generation = 0;
    f.session.pipelineManager = {
      isRecording: () => capturing,
      startRecording: async () => { capturing = true; generation++; return 'recording-id'; },
      getGeneration: () => generation,
      getState: () => ({ phase: capturing ? 'capturing' : 'ready' }),
      getVerification: () => undefined,
    } as unknown as typeof f.session.pipelineManager;
    const title = deferred<string>(); const reading = deferred<void>();
    jest.mocked(f.page.title).mockImplementation(() => { reading.resolve(); return title.promise; });
    const manager = { getSession: () => f.session, getSessionForLease: () => f.session, setSessionPhase: jest.fn() } as unknown as SessionManager;
    const response = createMockHttpResponse();
    const start = handleRecordStart(createMockHttpRequest({ method: 'POST', body: { page_callback_url: 'http://callback' } }),
      response, 'owned-tabs', manager, config);
    await reading.promise;
    f.session.pageLifecycleCleanup?.();
    capturing = false;
    title.resolve('Initial title after stop');
    await start;
    expect(fetchMock).not.toHaveBeenCalled();
    expect(f.page.listenerCount('framenavigated')).toBe(0);
    expect(response.statusCode).toBe(409);
  });

  it('registers an explicitly created tab once and uses the same callback and response identity', async () => {
    const f = fixture(); const { cleanup } = f.setup();
    let discovered: Promise<void> | undefined;
    f.session.context.newPage = jest.fn(async () => { discovered = f.open(); return f.page; });
    const manager = { getSession: () => f.session } as unknown as SessionManager;
    const response = createMockHttpResponse();
    try {
      await handleRecordNewPage(createMockHttpRequest({ method: 'POST', body: { url: 'https://fixture.invalid/first' } }), response, 'owned-tabs', manager, config);
      await discovered;
      expect(response.statusCode).toBe(201);
      expect(f.session.pages).toEqual([f.page]);
      expect(f.session.pageIdMap.size).toBe(1);
      const payload = JSON.parse(fetchMock.mock.calls[0][1]!.body as string);
      expect(response.getJSON().driver_page_id).toBe(payload.driverPageId);
      expect(f.session.pageToIdMap.get(f.page)).toBe(payload.driverPageId);
    } finally { cleanup(); }
  });
});
