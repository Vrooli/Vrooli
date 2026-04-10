import { ConsoleLogCollector, NetworkCollector } from '../../../src/telemetry/collector';
import { createMockPage } from '../../helpers';
import type { ConsoleMessage, Request, Response } from 'rebrowser-playwright';

type Listener<T> = (arg: T) => void;

const findListener = <T>(
  page: ReturnType<typeof createMockPage>,
  event: string
): Listener<T> | undefined => {
  const calls = (page.on as jest.Mock).mock.calls as Array<[string, Listener<T>]>;
  return calls.find(([name]) => name === event)?.[1];
};

const createConsoleMessage = (params: {
  type?: string;
  text?: string;
  location?: { url?: string; lineNumber?: number; columnNumber?: number };
} = {}): ConsoleMessage => {
  const { type = 'log', text = 'Test message', location = {} } = params;
  return {
    type: (): string => type,
    text: (): string => text,
    location: (): { url?: string; lineNumber?: number; columnNumber?: number } => location,
  } as unknown as ConsoleMessage;
};

const createRequest = (params: {
  url?: string;
  method?: string;
  resourceType?: string;
  failure?: { errorText: string } | null;
} = {}): Request => {
  const {
    url = 'https://example.com/api',
    method = 'GET',
    resourceType = 'xhr',
    failure = null,
  } = params;
  return {
    url: (): string => url,
    method: (): string => method,
    resourceType: (): string => resourceType,
    failure: (): { errorText: string } | null => failure,
  } as unknown as Request;
};

const createResponse = (params: {
  url?: string;
  status?: number;
  ok?: boolean;
  request: Request;
}): Response => {
  const {
    url = 'https://example.com/api',
    status = 200,
    ok = true,
    request,
  } = params;
  return {
    url: (): string => url,
    status: (): number => status,
    ok: (): boolean => ok,
    request: (): Request => request,
  } as unknown as Response;
};

describe('ConsoleLogCollector', () => {
  let mockPage: ReturnType<typeof createMockPage>;
  let collector: ConsoleLogCollector;

  beforeEach(() => {
    mockPage = createMockPage();
    collector = new ConsoleLogCollector(mockPage, 100);
  });

  describe('initialization', () => {
    it('should setup console listener', () => {
      const consoleListener = findListener<ConsoleMessage>(mockPage, 'console');
      expect(consoleListener).toEqual(expect.any(Function));
    });
  });

  describe('log collection', () => {
    it('should collect console logs', () => {
      // Simulate console event
      const mockMessage = createConsoleMessage({
        type: 'log',
        text: 'Test message',
        location: { url: '', lineNumber: 0, columnNumber: 0 },
      });

      const listener = findListener<ConsoleMessage>(mockPage, 'console');
      if (!listener) {
        throw new Error('Console listener not registered');
      }
      listener(mockMessage);

      const logs = collector.getLogs();

      expect(logs).toHaveLength(1);
      const [firstLog] = logs;
      if (!firstLog) {
        throw new Error('Expected a console log entry');
      }
      expect(firstLog.type).toBe('log'); // Note: 'type' not 'level'
      expect(firstLog.text).toBe('Test message');
      expect(firstLog.timestamp).toBeDefined();
    });

    it('should collect multiple log types', () => {
      const messages = [
        createConsoleMessage({ type: 'log', text: 'Log message' }),
        createConsoleMessage({ type: 'error', text: 'Error message' }),
        createConsoleMessage({ type: 'warning', text: 'Warning message' }),
        createConsoleMessage({ type: 'info', text: 'Info message' }),
      ];

      const listener = findListener<ConsoleMessage>(mockPage, 'console');
      if (!listener) {
        throw new Error('Console listener not registered');
      }
      messages.forEach((msg) => listener(msg));

      const logs = collector.getLogs();

      expect(logs).toHaveLength(4);
      const [firstLog, secondLog, thirdLog, fourthLog] = logs;
      if (!firstLog || !secondLog || !thirdLog || !fourthLog) {
        throw new Error('Expected four console log entries');
      }
      expect(firstLog.type).toBe('log');
      expect(secondLog.type).toBe('error');
      expect(thirdLog.type).toBe('warn'); // 'warning' maps to 'warn'
      expect(fourthLog.type).toBe('info');
    });

    it('should respect max entries limit', () => {
      // Create fresh mock page for this test to isolate listeners
      const freshMockPage = createMockPage();
      const smallCollector = new ConsoleLogCollector(freshMockPage, 3);

      const listener = findListener<ConsoleMessage>(freshMockPage, 'console');
      if (!listener) {
        throw new Error('Console listener not registered');
      }

      // Add 5 messages (exceeds limit of 3)
      for (let i = 0; i < 5; i++) {
        listener(createConsoleMessage({ type: 'log', text: `Message ${i}` }));
      }

      const logs = smallCollector.getLogs();

      expect(logs).toHaveLength(3);
      const [firstLog, , thirdLog] = logs;
      if (!firstLog || !thirdLog) {
        throw new Error('Expected three console log entries');
      }
      expect(firstLog.text).toBe('Message 2'); // Oldest retained
      expect(thirdLog.text).toBe('Message 4'); // Newest
    });

    it('should include timestamps', () => {
      const mockMessage = {
        ...createConsoleMessage({ type: 'log', text: 'Test message' }),
      };

      const before = new Date().toISOString();
      const listener = findListener<ConsoleMessage>(mockPage, 'console');
      if (!listener) {
        throw new Error('Console listener not registered');
      }
      listener(mockMessage);
      const after = new Date().toISOString();

      const logs = collector.getLogs();

      const [firstLog] = logs;
      if (!firstLog) {
        throw new Error('Expected a console log entry');
      }
      expect(firstLog.timestamp).toBeDefined();
      expect(firstLog.timestamp >= before).toBe(true);
      expect(firstLog.timestamp <= after).toBe(true);
    });

    it('should include location when available', () => {
      const mockMessage = createConsoleMessage({
        type: 'log',
        text: 'Test message',
        location: { url: 'https://example.com/script.js', lineNumber: 10, columnNumber: 5 },
      });

      const listener = findListener<ConsoleMessage>(mockPage, 'console');
      if (!listener) {
        throw new Error('Console listener not registered');
      }
      listener(mockMessage);

      const logs = collector.getLogs();

      const [firstLog] = logs;
      if (!firstLog) {
        throw new Error('Expected a console log entry');
      }
      expect(firstLog.location).toBe('https://example.com/script.js:10:5');
    });
  });

  describe('clear', () => {
    it('should clear all logs', () => {
      const mockMessage = createConsoleMessage({ type: 'log', text: 'Test message' });

      const listener = findListener<ConsoleMessage>(mockPage, 'console');
      if (!listener) {
        throw new Error('Console listener not registered');
      }
      listener(mockMessage);
      listener(mockMessage);

      collector.clear();

      const logs = collector.getLogs();
      expect(logs).toHaveLength(0);
    });
  });

  describe('getAndClear', () => {
    it('should return logs and clear', () => {
      const mockMessage = createConsoleMessage({ type: 'log', text: 'Test message' });

      const listener = findListener<ConsoleMessage>(mockPage, 'console');
      if (!listener) {
        throw new Error('Console listener not registered');
      }
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
      const requestListener = findListener<Request>(mockPage, 'request');
      expect(requestListener).toEqual(expect.any(Function));
    });

    it('should setup response listener', () => {
      const responseListener = findListener<Response>(mockPage, 'response');
      expect(responseListener).toEqual(expect.any(Function));
    });

    it('should setup request failed listener', () => {
      const requestFailedListener = findListener<Request>(mockPage, 'requestfailed');
      expect(requestFailedListener).toEqual(expect.any(Function));
    });
  });

  describe('event collection', () => {
    it('should collect response events after request', () => {
      // First trigger a request
      const requestListener = findListener<Request>(mockPage, 'request');
      const responseListener = findListener<Response>(mockPage, 'response');
      if (!requestListener || !responseListener) {
        throw new Error('Request/response listeners not registered');
      }

      const mockRequest = createRequest();

      requestListener(mockRequest);

      // Then trigger the response
      const mockResponse = createResponse({ request: mockRequest });

      responseListener(mockResponse);

      const events = collector.getEvents();

      expect(events).toHaveLength(1);
      const [firstEvent] = events;
      if (!firstEvent) {
        throw new Error('Expected a response event');
      }
      expect(firstEvent.type).toBe('response');
      expect(firstEvent.url).toBe('https://example.com/api');
      expect(firstEvent.status).toBe(200);
      expect(firstEvent.ok).toBe(true);
    });

    it('should collect request failure events', () => {
      const requestListener = findListener<Request>(mockPage, 'request');
      const failedListener = findListener<Request>(mockPage, 'requestfailed');
      if (!requestListener || !failedListener) {
        throw new Error('Request failure listeners not registered');
      }

      const mockRequest = createRequest({
        failure: { errorText: 'net::ERR_CONNECTION_REFUSED' },
      });

      requestListener(mockRequest);
      failedListener(mockRequest);

      const events = collector.getEvents();

      expect(events).toHaveLength(1);
      const [firstEvent] = events;
      if (!firstEvent) {
        throw new Error('Expected a failure event');
      }
      expect(firstEvent.type).toBe('failure'); // Note: 'failure' not 'failed'
      expect(firstEvent.url).toBe('https://example.com/api');
      expect(firstEvent.failure).toBe('net::ERR_CONNECTION_REFUSED'); // Note: 'failure' not 'error'
    });

    it('should respect max events limit', () => {
      // Create fresh mock page for this test to isolate listeners
      const freshMockPage = createMockPage();
      const smallCollector = new NetworkCollector(freshMockPage, 3);

      const requestListener = findListener<Request>(freshMockPage, 'request');
      const responseListener = findListener<Response>(freshMockPage, 'response');
      if (!requestListener || !responseListener) {
        throw new Error('Request/response listeners not registered');
      }

      // Add 5 request/response pairs (exceeds limit of 3)
      for (let i = 0; i < 5; i++) {
        const mockRequest = createRequest({ url: `https://example.com/api/${i}` });

        requestListener(mockRequest);

        const mockResponse = createResponse({ url: `https://example.com/api/${i}`, request: mockRequest });

        responseListener(mockResponse);
      }

      const events = smallCollector.getEvents();

      expect(events).toHaveLength(3);
      const [firstEvent, , thirdEvent] = events;
      if (!firstEvent || !thirdEvent) {
        throw new Error('Expected three events after trimming');
      }
      expect(firstEvent.url).toBe('https://example.com/api/2'); // Oldest retained
      expect(thirdEvent.url).toBe('https://example.com/api/4'); // Newest
    });

    it('should include timestamps from request time', () => {
      const requestListener = findListener<Request>(mockPage, 'request');
      const responseListener = findListener<Response>(mockPage, 'response');
      if (!requestListener || !responseListener) {
        throw new Error('Request/response listeners not registered');
      }

      const mockRequest = createRequest();

      const before = new Date().toISOString();
      requestListener(mockRequest);
      const after = new Date().toISOString();

      const mockResponse = createResponse({ request: mockRequest });

      responseListener(mockResponse);

      const events = collector.getEvents();

      const [firstEvent] = events;
      if (!firstEvent) {
        throw new Error('Expected a network event');
      }
      expect(firstEvent.timestamp).toBeDefined();
      expect(firstEvent.timestamp >= before).toBe(true);
      expect(firstEvent.timestamp <= after).toBe(true);
    });
  });

  describe('clear', () => {
    it('should clear all events', () => {
      const requestListener = findListener<Request>(mockPage, 'request');
      const responseListener = findListener<Response>(mockPage, 'response');
      if (!requestListener || !responseListener) {
        throw new Error('Request/response listeners not registered');
      }

      const mockRequest = createRequest();

      requestListener(mockRequest);

      const mockResponse = createResponse({ request: mockRequest });

      responseListener(mockResponse);

      collector.clear();

      const events = collector.getEvents();
      expect(events).toHaveLength(0);
    });
  });

  describe('getAndClear', () => {
    it('should return events and clear', () => {
      const requestListener = findListener<Request>(mockPage, 'request');
      const responseListener = findListener<Response>(mockPage, 'response');
      if (!requestListener || !responseListener) {
        throw new Error('Request/response listeners not registered');
      }

      const mockRequest = createRequest();

      requestListener(mockRequest);

      const mockResponse = createResponse({ request: mockRequest });

      responseListener(mockResponse);

      const events = collector.getAndClear();

      expect(events).toHaveLength(1);
      expect(collector.getEvents()).toHaveLength(0);
    });
  });
});
