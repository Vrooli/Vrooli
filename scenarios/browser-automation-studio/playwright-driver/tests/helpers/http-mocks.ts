import type { IncomingMessage, ServerResponse } from 'http';
import { EventEmitter } from 'events';

/**
 * Mock IncomingMessage (HTTP Request)
 */
type MockRequest = jest.Mocked<IncomingMessage> & {
  destroyed: boolean;
  destroy: jest.Mock<MockRequest, [Error?]>;
};

type MockResponse = jest.Mocked<ServerResponse> & {
  getBody: () => string;
  getJSON: () => Record<string, unknown>;
};

export function createMockRequest(options?: {
  method?: string;
  url?: string;
  headers?: Record<string, string>;
  body?: unknown;
  /** Delay before emitting body (ms). Used to test streaming behavior. */
  bodyDelay?: number;
}): MockRequest {
  const req = new EventEmitter() as jest.Mocked<IncomingMessage>;
  const mockReq = req as MockRequest;

  mockReq.method = options?.method || 'GET';
  mockReq.url = options?.url || '/';
  mockReq.headers = options?.headers || {};
  // Add destroyed property to match IncomingMessage interface
  mockReq.destroyed = false;
  // Add destroy method to match IncomingMessage interface (required by hardened body-parser)
  mockReq.destroy = jest.fn((error?: Error) => {
    mockReq.destroyed = true;
    mockReq.emit('close', error);
    return mockReq;
  });

  const delay = options?.bodyDelay ?? 0;

  // Simulate body streaming
  if (options?.body) {
    setTimeout(() => {
      if (mockReq.destroyed) return;
      const bodyStr = typeof options.body === 'string' ? options.body : JSON.stringify(options.body);
      mockReq.emit('data', Buffer.from(bodyStr));
      mockReq.emit('end');
    }, delay);
  } else {
    setTimeout(() => {
      if (mockReq.destroyed) return;
      mockReq.emit('end');
    }, delay);
  }

  return mockReq;
}

/**
 * Mock ServerResponse (HTTP Response)
 */
export function createMockResponse(): MockResponse {
  const res = new EventEmitter() as jest.Mocked<ServerResponse>;
  const mockRes = res as MockResponse;

  mockRes.statusCode = 200;
  mockRes.statusMessage = 'OK';
  mockRes.setHeader = jest.fn();
  mockRes.getHeader = jest.fn();
  mockRes.removeHeader = jest.fn();
  mockRes.writeHead = jest.fn();
  type WriteArgs = [chunk: unknown, encoding?: BufferEncoding, cb?: (error?: Error | null) => void];
  const writeCalls: WriteArgs[] = [];
  mockRes.write = ((chunk: unknown, encoding?: BufferEncoding, cb?: (error?: Error | null) => void) => {
    writeCalls.push([chunk, encoding, cb]);
    return true;
  }) as ServerResponse['write'];

  // ServerResponse.end has complex overloads - use Object.defineProperty to bypass TypeScript
  const endMock = jest.fn((data?: unknown) => {
    if (data) {
      mockRes.write(data);
    }
    mockRes.emit('finish');
    return mockRes;
  });
  Object.defineProperty(mockRes, 'end', {
    value: endMock,
    writable: true,
    configurable: true,
  });

  // Helper to get response body
  mockRes.getBody = (): string => {
    return writeCalls.map((call) => String(call[0] ?? '')).join('');
  };

  // Helper to get response JSON
  mockRes.getJSON = (): Record<string, unknown> => {
    const body = mockRes.getBody();
    if (!body) {
      return {};
    }
    const parsed: unknown = JSON.parse(body);
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
    throw new Error('Expected JSON object response');
  };

  return mockRes;
}

/**
 * Wait for response to finish
 */
export function waitForResponse(res: ServerResponse): Promise<void> {
  return new Promise((resolve) => {
    res.on('finish', () => resolve());
  });
}
