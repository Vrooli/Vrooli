/**
 * Unit Tests for RecordingContextInitializer
 *
 * Tests the core functionality of the context initializer:
 * - Route interception setup for event URL
 * - Script injection into HTML responses
 * - Event handling and parsing
 * - Injection statistics tracking
 */

import { jest, describe, beforeEach, afterEach, it, expect } from '@jest/globals';
import {
  RecordingContextInitializer,
  createRecordingContextInitializer,
  type InjectionStats,
  type RecordingEventHandler,
  type RawBrowserEvent,
} from '../../../src/recording';
import type { BrowserContext, Route, Request, APIResponse } from 'rebrowser-playwright';

// Mock logger
const mockLogger = {
  debug: jest.fn(),
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn(),
};

// Helper to create mock route
function createMockRoute(
  requestOverrides?: Partial<Request>,
  fetchResult?: { text: () => Promise<string>; headers: () => Record<string, string>; status?: () => number }
): { route: jest.Mocked<Route>; request: jest.Mocked<Request> } {
  const mockRequest = {
    url: jest.fn().mockReturnValue('https://example.com'),
    method: jest.fn().mockReturnValue('GET'),
    resourceType: jest.fn().mockReturnValue('document'),
    postData: jest.fn().mockReturnValue(null),
    ...requestOverrides,
  } as unknown as jest.Mocked<Request>;

  const mockFetchResult = fetchResult ?? {
    text: () => Promise.resolve('<html><head></head><body>Test</body></html>'),
    headers: () => ({ 'content-type': 'text/html' }),
    status: () => 200,
  };

  // Ensure status is defined
  if (!mockFetchResult.status) {
    mockFetchResult.status = () => 200;
  }

  const mockRoute = {
    request: jest.fn().mockReturnValue(mockRequest),
    continue: jest.fn().mockResolvedValue(undefined),
    fulfill: jest.fn().mockResolvedValue(undefined),
    fallback: jest.fn().mockResolvedValue(undefined),
    fetch: jest.fn().mockResolvedValue(mockFetchResult as unknown as APIResponse),
  } as unknown as jest.Mocked<Route>;

  return { route: mockRoute, request: mockRequest };
}

// Helper to create mock context
function createMockContext(): {
  context: jest.Mocked<BrowserContext>;
  routeHandlers: Map<string, (route: Route) => Promise<void>>;
} {
  const routeHandlers = new Map<string, (route: Route) => Promise<void>>();

  const mockContext = {
    route: jest.fn().mockImplementation(async (pattern: string, handler: (route: Route) => Promise<void>) => {
      routeHandlers.set(pattern, handler);
    }),
    pages: jest.fn().mockReturnValue([]),
    on: jest.fn(),
    off: jest.fn(),
  } as unknown as jest.Mocked<BrowserContext>;

  return { context: mockContext, routeHandlers };
}

// Helper to create mock page with event listeners
function createMockPage(): {
  page: any;
  pageListeners: Map<string, Array<(...args: unknown[]) => void>>;
  pageRouteHandlers: Map<string, (route: Route) => Promise<void>>;
} {
  const pageListeners = new Map<string, Array<(...args: unknown[]) => void>>();
  const pageRouteHandlers = new Map<string, (route: Route) => Promise<void>>();

  const mockPage = {
    url: jest.fn().mockReturnValue('https://example.com'),
    on: jest.fn().mockImplementation((event: string, handler: (...args: unknown[]) => void) => {
      const handlers = pageListeners.get(event) || [];
      handlers.push(handler);
      pageListeners.set(event, handlers);
    }),
    off: jest.fn(),
    route: jest.fn().mockImplementation(async (pattern: string, handler: (route: Route) => Promise<void>) => {
      pageRouteHandlers.set(pattern, handler);
    }),
  };

  return { page: mockPage, pageListeners, pageRouteHandlers };
}

describe('RecordingContextInitializer', () => {
  let initializer: RecordingContextInitializer;

  beforeEach(() => {
    jest.clearAllMocks();
    initializer = createRecordingContextInitializer({
      logger: mockLogger as any,
      diagnosticsEnabled: true,
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('initialization', () => {
    it('should set up context-level route for HTML injection', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      // Should have set up HTML injection route on context
      // Note: Event route is now set up per-page via page.route(), not context.route()
      expect(context.route).toHaveBeenCalledTimes(1);

      // Should have catch-all route for HTML injection
      const catchAllPattern = Array.from(routeHandlers.keys()).find(
        (k) => k === '**/*'
      );
      expect(catchAllPattern).toBeDefined();
    });

    it('should set up page-level event route on existing pages', async () => {
      const { page, pageRouteHandlers } = createMockPage();
      const mockContext = {
        route: jest.fn(),
        pages: jest.fn().mockReturnValue([page]),
        on: jest.fn(),
        off: jest.fn(),
      } as unknown as jest.Mocked<BrowserContext>;

      await initializer.initialize(mockContext);

      // Should have set up event route on page (not context)
      expect(page.route).toHaveBeenCalled();
      const eventRoutePattern = Array.from(pageRouteHandlers.keys()).find(
        (k) => k.includes('__vrooli_recording_event__')
      );
      expect(eventRoutePattern).toBeDefined();
    });

    it('should be idempotent (safe to call multiple times)', async () => {
      const { context } = createMockContext();

      await initializer.initialize(context);
      await initializer.initialize(context);

      // Should only set up HTML injection route once
      expect(context.route).toHaveBeenCalledTimes(1);
    });

    it('should mark as initialized after setup', async () => {
      const { context } = createMockContext();

      expect(initializer.isInitialized()).toBe(false);

      await initializer.initialize(context);

      expect(initializer.isInitialized()).toBe(true);
    });
  });

  describe('script injection', () => {
    it('should inject script into HTML responses with <head> tag', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      // Get the catch-all handler
      const handler = routeHandlers.get('**/*');
      expect(handler).toBeDefined();

      // Create mock route with HTML containing <head>
      const { route } = createMockRoute(
        { resourceType: jest.fn().mockReturnValue('document') as any },
        {
          text: () => Promise.resolve('<html><head></head><body>Test</body></html>'),
          headers: () => ({ 'content-type': 'text/html' }),
        }
      );

      await handler!(route);

      // Should have fulfilled with modified body
      expect(route.fulfill).toHaveBeenCalled();
      const fulfillCall = (route.fulfill as jest.Mock).mock.calls[0][0];
      expect(fulfillCall.body).toContain('<script>');
      expect(fulfillCall.body).toContain('__vrooli_recording');
    });

    it('should inject script into HTML responses with <HEAD> tag (uppercase)', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      const handler = routeHandlers.get('**/*');
      const { route } = createMockRoute(
        { resourceType: jest.fn().mockReturnValue('document') as any },
        {
          text: () => Promise.resolve('<html><HEAD></HEAD><body>Test</body></html>'),
          headers: () => ({ 'content-type': 'text/html' }),
        }
      );

      await handler!(route);

      expect(route.fulfill).toHaveBeenCalled();
      const fulfillCall = (route.fulfill as jest.Mock).mock.calls[0][0];
      expect(fulfillCall.body).toContain('<HEAD><script>');
    });

    it('should inject script after doctype when no head tag', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      const handler = routeHandlers.get('**/*');
      const { route } = createMockRoute(
        { resourceType: jest.fn().mockReturnValue('document') as any },
        {
          text: () => Promise.resolve('<!DOCTYPE html><html><body>Test</body></html>'),
          headers: () => ({ 'content-type': 'text/html' }),
        }
      );

      await handler!(route);

      expect(route.fulfill).toHaveBeenCalled();
      const fulfillCall = (route.fulfill as jest.Mock).mock.calls[0][0];
      expect(fulfillCall.body).toMatch(/<!DOCTYPE html><script>/);
    });

    it('should prepend script when no head or doctype', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      const handler = routeHandlers.get('**/*');
      const { route } = createMockRoute(
        { resourceType: jest.fn().mockReturnValue('document') as any },
        {
          text: () => Promise.resolve('<html><body>No head</body></html>'),
          headers: () => ({ 'content-type': 'text/html' }),
        }
      );

      await handler!(route);

      expect(route.fulfill).toHaveBeenCalled();
      const fulfillCall = (route.fulfill as jest.Mock).mock.calls[0][0];
      expect(fulfillCall.body).toMatch(/^<script>/);
    });

    it('should NOT inject into non-document resources', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      const handler = routeHandlers.get('**/*');
      const { route } = createMockRoute({
        resourceType: jest.fn().mockReturnValue('stylesheet') as any,
      });

      await handler!(route);

      // Should continue without injection
      expect(route.continue).toHaveBeenCalled();
      expect(route.fulfill).not.toHaveBeenCalled();
    });

    it('should NOT inject into non-HTML content types', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      const handler = routeHandlers.get('**/*');
      const { route } = createMockRoute(
        { resourceType: jest.fn().mockReturnValue('document') as any },
        {
          text: () => Promise.resolve('{"json": "data"}'),
          headers: () => ({ 'content-type': 'application/json' }),
        }
      );

      await handler!(route);

      // Should fulfill with original response (no modification)
      expect(route.fulfill).toHaveBeenCalledWith({ response: expect.anything() });
    });
  });

  describe('injection statistics', () => {
    it('should track successful injections', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      const handler = routeHandlers.get('**/*');
      const { route } = createMockRoute(
        { resourceType: jest.fn().mockReturnValue('document') as any },
        {
          text: () => Promise.resolve('<html><head></head><body>Test</body></html>'),
          headers: () => ({ 'content-type': 'text/html' }),
        }
      );

      await handler!(route);

      const stats = initializer.getInjectionStats();
      expect(stats.attempted).toBe(1);
      expect(stats.successful).toBe(1);
      expect(stats.methods.head).toBe(1);
    });

    it('should track skipped requests', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      const handler = routeHandlers.get('**/*');
      const { route } = createMockRoute({
        resourceType: jest.fn().mockReturnValue('image') as any,
      });

      await handler!(route);

      const stats = initializer.getInjectionStats();
      expect(stats.skipped).toBe(1);
      expect(stats.attempted).toBe(0);
    });

    it('should track injection methods separately', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      const handler = routeHandlers.get('**/*');

      // Test <head> method
      const { route: route1 } = createMockRoute(
        { resourceType: jest.fn().mockReturnValue('document') as any },
        {
          text: () => Promise.resolve('<html><head></head></html>'),
          headers: () => ({ 'content-type': 'text/html' }),
        }
      );
      await handler!(route1);

      // Test <HEAD> method
      const { route: route2 } = createMockRoute(
        { resourceType: jest.fn().mockReturnValue('document') as any },
        {
          text: () => Promise.resolve('<html><HEAD></HEAD></html>'),
          headers: () => ({ 'content-type': 'text/html' }),
        }
      );
      await handler!(route2);

      const stats = initializer.getInjectionStats();
      expect(stats.methods.head).toBe(1);
      expect(stats.methods.HEAD).toBe(1);
      expect(stats.successful).toBe(2);
    });
  });

  describe('event handling', () => {
    it('should parse events from POST data', async () => {
      const { page, pageRouteHandlers } = createMockPage();
      const mockContext = {
        route: jest.fn(),
        pages: jest.fn().mockReturnValue([page]),
        on: jest.fn(),
        off: jest.fn(),
      } as unknown as jest.Mocked<BrowserContext>;

      await initializer.initialize(mockContext);

      const receivedEvents: RawBrowserEvent[] = [];
      initializer.setEventHandler((event) => {
        receivedEvents.push(event);
      });

      // Event route is now on page, not context
      const eventRoutePattern = Array.from(pageRouteHandlers.keys()).find(
        (k) => k.includes('__vrooli_recording_event__')
      );
      const handler = pageRouteHandlers.get(eventRoutePattern!);

      const eventData: RawBrowserEvent = {
        actionType: 'click',
        timestamp: Date.now(),
        selector: { primary: '[data-testid="btn"]', candidates: [] },
        elementMeta: { tagName: 'BUTTON', isVisible: true, isEnabled: true },
        boundingBox: { x: 100, y: 100, width: 50, height: 30 },
        cursorPos: { x: 125, y: 115 },
        url: 'https://example.com',
        frameId: null,
      };

      const { route } = createMockRoute({
        url: jest.fn().mockReturnValue('https://example.com/__vrooli_recording_event__') as any,
        method: jest.fn().mockReturnValue('POST') as any,
        postData: jest.fn().mockReturnValue(JSON.stringify(eventData)) as any,
      });

      await handler!(route);

      expect(receivedEvents).toHaveLength(1);
      expect(receivedEvents[0].actionType).toBe('click');
      expect(receivedEvents[0].selector?.primary).toBe('[data-testid="btn"]');
    });

    it('should call event handler with parsed event', async () => {
      const { page, pageRouteHandlers } = createMockPage();
      const mockContext = {
        route: jest.fn(),
        pages: jest.fn().mockReturnValue([page]),
        on: jest.fn(),
        off: jest.fn(),
      } as unknown as jest.Mocked<BrowserContext>;

      await initializer.initialize(mockContext);

      const eventHandler = jest.fn<RecordingEventHandler>();
      initializer.setEventHandler(eventHandler);

      // Event route is now on page, not context
      const eventRoutePattern = Array.from(pageRouteHandlers.keys()).find(
        (k) => k.includes('__vrooli_recording_event__')
      );
      const handler = pageRouteHandlers.get(eventRoutePattern!);

      const { route } = createMockRoute({
        url: jest.fn().mockReturnValue('https://example.com/__vrooli_recording_event__') as any,
        method: jest.fn().mockReturnValue('POST') as any,
        postData: jest.fn().mockReturnValue(JSON.stringify({
          actionType: 'type',
          timestamp: Date.now(),
          payload: { text: 'hello' },
        })) as any,
      });

      await handler!(route);

      expect(eventHandler).toHaveBeenCalledTimes(1);
      expect(eventHandler).toHaveBeenCalledWith(
        expect.objectContaining({ actionType: 'type' })
      );
    });

    it('should handle malformed event data gracefully', async () => {
      const { page, pageRouteHandlers } = createMockPage();
      const mockContext = {
        route: jest.fn(),
        pages: jest.fn().mockReturnValue([page]),
        on: jest.fn(),
        off: jest.fn(),
      } as unknown as jest.Mocked<BrowserContext>;

      await initializer.initialize(mockContext);

      const eventHandler = jest.fn<RecordingEventHandler>();
      initializer.setEventHandler(eventHandler);

      // Event route is now on page, not context
      const eventRoutePattern = Array.from(pageRouteHandlers.keys()).find(
        (k) => k.includes('__vrooli_recording_event__')
      );
      const handler = pageRouteHandlers.get(eventRoutePattern!);

      const { route } = createMockRoute({
        url: jest.fn().mockReturnValue('https://example.com/__vrooli_recording_event__') as any,
        method: jest.fn().mockReturnValue('POST') as any,
        postData: jest.fn().mockReturnValue('not valid json') as any,
      });

      // Should not throw
      await handler!(route);

      // Error should be logged
      expect(mockLogger.error).toHaveBeenCalled();
    });

    it('should clear event handler when requested', async () => {
      const { page, pageRouteHandlers } = createMockPage();
      const mockContext = {
        route: jest.fn(),
        pages: jest.fn().mockReturnValue([page]),
        on: jest.fn(),
        off: jest.fn(),
      } as unknown as jest.Mocked<BrowserContext>;

      await initializer.initialize(mockContext);

      const eventHandler = jest.fn<RecordingEventHandler>();
      initializer.setEventHandler(eventHandler);
      initializer.clearEventHandler();

      // Event route is now on page, not context
      const eventRoutePattern = Array.from(pageRouteHandlers.keys()).find(
        (k) => k.includes('__vrooli_recording_event__')
      );
      const handler = pageRouteHandlers.get(eventRoutePattern!);

      const { route } = createMockRoute({
        url: jest.fn().mockReturnValue('https://example.com/__vrooli_recording_event__') as any,
        method: jest.fn().mockReturnValue('POST') as any,
        postData: jest.fn().mockReturnValue(JSON.stringify({ actionType: 'click' })) as any,
      });

      await handler!(route);

      // Handler should not be called after clearing
      expect(eventHandler).not.toHaveBeenCalled();
    });
  });

  describe('getBindingName', () => {
    it('should return the configured binding name', () => {
      const customInitializer = createRecordingContextInitializer({
        bindingName: 'custom_binding',
        logger: mockLogger as any,
      });

      expect(customInitializer.getBindingName()).toBe('custom_binding');
    });

    it('should return default binding name when not specified', () => {
      expect(initializer.getBindingName()).toBeDefined();
      expect(typeof initializer.getBindingName()).toBe('string');
    });
  });

  describe('redirect handling', () => {
    /**
     * CRITICAL: These tests verify correct redirect handling behavior.
     *
     * With rebrowser-playwright, if we return a 3xx response via route.fulfill(),
     * the browser follows the redirect but the new request does NOT go through
     * our route handler again (anti-detection feature). By following redirects
     * ourselves (maxRedirects: 10), we ensure we can inject into the final HTML.
     *
     * Note: The browser's URL bar will show the original URL (e.g., wikipedia.com)
     * not the redirect destination. This is acceptable for recording purposes.
     */

    it('should call route.fetch() with maxRedirects: 10 to follow redirects for injection', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      const handler = routeHandlers.get('**/*');
      const { route } = createMockRoute(
        { resourceType: jest.fn().mockReturnValue('document') as any },
        {
          text: () => Promise.resolve('<html><head></head><body>Test</body></html>'),
          headers: () => ({ 'content-type': 'text/html' }),
          status: () => 200,
        }
      );

      await handler!(route);

      // Verify fetch follows redirects (anti-detection workaround)
      expect(route.fetch).toHaveBeenCalledWith(
        expect.objectContaining({
          maxRedirects: 10,
        })
      );
    });

    it('should pass 301 redirect responses through without modification', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      const handler = routeHandlers.get('**/*');
      const { route } = createMockRoute(
        { resourceType: jest.fn().mockReturnValue('document') as any },
        {
          text: () => Promise.resolve(''),
          headers: () => ({ 'location': 'https://example.com/new-path' }),
          status: () => 301,
        }
      );

      await handler!(route);

      // Should fulfill with original response (no body modification)
      expect(route.fulfill).toHaveBeenCalledWith({ response: expect.anything() });

      // Body should NOT be modified (no script injection)
      const fulfillCall = (route.fulfill as jest.Mock).mock.calls[0][0];
      expect(fulfillCall.body).toBeUndefined();
    });

    it('should pass 302 redirect responses through without modification', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      const handler = routeHandlers.get('**/*');
      const { route } = createMockRoute(
        { resourceType: jest.fn().mockReturnValue('document') as any },
        {
          text: () => Promise.resolve(''),
          headers: () => ({ 'location': 'https://example.com/redirected' }),
          status: () => 302,
        }
      );

      await handler!(route);

      // Should fulfill with original response
      expect(route.fulfill).toHaveBeenCalledWith({ response: expect.anything() });
      const fulfillCall = (route.fulfill as jest.Mock).mock.calls[0][0];
      expect(fulfillCall.body).toBeUndefined();
    });

    it('should pass 307 redirect responses through without modification', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      const handler = routeHandlers.get('**/*');
      const { route } = createMockRoute(
        { resourceType: jest.fn().mockReturnValue('document') as any },
        {
          text: () => Promise.resolve(''),
          headers: () => ({ 'location': 'https://example.com/temp-redirect' }),
          status: () => 307,
        }
      );

      await handler!(route);

      // Should fulfill with original response
      expect(route.fulfill).toHaveBeenCalledWith({ response: expect.anything() });
      const fulfillCall = (route.fulfill as jest.Mock).mock.calls[0][0];
      expect(fulfillCall.body).toBeUndefined();
    });

    it('should track redirects as skipped in injection stats', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      const handler = routeHandlers.get('**/*');
      const { route } = createMockRoute(
        { resourceType: jest.fn().mockReturnValue('document') as any },
        {
          text: () => Promise.resolve(''),
          headers: () => ({ 'location': 'https://example.com/redirect' }),
          status: () => 301,
        }
      );

      await handler!(route);

      const stats = initializer.getInjectionStats();
      expect(stats.skipped).toBe(1);
      expect(stats.successful).toBe(0);
    });

    it('should still inject script into 200 OK responses', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      const handler = routeHandlers.get('**/*');
      const { route } = createMockRoute(
        { resourceType: jest.fn().mockReturnValue('document') as any },
        {
          text: () => Promise.resolve('<html><head></head><body>Success</body></html>'),
          headers: () => ({ 'content-type': 'text/html' }),
          status: () => 200,
        }
      );

      await handler!(route);

      // Should have injected script
      expect(route.fulfill).toHaveBeenCalled();
      const fulfillCall = (route.fulfill as jest.Mock).mock.calls[0][0];
      expect(fulfillCall.body).toContain('<script>');
    });

    it('should call route.fetch() with a timeout to prevent hanging', async () => {
      const { context, routeHandlers } = createMockContext();

      await initializer.initialize(context);

      const handler = routeHandlers.get('**/*');
      const { route } = createMockRoute(
        { resourceType: jest.fn().mockReturnValue('document') as any },
        {
          text: () => Promise.resolve('<html><head></head><body>Test</body></html>'),
          headers: () => ({ 'content-type': 'text/html' }),
          status: () => 200,
        }
      );

      await handler!(route);

      // Verify fetch was called with a timeout
      expect(route.fetch).toHaveBeenCalledWith(
        expect.objectContaining({
          timeout: expect.any(Number),
        })
      );

      // Timeout should be reasonable (> 10s to handle slow servers)
      const fetchCall = (route.fetch as jest.Mock).mock.calls[0][0];
      expect(fetchCall.timeout).toBeGreaterThanOrEqual(10000);
    });
  });

  describe('page event route setup (architectural note)', () => {
    /**
     * ARCHITECTURAL NOTE: Navigation listener setup was moved to pipeline-manager.ts
     *
     * With rebrowser-playwright, page.route() handlers do NOT persist across navigation.
     * After a page.goto() or link click, the event route is silently lost.
     *
     * The fix is implemented in pipeline-manager.ts (not here) which:
     * 1. Sets up 'load' event listeners on each page during recording
     * 2. Re-registers event routes after each navigation
     *
     * This separation allows:
     * - Context initialization to remain simple (one-time setup)
     * - Navigation listeners to only be active during recording
     * - No duplicate listeners when recording starts/stops multiple times
     *
     * @see pipeline-manager.ts - setupNavigationListeners() method
     */

    it('should set up page.route for event URL but NOT navigation listeners', async () => {
      const { page, pageListeners, pageRouteHandlers } = createMockPage();
      const mockContext = {
        route: jest.fn(),
        pages: jest.fn().mockReturnValue([page]),
        on: jest.fn(),
        off: jest.fn(),
      } as unknown as jest.Mocked<BrowserContext>;

      await initializer.initialize(mockContext);

      // Should have set up page.route for event URL
      expect(page.route).toHaveBeenCalled();
      const eventRoutePattern = Array.from(pageRouteHandlers.keys()).find(
        (k) => k.includes('__vrooli_recording_event__')
      );
      expect(eventRoutePattern).toBeDefined();

      // Should NOT set up 'load' listener (handled by pipeline-manager during recording)
      expect(pageListeners.has('load')).toBe(false);
    });

    it('should set up event route for new pages via context.on("page") but NOT navigation listeners', async () => {
      const contextPageListeners = new Map<string, Array<(...args: unknown[]) => void>>();
      const mockContext = {
        route: jest.fn(),
        pages: jest.fn().mockReturnValue([]),
        on: jest.fn().mockImplementation((event: string, handler: (...args: unknown[]) => void) => {
          const handlers = contextPageListeners.get(event) || [];
          handlers.push(handler);
          contextPageListeners.set(event, handlers);
        }),
        off: jest.fn(),
      } as unknown as jest.Mocked<BrowserContext>;

      await initializer.initialize(mockContext);

      // Simulate new page creation
      const { page, pageListeners, pageRouteHandlers } = createMockPage();
      const pageHandler = contextPageListeners.get('page')?.[0];
      expect(pageHandler).toBeDefined();

      await pageHandler!(page);

      // Should have set up page.route for event URL
      expect(page.route).toHaveBeenCalled();
      const eventRoutePattern = Array.from(pageRouteHandlers.keys()).find(
        (k) => k.includes('__vrooli_recording_event__')
      );
      expect(eventRoutePattern).toBeDefined();

      // Should NOT set up 'load' listener (handled by pipeline-manager during recording)
      expect(pageListeners.has('load')).toBe(false);
    });
  });

  describe('setupPageEventRoute', () => {
    it('should be idempotent by default (skip if already set up)', async () => {
      const { page, pageRouteHandlers } = createMockPage();
      const mockContext = {
        route: jest.fn(),
        pages: jest.fn().mockReturnValue([page]),
        on: jest.fn(),
        off: jest.fn(),
      } as unknown as jest.Mocked<BrowserContext>;

      await initializer.initialize(mockContext);

      const initialRouteCallCount = (page.route as jest.Mock).mock.calls.length;

      // Call setupPageEventRoute again (without force)
      await initializer.setupPageEventRoute(page, { force: false });

      // Should NOT have called page.route again
      expect((page.route as jest.Mock).mock.calls.length).toBe(initialRouteCallCount);
    });

    it('should re-register when force=true', async () => {
      const { page } = createMockPage();
      const mockContext = {
        route: jest.fn(),
        pages: jest.fn().mockReturnValue([page]),
        on: jest.fn(),
        off: jest.fn(),
      } as unknown as jest.Mocked<BrowserContext>;

      await initializer.initialize(mockContext);

      const initialRouteCallCount = (page.route as jest.Mock).mock.calls.length;

      // Call setupPageEventRoute with force=true
      await initializer.setupPageEventRoute(page, { force: true });

      // Should have called page.route again
      expect((page.route as jest.Mock).mock.calls.length).toBeGreaterThan(initialRouteCallCount);
    });
  });
});
