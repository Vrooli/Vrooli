import { handleRecordNavigate } from '../../../src/routes/record-mode';
import { createMockHttpRequest, createMockHttpResponse, createTestConfig } from '../../helpers';
import type { SessionManager } from '../../../src/session/manager';

// Minimal session manager stub to avoid spinning up Playwright
const mockPage = {
  on: jest.fn(),
  goto: jest.fn().mockResolvedValue(undefined),
  screenshot: jest.fn().mockResolvedValue(Buffer.from('image-bytes')),
  url: jest.fn().mockReturnValue('https://example.com'),
};
const mockSessionManager: Pick<SessionManager, 'getSession'> = {
  getSession: () => ({ page: mockPage } as any),
};

describe('Record Mode Routes', () => {
  const config = createTestConfig();

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('handles navigate requests without crashing when parsing body', async () => {
    const sessionId = 'session-123';

    const mockReq = createMockHttpRequest({
      method: 'POST',
      url: `/session/${sessionId}/record/navigate`,
      body: { url: 'https://example.com', capture: true },
    });
    const mockRes = createMockHttpResponse();

    await handleRecordNavigate(mockReq, mockRes, sessionId, mockSessionManager as SessionManager, config);

    expect(mockRes.statusCode).toBe(200);
    const payload = (mockRes as any).getJSON();
    expect(payload.url).toBe('https://example.com');
    expect(payload.screenshot).toContain('data:image/jpeg;base64,');
    expect(mockPage.goto).toHaveBeenCalledWith('https://example.com', { waitUntil: 'load', timeout: config.execution.navigationTimeoutMs });
  });
});
