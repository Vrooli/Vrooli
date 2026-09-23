import { streamRecordingEntry } from '../../../src/routes/record-mode/callback-streaming';
import { create } from '@bufbuild/protobuf';
import { TimelineEntrySchema } from '../../../src/proto/recording';
import { initRecordingBuffer, bufferTimelineEntry, getTimelineEntries, acknowledgeTimelineEntries, removeRecordingBuffer } from '../../../src/recording';
import { handleRecordNavigate, handleRecordReload, handleRecordGoBack, handleRecordGoForward, handleRecordActions, handleRecordActionsAck, handleRecordStop } from '../../../src/routes/record-mode';
import { handleRecordNavigationState, handleRecordNavigationStack } from '../../../src/routes/record-mode/recording-navigation';
import * as frames from '../../../src/routes/record-mode/recording-frames';
import * as pages from '../../../src/routes/record-mode/recording-pages';
import { createMockHttpRequest, createMockHttpResponse, createTestConfig } from '../../helpers';
import { SessionManager } from '../../../src/session/manager';

const mockHistoryCDP = { send: jest.fn(async () => ({ currentIndex: 0, entries: [{ id: 1, url: 'https://example.com', title: 'Example' }] })), detach: jest.fn(async () => {}) };

// Minimal session manager stub to avoid spinning up Playwright
const mockPage = {
  context: () => ({ newCDPSession: async () => mockHistoryCDP }),
  on: jest.fn(),
  goto: jest.fn().mockResolvedValue(undefined),
  screenshot: jest.fn().mockResolvedValue(Buffer.from('image-bytes')),
  url: jest.fn().mockReturnValue('https://example.com'),
  title: jest.fn().mockResolvedValue('Example'),
};
const mockSessionManager: Pick<SessionManager, 'getSession'> = {
  getSession: () => ({ page: mockPage } as unknown as ReturnType<SessionManager['getSession']>),
};

describe('recording navigation ownership and browser history [REQ:BAS-RH-J17] [REQ:BAS-RH-J04]', () => {
  const handlers = { navigate: handleRecordNavigate, reload: handleRecordReload, 'go-back': handleRecordGoBack, 'go-forward': handleRecordGoForward };
  const methods = { navigate: 'goto', reload: 'reload', 'go-back': 'goBack', 'go-forward': 'goForward' } as const;
  type Operation = keyof typeof handlers;
  const operations = Object.keys(handlers) as Operation[];
  const config = createTestConfig({ history: { callbackUrl: '', thumbnailEnabled: false } });
  let clearCache: jest.SpyInstance, historyCallback: jest.SpyInstance;
  beforeEach(() => {
    clearCache = jest.spyOn(frames, 'clearFrameCache');
    historyCallback = jest.spyOn(pages, 'emitHistoryCallback');
  });
  afterEach(() => jest.restoreAllMocks());

  function fixture(operation: Operation = 'navigate') {
    const id = `navigation-${operation}`;
    // Browser-created entries exist before any recording navigation command.
    const history = { currentIndex: operation === 'go-forward' ? 0 : 1, entries: [
      { id: 11, url: 'https://a.test', title: '' }, { id: 12, url: 'https://b.test', title: '' },
    ] };
    const cdp = {
      send: jest.fn(async (_method: string) => structuredClone(history)),
      detach: jest.fn(async () => {}),
    };
    const attach = jest.fn(async () => cdp);
    const page = {
      context: () => ({ newCDPSession: attach }),
      goto: jest.fn(async (next: string) => {
        history.entries.splice(history.currentIndex + 1, Infinity, { id: 13, url: next, title: '' });
        history.currentIndex++; return {};
      }),
      reload: jest.fn(async () => ({})),
      goBack: jest.fn(async (): Promise<object | null> => { history.currentIndex--; return null; }),
      goForward: jest.fn(async (): Promise<object | null> => { history.currentIndex++; return null; }),
      url: () => history.entries[history.currentIndex].url,
      title: jest.fn(async () => 'fixture title'),
      screenshot: jest.fn(async () => Buffer.from('fixture screenshot')),
    };
    const session = { page, phase: 'ready', ownerExecutionId: 'owner', leaseId: 'lease', leaseReleasedAt: undefined as Date | undefined };
    const activity = jest.fn();
    const manager = {
      getSession: () => { activity(); return session; }, peekSession: () => session,
      getSessionForLease: SessionManager.prototype.getSessionForLease,
      updateActivity: activity,
    } as unknown as SessionManager;
    const call = (op: Operation, body: Record<string, unknown> = {}) => {
      const response = createMockHttpResponse();
      const pending = handlers[op](createMockHttpRequest({ method: 'POST', body: { execution_id: 'owner', lease_id: 'lease', url: 'https://c.test', ...body } }), response, id, manager, config);
      return { response, pending };
    };
    const read = async (stack = false) => {
      const response = createMockHttpResponse();
      await (stack ? handleRecordNavigationStack : handleRecordNavigationState)(createMockHttpRequest(), response, id, manager, config);
      return response;
    };
    return { id, page, session, manager, call, read, history, cdp, attach };
  }

  function deferred() {
    let resolve!: () => void;
    const promise = new Promise<void>(done => { resolve = done; });
    return { promise, resolve };
  }

  describe.each(operations)('%s', operation => {
    it.each(['missing', 'stale', 'released', 'closing', 'body handoff'])('rejects %s ownership before browser or history effects', async kind => {
      const f = fixture(operation), before = structuredClone(f.history);
      if (kind === 'released') f.session.leaseReleasedAt = new Date();
      if (kind === 'closing') f.session.phase = 'closing';
      const body = kind === 'missing' ? { execution_id: undefined, lease_id: undefined }
        : kind === 'stale' ? { execution_id: 'old', lease_id: 'old' } : {};
      const call = f.call(operation, body);
      if (kind === 'body handoff') f.session.leaseId = 'replacement';
      await call.pending;
      expect(call.response.statusCode).toBe(kind === 'missing' ? 400 : 404);
      expect(f.page[methods[operation]]).not.toHaveBeenCalled();
      expect(f.manager.updateActivity).not.toHaveBeenCalled();
      expect(f.attach).not.toHaveBeenCalled();
      expect(f.history).toEqual(before);
    });

    it.each(['lease during navigation', 'page during navigation', 'lease during title'])('does not publish completion after %s changes', async kind => {
      const f = fixture(operation), entered = deferred(), proceed = deferred();
      const method = kind.endsWith('title') ? 'title' : methods[operation];
      if (method === 'title') f.page.title.mockImplementationOnce(async () => { entered.resolve(); await proceed.promise; return 'late title'; });
      else f.page[method].mockImplementationOnce(async () => { entered.resolve(); await proceed.promise; return {}; });
      try {
        const call = f.call(operation, { capture: true });
        await entered.promise;
        if (kind.startsWith('page')) f.session.page = { ...f.page };
        else f.session.leaseId = 'replacement';
        proceed.resolve(); await call.pending;
        expect(call.response.statusCode).toBe(404);
        expect(f.page.screenshot).not.toHaveBeenCalled();
        expect(clearCache).not.toHaveBeenCalled();
        expect(historyCallback).not.toHaveBeenCalled();
      } finally { proceed.resolve(); }
    });

    it.each(['attachment', 'query', 'detach'])('detaches history and rejects ownership lost during %s', async stage => {
      const f = fixture(operation), entered = deferred(), proceed = deferred();
      if (stage === 'attachment') f.attach.mockImplementationOnce(async () => { entered.resolve(); await proceed.promise; return f.cdp; });
      else if (stage === 'query') f.cdp.send.mockImplementationOnce(async () => { entered.resolve(); await proceed.promise; return structuredClone(f.history); });
      else f.cdp.detach.mockImplementationOnce(async () => { entered.resolve(); await proceed.promise; });
      try {
        const call = f.call(operation);
        await entered.promise; f.session.leaseId = 'replacement'; proceed.resolve(); await call.pending;
        expect(call.response.statusCode).toBe(404);
        expect(f.cdp.detach).toHaveBeenCalledTimes(1);
        expect(clearCache).not.toHaveBeenCalled();
        expect(historyCallback).not.toHaveBeenCalled();
        if (operation.startsWith('go-')) expect(f.page[methods[operation]]).not.toHaveBeenCalled();
      } finally { proceed.resolve(); }
    });

    it('preserves owned wait options and reports actual browser history including null responses', async () => {
      const f = fixture(operation);
      const call = f.call(operation, { wait_until: 'domcontentloaded', timeout_ms: 1234, capture: operation === 'navigate' });
      await call.pending;
      expect(call.response.statusCode).toBe(200);
      const args = operation === 'navigate' ? ['https://c.test', { waitUntil: 'domcontentloaded', timeout: 1234 }] : [{ waitUntil: 'domcontentloaded', timeout: 1234 }];
      expect(f.page[methods[operation]]).toHaveBeenCalledWith(...args);
      expect(call.response.getJSON()).toMatchObject({ session_id: f.id, url: f.page.url(), title: 'fixture title', can_go_back: operation !== 'go-back', can_go_forward: operation === 'go-back' });
      expect(f.manager.updateActivity).toHaveBeenCalledTimes(1);
      expect(f.cdp.send).toHaveBeenCalledWith('Page.getNavigationHistory');
      expect(f.cdp.detach.mock.calls.length).toBe(f.attach.mock.calls.length);
      if (operation === 'navigate') expect(call.response.getJSON().screenshot).toBe(`data:image/jpeg;base64,${Buffer.from('fixture screenshot').toString('base64')}`);
    });
  });

  it.each(['go-back', 'go-forward'] as const)('rejects %s when the browser did not move', async operation => {
    const f = fixture(operation), before = structuredClone(f.history);
    f.page[methods[operation]].mockResolvedValueOnce(null);
    const call = f.call(operation); await call.pending;
    expect(call.response.statusCode).toBe(400);
    expect(f.history).toEqual(before);
    expect(clearCache).not.toHaveBeenCalled();
    expect(f.cdp.detach).toHaveBeenCalledTimes(2);
  });

  it.each(['go-back', 'go-forward'] as const)('rejects %s beyond browser bounds without another navigation', async operation => {
    const f = fixture(operation);
    const valid = f.call(operation); await valid.pending;
    expect(valid.response.statusCode).toBe(200);
    const before = structuredClone(f.history);
    const rejected = f.call(operation); await rejected.pending;
    expect(rejected.response.statusCode).toBe(400);
    expect(f.page[methods[operation]]).toHaveBeenCalledTimes(1);
    expect(f.history).toEqual(before);
  });

  it('recognizes distinct same-URL entries without inferring movement from the URL', async () => {
    const f = fixture('go-back');
    f.history.entries[1].url = f.history.entries[0].url;
    const call = f.call('go-back'); await call.pending;
    expect(call.response.statusCode).toBe(200);
    expect(call.response.getJSON()).toMatchObject({ can_go_back: false, can_go_forward: true });
  });

  it('reads untitled browser-created entries and the newly selected tab without a parallel map', async () => {
    const f = fixture();
    expect((await f.read(true)).getJSON()).toEqual({ session_id: f.id, back_stack: [{ url: 'https://a.test', title: '' }], current: { url: 'https://b.test', title: '' }, forward_stack: [] });
    const tab = fixture('go-forward');
    f.session.page = tab.page;
    expect((await f.read()).getJSON()).toMatchObject({ url: 'https://a.test', can_go_back: false, can_go_forward: true });
    expect((await f.read(true)).getJSON()).toMatchObject({ back_stack: [], current: { url: 'https://a.test' }, forward_stack: [{ url: 'https://b.test', title: '' }] });
    expect(f.attach).toHaveBeenCalledTimes(1);
    expect(tab.cdp.detach).toHaveBeenCalledTimes(2);
  });

  it.each([false, true])('rejects a page replacement during history read (stack=%s)', async stack => {
    const f = fixture();
    f.cdp.send.mockImplementationOnce(async () => { f.session.page = { ...f.page }; return structuredClone(f.history); });
    const response = await f.read(stack);
    expect(response.statusCode).toBe(404);
    expect(f.cdp.detach).toHaveBeenCalledTimes(1);
  });

  it('detaches even when the browser history query fails', async () => {
    const f = fixture('go-back');
    f.cdp.send.mockRejectedValueOnce(new Error('history query failed'));
    const call = f.call('go-back'); await call.pending;
    expect(call.response.statusCode).toBe(500);
    expect(f.cdp.detach).toHaveBeenCalledTimes(1);
    expect(f.page.goBack).not.toHaveBeenCalled();
  });
});

describe('Record Mode Routes', () => {
  const config = createTestConfig();

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('handles navigate requests without crashing when parsing body', async () => {
    const sessionId = 'session-123';
    const session = { phase: 'ready', page: mockPage };
    const manager = { ...mockSessionManager, getSessionForLease: () => session, updateActivity: jest.fn() } as unknown as SessionManager;

    const mockReq = createMockHttpRequest({
      method: 'POST',
      url: `/session/${sessionId}/record/navigate`,
      body: { execution_id: 'owner', lease_id: 'lease', url: 'https://example.com', capture: true },
    });
    const mockRes = createMockHttpResponse();

    await handleRecordNavigate(mockReq, mockRes, sessionId, manager, config);

    expect(mockRes.statusCode).toBe(200);
    const payload = mockRes.getJSON();
    expect(payload.url).toBe('https://example.com');
    expect(payload.screenshot).toContain('data:image/jpeg;base64,');
    expect(mockPage.goto).toHaveBeenCalledWith('https://example.com', { waitUntil: 'load', timeout: config.execution.navigationTimeoutMs });
  });
  it('requires an explicit acknowledgement after non-destructive reads', async () => {
    const sessionId = 'pull-actions';
    const session = { phase: 'ready', page: mockPage };
    const manager = { ...mockSessionManager, getSessionForLease: jest.fn(() => session), updateActivity: jest.fn() } as unknown as SessionManager;
    const body = { execution_id: 'owner', lease_id: 'lease', entry_ids: ['pending'] };
    initRecordingBuffer(sessionId);
    bufferTimelineEntry(sessionId, create(TimelineEntrySchema, { id: 'pending', sequenceNum: 0 }));
    try {
      const read = (query = '') => {
        const res = createMockHttpResponse();
        handleRecordActions(createMockHttpRequest({ url: `/session/${sessionId}/record/actions${query}` }), res, sessionId, mockSessionManager as SessionManager);
        return res;
      };
      expect(read().getJSON().count).toBe(1);
      expect(read('?clear=true').statusCode).toBeGreaterThanOrEqual(400);
      expect(read().getJSON().count).toBe(1);
      const ack = createMockHttpResponse();
      await handleRecordActionsAck(createMockHttpRequest({ method: 'POST', body }), ack, sessionId, manager, config);
      expect(ack.statusCode).toBe(200);
      expect(ack.getJSON().entry_ids).toEqual(['pending']);
      expect(read().getJSON().count).toBe(0);
      const retry = createMockHttpResponse();
      await handleRecordActionsAck(createMockHttpRequest({ method: 'POST', body }), retry, sessionId, manager, config);
      expect(retry.statusCode).toBe(200);
      expect(manager.getSessionForLease).toHaveBeenCalledWith(sessionId, 'owner', 'lease');
    } finally {
      acknowledgeTimelineEntries(sessionId, getTimelineEntries(sessionId).map((entry) => entry.id));
      removeRecordingBuffer(sessionId);
    }
  });

  it.each([
    [503, { status: 'ok', entry_id: 'delivery' }],
    [200, { status: 'ok', entry_id: 'another-observation' }],
    [200, { status: 'ok' }],
  ])('requires the callback to acknowledge this exact observation (%s)', async (status, receipt) => {
    const entry = create(TimelineEntrySchema, { id: 'delivery' });
    const fetch = jest.spyOn(globalThis, 'fetch');
    try {
      fetch.mockImplementation(async () => new Response(JSON.stringify(receipt), { status }));
      await expect(streamRecordingEntry('http://fixture.test/commit', entry)).rejects.toThrow();
      fetch.mockImplementation(async () => new Response(JSON.stringify({ status: 'ok', entry_id: entry.id }), { status: 200 }));
      await expect(streamRecordingEntry('http://fixture.test/commit', entry)).resolves.toBeUndefined();
      expect(fetch).toHaveBeenCalledTimes(2);
    } finally { fetch.mockRestore(); }
  });

  it('returns the retained terminal receipt when stop is retried', async () => {
    const stoppedAt = '2026-09-22T00:00:00.000Z';
    const stop = jest.fn();
    const session = {
      phase: 'ready', page: mockPage,
      pipelineManager: {
        isRecording: () => false, getRecordingId: () => 'recording', getGeneration: () => 1,
        getRecordingData: () => ({ actionCount: 7, stoppedAt }), stopRecording: stop,
      },
    };
    const manager = { getSession: () => session, getSessionForLease: () => session, updateActivity: jest.fn(), setSessionPhase: jest.fn() } as unknown as SessionManager;
    const response = createMockHttpResponse();
    await handleRecordStop(createMockHttpRequest({ method: 'POST', body: { execution_id: 'owner', lease_id: 'lease' } }), response, 'stopped', manager);
    expect(response.statusCode).toBe(200);
    expect(response.getJSON()).toMatchObject({ action_count: 7, recording_id: 'recording', stopped_at: stoppedAt });
    expect(stop).not.toHaveBeenCalled();
  });

});
