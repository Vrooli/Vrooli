import {
  handleRecordFrame,
  handleRecordScreenshot,
  clearFrameCache,
  clearAllFrameCaches,
} from '../../../src/routes/record-mode/recording-frames';
import { createMockHttpRequest, createMockHttpResponse, createMockPage, createTestConfig } from '../../helpers';
import type { SessionManager } from '../../../src/session';
import { RECORDING_FRAME_CACHE_TTL_MS } from '../../../src/constants';

describe('recording frame routes', () => {
  const config = createTestConfig();
  let mockPage: ReturnType<typeof createMockPage>;
  let sessionManager: Pick<SessionManager, 'getSession'>;
  let nowSpy: jest.SpyInstance<number, []>;

  beforeEach(() => {
    mockPage = createMockPage({
      screenshot: jest.fn().mockResolvedValue(Buffer.from('frame-1')),
      viewportSize: jest.fn().mockReturnValue({ width: 1024, height: 768 }),
      title: jest.fn().mockResolvedValue('Test Page'),
      url: jest.fn().mockReturnValue('https://example.com'),
    });
    sessionManager = {
      getSession: () => ({ page: mockPage } as ReturnType<SessionManager['getSession']>),
    };
    nowSpy = jest.spyOn(Date, 'now').mockReturnValue(1000);
  });

  afterEach(() => {
    nowSpy.mockRestore();
    clearFrameCache('session-1');
    clearAllFrameCaches();
  });

  it('captures screenshot for recording screenshot endpoint', async () => {
    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/session-1/record/screenshot',
      body: { full_page: false, quality: 70 },
    });
    const res = createMockHttpResponse();

    await handleRecordScreenshot(req, res, 'session-1', sessionManager as SessionManager, config);

    expect(mockPage.screenshot).toHaveBeenCalledWith({
      fullPage: false,
      type: 'jpeg',
      quality: 70,
    });
    expect(res.statusCode).toBe(200);
  });

  it('caches frame responses within TTL', async () => {
    const req = createMockHttpRequest({
      method: 'GET',
      url: '/session/session-1/record/frame?quality=60&full_page=false',
    });
    const res1 = createMockHttpResponse();
    const res2 = createMockHttpResponse();

    await handleRecordFrame(req, res1, 'session-1', sessionManager as SessionManager, config);
    nowSpy.mockReturnValue(1000 + RECORDING_FRAME_CACHE_TTL_MS - 1);
    await handleRecordFrame(req, res2, 'session-1', sessionManager as SessionManager, config);

    expect(mockPage.screenshot).toHaveBeenCalledTimes(1);
    expect(res2.getJSON().content_hash).toBe(res1.getJSON().content_hash);
  });

  it('reuses cached data when content hash is unchanged after TTL', async () => {
    const req = createMockHttpRequest({
      method: 'GET',
      url: '/session/session-1/record/frame?quality=60&full_page=false',
    });
    const res1 = createMockHttpResponse();
    const res2 = createMockHttpResponse();

    await handleRecordFrame(req, res1, 'session-1', sessionManager as SessionManager, config);

    nowSpy.mockReturnValue(1000 + RECORDING_FRAME_CACHE_TTL_MS + 1);
    await handleRecordFrame(req, res2, 'session-1', sessionManager as SessionManager, config);

    expect(mockPage.screenshot).toHaveBeenCalledTimes(2);
    expect(res2.getJSON().image).toBe(res1.getJSON().image);
  });

  it('uses fullPage capture when requested', async () => {
    const req = createMockHttpRequest({
      method: 'GET',
      url: '/session/session-1/record/frame?quality=60&full_page=true',
    });
    const res = createMockHttpResponse();

    await handleRecordFrame(req, res, 'session-1', sessionManager as SessionManager, config);

    const [options] = mockPage.screenshot.mock.calls[0] ?? [];
    expect(options.fullPage).toBe(true);
    expect(options.clip).toBeUndefined();
  });
});
