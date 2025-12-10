import type { Page, Browser, BrowserContext, Frame, Response, Request } from 'playwright';

/**
 * Mock Playwright Page
 */
export function createMockPage(overrides?: Partial<Page>): jest.Mocked<Page> {
  // Create a shared locator that can be reconfigured in tests
  const mockFirstLocator = {
    innerHTML: jest.fn().mockResolvedValue('<span>inner content</span>'),
    textContent: jest.fn().mockResolvedValue('test text'),
    boundingBox: jest.fn().mockResolvedValue({ x: 0, y: 0, width: 100, height: 50 }),
    scrollIntoViewIfNeeded: jest.fn().mockResolvedValue(undefined),
    screenshot: jest.fn().mockResolvedValue(Buffer.from('element-screenshot')),
  };

  const mockLocator = {
    boundingBox: jest.fn().mockResolvedValue({ x: 0, y: 0, width: 100, height: 50 }),
    textContent: jest.fn().mockResolvedValue('test text'),
    getAttribute: jest.fn().mockResolvedValue('test-value'),
    isVisible: jest.fn().mockResolvedValue(true),
    count: jest.fn().mockResolvedValue(1),
    innerHTML: jest.fn().mockResolvedValue('<span>inner content</span>'),
    first: jest.fn().mockReturnValue(mockFirstLocator),
    scrollIntoViewIfNeeded: jest.fn().mockResolvedValue(undefined),
    screenshot: jest.fn().mockResolvedValue(Buffer.from('element-screenshot')),
  };

  const mockPage = {
    goto: jest.fn().mockResolvedValue(null),
    click: jest.fn().mockResolvedValue(undefined),
    hover: jest.fn().mockResolvedValue(undefined),
    type: jest.fn().mockResolvedValue(undefined),
    fill: jest.fn().mockResolvedValue(undefined),
    textContent: jest.fn().mockResolvedValue('test text'),
    focus: jest.fn().mockResolvedValue(undefined),
    waitForEvent: jest.fn().mockResolvedValue(null),
    waitForSelector: jest.fn().mockResolvedValue(null),
    waitForTimeout: jest.fn().mockResolvedValue(undefined),
    screenshot: jest.fn().mockResolvedValue(Buffer.from('fake-screenshot')),
    content: jest.fn().mockResolvedValue('<html><body>Test</body></html>'),
    locator: jest.fn().mockReturnValue(mockLocator),
    $: jest.fn().mockResolvedValue({ boundingBox: jest.fn().mockResolvedValue({ x: 0, y: 0, width: 100, height: 50 }) }),
    $$: jest.fn().mockResolvedValue([]),
    evaluate: jest.fn().mockResolvedValue({ result: 'test' }),
    setInputFiles: jest.fn().mockResolvedValue(undefined),
    on: jest.fn(),
    removeListener: jest.fn(),
    close: jest.fn().mockResolvedValue(undefined),
    isClosed: jest.fn().mockReturnValue(false),
    url: jest.fn().mockReturnValue('https://example.com'),
    title: jest.fn().mockResolvedValue('Test Page'),
    viewport: jest.fn().mockReturnValue({ width: 1280, height: 720 }),
    viewportSize: jest.fn().mockReturnValue({ width: 1280, height: 720 }),
    context: jest.fn(),
    frames: jest.fn().mockReturnValue([]),
    mainFrame: jest.fn(),
    isVisible: jest.fn().mockResolvedValue(true),
    isHidden: jest.fn().mockResolvedValue(false),
    isEnabled: jest.fn().mockResolvedValue(true),
    isDisabled: jest.fn().mockResolvedValue(false),
    getAttribute: jest.fn().mockResolvedValue(''),
    route: jest.fn().mockResolvedValue(undefined),
    unroute: jest.fn().mockResolvedValue(undefined),
    ...overrides,
  } as unknown as jest.Mocked<Page>;

  return mockPage;
}

/**
 * Mock Playwright BrowserContext
 */
export function createMockContext(overrides?: Partial<BrowserContext>): jest.Mocked<BrowserContext> {
  const mockContext = {
    newPage: jest.fn().mockResolvedValue(createMockPage()),
    pages: jest.fn().mockReturnValue([]),
    close: jest.fn().mockResolvedValue(undefined),
    clearCookies: jest.fn().mockResolvedValue(undefined),
    clearPermissions: jest.fn().mockResolvedValue(undefined),
    addCookies: jest.fn().mockResolvedValue(undefined),
    cookies: jest.fn().mockResolvedValue([]),
    storageState: jest.fn().mockResolvedValue({ cookies: [], origins: [] }),
    tracing: {
      start: jest.fn().mockResolvedValue(undefined),
      stop: jest.fn().mockResolvedValue(undefined),
    },
    ...overrides,
  } as unknown as jest.Mocked<BrowserContext>;

  return mockContext;
}

/**
 * Mock Playwright Browser
 */
export function createMockBrowser(overrides?: Partial<Browser>): jest.Mocked<Browser> {
  const mockBrowser = {
    newContext: jest.fn().mockResolvedValue(createMockContext()),
    close: jest.fn().mockResolvedValue(undefined),
    isConnected: jest.fn().mockReturnValue(true),
    contexts: jest.fn().mockReturnValue([]),
    version: jest.fn().mockReturnValue('mock-browser-version'),
    ...overrides,
  } as unknown as jest.Mocked<Browser>;

  return mockBrowser;
}

/**
 * Mock Playwright Frame
 */
export function createMockFrame(overrides?: Partial<Frame>): jest.Mocked<Frame> {
  const mockFrame = {
    name: jest.fn().mockReturnValue('test-frame'),
    url: jest.fn().mockReturnValue('https://example.com/frame'),
    parentFrame: jest.fn().mockReturnValue(null),
    childFrames: jest.fn().mockReturnValue([]),
    locator: jest.fn().mockReturnValue({
      boundingBox: jest.fn().mockResolvedValue({ x: 0, y: 0, width: 100, height: 50 }),
    }),
    ...overrides,
  } as unknown as jest.Mocked<Frame>;

  return mockFrame;
}

/**
 * Mock Playwright Response
 */
export function createMockResponse(overrides?: Partial<Response>): jest.Mocked<Response> {
  const mockResponse = {
    status: jest.fn().mockReturnValue(200),
    statusText: jest.fn().mockReturnValue('OK'),
    url: jest.fn().mockReturnValue('https://example.com'),
    headers: jest.fn().mockReturnValue({}),
    body: jest.fn().mockResolvedValue(Buffer.from('response body')),
    json: jest.fn().mockResolvedValue({}),
    text: jest.fn().mockResolvedValue('response text'),
    ...overrides,
  } as unknown as jest.Mocked<Response>;

  return mockResponse;
}

/**
 * Mock Playwright Request
 */
export function createMockRequest(overrides?: Partial<Request>): jest.Mocked<Request> {
  const mockRequest = {
    url: jest.fn().mockReturnValue('https://example.com'),
    method: jest.fn().mockReturnValue('GET'),
    headers: jest.fn().mockReturnValue({}),
    postData: jest.fn().mockReturnValue(null),
    resourceType: jest.fn().mockReturnValue('document'),
    ...overrides,
  } as unknown as jest.Mocked<Request>;

  return mockRequest;
}

/**
 * Setup common Playwright mocks for tests
 */
export function setupPlaywrightMocks() {
  jest.mock('playwright', () => ({
    chromium: {
      launch: jest.fn().mockResolvedValue(createMockBrowser()),
    },
  }));
}
