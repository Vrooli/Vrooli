import { ConsoleLogCollector, NetworkCollector } from '../../../src/telemetry/collector';
import { createMockPage } from '../../helpers';
import type { ConsoleMessage, Request, Response } from 'playwright';

describe('ConsoleLogCollector', () => {
  let mockPage: ReturnType<typeof createMockPage>;
  let collector: ConsoleLogCollector;

  beforeEach(() => {
    mockPage = createMockPage();
    collector = new ConsoleLogCollector(mockPage, 100);
  });

  describe('initialization', () => {
    it('should setup console listener', () => {
      expect(mockPage.on).toHaveBeenCalledWith('console', expect.any(Function));
    });
  });

  describe('log collection', () => {
    it('should collect console logs', () => {
      // Simulate console event
      const mockMessage = {
        type: () => 'log',
        text: () => 'Test message',
        location: () => ({ url: '', lineNumber: 0, columnNumber: 0 }),
      } as unknown as ConsoleMessage;

      const listener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'console')?.[1];
      listener(mockMessage);

      const logs = collector.getLogs();

      expect(logs).toHaveLength(1);
      expect(logs[0].type).toBe('log'); // Note: 'type' not 'level'
      expect(logs[0].text).toBe('Test message');
      expect(logs[0].timestamp).toBeDefined();
    });

    it('should collect multiple log types', () => {
      const messages = [
        { type: () => 'log', text: () => 'Log message', location: () => ({}) },
        { type: () => 'error', text: () => 'Error message', location: () => ({}) },
        { type: () => 'warning', text: () => 'Warning message', location: () => ({}) },
        { type: () => 'info', text: () => 'Info message', location: () => ({}) },
      ] as unknown as ConsoleMessage[];

      const listener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'console')?.[1];
      messages.forEach((msg) => listener(msg));

      const logs = collector.getLogs();

      expect(logs).toHaveLength(4);
      expect(logs[0].type).toBe('log');
      expect(logs[1].type).toBe('error');
      expect(logs[2].type).toBe('warn'); // 'warning' maps to 'warn'
      expect(logs[3].type).toBe('info');
    });

    it('should respect max entries limit', () => {
      // Create fresh mock page for this test to isolate listeners
      const freshMockPage = createMockPage();
      const smallCollector = new ConsoleLogCollector(freshMockPage, 3);

      const listener = (freshMockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'console')?.[1];

      // Add 5 messages (exceeds limit of 3)
      for (let i = 0; i < 5; i++) {
        listener({
          type: () => 'log',
          text: () => `Message ${i}`,
          location: () => ({}),
        } as unknown as ConsoleMessage);
      }

      const logs = smallCollector.getLogs();

      expect(logs).toHaveLength(3);
      expect(logs[0].text).toBe('Message 2'); // Oldest retained
      expect(logs[2].text).toBe('Message 4'); // Newest
    });

    it('should include timestamps', () => {
      const mockMessage = {
        type: () => 'log',
        text: () => 'Test message',
        location: () => ({}),
      } as unknown as ConsoleMessage;

      const before = new Date().toISOString();
      const listener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'console')?.[1];
      listener(mockMessage);
      const after = new Date().toISOString();

      const logs = collector.getLogs();

      expect(logs[0].timestamp).toBeDefined();
      expect(logs[0].timestamp >= before).toBe(true);
      expect(logs[0].timestamp <= after).toBe(true);
    });

    it('should include location when available', () => {
      const mockMessage = {
        type: () => 'log',
        text: () => 'Test message',
        location: () => ({ url: 'https://example.com/script.js', lineNumber: 10, columnNumber: 5 }),
      } as unknown as ConsoleMessage;

      const listener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'console')?.[1];
      listener(mockMessage);

      const logs = collector.getLogs();

      expect(logs[0].location).toBe('https://example.com/script.js:10:5');
    });
  });

  describe('clear', () => {
    it('should clear all logs', () => {
      const mockMessage = {
        type: () => 'log',
        text: () => 'Test message',
        location: () => ({}),
      } as unknown as ConsoleMessage;

      const listener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'console')?.[1];
      listener(mockMessage);
      listener(mockMessage);

      collector.clear();

      const logs = collector.getLogs();
      expect(logs).toHaveLength(0);
    });
  });

  describe('getAndClear', () => {
    it('should return logs and clear', () => {
      const mockMessage = {
        type: () => 'log',
        text: () => 'Test message',
        location: () => ({}),
      } as unknown as ConsoleMessage;

      const listener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'console')?.[1];
      listener(mockMessage);

      const logs = collector.getAndClear();

      expect(logs).toHaveLength(1);
      expect(collector.getLogs()).toHaveLength(0);
    });
  });
});

describe('NetworkCollector', () => {
  let mockPage: ReturnType<typeof createMockPage>;
  let collector: NetworkCollector;

  beforeEach(() => {
    mockPage = createMockPage();
    collector = new NetworkCollector(mockPage, 100);
  });

  describe('initialization', () => {
    it('should setup request listener', () => {
      expect(mockPage.on).toHaveBeenCalledWith('request', expect.any(Function));
    });

    it('should setup response listener', () => {
      expect(mockPage.on).toHaveBeenCalledWith('response', expect.any(Function));
    });

    it('should setup request failed listener', () => {
      expect(mockPage.on).toHaveBeenCalledWith('requestfailed', expect.any(Function));
    });
  });

  describe('event collection', () => {
    it('should collect response events after request', () => {
      // First trigger a request
      const requestListener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'request')?.[1];
      const responseListener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'response')?.[1];

      const mockRequest = {
        url: () => 'https://example.com/api',
        method: () => 'GET',
        resourceType: () => 'xhr',
      } as unknown as Request;

      requestListener(mockRequest);

      // Then trigger the response
      const mockResponse = {
        url: () => 'https://example.com/api',
        status: () => 200,
        ok: () => true,
        request: () => mockRequest,
      } as unknown as Response;

      responseListener(mockResponse);

      const events = collector.getEvents();

      expect(events).toHaveLength(1);
      expect(events[0].type).toBe('response');
      expect(events[0].url).toBe('https://example.com/api');
      expect(events[0].status).toBe(200);
      expect(events[0].ok).toBe(true);
    });

    it('should collect request failure events', () => {
      const requestListener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'request')?.[1];
      const failedListener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'requestfailed')?.[1];

      const mockRequest = {
        url: () => 'https://example.com/api',
        method: () => 'GET',
        resourceType: () => 'xhr',
        failure: () => ({ errorText: 'net::ERR_CONNECTION_REFUSED' }),
      } as unknown as Request;

      requestListener(mockRequest);
      failedListener(mockRequest);

      const events = collector.getEvents();

      expect(events).toHaveLength(1);
      expect(events[0].type).toBe('failure'); // Note: 'failure' not 'failed'
      expect(events[0].url).toBe('https://example.com/api');
      expect(events[0].failure).toBe('net::ERR_CONNECTION_REFUSED'); // Note: 'failure' not 'error'
    });

    it('should respect max events limit', () => {
      // Create fresh mock page for this test to isolate listeners
      const freshMockPage = createMockPage();
      const smallCollector = new NetworkCollector(freshMockPage, 3);

      const requestListener = (freshMockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'request')?.[1];
      const responseListener = (freshMockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'response')?.[1];

      // Add 5 request/response pairs (exceeds limit of 3)
      for (let i = 0; i < 5; i++) {
        const mockRequest = {
          url: () => `https://example.com/api/${i}`,
          method: () => 'GET',
          resourceType: () => 'xhr',
        } as unknown as Request;

        requestListener(mockRequest);

        const mockResponse = {
          url: () => `https://example.com/api/${i}`,
          status: () => 200,
          ok: () => true,
          request: () => mockRequest,
        } as unknown as Response;

        responseListener(mockResponse);
      }

      const events = smallCollector.getEvents();

      expect(events).toHaveLength(3);
      expect(events[0].url).toBe('https://example.com/api/2'); // Oldest retained
      expect(events[2].url).toBe('https://example.com/api/4'); // Newest
    });

    it('should include timestamps from request time', () => {
      const requestListener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'request')?.[1];
      const responseListener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'response')?.[1];

      const mockRequest = {
        url: () => 'https://example.com/api',
        method: () => 'GET',
        resourceType: () => 'xhr',
      } as unknown as Request;

      const before = new Date().toISOString();
      requestListener(mockRequest);
      const after = new Date().toISOString();

      const mockResponse = {
        url: () => 'https://example.com/api',
        status: () => 200,
        ok: () => true,
        request: () => mockRequest,
      } as unknown as Response;

      responseListener(mockResponse);

      const events = collector.getEvents();

      expect(events[0].timestamp).toBeDefined();
      expect(events[0].timestamp >= before).toBe(true);
      expect(events[0].timestamp <= after).toBe(true);
    });
  });

  describe('clear', () => {
    it('should clear all events', () => {
      const requestListener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'request')?.[1];
      const responseListener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'response')?.[1];

      const mockRequest = {
        url: () => 'https://example.com/api',
        method: () => 'GET',
        resourceType: () => 'xhr',
      } as unknown as Request;

      requestListener(mockRequest);

      const mockResponse = {
        url: () => 'https://example.com/api',
        status: () => 200,
        ok: () => true,
        request: () => mockRequest,
      } as unknown as Response;

      responseListener(mockResponse);

      collector.clear();

      const events = collector.getEvents();
      expect(events).toHaveLength(0);
    });
  });

  describe('getAndClear', () => {
    it('should return events and clear', () => {
      const requestListener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'request')?.[1];
      const responseListener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'response')?.[1];

      const mockRequest = {
        url: () => 'https://example.com/api',
        method: () => 'GET',
        resourceType: () => 'xhr',
      } as unknown as Request;

      requestListener(mockRequest);

      const mockResponse = {
        url: () => 'https://example.com/api',
        status: () => 200,
        ok: () => true,
        request: () => mockRequest,
      } as unknown as Response;

      responseListener(mockResponse);

      const events = collector.getAndClear();

      expect(events).toHaveLength(1);
      expect(collector.getEvents()).toHaveLength(0);
    });
  });
});
