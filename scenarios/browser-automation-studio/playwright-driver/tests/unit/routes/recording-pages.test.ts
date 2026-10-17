import type { Page, Frame } from 'rebrowser-playwright';
import {
  createMockHttpRequest,
  createMockHttpResponse,
  createTestConfig,
  installFetchMock,
} from '../../helpers';
import { SessionManager } from '../../../src/session';
import { DriverPageBindings } from '../../../src/session/page-bindings';

function pageBindings(bindings: Array<[string, Page]>): DriverPageBindings {
  const pageBindings = new DriverPageBindings();
  bindings.forEach(([id, page]) => pageBindings.register(page, id));
  return pageBindings;
}

function mockContext(initialPages: Page[], createPage?: Page) {
  let pages = [...initialPages];
  return {
    pages: jest.fn(() => pages.filter(page => !page.isClosed?.())),
    newPage: jest.fn(async () => {
      if (!createPage) throw new Error('no test page configured');
      pages.push(createPage);
      return createPage;
    }),
    removePage: (page: Page) => { pages = pages.filter(candidate => candidate !== page); },
    setPages: (next: Page[]) => { pages = [...next]; },
  };
}

jest.mock('../../../src/session/frame-cache', () => ({
  clearFrameCache: jest.fn(),
}));

jest.mock('../../../src/utils', () => ({
  ...jest.requireActual('../../../src/utils'),
  logger: {
    info: jest.fn(),
    warn: jest.fn(),
    error: jest.fn(),
    debug: jest.fn(),
  },
}));

import {
  captureThumbnail,
  emitHistoryCallback,
  handleRecordActivePage,
  handleRecordNewPage,
  handleRecordClosePage,
  unregisterRecordingPage,
} from '../../../src/routes/record-mode/recording-pages';

describe('recording pages', () => {
  const config = createTestConfig({
    history: {
      callbackUrl: 'https://history.example.com/callback',
      thumbnailEnabled: true,
      thumbnailQuality: 60,
    },
  });

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('captureThumbnail', () => {
    it('returns a base64 thumbnail when screenshot succeeds', async () => {
      const page = {
        screenshot: jest.fn().mockResolvedValue(Buffer.from('thumb')),
      } as unknown as Page;

      const result = await captureThumbnail(page, 70);

      expect(result).toBe(Buffer.from('thumb').toString('base64'));
      expect(page.screenshot).toHaveBeenCalledWith({ type: 'jpeg', quality: 70, fullPage: false });
    });

    it('returns undefined when screenshot fails', async () => {
      const page = {
        screenshot: jest.fn().mockRejectedValue(new Error('boom')),
      } as unknown as Page;

      const result = await captureThumbnail(page, 70);

      expect(result).toBeUndefined();
    });
  });

  describe('emitHistoryCallback', () => {
    it('skips callback when url is not configured', async () => {
      const localConfig = createTestConfig({ history: { callbackUrl: '' } });
      const fetchMock = installFetchMock();

      await emitHistoryCallback(localConfig, 'session-1', 'https://example.com', 'Example', 'navigate');

      expect(fetchMock).not.toHaveBeenCalled();
    });

    it('sends callback and ignores non-ok responses', async () => {
      const fetchMock = installFetchMock();
      fetchMock.mockResolvedValue({
        ok: false,
        status: 500,
        statusText: 'Server error',
      } as Response);

      await emitHistoryCallback(config, 'session-2', 'https://example.com', 'Example', 'navigate', 'thumb');

      expect(fetchMock).toHaveBeenCalledTimes(1);
      const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(url).toBe('https://history.example.com/callback');
      expect(init.method).toBe('POST');
      expect(init.headers).toEqual({ 'Content-Type': 'application/json' });
      expect(String(init.body)).toContain('session-2');
      expect(String(init.body)).toContain('https://example.com');
    });
  });

  describe('handleRecordNewPage', () => {
    it('creates a new page and updates session tracking', async () => {
      const newPage = {
        goto: jest.fn().mockResolvedValue(undefined),
        title: jest.fn().mockResolvedValue('New Page'),
        url: jest.fn().mockReturnValue('https://example.com'),
        isClosed: () => false,
      } as unknown as Page;

      const session = {
        phase: 'ready', ownerExecutionId: 'owner', leaseId: 'lease',
        context: mockContext([], newPage),
        pageBindings: pageBindings([]),
        frameStack: [{} as Frame],
        page: undefined as Page | undefined,
      };

      const sessionManager = {
        getSession: jest.fn().mockReturnValue(session),
        getSessionForLease: jest.fn().mockReturnValue(session), updateActivity: jest.fn(),
      } as unknown as SessionManager;

      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/abc/record/new-page',
        body: { execution_id: 'owner', lease_id: 'lease', url: 'https://example.com' },
      });
      const res = createMockHttpResponse();

      await handleRecordNewPage(req, res, 'abc', sessionManager, config);

      expect(res.statusCode).toBe(201);
      const payload = res.getJSON();
      expect(payload.url).toBe('https://example.com');
      expect(payload.title).toBe('New Page');
      expect(session.context.pages()).toHaveLength(1);
      expect(session.page).toBe(newPage);
      expect(session.frameStack).toEqual([]);
    });

    // [REQ:BAS-RH-J01] [REQ:BAS-RH-J07] Failed navigation cannot commit an error page.
    it.each([
      ['https://unreachable.test', false],
      ['about:blank', false],
      ['https://unreachable.test', true],
    ])('rejects failed navigation to %s, cleanup failure=%s', async (url, cleanupFails) => {
      const original = {} as Page;
      const newPage = {
        goto: jest.fn().mockRejectedValue(new Error('controlled navigation failure')),
        close: cleanupFails
          ? jest.fn().mockRejectedValue(new Error('controlled close failure'))
          : jest.fn().mockResolvedValue(undefined),
        title: jest.fn().mockResolvedValue('Error page'),
        url: jest.fn().mockReturnValue('chrome-error://chromewebdata/'),
      } as unknown as Page;
      const frame = {} as Frame;
      const session = {
        phase: 'ready', ownerExecutionId: 'owner', leaseId: 'lease',
        context: mockContext([original, newPage], newPage),
        pageBindings: pageBindings([['original', original], ['new', newPage]]),
        frameStack: [frame],
        page: original,
      };
      const manager = { getSession: jest.fn().mockReturnValue(session), getSessionForLease: jest.fn().mockReturnValue(session), updateActivity: jest.fn() } as unknown as SessionManager;
      const req = createMockHttpRequest({ method: 'POST', body: { execution_id: 'owner', lease_id: 'lease', url } });
      const res = createMockHttpResponse();
      await handleRecordNewPage(req, res, 'abc', manager, config);
      expect(res.statusCode).toBeGreaterThanOrEqual(400);
      expect(JSON.stringify(res.getJSON())).toContain('controlled navigation failure');
      expect(newPage.close).toHaveBeenCalledTimes(1);
      expect(session.page).toBe(original);
      expect(session.context.pages().indexOf(session.page)).toBe(0);
      expect(session.frameStack).toEqual([frame]);
      if (cleanupFails) {
        expect(JSON.stringify(res.getJSON())).toContain('controlled close failure');
      } else {
        expect(session.context.pages()).toContain(original);
        expect(session.pageBindings.has(newPage)).toBe(false);
        expect(session.pageBindings.getPage('new')).toBeUndefined();
        expect(session.pageBindings.getPage('original')).toBe(original);
      }
    });
  });

  describe('handleRecordActivePage', () => {
    it('returns 400 when page_id is missing', async () => {
      const session = {
        phase: 'ready', ownerExecutionId: 'owner', leaseId: 'lease',
        context: mockContext([]),
        pageBindings: pageBindings([]),
        page: undefined as Page | undefined,
      };
      const sessionManager = {
        getSession: jest.fn().mockReturnValue(session),
        getSessionForLease: jest.fn().mockReturnValue(session), updateActivity: jest.fn(),
      } as unknown as SessionManager;

      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/abc/record/active-page',
        body: { execution_id: 'owner', lease_id: 'lease',},
      });
      const res = createMockHttpResponse();

      await handleRecordActivePage(req, res, 'abc', sessionManager, config);

      expect(res.statusCode).toBe(400);
      expect(res.getJSON().error).toBe('MISSING_PAGE_ID');
    });

    it('returns 404 when page id is unknown', async () => {
      const session = {
        phase: 'ready', ownerExecutionId: 'owner', leaseId: 'lease',
        context: mockContext([]),
        pageBindings: pageBindings([['known', {} as Page]]),
        page: undefined as Page | undefined,
      };
      const sessionManager = {
        getSession: jest.fn().mockReturnValue(session),
        getSessionForLease: jest.fn().mockReturnValue(session), updateActivity: jest.fn(),
      } as unknown as SessionManager;

      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/abc/record/active-page',
        body: { execution_id: 'owner', lease_id: 'lease', page_id: 'missing' },
      });
      const res = createMockHttpResponse();

      await handleRecordActivePage(req, res, 'abc', sessionManager, config);

      expect(res.statusCode).toBe(404);
      expect(res.getJSON().available_page_ids).toEqual(['known']);
    });

    it('returns 410 when page is closed', async () => {
      const page = { isClosed: jest.fn().mockReturnValue(true) } as unknown as Page;
      const session = {
        phase: 'ready', ownerExecutionId: 'owner', leaseId: 'lease',
        context: mockContext([page]),
        pageBindings: pageBindings([['page-1', page]]),
        page,
      };
      const sessionManager = {
        getSession: jest.fn().mockReturnValue(session),
        getSessionForLease: jest.fn().mockReturnValue(session), updateActivity: jest.fn(),
      } as unknown as SessionManager;

      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/abc/record/active-page',
        body: { execution_id: 'owner', lease_id: 'lease', page_id: 'page-1' },
      });
      const res = createMockHttpResponse();

      await handleRecordActivePage(req, res, 'abc', sessionManager, config);

      expect(res.statusCode).toBe(410);
      expect(res.getJSON().error).toBe('PAGE_CLOSED');
    });

    it('switches active page and returns payload', async () => {
      const pageA = { isClosed: jest.fn().mockReturnValue(false) } as unknown as Page;
      const pageB = {
        isClosed: jest.fn().mockReturnValue(false),
        url: jest.fn().mockReturnValue('https://example.com'),
        isClosed: () => false,
        title: jest.fn().mockResolvedValue('Example'),
      } as unknown as Page;

      const session = {
        phase: 'ready', ownerExecutionId: 'owner', leaseId: 'lease',
        context: mockContext([pageA, pageB]),
        pageBindings: pageBindings([['page-a', pageA], ['page-b', pageB]]),
        page: pageA,
        frameStack: [{} as Frame],
      };

      const sessionManager = {
        getSession: jest.fn().mockReturnValue(session),
        getSessionForLease: jest.fn().mockReturnValue(session), updateActivity: jest.fn(),
      } as unknown as SessionManager;

      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/abc/record/active-page',
        body: { execution_id: 'owner', lease_id: 'lease', page_id: 'page-b' },
      });
      const res = createMockHttpResponse();

      await handleRecordActivePage(req, res, 'abc', sessionManager, config);

      expect(res.statusCode).toBe(200);
      const payload = res.getJSON();
      expect(payload.active_page_id).toBe('page-b');
      expect(session.page).toBe(pageB);
      expect(session.frameStack).toEqual([]);
      expect(session.context.pages().indexOf(session.page)).toBe(1);
    });
  });
});


describe('tab command authority [REQ:BAS-RH-J03] [REQ:BAS-RH-J17]', () => {
  const config = createTestConfig();
  const handlers = { create: handleRecordNewPage, activate: handleRecordActivePage };
  function deferred<T>() {
    let resolve!: (value: T) => void;
    const promise = new Promise<T>(done => { resolve = done; });
    return { promise, resolve };
  }
  function fixture(operation: keyof typeof handlers) {
    const original = { isClosed: () => false } as Page;
    const target = {
      goto: jest.fn().mockResolvedValue(undefined), title: jest.fn().mockResolvedValue('Target'),
      url: () => 'https://fixture.test/target', isClosed: () => false,
      close: jest.fn().mockResolvedValue(undefined),
    } as unknown as Page;
    const context = mockContext([original], target);
    const session = {
      id: 'owned-tabs', phase: 'ready', ownerExecutionId: 'owner', leaseId: 'lease',
      leaseReleasedAt: undefined as Date | undefined,
      context,
      page: original,
      pageBindings: pageBindings([['original', original]]),
      frameStack: [{} as Frame],
    };
    if (operation === 'activate') session.pageBindings.register(target, 'target');
    const manager = {
      getSession: () => session, peekSession: () => session,
      getSessionForLease: SessionManager.prototype.getSessionForLease, updateActivity: jest.fn(),
    } as unknown as SessionManager;
    const call = (body: Record<string, unknown> = {}) => {
      const response = createMockHttpResponse();
      const pending = handlers[operation](createMockHttpRequest({method: 'POST', body: {
        execution_id: 'owner', lease_id: 'lease', url: 'https://fixture.test/target', page_id: 'target', ...body,
      }}), response, 'owned-tabs', manager, config);
      return { response, pending };
    };
    return { session, manager, original, target, call };
  }
  describe.each(['create', 'activate'] as const)('%s', operation => {
    it.each(['missing', 'wrong', 'released', 'closing', 'body handoff'])('rejects %s authority before effects', async fault => {
      const f = fixture(operation);
      if (fault === 'released') f.session.leaseReleasedAt = new Date();
      if (fault === 'closing') f.session.phase = 'closing';
      const body = fault === 'missing' ? {execution_id: undefined, lease_id: undefined}
        : fault === 'wrong' ? {execution_id: 'wrong', lease_id: 'wrong'} : {};
      const call = f.call(body);
      if (fault === 'body handoff') f.session.leaseId = 'replacement';
      await call.pending;
      expect(call.response.statusCode).toBe(fault === 'missing' ? 400 : 404);
      expect(f.session.context.newPage).not.toHaveBeenCalled();
      expect(f.target.title).not.toHaveBeenCalled();
      expect(f.target.goto).not.toHaveBeenCalled();
      expect(f.session.page).toBe(f.original);
      expect(f.session.context.pages().indexOf(f.session.page)).toBe(0);
      expect(f.session.frameStack).toHaveLength(1);
      expect(f.manager.updateActivity).not.toHaveBeenCalled();
    });
    it('accepts the current lease', async () => {
      const f = fixture(operation);const call = f.call();await call.pending;
      expect(call.response.statusCode).toBe(operation === 'create' ? 201 : 200);
      expect(f.session.page).toBe(f.target);
    });
    it('cannot select or acknowledge after ownership changes during page title', async () => {
      const f = fixture(operation), started = deferred<void>(), title = deferred<string>();
      jest.mocked(f.target.title).mockImplementation(() => {started.resolve();return title.promise;});
      const call = f.call();await started.promise;f.session.leaseId = 'replacement';title.resolve('Late title');await call.pending;
      expect(call.response.statusCode).toBe(404);
      expect(f.session.page).toBe(f.original);
      expect(f.session.context.pages().indexOf(f.session.page)).toBe(0);
      expect(f.session.frameStack).toHaveLength(1);
      expect(f.target.close).toHaveBeenCalledTimes(operation === 'create' ? 1 : 0);
      if (operation === 'create') expect(f.session.pageBindings.has(f.target)).toBe(false);
    });
  });
  it.each(['acquire', 'navigate'])('disposes the command page after handoff during %s', async stage => {
    const f = fixture('create'), started = deferred<void>(), gate = deferred<void>();
    if (stage === 'acquire') f.session.context.newPage.mockImplementation(async () => {started.resolve();await gate.promise;return f.target;});
    else jest.mocked(f.target.goto).mockImplementation(async () => {started.resolve();await gate.promise;return null;});
    const call = f.call();await started.promise;f.session.leaseId = 'replacement';gate.resolve();await call.pending;
    expect(call.response.statusCode).toBe(404);
    expect(f.target.close).toHaveBeenCalledTimes(1);
    expect(f.target.title).not.toHaveBeenCalled();
    expect(f.session.page).toBe(f.original);
    expect(f.session.context.pages()).toContain(f.original);
    if (stage === 'acquire') expect(f.target.goto).not.toHaveBeenCalled();
  });
});

describe('tab removal selection [REQ:BAS-RH-J03]', () => {
  it.each(['active', 'inactive', 'last'])('keeps a valid selection when removing %s page', kind => {
    const first = {isClosed: () => false} as Page;
    const target = {isClosed: () => true} as Page;
    const third = {isClosed: () => false} as Page;
    const context = mockContext(kind === 'last' ? [target] : [first, target, third]);
    const session = {
      id: 'close-tabs', page: kind === 'inactive' ? third : target,
      context,
      pageBindings: pageBindings(
        kind === 'last' ? [['target', target]] : [['first', first], ['target', target], ['third', third]]),
      frameStack: [{} as Frame],
    };
    unregisterRecordingPage(session as unknown as ReturnType<SessionManager['getSession']>, target);
    expect(session.context.pages()).not.toContain(target);
    expect(session.pageBindings.getPage('target')).toBeUndefined();
    expect(session.pageBindings.has(target)).toBe(false);
    expect(session.context.pages().indexOf(session.page)).toBe(kind === 'last' ? -1 : kind === 'inactive' ? 1 : 0);
    expect(session.page).toBe(kind === 'last' ? target : kind === 'inactive' ? third : first);
    expect(session.frameStack).toHaveLength(kind === 'inactive' ? 1 : 0);
  });
});

describe('leased browser tab closure [REQ:BAS-RH-J03] [REQ:BAS-RH-J17]', () => {
  const config = createTestConfig();
  function fixture() {
    const first = {isClosed: () => false} as Page;
    let targetClosed = false;
    const target = {isClosed: () => targetClosed, close: jest.fn(async () => {targetClosed = true;context.removePage(target);})} as unknown as Page;
    const context = mockContext([first, target]);
    const session = {
      id: 'close-tabs', phase: 'ready', ownerExecutionId: 'owner', leaseId: 'lease',
      leaseReleasedAt: undefined as Date | undefined,
      page: target, context, pageBindings: pageBindings([['first', first], ['target', target]]), frameStack: [{} as Frame],
    };
    const manager = {
      getSession: () => session, peekSession: () => session,
      getSessionForLease: SessionManager.prototype.getSessionForLease, updateActivity: jest.fn(),
    } as unknown as SessionManager;
    const call = (body: Record<string, unknown> = {}) => {
      const response = createMockHttpResponse();
      const pending = handleRecordClosePage(createMockHttpRequest({method: 'POST', body: {
        execution_id: 'owner', lease_id: 'lease', page_id: 'target', ...body,
      }}), response, 'close-tabs', manager, config);
      return {response, pending};
    };
    return {first, target, session, manager, context, closeTarget: () => {targetClosed = true;context.removePage(target);}, call};
  }
  it.each(['missing lease', 'wrong lease', 'released', 'closing', 'body handoff', 'missing page', 'unknown page'])('rejects %s before browser effects', async fault => {
    const f = fixture();
    if (fault === 'released') f.session.leaseReleasedAt = new Date();
    if (fault === 'closing') f.session.phase = 'closing';
    const body = fault === 'missing lease' ? {execution_id: undefined, lease_id: undefined}
      : fault === 'wrong lease' ? {execution_id: 'wrong', lease_id: 'wrong'}
      : fault === 'missing page' ? {page_id: undefined} : fault === 'unknown page' ? {page_id: 'absent'} : {};
    const call = f.call(body);
    if (fault === 'body handoff') f.session.leaseId = 'replacement';
    await call.pending;
    expect(call.response.statusCode).toBe(fault === 'missing lease' || fault === 'missing page' ? 400 : 404);
    expect(f.target.close).not.toHaveBeenCalled();
    expect(f.session.page).toBe(f.target);
    expect(f.session.pageBindings.getPage('target')).toBe(f.target);
  });
  it.each([false, true])('closes the selected browser page with callback-first=%s', async callbackFirst => {
    const f = fixture();
    if (callbackFirst) jest.mocked(f.target.close).mockImplementation(async () => {f.closeTarget(); unregisterRecordingPage(f.session as unknown as ReturnType<SessionManager['getSession']>, f.target);});
    const call = f.call(); await call.pending;
    expect(f.target.close).toHaveBeenCalledTimes(1);
    expect(call.response.statusCode).toBe(200);
    expect(call.response.getJSON()).toEqual({closed_page_id: 'target', active_page_id: 'first'});
    expect(f.session.page).toBe(f.first);
    expect(f.session.context.pages()).toEqual([f.first]);
    expect(f.session.context.pages().indexOf(f.session.page)).toBe(0);
    expect(f.session.frameStack).toHaveLength(0);
  });
  it('closes the last page without inventing a selected page', async () => {
    const f = fixture();
    f.session.pageBindings.retain(f.target);
    f.context.setPages([f.target]);
    const call = f.call(); await call.pending;
    expect(call.response.getJSON()).toEqual({closed_page_id: 'target', active_page_id: ''});
    expect(f.session.context.pages()).toHaveLength(0);
    expect(f.session.context.pages().indexOf(f.session.page)).toBe(-1);
  });
  it('preserves the open tab after a browser close failure', async () => {
    const f = fixture(); jest.mocked(f.target.close).mockRejectedValue(new Error('browser close failed'));
    const call = f.call(); await call.pending;
    expect(call.response.statusCode).toBe(500);
    expect(f.session.page).toBe(f.target);
    expect(f.session.context.pages()).toEqual([f.first, f.target]);
    expect(f.session.frameStack).toHaveLength(1);
  });
  it('does not commit selection to a replacement owner after closing completes', async () => {
    const f = fixture(); let finish!: () => void, started!: () => void;
    const entered = new Promise<void>(resolve => {started = resolve;});
    const pendingClose = new Promise<void>(resolve => {finish = resolve;});
    jest.mocked(f.target.close).mockImplementation(() => {started(); return pendingClose;});
    const call = f.call(); await entered; f.session.leaseId = 'replacement'; finish(); await call.pending;
    expect(call.response.statusCode).toBe(404);
    expect(f.session.page).toBe(f.target);
    expect(f.session.context.pages()).toEqual([f.first, f.target]);
    expect(f.session.frameStack).toHaveLength(1);
  });
});
