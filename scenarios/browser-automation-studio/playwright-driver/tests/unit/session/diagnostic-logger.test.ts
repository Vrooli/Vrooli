import type { BrowserContext, Page, Request, Response, Frame } from 'rebrowser-playwright';

const mockLogger = {
  info: jest.fn(),
  warn: jest.fn(),
  debug: jest.fn(),
  error: jest.fn(),
};

jest.mock('../../../src/utils', () => ({
  logger: mockLogger,
  LogContext: {
    SESSION: 'session',
  },
  scopedLog: (_context: string, message: string) => message,
}));

describe('diagnostic logger', () => {
  const originalEnv = process.env.DIAGNOSTIC_LOGGING;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterEach(() => {
    process.env.DIAGNOSTIC_LOGGING = originalEnv;
    jest.resetModules();
  });

  it('exits early when diagnostic logging is disabled', async () => {
    process.env.DIAGNOSTIC_LOGGING = 'false';
    jest.resetModules();
    const { setupDiagnosticLogging } = await import('../../../src/session/diagnostic-logger');

    const context = {
      on: jest.fn(),
      pages: jest.fn().mockReturnValue([]),
    } as unknown as BrowserContext;

    setupDiagnosticLogging(context, 'session-1');

    expect(context.on).not.toHaveBeenCalled();
    expect(mockLogger.debug).toHaveBeenCalled();
  });

  it('attaches page diagnostics when enabled', async () => {
    process.env.DIAGNOSTIC_LOGGING = 'true';
    jest.resetModules();
    const { setupDiagnosticLogging } = await import('../../../src/session/diagnostic-logger');

    const pageHandlers = new Map<string, (value: unknown) => void>();
    const page = {
      url: jest.fn().mockReturnValue('https://www.google.com'),
      mainFrame: jest.fn().mockReturnValue({} as Frame),
      on: jest.fn((event: string, handler: (value: unknown) => void) => {
        pageHandlers.set(event, handler);
      }),
      evaluate: jest.fn(),
    } as unknown as Page;

    const context = {
      on: jest.fn((event: string, handler: (value: unknown) => void) => {
        if (event === 'page') {
          handler(page);
        }
      }),
      pages: jest.fn().mockReturnValue([page]),
    } as unknown as BrowserContext;

    setupDiagnosticLogging(context, 'session-2');

    const frame = {
      url: () => 'https://www.google.com/search?q=test',
    } as Frame;
    pageHandlers.get('framenavigated')?.(frame);

    const request = {
      url: () => 'https://www.google.com/sw.js',
      method: () => 'GET',
      resourceType: () => 'serviceworker',
      headers: () => ({ 'sec-ch-ua': 'ua', referer: 'https://www.google.com/' }),
      failure: () => ({ errorText: 'net::ERR_FAILED' }),
    } as Request;
    pageHandlers.get('request')?.(request);

    const response = {
      url: () => 'https://www.google.com/redirect',
      status: () => 302,
      headers: () => ({ location: 'https://www.google.com/target' }),
    } as Response;
    pageHandlers.get('response')?.(response);

    pageHandlers.get('requestfailed')?.(request);

    const consoleMessage = {
      type: () => 'error',
      text: () => 'Service Worker registration failed',
    };
    pageHandlers.get('console')?.(consoleMessage);

    expect(mockLogger.info).toHaveBeenCalled();
    expect(mockLogger.warn).toHaveBeenCalled();
  });

  it('logs configuration helpers with redaction when enabled', async () => {
    process.env.DIAGNOSTIC_LOGGING = 'true';
    jest.resetModules();
    const {
      logAntiDetectionApplied,
      logClientHints,
      logAdBlockerConfig,
      logContextOptions,
    } = await import('../../../src/session/diagnostic-logger');

    logAntiDetectionApplied('session-3', ['patch-1', 'patch-2']);
    logClientHints('session-3', 'Mozilla/5.0', { 'sec-ch-ua': 'ua' });
    logAdBlockerConfig('session-3', 'strict', ['example.com']);
    logContextOptions('session-3', {
      proxy: { server: 'proxy', username: 'user', password: 'secret' },
      storageState: { cookies: [] },
      viewport: { width: 800, height: 600 },
    });

    const lastCallArgs = mockLogger.info.mock.calls[mockLogger.info.mock.calls.length - 1];
    const payload = lastCallArgs?.[1] as { options?: Record<string, unknown> } | undefined;
    expect(payload?.options).toBeDefined();
    expect(payload?.options?.proxy).toEqual({ server: 'proxy', username: 'user', password: '[REDACTED]' });
    expect(payload?.options?.storageState).toBe('[PRESENT]');
  });
});
