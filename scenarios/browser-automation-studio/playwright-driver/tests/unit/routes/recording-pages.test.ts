import type { Page } from 'rebrowser-playwright';
import {
  createMockHttpRequest,
  createMockHttpResponse,
  createTestConfig,
  installFetchMock,
} from '../../helpers';
import type { SessionManager } from '../../../src/session';

jest.mock('../../../src/routes/record-mode/recording-frames', () => ({
  clearFrameCache: jest.fn(),
}));

jest.mock('../../../src/utils', () => ({
  logger: {
    info: jest.fn(),
    warn: jest.fn(),
  },
}));

import {
  captureThumbnail,
  emitHistoryCallback,
  handleRecordActivePage,
  handleRecordNewPage,
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
      } as unknown as Page;

      const session = {
        context: { newPage: jest.fn().mockResolvedValue(newPage) },
        pages: [] as Page[],
        pageIdMap: new Map<string, Page>(),
        pageToIdMap: new Map<Page, string>(),
        currentPageIndex: 0,
        page: undefined as Page | undefined,
      };

      const sessionManager = {
        getSession: jest.fn().mockReturnValue(session),
      } as unknown as SessionManager;

      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/abc/record/new-page',
        body: { url: 'https://example.com' },
      });
      const res = createMockHttpResponse();

      await handleRecordNewPage(req, res, 'abc', sessionManager, config);

      expect(res.statusCode).toBe(201);
      const payload = res.getJSON();
      expect(payload.url).toBe('https://example.com');
      expect(payload.title).toBe('New Page');
      expect(session.pages).toHaveLength(1);
      expect(session.page).toBe(newPage);
    });

    it('continues when navigation fails', async () => {
      const newPage = {
        goto: jest.fn().mockRejectedValue(new Error('nav error')),
        title: jest.fn().mockResolvedValue('Fallback'),
        url: jest.fn().mockReturnValue('about:blank'),
      } as unknown as Page;

      const session = {
        context: { newPage: jest.fn().mockResolvedValue(newPage) },
        pages: [] as Page[],
        pageIdMap: new Map<string, Page>(),
        pageToIdMap: new Map<Page, string>(),
        currentPageIndex: 0,
        page: undefined as Page | undefined,
      };

      const sessionManager = {
        getSession: jest.fn().mockReturnValue(session),
      } as unknown as SessionManager;

      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/abc/record/new-page',
        body: { url: 'https://example.com' },
      });
      const res = createMockHttpResponse();

      await handleRecordNewPage(req, res, 'abc', sessionManager, config);

      expect(res.statusCode).toBe(201);
      expect(session.page).toBe(newPage);
    });
  });

  describe('handleRecordActivePage', () => {
    it('returns 400 when page_id is missing', async () => {
      const session = {
        pageIdMap: new Map<string, Page>(),
        pageToIdMap: new Map<Page, string>(),
        pages: [] as Page[],
        page: undefined as Page | undefined,
      };
      const sessionManager = {
        getSession: jest.fn().mockReturnValue(session),
      } as unknown as SessionManager;

      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/abc/record/active-page',
        body: {},
      });
      const res = createMockHttpResponse();

      await handleRecordActivePage(req, res, 'abc', sessionManager, config);

      expect(res.statusCode).toBe(400);
      expect(res.getJSON().error).toBe('MISSING_PAGE_ID');
    });

    it('returns 404 when page id is unknown', async () => {
      const session = {
        pageIdMap: new Map<string, Page>([['known', {} as Page]]),
        pageToIdMap: new Map<Page, string>(),
        pages: [] as Page[],
        page: undefined as Page | undefined,
      };
      const sessionManager = {
        getSession: jest.fn().mockReturnValue(session),
      } as unknown as SessionManager;

      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/abc/record/active-page',
        body: { page_id: 'missing' },
      });
      const res = createMockHttpResponse();

      await handleRecordActivePage(req, res, 'abc', sessionManager, config);

      expect(res.statusCode).toBe(404);
      expect(res.getJSON().available_page_ids).toEqual(['known']);
    });

    it('returns 410 when page is closed', async () => {
      const page = { isClosed: jest.fn().mockReturnValue(true) } as unknown as Page;
      const session = {
        pageIdMap: new Map<string, Page>([['page-1', page]]),
        pageToIdMap: new Map<Page, string>([[page, 'page-1']]),
        pages: [page],
        page,
      };
      const sessionManager = {
        getSession: jest.fn().mockReturnValue(session),
      } as unknown as SessionManager;

      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/abc/record/active-page',
        body: { page_id: 'page-1' },
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
        title: jest.fn().mockResolvedValue('Example'),
      } as unknown as Page;

      const session = {
        pageIdMap: new Map<string, Page>([['page-a', pageA], ['page-b', pageB]]),
        pageToIdMap: new Map<Page, string>([[pageA, 'page-a'], [pageB, 'page-b']]),
        pages: [pageA, pageB],
        page: pageA,
        currentPageIndex: 0,
      };

      const sessionManager = {
        getSession: jest.fn().mockReturnValue(session),
      } as unknown as SessionManager;

      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/abc/record/active-page',
        body: { page_id: 'page-b' },
      });
      const res = createMockHttpResponse();

      await handleRecordActivePage(req, res, 'abc', sessionManager, config);

      expect(res.statusCode).toBe(200);
      const payload = res.getJSON();
      expect(payload.active_page_id).toBe('page-b');
      expect(session.page).toBe(pageB);
      expect(session.currentPageIndex).toBe(1);
    });
  });
});
