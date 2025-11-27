import { ConsoleLogCollector, NetworkCollector } from '../../../src/telemetry/collector';
import { createMockPage, createMockRequest, createMockResponse } from '../../helpers';
import type { ConsoleMessage } from 'playwright';

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
      } as ConsoleMessage;

      const listener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'console')?.[1];
      listener(mockMessage);

      const logs = collector.getLogs();

      expect(logs).toHaveLength(1);
      expect(logs[0].level).toBe('log');
      expect(logs[0].text).toBe('Test message');
      expect(logs[0].timestamp).toBeDefined();
    });

    it('should collect multiple log levels', () => {
      const messages = [
        { type: () => 'log', text: () => 'Log message' },
        { type: () => 'error', text: () => 'Error message' },
        { type: () => 'warning', text: () => 'Warning message' },
        { type: () => 'info', text: () => 'Info message' },
      ] as ConsoleMessage[];

      const listener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'console')?.[1];
      messages.forEach((msg) => listener(msg));

      const logs = collector.getLogs();

      expect(logs).toHaveLength(4);
      expect(logs[0].level).toBe('log');
      expect(logs[1].level).toBe('error');
      expect(logs[2].level).toBe('warning');
      expect(logs[3].level).toBe('info');
    });

    it('should respect max entries limit', () => {
      const smallCollector = new ConsoleLogCollector(mockPage, 3);

      const listener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'console')?.[1];

      // Add 5 messages (exceeds limit of 3)
      for (let i = 0; i < 5; i++) {
        listener({ type: () => 'log', text: () => `Message ${i}` } as ConsoleMessage);
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
      } as ConsoleMessage;

      const before = new Date().toISOString();
      const listener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'console')?.[1];
      listener(mockMessage);
      const after = new Date().toISOString();

      const logs = collector.getLogs();

      expect(logs[0].timestamp).toBeDefined();
      expect(logs[0].timestamp >= before).toBe(true);
      expect(logs[0].timestamp <= after).toBe(true);
    });
  });

  describe('reset', () => {
    it('should clear all logs', () => {
      const mockMessage = {
        type: () => 'log',
        text: () => 'Test message',
      } as ConsoleMessage;

      const listener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'console')?.[1];
      listener(mockMessage);
      listener(mockMessage);

      collector.reset();

      const logs = collector.getLogs();
      expect(logs).toHaveLength(0);
    });
  });

  describe('cleanup', () => {
    it('should remove listener', () => {
      collector.cleanup();

      expect(mockPage.removeListener).toHaveBeenCalledWith('console', expect.any(Function));
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
    it('should collect request events', () => {
      const mockRequest = createMockRequest({
        url: () => 'https://example.com/api',
        method: () => 'GET',
        headers: () => ({ 'User-Agent': 'test' }),
        resourceType: () => 'xhr',
      });

      const listener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'request')?.[1];
      listener(mockRequest);

      const events = collector.getEvents();

      expect(events).toHaveLength(1);
      expect(events[0].type).toBe('request');
      expect(events[0].url).toBe('https://example.com/api');
      expect(events[0].method).toBe('GET');
      expect(events[0].resource_type).toBe('xhr');
    });

    it('should collect response events', () => {
      const mockRequest = createMockRequest({
        url: () => 'https://example.com/api',
        method: () => 'GET',
      });

      const mockResponse = createMockResponse({
        status: () => 200,
        statusText: () => 'OK',
        url: () => 'https://example.com/api',
        request: () => mockRequest,
      } as any);

      const listener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'response')?.[1];
      listener(mockResponse);

      const events = collector.getEvents();

      expect(events).toHaveLength(1);
      expect(events[0].type).toBe('response');
      expect(events[0].url).toBe('https://example.com/api');
      expect(events[0].status).toBe(200);
    });

    it('should collect request failure events', () => {
      const mockRequest = createMockRequest({
        url: () => 'https://example.com/api',
        method: () => 'GET',
        failure: () => ({ errorText: 'net::ERR_CONNECTION_REFUSED' }),
      } as any);

      const listener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'requestfailed')?.[1];
      listener(mockRequest);

      const events = collector.getEvents();

      expect(events).toHaveLength(1);
      expect(events[0].type).toBe('failed');
      expect(events[0].url).toBe('https://example.com/api');
      expect(events[0].error).toBe('net::ERR_CONNECTION_REFUSED');
    });

    it('should respect max entries limit', () => {
      const smallCollector = new NetworkCollector(mockPage, 3);

      const requestListener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'request')?.[1];

      // Add 5 requests (exceeds limit of 3)
      for (let i = 0; i < 5; i++) {
        const req = createMockRequest({
          url: () => `https://example.com/api/${i}`,
          method: () => 'GET',
        });
        requestListener(req);
      }

      const events = smallCollector.getEvents();

      expect(events).toHaveLength(3);
      expect(events[0].url).toBe('https://example.com/api/2'); // Oldest retained
      expect(events[2].url).toBe('https://example.com/api/4'); // Newest
    });

    it('should include timestamps', () => {
      const mockRequest = createMockRequest({
        url: () => 'https://example.com/api',
        method: () => 'GET',
      });

      const before = new Date().toISOString();
      const listener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'request')?.[1];
      listener(mockRequest);
      const after = new Date().toISOString();

      const events = collector.getEvents();

      expect(events[0].timestamp).toBeDefined();
      expect(events[0].timestamp >= before).toBe(true);
      expect(events[0].timestamp <= after).toBe(true);
    });
  });

  describe('reset', () => {
    it('should clear all events', () => {
      const mockRequest = createMockRequest({
        url: () => 'https://example.com/api',
        method: () => 'GET',
      });

      const listener = (mockPage.on as jest.Mock).mock.calls.find((call) => call[0] === 'request')?.[1];
      listener(mockRequest);
      listener(mockRequest);

      collector.reset();

      const events = collector.getEvents();
      expect(events).toHaveLength(0);
    });
  });

  describe('cleanup', () => {
    it('should remove all listeners', () => {
      collector.cleanup();

      expect(mockPage.removeListener).toHaveBeenCalledWith('request', expect.any(Function));
      expect(mockPage.removeListener).toHaveBeenCalledWith('response', expect.any(Function));
      expect(mockPage.removeListener).toHaveBeenCalledWith('requestfailed', expect.any(Function));
    });
  });
});
