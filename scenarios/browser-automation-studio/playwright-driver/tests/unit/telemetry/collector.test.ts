import { ConsoleLogCollector, NetworkCollector } from '../../../src/telemetry/collector';
import { createMockPage } from '../../helpers';
import { EventEmitter } from 'node:events';
import type { Page, CDPSession, Request, Response } from 'rebrowser-playwright';

type Listener<T> = (arg: T) => void;

const findListener = <T>(
  page: ReturnType<typeof createMockPage>,
  event: string
): Listener<T> | undefined => {
  const calls = (page.on as jest.Mock).mock.calls as Array<[string, Listener<T>]>;
  return calls.find(([name]) => name === event)?.[1];
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

describe('ConsoleLogCollector native event ownership', () => {
  const fixture = (limit = 3) => {
    const session = Object.assign(new EventEmitter(), {
      send: jest.fn().mockResolvedValue({}), detach: jest.fn().mockResolvedValue(undefined),
    });
    const newCDPSession = jest.fn().mockResolvedValue(session);
    const page = { context: () => ({ newCDPSession }), isClosed: () => false } as unknown as Page;
    const collector = new ConsoleLogCollector(page, limit);
    const emit = (text: string, timestamp = Date.now() + 1, type = 'log') => session.emit('Runtime.consoleAPICalled', {
      type, timestamp, args: [{ value: text }], stackTrace: { callFrames: [{ url: 'https://fixture.test/script.js', lineNumber: 10, columnNumber: 5 }] },
    });
    return { session, newCDPSession, page, collector, emit };
  };

  it('retains only current bounded events with normalized text and location', async () => {
    const f = fixture();
    await f.collector.start();
    f.emit('old history', 0);
    for (let i = 0; i < 5; i++) f.emit(`message ${i}`, Date.now() + 1, i === 4 ? 'warning' : 'log');
    const logs = f.collector.getLogs();
    expect(logs.map(x => x.text)).toEqual(['message 2', 'message 3', 'message 4']);
    expect(logs[2]).toMatchObject({ type: 'warn', location: 'https://fixture.test/script.js:10:5' });
    expect(new Date(logs[2]!.timestamp).getTime()).toBeGreaterThan(0);
    logs.length = 0;
    expect(f.collector.getAndClear()).toHaveLength(3);
    expect(f.collector.getLogs()).toEqual([]);
    f.emit('clear me'); f.collector.clear(); expect(f.collector.getLogs()).toEqual([]);
    await f.collector.dispose();
    f.emit('too late'); expect(f.collector.getLogs()).toEqual([]);
    expect(f.session.listenerCount('Runtime.consoleAPICalled')).toBe(0);
    expect(f.session.detach).toHaveBeenCalledTimes(1);
  });

  it('keeps primitive and unavailable remote-object descriptions without evaluating them', async () => {
    const f = fixture(); await f.collector.start();
    f.session.emit('Runtime.consoleAPICalled', { type: 'error', timestamp: Date.now() + 1,
      args: [{ value: 'message' }, { value: 42 }, { value: null }, { unserializableValue: 'NaN' }, { type: 'object', description: 'Object' }],
    });
    expect(f.collector.getLogs()[0]).toMatchObject({ type: 'error', text: 'message 42 null NaN Object' });
    expect(f.session.send).toHaveBeenCalledTimes(1);
    await f.collector.dispose();
  });

  it('does not acknowledge readiness before Runtime.enable settles', async () => {
    const f = fixture(); let enable!: () => void;
    f.session.send.mockReturnValueOnce(new Promise<void>(resolve => { enable = resolve; }));
    let ready = false; const starting = f.collector.start().then(() => { ready = true; });
    await Promise.resolve(); await Promise.resolve(); expect(ready).toBe(false);
    enable(); await starting; expect(ready).toBe(true);
    await Promise.all([f.collector.dispose(), f.collector.dispose()]);
    expect(f.session.detach).toHaveBeenCalledTimes(1);
  });

  it('detaches a session acquired after disposal started without enabling it', async () => {
    const f = fixture(); let attach!: (session: CDPSession) => void;
    f.newCDPSession.mockReturnValueOnce(new Promise<CDPSession>(resolve => { attach = resolve; }));
    const starting = f.collector.start(); const closing = f.collector.dispose();
    attach(f.session as unknown as CDPSession);
    await Promise.all([starting, closing]);
    expect(f.session.send).not.toHaveBeenCalled();
    expect(f.session.detach).toHaveBeenCalledTimes(1);
    await expect(f.collector.start()).rejects.toThrow('disposed');
  });

  it('releases its attachment when native initialization fails', async () => {
    const f = fixture(); f.session.send.mockRejectedValueOnce(new Error('native initialization failed'));
    await expect(f.collector.start()).rejects.toThrow('native initialization failed');
    await f.collector.dispose();
    expect(f.session.detach).toHaveBeenCalledTimes(1);
    expect(f.session.listenerCount('Runtime.consoleAPICalled')).toBe(0);
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
